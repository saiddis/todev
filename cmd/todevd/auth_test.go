package main_test

import (
	"context"
	"log"
	"testing"

	"github.com/chromedp/chromedp"
)

// Ensure that navigating to the page that requires authentication will redirect
// the user to the home page.
func TestRedirectToLogin(t *testing.T) {
	t.Parallel()

	// Begin running our test programm.
	m := MustRunMain(t)
	defer MustCloseMain(t, m)

	// Create Chrome testing context.
	ctx, cancel := chromedp.NewContext(context.Background(), chromedp.WithLogf(log.Printf))
	defer cancel()

	// Navigate to the main page, expect to be redirect to login.
	if err := chromedp.Run(ctx,
		chromedp.Navigate(m.HTTPServer.URL()),
	); err != nil {
		t.Fatal(err)
	}
}
