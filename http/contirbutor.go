package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/saiddis/todev"
)

// registerContributorRoutes is a helper function for registering contirbutor
// routes.
func (s *Server) registerContributorRoutes(r *mux.Router) {
	// Create contributor via invite code
	r.HandleFunc("/invite/{code}", s.handleContirbutorNew).Methods("GET")
	r.HandleFunc("/invite/{code}", s.handleContributorCreate).Methods("POST")

	// Update contributor
	r.HandleFunc("/contributor/{id}", s.handleContritbutorUpdate).Methods("PATCH")

	// Delete contributor
	r.HandleFunc("/contributor/{id}", s.handleContritbutorDelete).Methods("DELETE")
}

// handleContirbutorNew handles the "GET /invite/:code" route. This route uses
// the repo's invite code to allow users to join an existing repo.
func (s *Server) handleContirbutorNew(w http.ResponseWriter, r *http.Request) {
	userID := todev.UserIDFromContext(r.Context())

	code := mux.Vars(r)["code"]

	repos, _, err := s.RepoService.FindRepos(r.Context(), todev.RepoFilter{InviteCode: &code})
	if err != nil {
		Error(w, r, fmt.Errorf("error retrieving repos: %v", err))
		return
	} else if len(repos) == 0 {
		Error(w, r, todev.Errorf(todev.ENOTFOUND, "Invalid invitation URL."))
		return
	}

	if contributors, _, err := s.ContributorService.FindContributors(r.Context(), todev.ContributorFilter{
		RepoID: &repos[0].ID,
		UserID: &userID,
	}); err != nil {
		Error(w, r, fmt.Errorf("error retrieving contributors: %v", err))
		return
	} else if len(contributors) != 0 {
		SetFlash(w, "You are already a contributing for this repo.")
		http.Redirect(w, r, fmt.Sprintf("/repos/%d", contributors[0].RepoID), http.StatusFound)
		return
	}
}

// handleContributorCreate handles the "POST /repos/:code" route. This route
// adds a new contributor for the current user's user ID to a repo.
func (s *Server) handleContributorCreate(w http.ResponseWriter, r *http.Request) {
	userID := todev.UserIDFromContext(r.Context())

	code := mux.Vars(r)["code"]

	repos, _, err := s.RepoService.FindRepos(r.Context(), todev.RepoFilter{InviteCode: &code})
	if err != nil {
		Error(w, r, fmt.Errorf("error retrieving repos: %v", err))
		return
	} else if len(repos) == 0 {
		Error(w, r, todev.Errorf(todev.ENOTFOUND, "Invalid invitation URL."))
		return
	}

	contributor := &todev.Contributor{
		RepoID: repos[0].ID,
		UserID: userID,
	}
	if err = s.ContributorService.CreateContributor(r.Context(), contributor); err != nil {
		Error(w, r, fmt.Errorf("error creating contirbutor: %v", err))
		return
	}
}

// handleContritbutorUpdate handles the "PATCH /contributor/:id" route. This route
// is only called via JSON API on the repo view page.
func (s *Server) handleContritbutorUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		Error(w, r, todev.Errorf(todev.EINVALID, "Invalid id format."))
		return
	}

	// Force application/json output.
	r.Header.Set("Accept", "application/json")

	var upd todev.ContributorUpdate
	if err = json.NewDecoder(r.Body).Decode(&upd); err != nil {
		Error(w, r, todev.Errorf(todev.EINVALID, "Invalid JSON body."))
		return
	}

	contritbutor, err := s.ContributorService.UpdateContributor(r.Context(), id, upd)
	if err != nil {
		Error(w, r, fmt.Errorf("error updating contritbutor: %v", err))
		return
	}

	if err = todev.JSON(w, http.StatusOK, contritbutor); err != nil {
		Error(w, r, fmt.Errorf("error writing response: %v", err))
		return
	}
}

// handleContritbutorDelete handles the "DELETE /contributor/:id" route.
// This route deletes the given contritbutor and redirects the user.
func (s *Server) handleContritbutorDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		Error(w, r, todev.Errorf(todev.EINVALID, "Invalid id format."))
		return
	}

	contritbutor, err := s.ContributorService.FindContributorByID(r.Context(), id)
	if err != nil {
		Error(w, r, fmt.Errorf("error retrieving contritbutor by ID: %v", err))
		return
	}

	if err = s.ContributorService.DeleteContributor(r.Context(), id); err != nil {
		Error(w, r, fmt.Errorf("error deleting contritbutor: %v", err))
		return
	}

	SetFlash(w, "Contributor successfully deleted.")

	// If user is owner then redirect back to the repo's view page. However,
	// if user was just a member then they won't be able to see the repo anymore
	// so redirect them to the home page.
	if contritbutor.OwnerID == todev.UserIDFromContext(r.Context()) {
		http.Redirect(w, r, fmt.Sprintf("/repos/%d", contritbutor.RepoID), http.StatusFound)
	} else {
		http.Redirect(w, r, fmt.Sprintf("/repos"), http.StatusFound)
	}
}
