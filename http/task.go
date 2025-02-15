package http

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/saiddis/todev"
	"github.com/saiddis/todev/http/json"
)

// registerTaskRoutes is a helper function for registering task routes.
func (s *Server) registerTaskRoutes(r *mux.Router) {
	// Listing of all tasks user can view.
	r.HandleFunc("/tasks", s.handleTasksFind).Methods("GET")

	// API endpoint for creating tasks.
	r.HandleFunc("/tasks", s.handleTaskCreate).Methods("POST")

	// Update task
	r.HandleFunc("/tasks/{id}", s.handleTaskUpdate).Methods("PATCH")

	// Delete task
	r.HandleFunc("/tasks/{id}", s.handleTaskDelete).Methods("DELETE")

	// Attach contributor.
	r.HandleFunc("/tasks/{taskID}/contributor/{contributorID}", s.handleTaskAttachContributor).Methods("POST")

	// Unattach contributor.
	r.HandleFunc("/tasks/{taskID}/contributor/{contributorID}", s.handleTaskUnattachContributor).Methods("DELETE")
}

// handleTaskRepoView handles the "GET /tasks" route. This route retrieves all
// tasks for the current user.
func (s *Server) handleTasksFind(w http.ResponseWriter, r *http.Request) {
	var filter todev.TaskFilter
	if err := json.Decode(r.Body, &filter); err != nil {
		Error(w, r, todev.Errorf(todev.EINVALID, "Invalid JSON body"))
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			LogError(r, fmt.Errorf("error closing request body: %v", err))
		}
	}()

	tasks, n, err := s.TaskService.FindTasks(r.Context(), filter)
	if err != nil {
		Error(w, r, fmt.Errorf("error retrieving tasks: %v", err))
		return
	}

	switch r.Header.Get("Accept") {
	case "application/json":
		if err = json.Encode(json.FindTasksResponse{Tasks: tasks, N: n}, w); err != nil {
			LogError(r, fmt.Errorf("error writing response: %v", err))
			return
		}
	}
}

// handleTaskCreate handles the "POST /tasks" route.
func (s *Server) handleTaskCreate(w http.ResponseWriter, r *http.Request) {
	var task todev.Task
	switch r.Header.Get("Content-type") {
	case "application/json":
		if err := json.Decode(r.Body, &task); err != nil {
			Error(w, r, todev.Errorf(todev.EINVALID, "Invalid JSON body"))
			return
		}
		defer func() {
			if err := r.Body.Close(); err != nil {
				LogError(r, fmt.Errorf("error closing request body: %v", err))
			}
		}()
	default:
		task.Description = r.PostFormValue("description")
	}

	err := s.TaskService.CreateTask(r.Context(), &task)
	if err != nil {
		Error(w, r, fmt.Errorf("error creating task: %v", err))
		return
	}

	switch r.Header.Get("Accept") {
	case "application/json":
		w.WriteHeader(http.StatusCreated)
		if err = json.Encode(task, w); err != nil {
			LogError(r, err)
			return
		}
	}
}

// handleTaskUpdate handles the "PATCH /tasks/:id" route. This route is only
// called via JSON API on the repo view page.
func (s *Server) handleTaskUpdate(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		Error(w, r, todev.Errorf(todev.EINVALID, "Invalid ID format"))
		return
	}

	r.Header.Set("Accept", "application/json")

	var upd todev.TaskUpdate
	if err = json.Decode(r.Body, &upd); err != nil {
		Error(w, r, todev.Errorf(todev.EINVALID, "Invalid JSON body"))
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil {
			LogError(r, fmt.Errorf("error closing request body: %v", err))
		}
	}()

	if task, err := s.TaskService.UpdateTask(r.Context(), id, upd); err != nil {
		Error(w, r, fmt.Errorf("error updating task: %v", err))
		return
	} else if err = json.Write(w, http.StatusOK, task); err != nil {
		Error(w, r, fmt.Errorf("error writing response: %v", err))
		return
	}

}

// handleTaskDelete handles the "DELETE /task/:id" route.
func (s *Server) handleTaskDelete(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		Error(w, r, todev.Errorf(todev.EINVALID, "Invalid ID format"))
		return
	}

	if err = s.TaskService.DeleteTask(r.Context(), id); err != nil {
		Error(w, r, fmt.Errorf("error deleting task by ID=%d: %v", id, err))
		return
	} else if err = json.Encode("{}", w); err != nil {
		Error(w, r, fmt.Errorf("error writing response: %v", err))
		return
	}
}

func (s *Server) handleTaskAttachContributor(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.Atoi(mux.Vars(r)["taskID"])
	if err != nil {
		Error(w, r, todev.Errorf(todev.EINVALID, "Invalid ID format"))
		return
	}
	contributorID, err := strconv.Atoi(mux.Vars(r)["contributorID"])
	if err != nil {
		Error(w, r, todev.Errorf(todev.EINVALID, "Invalid ID format"))
		return
	}

	r.Header.Set("Accept", "application/json")

	if task, err := s.TaskService.FindTaskByID(r.Context(), taskID); err != nil {
		Error(w, r, fmt.Errorf("error retrieving task by ID=%d: %v", taskID, err))
		return
	} else if err = s.TaskService.AttachContributor(r.Context(), task, contributorID); err != nil {
		Error(w, r, fmt.Errorf("error contributor ID for task with ID=%d: %v", taskID, err))
		// LogError(r, fmt.Errorf("error contributor ID for task with ID=%d: %v", taskID, err))
		return
	} else if err = json.Encode(task, w); err != nil {
		LogError(r, err)
		return
	}
}

func (s *Server) handleTaskUnattachContributor(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.Atoi(mux.Vars(r)["taskID"])
	if err != nil {
		Error(w, r, todev.Errorf(todev.EINVALID, "Invalid ID format"))
		return
	}
	contributorID, err := strconv.Atoi(mux.Vars(r)["contributorID"])
	if err != nil {
		Error(w, r, todev.Errorf(todev.EINVALID, "Invalid ID format"))
		return
	}

	r.Header.Set("Accept", "application/json")

	if task, err := s.TaskService.FindTaskByID(r.Context(), taskID); err != nil {
		Error(w, r, fmt.Errorf("error deleting task by ID=%d: %v", taskID, err))
		return
	} else if err = s.TaskService.UnattachContributor(r.Context(), task, contributorID); err != nil {
		Error(w, r, fmt.Errorf("error contributor ID for task with ID=%d: %v", taskID, err))
		return
	} else if err = json.Encode(task, w); err != nil {
		LogError(r, err)
		return
	}
}
