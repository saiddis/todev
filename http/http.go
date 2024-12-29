package http

import (
	"context"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/saiddis/todev"
	"github.com/saiddis/todev/http/html"
)

var (
	errorCount = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "todev_http_error_count",
		Help: "Total number of errors by error code",
	}, []string{"code"})
)

type Client struct {
	URL string
}

func NewClient(u string) *Client {
	return &Client{URL: u}
}

// newRequest returns a new HTTP request but adds current user's API key and sets
// the accept and content type header to use JSON.
func (c *Client) newRequest(ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, c.URL+url, body)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	if user := todev.UserFromContext(ctx); user != nil && user.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+user.APIKey)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-type", "application/json")

	return req, nil

}

// SessionCookieName is the name of the cookie used to store the session.
const SessionCookieName = "session"

// Session represensts session data strored in a secure cookie.
type Session struct {
	UserID      int    `json:"userID"`
	RedirectURL string `json:"redirectURL"`
	State       string `json:"state"`
	AvatarURL   string `json:"avatarURL"`
}

// SetFlash sets the flash cookie for the next request to read.
func SetFlash(w http.ResponseWriter, s string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "flash",
		Value:    s,
		Path:     "/",
		HttpOnly: true,
	})
}

// Error writes error and status code to the w and increments errors metric count.
func Error(w http.ResponseWriter, r *http.Request, err error) {
	code, message := todev.ErrorCode(err), todev.ErrorMessage(err)

	errorCount.WithLabelValues(code).Inc()

	if code == todev.EINTERNAL {
		LogError(r, err)
	}

	switch r.Header.Get("Accept") {
	case "appilcation/json":
		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(ErrorStatusCode(code))
		json.NewEncoder(w).Encode(&ErrorResponse{Error: message})
	default:
		w.WriteHeader(ErrorStatusCode(code))
		tmplData := html.Errortemplate{
			StatusCode: ErrorStatusCode(code),
			Header:     "An error has occured.",
			Message:    message,
		}
		if tmpl, err := template.ParseFS(templateFiles, "html/base.html", "html/error.html"); err != nil {
			LogError(r, fmt.Errorf("error parsing html file: %v", err))
			return
		} else if err = tmpl.Execute(w, tmplData); err != nil {
			LogError(r, fmt.Errorf("error executing template: %v", err))
			return
		}
	}
}

// ErrorResponse represents a JSON structure for error output.
type ErrorResponse struct {
	Error string `json:"error"`
}

// parseResponseError parses a JSON-formatted error response.
func parseResponseError(resp *http.Response) error {
	defer resp.Body.Close()

	buf, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var errorResponse ErrorResponse
	if err := json.Unmarshal(buf, &errorResponse); err != nil {
		message := strings.TrimSpace(string(buf))
		if message == "" {
			message = "Empty response from server."
		}
		return todev.Errorf(FromErrorStatusCode(resp.StatusCode), message)
	}
	return todev.Errorf(FromErrorStatusCode(resp.StatusCode), errorResponse.Error)
}

// LogError logs an error with the HTTP route information.
func LogError(r *http.Request, err error) {
	log.Printf("[http] error: %s %s: %s", r.Method, r.URL.Path, err)
}

var codes = map[string]int{
	todev.ECONFLICT:       http.StatusConflict,
	todev.EINVALID:        http.StatusBadRequest,
	todev.ENOTFOUND:       http.StatusNotFound,
	todev.ENOTIMPLEMENTED: http.StatusNotImplemented,
	todev.EUNAUTHORIZED:   http.StatusUnauthorized,
	todev.EINTERNAL:       http.StatusInternalServerError,
}

// ErrorStatusCode returns the associated HTTP status code for a todev error code.
func ErrorStatusCode(code string) int {
	if v, ok := codes[code]; ok {
		return v
	}
	return http.StatusInternalServerError
}

// FromErrorStatusCode returns the associated todev error for an HTTP status code.
func FromErrorStatusCode(code int) string {
	for k, v := range codes {
		if v == code {
			return k
		}
	}
	return todev.EINTERNAL
}
