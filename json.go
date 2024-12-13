package todev

import (
	"encoding/json"
	"net/http"
)

// Respond marshals and writes payload into w.
// EINTERNAL on error.
//
// Writes to w only if no error occured, so as to let client code to
// handle the error.
func JSON(w http.ResponseWriter, code int, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return Errorf(EINTERNAL, "error marshalling into json: %v", err)
	}
	w.Header().Add("Content-type", "application/json")
	w.WriteHeader(code)
	w.Write(jsonData)
	return nil
}

// IndexResponse represents response payload for the main page.
type IndexResponse struct {
	// List of repos user ownes or is member of.
	Repos []*Repo

	// Most recently updated contritbutors.
	Contributors []*Contributor
}

// ErrorResponse represents error response payload for an error.
type ErrorResponse struct {
	Header  string
	Message string
}

// FindReposResponse represents payload for "GET /repos".
type FindReposResponse struct {
	Repos []*Repo `json:"repos"`
	N     int     `json:"n"`
}

// FindTasksResponse represents payload for "GET /tasks".
type FindTasksResponse struct {
	Tasks []*Task `json:"tasks"`
	N     int     `json:"n"`
}
