package http_test

import (
	"net/http"
	"net/url"
	"testing"

	todevhttp "github.com/saiddis/todev/http"
)

func TestLogin_OAuth_GitHub(t *testing.T) {
	s := MustOpenServer(t)
	defer MustCloseServer(t, s)

	// Disable redirects for testing OAuth.
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Fetch our OAuth redirection URL and ensure it is redirecting us.
	resp, err := client.Get(s.URL() + "/oauth/github")
	if err != nil {
		t.Fatal(err)
	} else if err = resp.Body.Close(); err != nil {
		t.Fatal(err)
	} else if got, want := resp.StatusCode, http.StatusFound; got != want {
		t.Fatalf("StatusCode=%d, want %d", got, want)
	}

	// Read session from cookie & ensure a state variable is set.
	var session todevhttp.Session
	if err = s.UnmarshalSession(resp.Cookies()[0].Value, &session); err != nil {
		t.Fatal(err)
	} else if session.State == "" {
		t.Fatal("expectex oauth state in session")
	}

	// Parse location and verify that the URL is correct and that we have the
	// client ID to our configured ID and that state matches our session state.
	if loc, err := url.Parse(resp.Header.Get("Location")); err != nil {
		t.Fatal(err)
	} else if got, want := loc.Host, "github.com"; got != want {
		t.Fatalf("Location=%s, want %s", got, want)
	} else if got, want := loc.Path, "/login/oauth/authorize"; got != want {
		t.Fatalf("Location.Path=%s, want %s", got, want)
	} else if got, want := loc.Query().Get("client_id"), TestGitHubClientID; got != want {
		t.Fatalf("Location.Query.client_id=%s, want %s", got, want)
	} else if got, want := loc.Query().Get("state"), session.State; got != want {
		t.Fatalf("Location.Query.state=%s, want %s", got, want)
	}
}
