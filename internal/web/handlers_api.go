package web

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
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

func (s *Server) handleRunAction(w http.ResponseWriter, r *http.Request) {
	scope := GetScopeConstraint(r.Context())

	// Example Path: /api/runs/{id}/cancel
	path := r.URL.Path[len("/api/runs/"):]

	// simple manual routing
	idSize := 36 // UUID
	if len(path) < idSize {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}
	id := path[:idSize]
	action := path[idSize:]

	run, err := s.store.GetRun(id)
	if err != nil || run.Scope != scope {
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}

	if action == "/cancel" {
		// Update Store
		run.Status = types.TaskStatusCanceled
		s.store.SaveRun(run)

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

type evidenceNode struct {
	Item        types.MemoryItem `json:"item"`
	DerivedFrom []evidenceNode   `json:"derived_from,omitempty"`
	EvidenceOf  []evidenceNode   `json:"evidence_of,omitempty"`
}

func (s *Server) handleMemoryDetail(w http.ResponseWriter, r *http.Request) {
	scope := GetScopeConstraint(r.Context())
	if s.memoryInspector == nil {
		http.Error(w, "Memory store does not support detail lookup", http.StatusNotImplemented)
		return
	}

	path := strings.TrimPrefix(r.URL.Path, "/api/memory/")
	parts := strings.Split(strings.Trim(path, "/"), "/")
	if len(parts) != 2 || parts[0] == "" {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}
	memoryID := parts[0]
	action := parts[1]

	item, ok, err := s.memoryInspector.GetMemory(r.Context(), memoryID)
	if err != nil {
		http.Error(w, "Failed to load memory item", http.StatusInternalServerError)
		return
	}
	if !ok || item.Scope != scope {
		http.Error(w, "Memory item not found", http.StatusNotFound)
		return
	}

	switch action {
	case "explain":
		s.handleMemoryExplain(w, r, item, scope)
	case "evidence":
		chain, err := s.buildEvidenceChain(r.Context(), memoryID, scope, map[string]bool{}, 0)
		if err != nil {
			http.Error(w, "Failed to build evidence chain", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(chain)
	default:
		http.Error(w, "Action not found", http.StatusNotFound)
	}
}

func (s *Server) handleMemoryExplain(w http.ResponseWriter, r *http.Request, item types.MemoryItem, scope string) {
	if s.memoryRouter == nil {
		http.Error(w, "Explain router not available", http.StatusNotImplemented)
		return
	}

	queryText := strings.TrimSpace(item.Text)
	if queryText == "" {
		queryText = strings.TrimSpace(item.BM25Text)
	}
	if queryText == "" {
		http.Error(w, "Memory item has no searchable text", http.StatusBadRequest)
		return
	}

	result, err := s.memoryRouter.SearchExplain(r.Context(), queryText, scope, 10, true)
	if err != nil {
		http.Error(w, "Explain query failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result.ExplainedHits)
}

func (s *Server) buildEvidenceChain(ctx context.Context, id, scope string, visited map[string]bool, depth int) (evidenceNode, error) {
	const maxDepth = 20
	if depth > maxDepth {
		return evidenceNode{}, errors.New("evidence chain exceeded max depth")
	}

	item, ok, err := s.memoryInspector.GetMemory(ctx, id)
	if err != nil {
		return evidenceNode{}, err
	}
	if !ok || item.Scope != scope {
		return evidenceNode{}, errors.New("evidence item not found")
	}

	node := evidenceNode{Item: item}
	if visited[id] {
		return node, nil
	}
	visited[id] = true
	defer delete(visited, id)

	for _, parentID := range item.DerivedFrom {
		child, err := s.buildEvidenceChain(ctx, parentID, scope, visited, depth+1)
		if err == nil {
			node.DerivedFrom = append(node.DerivedFrom, child)
		}
	}
	for _, childID := range item.EvidenceOf {
		child, err := s.buildEvidenceChain(ctx, childID, scope, visited, depth+1)
		if err == nil {
			node.EvidenceOf = append(node.EvidenceOf, child)
		}
	}

	return node, nil
}
