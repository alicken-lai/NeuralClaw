package web

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"neuralclaw/internal/agent"
	"neuralclaw/internal/observability"
	"neuralclaw/pkg/types"
)

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	scope := GetScopeConstraint(r.Context())

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid form", http.StatusBadRequest)
		return
	}

	t := types.Task{
		ID:        uuid.New().String(),
		Title:     r.FormValue("title"),
		Prompt:    r.FormValue("prompt"),
		Scope:     scope,
		Priority:  1, // default
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Status:    types.TaskStatusQueued,
	}

	if err := s.store.SaveTask(t); err != nil {
		http.Error(w, "Save failed", http.StatusInternalServerError)
		return
	}

	observability.Logger.Info("Created task via Web", zap.String("id", t.ID))

	// If HTMX requested this, return a redirect or a snippet, we'll just redirect to tasks
	w.Header().Set("HX-Redirect", "/web/tasks")
	http.Redirect(w, r, "/web/tasks", http.StatusSeeOther)
}

func (s *Server) handleTaskAction(w http.ResponseWriter, r *http.Request) {
	scope := GetScopeConstraint(r.Context())

	// Example Path: /api/tasks/{id}/dispatch
	path := r.URL.Path[len("/api/tasks/"):]

	// simple manual routing
	idSize := 36 // UUID
	if len(path) < idSize {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id := path[:idSize]
	action := path[idSize:]

	task, err := s.store.GetTask(id)
	if err != nil || task.Scope != scope {
		http.Error(w, "Task not found", http.StatusNotFound)
		return
	}

	if action == "/dispatch" {
		req := agent.DispatchRequest{
			TaskID:   task.ID,
			Scope:    task.Scope,
			Prompt:   task.Prompt,
			Priority: task.Priority,
		}

		run, err := s.dispatcher.Dispatch(r.Context(), req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Update Store
		s.store.SaveRun(run)
		task.Status = types.TaskStatusRunning
		runID := run.ID
		task.RunID = &runID
		s.store.SaveTask(task)

		w.Header().Set("HX-Redirect", "/web/runs/"+run.ID)
		w.WriteHeader(http.StatusOK)
		return
	}

	http.Error(w, "Action not found", http.StatusNotFound)
}

func (s *Server) handleRunAPI(w http.ResponseWriter, r *http.Request) {
	scope := GetScopeConstraint(r.Context())
	path := r.URL.Path[len("/api/runs/"):]

	idSize := 36
	if len(path) < idSize {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id := path[:idSize]
	sub := path[idSize:]

	run, err := s.store.GetRun(id)
	if err != nil || run.Scope != scope {
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}

	if sub == "/events" {
		// SSE Broker handles it
		s.broker.ServeHTTP(w, r, run.ID)
		return
	}

	// Default: return run JSON
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(run)
}
