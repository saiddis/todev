package json

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/saiddis/todev"
)

// Write marshals and writes payload into w.
// EINTERNAL on error.
//
// Writes to w only if no error occured, so as to let client code to
// handle the error.
func Write(w http.ResponseWriter, code int, payload interface{}) error {
	jsonData, err := json.Marshal(payload)
	if err != nil {
		return todev.Errorf(todev.EINTERNAL, "error marshalling into json: %v", err)
	}
	w.Header().Add("Content-type", "application/json")
	w.WriteHeader(code)
	w.Write(jsonData)
	return nil
}

// Encode encodes src into a writer dst.
func Encode(src any, dst io.Writer) error {
	err := json.NewEncoder(dst).Encode(src)
	if err != nil {
		return err
	}
	return nil
}

// Decode decodes response/request body into a struct.
func Decode(body io.ReadCloser, dst any) error {
	if err := json.NewDecoder(body).Decode(dst); err != nil {
		return err
	}
	return nil
}

// IndexResponse represents response payload for the main page.
type IndexResponse struct {
	// List of repos user ownes or is member of.
	Repos []*todev.Repo `json:"repos"`

	// Most recently updated contritbutors.
	Contributors []*todev.Contributor `json:"contributors"`
}

// ErrorResponse represents error response payload for an error.
type ErrorResponse struct {
	Header  string `json:"header"`
	Message string `json:"message"`
}

// FindReposResponse represents payload for "GET /repos".
type FindReposResponse struct {
	Repos []*todev.Repo `json:"repos"`
	N     int           `json:"n"`
}

// FindTasksResponse represents payload for "GET /tasks".
type FindTasksResponse struct {
	Tasks []*todev.Task `json:"tasks"`
	N     int           `json:"n"`
}
