package main_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/chromedp/cdproto/network"
	"github.com/chromedp/chromedp"
	"github.com/saiddis/todev"
	"github.com/saiddis/todev/cmd/todevd"
	"github.com/saiddis/todev/http"
)

// MustRunMain is a test helper function that executes Main in a temporary path.
func MustRunMain(tb testing.TB) *main.Main {
	tb.Helper()

	m := main.NewMain()
	// m.Config.DB.DSN = filepath.Join(tb.TempDir(), "todev")
	m.Config.HTTP.Addr = ":0"
	m.Config.Github.ClientID = strings.Repeat("00", 10)
	m.Config.Github.ClientSecret = strings.Repeat("00", 20)
	m.Config.HTTP.HashKey = strings.Repeat("00", 64)
	m.Config.HTTP.BlockKey = strings.Repeat("00", 32)

	if err := m.Run(context.Background()); err != nil {
		tb.Fatal(err)
	}

	return m
}

// MustCloseMain closes the programm.
func MustCloseMain(tb testing.TB, m *main.Main) {
	tb.Helper()
	if err := m.Close(); err != nil {
		tb.Fatal(err)
	}
}

// MustCreateUser is test helper for creating a new user in the system by colling
// the underlying DB service directly.
func MustCreateUser(tb testing.TB, m *main.Main, user *todev.User) (*todev.User, context.Context) {
	tb.Helper()
	if err := m.UserService.CreateUser(context.Background(), user); err != nil {
		tb.Fatalf("error creating user: %v", err)
	}
	return user, todev.NewContextWithUser(context.Background(), user)
}

// Login returns a Chrome action that generates a secure cookie and attaches it
// to the browser. This approach is used to avoid OAuth communication with GitHub.
func Login(ctx context.Context, m *main.Main) chromedp.Action {
	return chromedp.ActionFunc(func(ctx context.Context) error {
		// Generate cookie value from the server.
		value, err := m.HTTPServer.MarshalSession(http.Session{
			UserID: todev.UserIDFromContext(ctx),
		})
		if err != nil {
			return fmt.Errorf("error marshalling session: %v", err)
		}

		// Add cookie to browser.
		if err = network.SetCookie(http.SessionCookieName, value).WithDomain("localhost").Do(ctx); err != nil {
			return fmt.Errorf("error setting session cookie: %v", err)
		}
		return nil
	})
}
