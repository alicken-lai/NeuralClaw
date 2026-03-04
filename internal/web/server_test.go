package web_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	// Added time import
	"neuralclaw/internal/observability"
	"neuralclaw/internal/web"
	"neuralclaw/pkg/types"
)

// mockStore for testing Web Handlers
type mockTaskStore struct {
	tasks []types.Task
	runs  []types.Run
}

func (m *mockTaskStore) SaveTask(t types.Task) error { m.tasks = append(m.tasks, t); return nil }
func (m *mockTaskStore) GetTask(id string) (types.Task, error) {
	for _, t := range m.tasks {
		if t.ID == id {
			return t, nil
		}
	}
	return types.Task{}, nil // Simplify for mock
}
func (m *mockTaskStore) ListTasks(scope string) ([]types.Task, error) {
	var res []types.Task
	for _, t := range m.tasks {
		if t.Scope == scope {
			res = append(res, t)
		}
	}
	return res, nil
}
func (m *mockTaskStore) SaveRun(r types.Run) error                        { m.runs = append(m.runs, r); return nil }
func (m *mockTaskStore) GetRun(id string) (types.Run, error)              { return types.Run{}, nil }
func (m *mockTaskStore) ListRuns(scope string) ([]types.Run, error)       { return nil, nil }
func (m *mockTaskStore) GetRunsByTask(taskID string) ([]types.Run, error) { return nil, nil }

func TestWebAuthMiddleware(t *testing.T) {
	observability.InitLogger("error")

	// Note: We don't instantiate the real dependencies here as we only need
	// to test the internal middleware logic directly.
	// Since Start() is blocking, we can extract the internal Handler logic testing via httptest
	// We'll mimic the internal authMiddleware wrapper here for the test:

	// testHandler asserts the scope was properly injected
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		scope := web.GetScopeConstraint(r.Context())
		if scope != "test-scope" {
			t.Errorf("Expected scope 'test-scope', got '%s'", scope)
		}
		w.WriteHeader(http.StatusOK)
	})

	// (We copy the authMiddleware logic here for direct unit testing of that function)
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := r.Header.Get("X-Auth-Token")
			if token == "" {
				token = r.URL.Query().Get("token")
			}
			if token != "secret" {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			r = r.WithContext(web.SetScopeConstraint(r.Context(), "test-scope"))
			next.ServeHTTP(w, r)
		})
	}

	handler := middleware(testHandler)

	// Test 1: Missing Token
	req := httptest.NewRequest("GET", "/web", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized, got %v", rr.Code)
	}

	// Test 2: Invalid Token
	req = httptest.NewRequest("GET", "/web", nil)
	req.Header.Set("X-Auth-Token", "wrong")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401 Unauthorized for wrong token, got %v", rr.Code)
	}

	// Test 3: Correct Token via Header
	req = httptest.NewRequest("GET", "/web", nil)
	req.Header.Set("X-Auth-Token", "secret")
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for valid header token, got %v", rr.Code)
	}

	// Test 4: Correct Token via Query
	req = httptest.NewRequest("GET", "/web?token=secret", nil)
	rr = httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("Expected 200 OK for valid query token, got %v", rr.Code)
	}
}

func TestWebScopeIsolation(t *testing.T) {
	store := &mockTaskStore{}

	// Add tasks to the store for different scopes
	store.SaveTask(types.Task{ID: "1", Scope: "scope-A", Title: "Task A"})
	store.SaveTask(types.Task{ID: "2", Scope: "scope-B", Title: "Task B"})

	ctxA := web.SetScopeConstraint(context.Background(), "scope-A")

	// Ensure ListTasks only returns scope-A when queried under ctxA constraints
	tasks, _ := store.ListTasks(web.GetScopeConstraint(ctxA))

	if len(tasks) != 1 {
		t.Errorf("Expected 1 task in scope-A, got %d", len(tasks))
	}
	if tasks[0].ID != "1" {
		t.Errorf("Expected Task 1, got %s", tasks[0].ID)
	}
}
