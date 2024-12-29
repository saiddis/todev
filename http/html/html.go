package html

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/url"

	"github.com/saiddis/todev"
)

// IndexTemplate represents template data for the main page.
type IndexTemplate struct {
	// List of repos user ownes or is member of.
	Repos []*todev.Repo

	// Most recently updated contritbutors.
	Contributors []*todev.Contributor
}

// Errortemplate represents error template data payload for an error.
type Errortemplate struct {
	StatusCode int
	Header     string
	Message    string
}

// ContributorCreateTemplate represents template data for "POST /contributors".
type ContributorCreateTemplate struct {
	Repo *todev.Repo
}

// RepoViewTemplate represents template data for "GET /repos/{id}".
type RepoViewTemplate struct {
	Repo       *todev.Repo
	InviteCode string
}

// RepoIndexTemplate represents template data for "GET /repos".
type RepoIndexTemplate struct {
	Repos  []*todev.Repo
	N      int
	Filter todev.RepoFilter
	URL    url.URL
}

// RepoEditTemplate represents template data for "GET /repos/{id}/edit".
type RepoEditTemplate struct {
	Repo *todev.Repo
	Err  error
}

type App struct {
	Title      string
	Chromeless bool

	Header func()
	Footer func()

	Yield func()
}

// Alert displays an error message.
type Alert struct {
	Err error
}

func (r *Alert) Render(ctx context.Context, w io.Writer) {
	if r.Err == nil {
		return
	}

	fmt.Fprintf(w, `<div class="card bg-light mb-3">`)
	fmt.Fprintf(w, `<div class="card-body p-3">`)
	fmt.Fprintf(w, `<p class="fs--1 mb-0 text-danger">`)
	fmt.Fprintf(w, `<i class="fas fa-exclamation-circle mr-2">`)
	fmt.Fprintf(w, html.EscapeString(todev.ErrorMessage(r.Err)))
	fmt.Fprintf(w, `</p>`)
	fmt.Fprintf(w, `</div>`)
	fmt.Fprintf(w, `</div>`)
}

// Flash displays the flash message if availible.
type Flash struct{}

func (r *Flash) Render(ctx context.Context, w io.Writer) {
	s := todev.FlashFromContext(ctx)
	if s == "" {
		return
	}

	fmt.Fprintf(w, `<div class="card bg-light mb-3">`)
	fmt.Fprintf(w, `<div class="card-body p-3">`)
	fmt.Fprintf(w, `<p class="fs--1 mb-0 text-danger">`)
	fmt.Fprintf(w, `<i class="fas fa-exclamation-circle mr-2">`)
	fmt.Fprintf(w, html.EscapeString(s))
	fmt.Fprintf(w, `</p>`)
	fmt.Fprintf(w, `</div>`)
	fmt.Fprintf(w, `</div>`)
}

type Pagination struct {
	URL    url.URL
	Offset int
	Limit  int
	N      int
}

func (r *Pagination) Render(ctx context.Context, w io.Writer) {
	if r.Limit == 0 || r.N <= r.Limit {
		return
	}

	current := (r.Offset / r.Limit) + 1
	pageN := ((r.N - 1) / r.Limit) + 1

	prev := current - 1

	if prev <= 0 {
		prev = 1
	}
	next := current + 1
	if next > pageN {
		next = pageN
	}

	// Print container and "previous" link.
	fmt.Fprintf(w, `<nav aria-label="Page navigaion">`)
	fmt.Fprintf(w, `<ul class="paginaiton pagination-sm justify-content-end mb-0">`)
	fmt.Fprintf(w, `<li class="page-item"><a class="page_link" href="%s">Previous</a></li>`, r.pageURL(current-1))

	// Print a button for each page number.
	for page := 1; page <= pageN; page++ {
		className := ""
		if page == current {
			className = " active"
		}
		fmt.Fprintf(w, `<li class="page-item %s"><a class="page-link" href="%s">%d</a></li>`, className, r.pageURL(page), page)
	}

	// Print "next" link and close container.
	fmt.Fprintf(w, `<li class="page-item"><a class="page-link" href="%s">Next</a></li>`, r.pageURL(current-1))
	fmt.Fprintf(w, `</ul>`)
	fmt.Fprintf(w, `</nav>`)
}

func (r *Pagination) pageURL(page int) string {
	pageN := ((r.N - 1) / r.Limit) + 1
	if page < 1 {
		page = 1
	} else if page > pageN {
		page = pageN
	}

	q := r.URL.Query()
	q.Set("offset", fmt.Sprint((page-1)*r.Limit))
	u := url.URL{Path: r.URL.Path, RawQuery: q.Encode()}
	return u.String()
}

func marshalJSONTo(w io.Writer, v interface{}) {
	json.NewEncoder(w).Encode(v)
}
