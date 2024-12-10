package http

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/google/go-github/v32/github"
	"github.com/gorilla/mux"
	"github.com/saiddis/todev"
	"golang.org/x/oauth2"
)

// registerAuthRoutes is a helper function to register auth routes to the router.
func (s *Server) registerAuthRoutes(r *mux.Router) {
	r.HandleFunc("/logout", s.handleLogout).Methods("DELETE")
	r.HandleFunc("/oauth/github", s.handleOAuthGitHub).Methods("GET")
	r.HandleFunc("/oauth/github/callback", s.handleOAuthGitHubCallback).Methods("GET")
}

// TODO: implement handleLogin
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
}

// handleLogout handles the "DELETE /logout" route. It clears the session cookie
// and redirects the user to the home page.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	// Clear session cookie on HTTP response.
	if err := s.setSession(w, Session{}); err != nil {
		Error(w, r, err)
		return
	}

	http.Redirect(w, r, "/", http.StatusFound)
}

// handleOAuthGitHub handles the "GET /oauth/github" route. It generates a random
// state variable and redirects the user to the GitHub OAuth endpoint.
//
// After authentication, user will be redirected back to the callback page where
// we can store the returned OAuth tokens.
func (s *Server) handleOAuthGitHub(w http.ResponseWriter, r *http.Request) {
	session, err := s.session(r)
	if err != nil {
		Error(w, r, err)
		return
	}

	state := make([]byte, 64)
	if _, err = io.ReadFull(rand.Reader, state); err != nil {
		Error(w, r, err)
		return
	}

	session.State = hex.EncodeToString(state)

	if err = s.setSession(w, session); err != nil {
		Error(w, r, err)
		return
	}

	http.Redirect(w, r, s.OAuth2Config().AuthCodeURL(session.State), http.StatusFound)

}

// handleOAuthGitHubCallback handles the "GET /oauth/github/callback" route.
// It validates the required OAuth state that we generated previously, looks up
// the current user's information, and creates an "Auth" object in the database.
func (s *Server) handleOAuthGitHubCallback(w http.ResponseWriter, r *http.Request) {
	// Read form variables passed in from GitHub.
	state, code := r.FormValue("state"), r.FormValue("code")

	session, err := s.session(r)
	if err != nil {
		Error(w, r, fmt.Errorf("error reading session: %v", err))
		return
	}

	// Validate that session matches session state.
	if state != session.State {
		Error(w, r, fmt.Errorf("oauth state mismatch"))
		return
	}

	// Exchange code for OAuth tokens.
	tok, err := s.OAuth2Config().Exchange(r.Context(), code)
	if err != nil {
		Error(w, r, fmt.Errorf("oauth exchage error: %v", err))
		return
	}

	// Create a new GitHub API token.
	client := github.NewClient(oauth2.NewClient(r.Context(), oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: tok.AccessToken},
	)))

	// Fetch user information for the currently authenticated user.
	// Require that we at least recieve a user ID from GitHub.

	u, _, err := client.Users.Get(r.Context(), "")
	if err != nil {
		Error(w, r, fmt.Errorf("user ID not returned by GitHub, cannot authenticate user"))
		return
	}

	// Email is not nessarily availible for all accounts. If it is, store it
	// so we can link together multiple OAuth providers in the future.
	var name string
	if u.Name != nil {
		name = *u.Name
	} else if u.Login != nil {
		name = *u.Login
	}
	var email string
	if u.Email != nil {
		email = *u.Email
	}

	// Create an authentication object with the associated user.
	auth := &todev.Auth{
		Source:       todev.AuthSourceGitHub,
		SourceID:     strconv.FormatInt(*u.ID, 10),
		AccessToken:  tok.AccessToken,
		RefreshToken: tok.RefreshToken,
		User: &todev.User{
			Name:  name,
			Email: email,
		},
	}
	if !tok.Expiry.IsZero() {
		auth.Expiry = tok.Expiry
	}

	// Create the "Auth" object in the database. The AuthService will lookup
	// the user by email if they already exist. Otherwise, a new user will be
	// creaeted and the user's ID will be set to auth.UserID.
	if err = s.AuthService.CreateAuth(r.Context(), auth); err != nil {
		Error(w, r, fmt.Errorf("error creating auth: %v", err))
		return
	}

	// Restore redirect URL stored on login.
	redirectURL := session.RedirectURL

	// Update browser session to store user's ID and clear OAuth state.
	session.UserID = auth.UserID
	session.RedirectURL = ""
	session.State = ""
	if err = s.setSession(w, session); err != nil {
		Error(w, r, fmt.Errorf("error setting session cookie: %v", err))
		return
	}

	// Redirect to stored URL or, if not availible, to the home page.
	if redirectURL == "" {
		redirectURL = "/"
	}

	http.Redirect(w, r, redirectURL, http.StatusFound)
}
