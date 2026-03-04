package web

import (
	"context"
	"embed"
	"net/http"
	"strings"

	"go.uber.org/zap"

	"neuralclaw/internal/agent"
	"neuralclaw/internal/observability"
	"neuralclaw/internal/taskstore"
)

//go:embed templates/*
var templatesFS embed.FS

type Server struct {
	addr       string
	authToken  string
	scope      string // The active scope constraints out of the CLI
	store      taskstore.Store
	dispatcher *agent.Dispatcher
	broker     *SSEBroker
}

func NewServer(addr, authToken, scope string, store taskstore.Store, dispatcher *agent.Dispatcher) *Server {
	return &Server{
		addr:       addr,
		authToken:  authToken,
		scope:      scope,
		store:      store,
		dispatcher: dispatcher,
		broker:     NewSSEBroker(),
	}
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	// UI Routes
	mux.HandleFunc("/web", s.handleDashboard)
	mux.HandleFunc("/web/tasks", s.handleTasks)
	mux.HandleFunc("/web/runs", s.handleRuns)
	mux.HandleFunc("/web/runs/", s.handleRunDetail)

	// API Routes (Actions / JSON)
	mux.HandleFunc("POST /api/tasks", s.handleCreateTask)
	mux.HandleFunc("POST /api/tasks/", s.handleTaskAction) // /api/tasks/{id}/dispatch
	mux.HandleFunc("/api/runs/", s.handleRunAPI)           // /api/runs/{id} and /api/runs/{id}/events

	// Global Middleware Setup
	handler := s.authMiddleware(mux)

	observability.Logger.Info("Starting Web GUI",
		zap.String("addr", s.addr),
		zap.String("scope", s.scope),
		zap.Bool("auth_enabled", s.authToken != ""),
	)

	return http.ListenAndServe(s.addr, handler)
}

// authMiddleware enforces a dev token via `X-Auth-Token` header or `?token=` query param
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if s.authToken != "" {
			token := r.Header.Get("X-Auth-Token")
			if token == "" {
				token = r.URL.Query().Get("token")
			}
			// Let HTMX redirect to an unauthorized page or block
			if token != s.authToken {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
		}

		// Security: we strictly enforce Single-Scope isolation per UI server process.
		// A user looking at the UI shouldn't be allowed to manipulate other scopes.
		r = r.WithContext(SetScopeConstraint(r.Context(), s.scope))

		next.ServeHTTP(w, r)
	})
}

// Scope context helpers
type contextKey string

const scopeKey contextKey = "scope"

func SetScopeConstraint(ctx context.Context, scope string) context.Context {
	return context.WithValue(ctx, scopeKey, scope)
}

func GetScopeConstraint(ctx context.Context) string {
	if s, ok := ctx.Value(scopeKey).(string); ok {
		return s
	}
	return "global"
}

// Simple path parsing helper
func shiftPath(p string) (head, tail string) {
	p = strings.TrimPrefix(p, "/")
	i := strings.Index(p, "/")
	if i < 0 {
		return p, "/"
	}
	return p[:i], p[i:]
}
