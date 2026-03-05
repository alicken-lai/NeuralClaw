package web

import (
	"html/template"
	"net/http"

	"neuralclaw/internal/observability"
	"neuralclaw/pkg/types"

	"go.uber.org/zap"
)

func (s *Server) renderTemplate(w http.ResponseWriter, tmplName string, data interface{}) {
	tmplPath := "templates/" + tmplName
	layoutPath := "templates/layout.html"

	t, err := template.ParseFS(templatesFS, layoutPath, tmplPath)
	if err != nil {
		observability.Logger.Error("Template parse error", zap.Error(err), zap.String("tmpl", tmplName))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	if err := t.ExecuteTemplate(w, "layout", data); err != nil {
		observability.Logger.Error("Template execute error", zap.Error(err))
	}
}

func (s *Server) handleDashboard(w http.ResponseWriter, r *http.Request) {
	scope := GetScopeConstraint(r.Context())

	tasks, _ := s.store.ListTasks(scope)
	runs, _ := s.store.ListRuns(scope)

	data := struct {
		Scope          string
		TotalTasks     int
		TotalRuns      int
		RecentMemories []types.MemoryItem
	}{
		Scope:      scope,
		TotalTasks: len(tasks),
		TotalRuns:  len(runs),
	}
	if s.memoryInspector != nil {
		recent, err := s.memoryInspector.ListMemories(r.Context(), scope, 12)
		if err == nil {
			data.RecentMemories = recent
		}
	}

	s.renderTemplate(w, "dashboard.html", data)
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) {
	scope := GetScopeConstraint(r.Context())
	tasks, _ := s.store.ListTasks(scope)

	data := struct {
		Scope string
		Tasks interface{}
	}{
		Scope: scope,
		Tasks: tasks,
	}

	s.renderTemplate(w, "tasks.html", data)
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) {
	scope := GetScopeConstraint(r.Context())
	runs, _ := s.store.ListRuns(scope)

	data := struct {
		Scope string
		Runs  interface{}
	}{
		Scope: scope,
		Runs:  runs,
	}

	s.renderTemplate(w, "runs.html", data)
}

func (s *Server) handleRunDetail(w http.ResponseWriter, r *http.Request) {
	scope := GetScopeConstraint(r.Context())
	runID := r.URL.Path[len("/web/runs/"):]

	run, err := s.store.GetRun(runID)
	if err != nil || run.Scope != scope {
		http.Error(w, "Run not found", http.StatusNotFound)
		return
	}

	data := struct {
		Scope string
		Run   interface{}
	}{
		Scope: scope,
		Run:   run,
	}

	s.renderTemplate(w, "run_detail.html", data)
}
