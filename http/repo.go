package http

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/saiddis/todev"
)

// registerRepoRoutes is a helper function for registering repo routes.
func (s *Server) registerRepoRoutes(r *mux.Router) {
	// Listing of all repos user is a member of.
	r.HandleFunc("/repos", s.handleRepoIndex).Methods("GET")

	// API endpoint for creating repo.
	r.HandleFunc("/repos", s.handleRepoCreate).Methods("POST")

	// HTML form for creating repos.
	r.HandleFunc("/repos/new", s.handleRepoCreate).Methods("POST")

	// View a single repo.
	r.HandleFunc("/repos/{id}", s.handleRepoView).Methods("GET")

	// HTML form for updating an existing repo.
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
		if err := json.NewDecoder(r.Body).Decode(&filter); err != nil {
			Error(w, r, todev.Errorf(todev.EINVALID, "Invalid JSON body"))
			return
		}
	default:
		offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
		filter.Offset = offset
		filter.Limit = 20
	}

	repos, n, err := s.RepoService.FindRepos(r.Context(), filter)
	if err != nil {
		Error(w, r, fmt.Errorf("error retrieving repos: %v", err))
		return
	}

	switch r.Header.Get("Accept") {
	case "application/json":
		if err = todev.JSON(w, http.StatusOK, todev.FindReposResponse{
			Repos: repos,
			N:     n,
		}); err != nil {
			LogError(r, err)
			return
		}
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
		Error(w, r, err)
		return
	}

	switch r.Header.Get("Accept") {
	case "application/json":
		if err = todev.JSON(w, http.StatusFound, repo); err != nil {
			LogError(r, err)
			return
		}
	}
}

// handleRepoCreate handles the "POST /repos" and "POST /repos/new" route.
func (s *Server) handleRepoCreate(w http.ResponseWriter, r *http.Request) {
	var repo todev.Repo
	switch r.Header.Get("Content-type") {
	case "application/json":
		if err := json.NewDecoder(r.Body).Decode(&repo); err != nil {
			Error(w, r, todev.Errorf(todev.EINVALID, "Invalid JSON body"))
			return
		}
	default:
		repo.Name = r.PostFormValue("name")
	}

	err := s.RepoService.CreateRepo(r.Context(), &repo)
	if err != nil {
		Error(w, r, fmt.Errorf("error creating repo: %v", err))
		return
	}

	switch r.Header.Get("Accept") {
	case "application/json":
		if err := todev.JSON(w, http.StatusCreated, repo); err != nil {
			LogError(r, err)
			return
		}
	}
}

// TODO: implement repo edit route with HTML.

// handleRepoEdit handles the "GET /repos/:id/edit" route. This route fetches
// the underlying repo and renders it an HTML form.
// func (s *Server) handleRepoEdit(w http.ResponseWriter, r *http.Request) {
// 	// Parse repo ID from the path.
// 	id, err := strconv.Atoi(mux.Vars(r)["id"])
// 	if err != nil {
// 		Error(w, r, todev.Errorf(todev.EINTERNAL, "Invalid ID format"))
// 		return
// 	}
//
// }

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
	if err != nil {
		Error(w, r, err)
		return
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
		todev.JSON(w, http.StatusOK, []byte("{}"))
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
	if err = json.NewDecoder(resp.Body).Decode(&repo); err != nil {
		return nil, err
	}

	return &repo, nil
}

// FindRepos retrieves a list of repos based on a filter. Only returns repos
// that the user ownes or is a member of. Also returns a count of total matching
// repos.
func (s *RepoService) FindRepos(ctx context.Context, filter todev.RepoFilter) ([]*todev.Repo, int, error) {
	// Marshal filter into JSON format.
	body, err := json.Marshal(filter)
	if err != nil {
		return nil, 0, fmt.Errorf("error marshalling: %v", err)
	}

	// Create request with API key.
	req, err := s.Client.newRequest(ctx, "GET", "/repos", bytes.NewReader(body))
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
	defer resp.Body.Close()

	// Unmarshal result set of repos and total repo count.
	var jsonResponse todev.FindReposResponse
	if err = json.NewDecoder(resp.Body).Decode(&jsonResponse); err != nil {
		return nil, 0, fmt.Errorf("error decoding reponse: %v", err)
	}
	return jsonResponse.Repos, jsonResponse.N, nil
}

// CreateRepo creates a new repo and assigns the current user as the owner.
// The owner will automatically be added as a member of the new repo.
func (s *RepoService) CreateRepo(ctx context.Context, repo *todev.Repo) error {
	body, err := json.Marshal(repo)
	if err != nil {
		return err
	}

	// Create request with API key.
	req, err := s.Client.newRequest(ctx, "POST", "/repos", bytes.NewReader(body))
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

	if err = json.NewDecoder(req.Body).Decode(&repo); err != nil {
		return err
	}
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
