package http

import (
	"context"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"net/http"
	_ "net/http/pprof"
	"path"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/securecookie"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/saiddis/todev"
	"golang.org/x/crypto/acme/autocert"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
)

// Generic HTTP metrics.
var (
	requestCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "todev_http_request_count",
		Help: "Total number of request by route",
	}, []string{"method", "path"})

	requestSeconds = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "todev_http_request_seconds",
		Help: "Total amount of request time by route, in seconds",
	}, []string{"method", "path"})
)

const ShutdownTimeout = 1 * time.Second

// Server represents an HTTP server. It is meant to wrap all HTTP functionality
// used by the application.
type Server struct {
	ln     net.Listener
	server *http.Server
	router *mux.Router
	sc     *securecookie.SecureCookie

	// Bind address and domain for the server's listeners. If domain is
	// specified, server is run on TLS using acme/autocert.
	Addr   string
	Domain string

	// Keys used for secure cookie encryption.
	HashKey  string
	BlockKey string

	// GitHub OAuth settings.
	GitHubClientID     string
	GitHubClientSecret string

	// Services used by the various HTTP routes.
	AuthService        todev.AuthService
	RepoService        todev.RepoService
	ContributorService todev.ContributorService
	TaskService        todev.TaskService
	UserService        todev.UserService
	EventService       todev.EventService
}

// NewServer returns a new instance of server.
func NewServer() *Server {
	// Create a new server that wraps the net/http server and adds gorilla router.
	s := &Server{
		server: &http.Server{},
		router: mux.NewRouter(),
	}

	// report panics to external serveces.
	s.router.Use(reportPanic)

	// Our router is wrapper by another fuction handler to perform some
	// middleware-like tasks that cannot be performed by actual middleware.
	// This includes changing route paths for JSON endpoints and overriding methods.
	s.server.Handler = http.HandlerFunc(s.serveHTTP)

	// Setup error handling routes.
	s.router.NotFoundHandler = http.HandlerFunc(s.handleNotFound)

	// Setup endpoint to display deployed version.
	s.router.HandleFunc("/debug/version", s.handleVersion).Methods("GET")
	s.router.HandleFunc("/debug/commint", s.handleCommit).Methods("GET")

	// Setup a base router that excludes asset handling.
	router := s.router.PathPrefix("/").Subrouter()
	router.Use(s.authenticate)
	router.Use(s.loadFlash)
	router.Use(trackMetrics)

	// Handle authentication check within handler funciton for home page.
	router.HandleFunc("/", s.handleIndex).Methods("GET")

	// Registers unauthenticated routes.
	{
		r := s.router.PathPrefix("/").Subrouter()
		r.Use(s.requireNoAuth)
		s.registerAuthRoutes(r)
	}

	// Register authenticated routes.
	{
		r := router.PathPrefix("/").Subrouter()
		r.Use(s.requireAuth)
		s.registerRepoRoutes(r)
		s.registerContributorRoutes(r)
		s.registerTaskRoutes(r)
		s.registerEventRoutes(r)
	}

	return s
}

// UseTLS returns true if the cert and key file are specified.
func (s *Server) UseTLS() bool {
	return s.Domain != ""
}

// Scheme returns the URl scheme for the server.
func (s *Server) Scheme() string {
	if s.UseTLS() {
		return "https"
	}
	return "http"
}

// Port returns the TCP port for the running server.
// This is useful in tests where we allocate a random port by using ":0".
func (s *Server) Port() int {
	if s.ln == nil {
		return 0
	}
	return s.ln.Addr().(*net.TCPAddr).Port
}

// URL returns the local base URL for the running server.
func (s *Server) URL() string {
	scheme, port := s.Scheme(), s.Port()

	domain := "localhost"
	if s.Domain != "" {
		domain = s.Domain
	}

	if (scheme == "http" && port == 80) || (scheme == "https" && port == 443) {
		return fmt.Sprintf("%s://%s", s.Scheme(), domain)
	}

	return fmt.Sprintf("%s://%s:%d", s.Scheme(), domain, s.Port())
}

// Open validates the server options and begins listening on the bind address.
func (s *Server) Open() (err error) {

	if err = s.openSecureCookie(); err != nil {
		return err
	}

	if s.GitHubClientID == "" {
		return fmt.Errorf("github client id required")
	} else if s.GitHubClientSecret == "" {
		return fmt.Errorf("github client secret required")
	}

	if s.Domain != "" {
		s.ln = autocert.NewListener(s.Domain)
	} else {
		if s.ln, err = net.Listen("tcp", s.Addr); err != nil {
			return err
		}
	}

	go s.server.Serve(s.ln)
	return nil
}

// openSecureCookie validates and decodes the block and hash key and initializes
// our secure cookie implementation.
func (s *Server) openSecureCookie() error {
	if s.HashKey == "" {
		return fmt.Errorf("hash key required")
	} else if s.BlockKey == "" {
		return fmt.Errorf("block key required")
	}

	hashKey, err := hex.DecodeString(s.HashKey)
	if err != nil {
		return fmt.Errorf("invalid hash key")
	}
	blockKey, err := hex.DecodeString(s.BlockKey)
	if err != nil {
		return fmt.Errorf("invalid block key")
	}

	s.sc = securecookie.New(hashKey, blockKey)
	s.sc.SetSerializer(securecookie.JSONEncoder{})

	return nil
}

// Close gracefully shuts down the server.
func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// OAuth2Config returns the GitHub OAuth2 configuration.
func (s *Server) OAuth2Config() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.GitHubClientID,
		ClientSecret: s.GitHubClientSecret,
		Scopes:       []string{},
		Endpoint:     github.Endpoint,
	}
}

func (s *Server) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		switch v := r.PostFormValue("_method"); v {
		case http.MethodGet, http.MethodPost, http.MethodPatch, http.MethodDelete:
			r.Method = v
		}
	}

	switch ext := path.Ext(r.URL.Path); ext {
	case ".json":
		r.Header.Set("Accept", "application/json")
		r.Header.Set("Content-type", "application/json")
		r.URL.Path = strings.TrimSuffix(r.URL.Path, ext)
	case ".csv":
		r.Header.Set("Accept", "text/csv")
		r.URL.Path = strings.TrimSuffix(r.URL.Path, ext)
	}

	s.router.ServeHTTP(w, r)
}

// authenticate is middleware for loading session data from a cookie or API key header.
func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if v := r.Header.Get("Authorization"); strings.HasPrefix(v, "Bearer ") {
			apiKey := strings.TrimPrefix(v, "Bearer ")

			users, _, err := s.UserService.FindUsers(r.Context(), todev.UserFilter{APIKey: &apiKey})
			if err != nil {
				Error(w, r, err)
				return
			} else if len(users) == 0 {
				Error(w, r, todev.Errorf(todev.EUNAUTHORIZED, "Invalid API key."))
				return
			}

			r = r.WithContext(todev.NewContextWithUser(r.Context(), users[0]))

			next.ServeHTTP(w, r)
			return
		}

		session, _ := s.session(r)

		if session.UserID != 0 {
			if user, err := s.UserService.FindUserByID(r.Context(), session.UserID); err != nil {
				log.Printf("error retrieving session user; id=%d err=%s", session.UserID, err)
			} else {
				r = r.WithContext(todev.NewContextWithUser(r.Context(), user))
			}
		}

		next.ServeHTTP(w, r)
	})
}

// requireNoAuth is middleware for requiring no authentication.
// This is used if a user goes to log in but is already logged in.
func (s *Server) requireNoAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if userID := todev.UserIDFromContext(r.Context()); userID != 0 {
			http.Redirect(w, r, "/", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// requireAuth
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If user is logged in delegate to next HTTP handler.
		if userID := todev.UserIDFromContext(r.Context()); userID != 0 {
			next.ServeHTTP(w, r)
			return
		}
		// Otherwise save the current URL (without scheme/host).
		redirectURL := r.URL
		redirectURL.Scheme, redirectURL.Host = "", ""

		// Save the URL to the session and redirect to the log in page.
		// On successful login, the user will be redirected to their original location.
		session, _ := s.session(r)
		session.RedirectURL = redirectURL.String()
		if err := s.setSession(w, session); err != nil {
			log.Printf("http: cannot set session: %s", err)
		}
		http.Redirect(w, r, "/login", http.StatusFound)
	})

}

// loadFlash is middleware for reading flash data from the cookie.
// Data is only loaded once and then immediately cleared.
func (s *Server) loadFlash(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if cookie, _ := r.Cookie("flash"); cookie != nil {
			SetFlash(w, "")
			r = r.WithContext(todev.NewContextWithFlash(r.Context(), cookie.Value))
		}

		next.ServeHTTP(w, r)
	})
}

// trackMetrics is middleware for tracking the request count and timing per route.
func trackMetrics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t := time.Now()
		tmpl := requestPathTemplate(r)

		next.ServeHTTP(w, r)

		// Track total time unless it is the WebSocket endpoint for events.
		if tmpl != "" && tmpl != "/events" {
			requestCount.WithLabelValues(r.Method, tmpl).Inc()
			requestSeconds.WithLabelValues(r.Method, tmpl).Add(float64(time.Since(t).Seconds()))
		}

	})
}

// requestPathTemplate returns the route path template for r.
func requestPathTemplate(r *http.Request) string {
	route := mux.CurrentRoute(r)
	if route == nil {
		return ""
	}
	tmpl, _ := route.GetPathTemplate()
	return tmpl
}

// reportPanic is middleware for catching panics and reporting them.
func reportPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				todev.ReportPanic(err)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	resp := todev.ErrorResponse{
		Header:  "Your page cannot be found.",
		Message: "Sorry, it looks like we can't find what you're looking for.",
	}

	todev.JSON(w, http.StatusNotFound, resp)
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if todev.UserIDFromContext(r.Context()) == 0 {
		http.Redirect(w, r, "/login", http.StatusNotFound)
		return
	}

	var err error
	var resp todev.IndexResponse

	if resp.Repos, _, err = s.RepoService.FindRepos(r.Context(), todev.RepoFilter{}); err != nil {
		Error(w, r, err)
		return
	} else if len(resp.Repos) == 0 {
		http.Redirect(w, r, "/repos", http.StatusFound)
	}

	if resp.Contributors, _, err = s.ContributorService.FindContributors(r.Context(), todev.ContributorFilter{
		Limit:  20,
		SortBy: "updated_at_desc",
	}); err != nil {
		Error(w, r, err)
		return
	}

	if err = todev.JSON(w, http.StatusOK, resp); err != nil {
		Error(w, r, err)
		return
	}
}

// handleVersion displays the deployed version.
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "text/plain")
	w.Write([]byte(todev.Version))
}

// handleCommit displays the deployed commit.
func (s *Server) handleCommit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-type", "text/plain")
	w.Write([]byte(todev.Commit))
}

// session returns session data from the secure key.
func (s *Server) session(r *http.Request) (Session, error) {
	cookie, err := r.Cookie(SessionCookieName)
	if err != nil {
		return Session{}, nil
	}

	var session Session
	if err = s.UnmarshalSession(cookie.Value, &session); err != nil {
		return Session{}, err
	}

	return session, nil
}

// setSession cretes a secure cookie with session data.
func (s *Server) setSession(w http.ResponseWriter, session Session) error {
	buf, err := s.MarshalSession(session)
	if err != nil {
		return err
	}

	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    buf,
		Path:     "/",
		Expires:  time.Now().Add(30 * 24 * time.Hour),
		Secure:   s.UseTLS(),
		HttpOnly: true,
	})
	return nil
}

// MarshalSession encodes session data to string.
// This is exported to allow unit tests to generate fake sessions.
func (s *Server) MarshalSession(session Session) (string, error) {
	return s.sc.Encode(SessionCookieName, session)
}

// UnmarshalSession decodes session data into a session object.
// This is exported to allow unit tests to generate fake sessions.
func (s *Server) UnmarshalSession(data string, session *Session) error {
	return s.sc.Decode(SessionCookieName, data, &session)
}

// ListenAndServeTLSRedirect runs an HTTP server on port 80 to redirect users to
// the TLS enabled port 443 server.
func ListenAndServeTLSRedirect(domain string) error {
	return http.ListenAndServe(":80", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://"+domain, http.StatusFound)
	}))
}

// ListenAndServeDebug runs an http server with /debug endpoints (e.g. pprof, vars).
func ListenAndServeDebug() error {
	h := http.NewServeMux()
	h.Handle("/metrics", promhttp.Handler())
	return http.ListenAndServe(":6060", h)
}
