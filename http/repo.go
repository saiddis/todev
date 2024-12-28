package http

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/saiddis/todev"
	"github.com/saiddis/todev/http/html"
	"github.com/saiddis/todev/http/json"
)

// registerRepoRoutes is a helper function for registering repo routes.
func (s *Server) registerRepoRoutes(r *mux.Router) {
	// Listing of all repos user is a member of.
	r.HandleFunc("/repos", s.handleRepoIndex).Methods("GET")

	// API endpoint for creating repo.
	r.HandleFunc("/repos", s.handleRepoCreate).Methods("POST")

	// HTML form for creating repos.
	r.HandleFunc("/repos/new", s.handleRepoNew).Methods("GET")
	r.HandleFunc("/repos/new", s.handleRepoCreate).Methods("POST")

	// View a single repo.
	r.HandleFunc("/repos/{id}", s.handleRepoView).Methods("GET")

	// HTML form for updating an existing repo.
	r.HandleFunc("repo/{id}/edit", s.handleRepoEdit).Methods("GET")
	r.HandleFunc("repo/{id}/edit", s.handleRepoUpdate).Methods("PATCH")

	// Removing a repo.
	r.HandleFunc("repo/{id}", s.handleRepoDelete).Methods("DELETE")
}

// handleRepoIndex handles the "GET /repos" route. This route can optionaly accept
// filter arguments and output a list of all the repos that the current user is a
// member of.
//
// The endpoint works with HTML, JSON and CSV formats.
func (s *Server) handleRepoIndex(w http.ResponseWriter, r *http.Request) {
	var filter todev.RepoFilter
	switch r.Header.Get("Content-type") {
	case "application/json":
		if err := json.Decode(r.Body, &filter); err != nil {
			Error(w, r, todev.Errorf(todev.EINVALID, "Invalid JSON body"))
			return
		}
		defer func() {
			if err := r.Body.Close(); err != nil {
				LogError(r, fmt.Errorf("error closing request body: %v", err))
			}
		}()
	default:
		filter.Offset, _ = strconv.Atoi(r.URL.Query().Get("offset"))
		filter.Limit = 20
	}

	repos, n, err := s.RepoService.FindRepos(r.Context(), filter)
	if err != nil {
		Error(w, r, fmt.Errorf("error retrieving repos: %v", err))
		return
	}

	switch r.Header.Get("Accept") {
	case "application/json":
		w.Header().Set("Content-type", "application/json")
		if err = json.Encode(json.FindReposResponse{Repos: repos, N: n}, w); err != nil {
			LogError(r, err)
			return
		}
	default:
		tmplData := html.RepoIndexTemplate{Repos: repos, N: n, Filter: filter, URL: *r.URL}
		if tmpl, err := template.ParseFS(templateFiles, "html/base.html", "html/repoIndex.html"); err != nil {
			LogError(r, fmt.Errorf("error parsing html file: %v", err))
			return
		} else if err = tmpl.Execute(w, tmplData); err != nil {
			LogError(r, fmt.Errorf("error executing template: %v", err))
			return
		}
	}
}

// handleRepoNew handles the "GET /repos/new" route.
// It renders a new HTML form for editing a new repo.
func (s *Server) handleRepoNew(w http.ResponseWriter, r *http.Request) {
	tmplData := html.RepoEditTemplate{Repo: &todev.Repo{}}

	if tmpl, err := template.ParseFS(templateFiles, "html/base.html", "html/repoEdit.html"); err != nil {
		LogError(r, fmt.Errorf("error parsing html file: %v", err))
		return
	} else if err = tmpl.Execute(w, tmplData); err != nil {
		LogError(r, fmt.Errorf("error executing template: %v", err))
		return
	}

}

// handleRepoView handles the "GET /repos/:id" route.
func (s *Server) handleRepoView(w http.ResponseWriter, r *http.Request) {
	// Parse ID from path.
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		Error(w, r, err)
		return
	}

	repo, err := s.RepoService.FindRepoByID(r.Context(), id)
	if err != nil {
		Error(w, r, fmt.Errorf("error retrieving repo by ID: %v", err))
		return
	} else if repo.Contributors, _, err = s.ContributorService.FindContributors(r.Context(), todev.ContributorFilter{RepoID: &repo.ID}); err != nil {
		Error(w, r, fmt.Errorf("error retrieving repo contributors: %v", err))
		return
	} else if repo.Tasks, _, err = s.TaskService.FindTasks(r.Context(), todev.TaskFilter{RepoID: &repo.ID}); err != nil {
		Error(w, r, fmt.Errorf("error retrieving repo tasks: %v", err))
		return
	}

	switch r.Header.Get("Accept") {
	case "application/json":
		if err = json.Encode(repo, w); err != nil {
			LogError(r, err)
			return
		}
	default:
		tmplData := html.RepoViewTemplate{
			Repo:       repo,
			InviteCode: fmt.Sprintf("%s/invite/%s", s.URL(), repo.InviteCode),
		}

		if tmpl, err := template.ParseFS(templateFiles, "html/base.html", "html/repoView.html"); err != nil {
			LogError(r, fmt.Errorf("error parsing html file: %v", err))
			return
		} else if err = tmpl.Execute(w, tmplData); err != nil {
			LogError(r, fmt.Errorf("error executing template: %v", err))
			return
		}
	}
}

// handleRepoCreate handles the "POST /repos" and "POST /repos/new" route.
func (s *Server) handleRepoCreate(w http.ResponseWriter, r *http.Request) {
	var repo todev.Repo
	switch r.Header.Get("Content-type") {
	case "application/json":
		if err := json.Decode(r.Body, &repo); err != nil {
			Error(w, r, todev.Errorf(todev.EINVALID, "Invalid JSON body"))
			return
		}
		defer func() {
			if err := r.Body.Close(); err != nil {
				LogError(r, fmt.Errorf("error closing response body: %v", err))
			}
		}()
	default:
		repo.Name = r.PostFormValue("name")
	}

	err := s.RepoService.CreateRepo(r.Context(), &repo)

	switch r.Header.Get("Accept") {
	case "application/json":
		if err != nil {
			Error(w, r, fmt.Errorf("error creating repo: %v", err))
			return
		}

		w.Header().Set("Content-type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if err = json.Encode(repo, w); err != nil {
			LogError(r, err)
			return
		}
	default:
		if todev.ErrorCode(err) == todev.EINTERNAL {
			Error(w, r, err)
			return
		} else if err != nil {
			tmplData := html.RepoEditTemplate{Repo: &repo, Err: fmt.Errorf("error creating repo: %v", err)}

			if tmpl, err := template.ParseFS(templateFiles, "html/base.html", "html/repoEdit.html"); err != nil {
				LogError(r, fmt.Errorf("error parsing html file: %v", err))
				return
			} else if err = tmpl.Execute(w, tmplData); err != nil {
				LogError(r, fmt.Errorf("error executing template: %v", err))
				return
			}
		}

		SetFlash(w, "Repo successfully created.")
		http.Redirect(w, r, fmt.Sprintf("/repos/%d", repo.ID), http.StatusFound)
	}

}

// handleRepoEdit handles the "GET /repos/:id/edit" route. This route fetches
// the underlying repo and renders it an HTML form.
func (s *Server) handleRepoEdit(w http.ResponseWriter, r *http.Request) {
	// Parse repo ID from the path.
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		Error(w, r, todev.Errorf(todev.EINTERNAL, "Invalid ID format"))
		return
	}

	repo, err := s.RepoService.FindRepoByID(r.Context(), id)
	if err != nil {
		Error(w, r, fmt.Errorf("error retrieving repo by ID: %v", err))
		return
	}

	tmplData := html.RepoEditTemplate{Repo: repo}

	if tmpl, err := template.ParseFS(templateFiles, "html/base.html", "html/repoEdit.html"); err != nil {
		LogError(r, fmt.Errorf("error parsing html file: %v", err))
		return
	} else if err = tmpl.Execute(w, tmplData); err != nil {
		LogError(r, fmt.Errorf("error executing template: %v", err))
		return
	}

}

// handleRepoUpdate handles the "PATCH /repos/:id/edit" route. This route reads
// the udpated fields and issues an updat in the database. On success, it redirects
// repo's view page.
func (s *Server) handleRepoUpdate(w http.ResponseWriter, r *http.Request) {
	// Parse repo ID from the path.
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		Error(w, r, todev.Errorf(todev.EINVALID, "Invalid ID format"))
		return
	}

	var upd todev.RepoUpdate
	name := r.PostFormValue("name")
	upd.Name = &name

	repo, err := s.RepoService.UpdateRepo(r.Context(), id, upd)
	if todev.ErrorCode(err) == todev.EINTERNAL {
		Error(w, r, err)
		return
	} else if err != nil {
		tmplData := html.RepoEditTemplate{Repo: repo}

		if tmpl, err := template.ParseFS(templateFiles, "html/base.html", "html/repoEdit.html"); err != nil {
			LogError(r, fmt.Errorf("error parsing html file: %v", err))
			return
		} else if err = tmpl.Execute(w, tmplData); err != nil {
			LogError(r, fmt.Errorf("error executing template: %v", err))
			return
		}
	}

	SetFlash(w, "Repo successfully updated.")
	http.Redirect(w, r, fmt.Sprintf("/repos/%d", repo.ID), http.StatusFound)
}

// handleRepoDelete handles the "DELETE /repos/:id" route. This route permanently
// deletes the repo and all its contritbutors and redirects to the repo listing
// page.
func (s *Server) handleRepoDelete(w http.ResponseWriter, r *http.Request) {
	// Parse repo ID from the path.
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		Error(w, r, todev.Errorf(todev.EINVALID, "Invalid ID format"))
		return
	}

	if err = s.RepoService.DeleteRepo(r.Context(), id); err != nil {
		Error(w, r, err)
		return
	}

	switch r.Header.Get("Accept") {
	case "application/json":
		json.Write(w, http.StatusOK, []byte("{}"))
	default:
		SetFlash(w, "Repo successfully  deleted.")
		http.Redirect(w, r, "/repos", http.StatusFound)
	}
}

// RepoService implements the todev.RepoService over the HTTP protocol.
type RepoService struct {
	Client *Client
}

func NewRepoService(client *Client) *RepoService {
	return &RepoService{Client: client}
}

// FindRepoByID retrieves a single repo by ID along with associated contributors.
// Only the repo owner and contributors can see the repo.
func (s *RepoService) FindRepoByID(ctx context.Context, id int) (*todev.Repo, error) {
	req, err := s.Client.newRequest(ctx, "GET", fmt.Sprintf("/repos/%d", id), nil)
	if err != nil {
		return nil, err
	}

	// Issue request. If any other status besides 200, then treats as an error.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	} else if resp.StatusCode != http.StatusOK {
		return nil, parseResponseError(resp)
	}

	defer resp.Body.Close()

	var repo todev.Repo
	if err = json.Decode(resp.Body, &repo); err != nil {
		return nil, err
	}
	defer func() {
		if err := req.Body.Close(); err != nil {
			LogError(req, fmt.Errorf("error closing request body: %v", err))
		}
	}()

	return &repo, nil
}

// FindRepos retrieves a list of repos based on a filter. Only returns repos
// that the user ownes or is a member of. Also returns a count of total matching
// repos.
func (s *RepoService) FindRepos(ctx context.Context, filter todev.RepoFilter) ([]*todev.Repo, int, error) {
	buf := bytes.NewBuffer(make([]byte, 0))
	if err := json.Encode(filter, buf); err != nil {
		return nil, 0, fmt.Errorf("error creating request: %v", err)
	}

	// Create request with API key.
	req, err := s.Client.newRequest(ctx, "GET", "/repos", buf)
	if err != nil {
		return nil, 0, fmt.Errorf("error creating request: %v", err)
	}

	// Issue request. Any non-200 code is considered an error.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("error making request: %v", err)
	} else if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("error status code: %d", resp.StatusCode)
	}

	// Unmarshal result set of repos and total repo count.
	var jsonResponse json.FindReposResponse
	if err = json.Decode(resp.Body, &jsonResponse); err != nil {
		return nil, 0, fmt.Errorf("error decoding reponse: %v", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			LogError(req, fmt.Errorf("error closing response body: %v", err))
		}
	}()
	return jsonResponse.Repos, jsonResponse.N, nil
}

// CreateRepo creates a new repo and assigns the current user as the owner.
// The owner will automatically be added as a member of the new repo.
func (s *RepoService) CreateRepo(ctx context.Context, repo *todev.Repo) error {
	buf := bytes.NewBuffer(make([]byte, 0))
	if err := json.Encode(repo, buf); err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}

	// Create request with API key.
	req, err := s.Client.newRequest(ctx, "POST", "/repos", buf)
	if err != nil {
		return err
	}

	// Issue request. Any non-201 code is considered an error.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusCreated {
		log.Printf("status code: %d", resp.StatusCode)
		return parseResponseError(resp)
	}
	defer resp.Body.Close()

	if err = json.Decode(req.Body, &repo); err != nil {
		return err
	}
	defer func() {
		if err := req.Body.Close(); err != nil {
			LogError(req, fmt.Errorf("error closing request body: %v", err))
		}
	}()
	return nil
}

// UpdateRepo is not implemented by the HTTP service.
func (s *RepoService) UpdateRepo(ctx context.Context, id int, upd todev.RepoUpdate) (*todev.Repo, error) {
	return nil, todev.Errorf(todev.ENOTIMPLEMENTED, "Not implemented.")
}

// DeleteRepo permanently removes a repo by ID. Only the repo owner can delete
// a repo.
func (s *RepoService) DeleteRepo(ctx context.Context, id int) error {
	// Create request with API key.
	req, err := s.Client.newRequest(ctx, "POST", fmt.Sprintf("/repos/%d", id), nil)
	if err != nil {
		return err
	}

	// Issue request. Any non-200 code is considered an error.
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	} else if resp.StatusCode != http.StatusOK {
		return parseResponseError(resp)
	}
	defer resp.Body.Close()

	return nil
}
