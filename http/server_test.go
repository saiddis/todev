package http_test

import (
	"context"
	"io"
	"net/http"
	"testing"

	"github.com/saiddis/todev"
	todevhttp "github.com/saiddis/todev/http"
	"github.com/saiddis/todev/mock"
)

// Default configuration settings for the test server.
const (
	TestHashKey            = "0000000000000000"
	TestBlockKey           = "00000000000000000000000000000000"
	TestGitHubClientID     = "00000000000000000000"
	TestGitHubClientSecret = "0000000000000000000000000000000000000000"
)

type Server struct {
	*todevhttp.Server

	AuthService        mock.AuthService
	UserService        mock.UserService
	ContributorService mock.ContributorService
	TaskService        mock.TaskService
	RepoService        mock.RepoService
	EventService       mock.EventService
}

// MustOpenServer is a test helper function for starting a new test HTTP server.
// Fail on error.
func MustOpenServer(tb testing.TB) *Server {
	tb.Helper()

	s := &Server{Server: todevhttp.NewServer()}
	s.HashKey = TestHashKey
	s.BlockKey = TestBlockKey
	s.GitHubClientID = TestGitHubClientID
	s.GitHubClientSecret = TestGitHubClientSecret

	s.Server.AuthService = &s.AuthService
	s.Server.UserService = &s.UserService
	s.Server.ContributorService = &s.ContributorService
	s.Server.TaskService = &s.TaskService
	s.Server.RepoService = &s.RepoService
	s.Server.EventService = &s.EventService

	if err := s.Open(); err != nil {
		tb.Fatal(err)
	}
	return s
}

// MustCloseServer is test helper function for shutting down the server.
// Fail on error.
func MustCloseServer(tb testing.TB, s *Server) {
	tb.Helper()
	if err := s.Close(); err != nil {
		tb.Fatal(err)
	}
}

// MustNewRequest creates a new HTTP request using the server's base URL and
// attaching a user session based on the context.
func (s *Server) MustNewRequest(tb testing.TB, ctx context.Context, method, url string, body io.Reader) *http.Request {
	tb.Helper()

	r, err := http.NewRequest(method, s.URL()+url, body)
	if err != nil {
		tb.Fatal(err)
	}

	if user := todev.UserFromContext(ctx); user != nil {
		data, err := s.MarshalSession(todevhttp.Session{UserID: user.ID})
		if err != nil {
			tb.Fatal(err)
		}
		r.AddCookie(&http.Cookie{
			Name:  todevhttp.SessionCookieName,
			Value: data,
			Path:  "/",
		})
	}

	return r
}
