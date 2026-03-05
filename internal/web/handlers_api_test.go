package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"neuralclaw/internal/memory"
	"neuralclaw/internal/observability"
	"neuralclaw/pkg/types"
)

type fakeEmbedder struct{}

func (f *fakeEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	out := make([][]float32, 0, len(texts))
	for range texts {
		out = append(out, []float32{1, 0, 0})
	}
	return out, nil
}

type fakeMemoryStore struct {
	items map[string]types.MemoryItem
}

func (m *fakeMemoryStore) Upsert(ctx context.Context, items []types.MemoryItem) error {
	if m.items == nil {
		m.items = map[string]types.MemoryItem{}
	}
	for _, item := range items {
		m.items[item.ID] = item
	}
	return nil
}

func (m *fakeMemoryStore) Query(ctx context.Context, q types.Query) (types.QueryResult, error) {
	item := m.items["m-1"]
	return types.QueryResult{
		Items: []types.MemoryItem{item},
		ExplainedHits: []types.ExplainedHit{
			{
				Item: item,
				Score: types.ScoreBreakdown{
					VectorScore: 0.9,
					BM25Score:   0.7,
					FinalScore:  0.8,
				},
			},
		},
		TotalFound: 1,
	}, nil
}

func (m *fakeMemoryStore) Delete(ctx context.Context, filter types.Filter) error { return nil }
func (m *fakeMemoryStore) Health(ctx context.Context) error                      { return nil }

func (m *fakeMemoryStore) GetMemory(ctx context.Context, id string) (types.MemoryItem, bool, error) {
	item, ok := m.items[id]
	return item, ok, nil
}

func (m *fakeMemoryStore) ListMemories(ctx context.Context, scope string, limit int) ([]types.MemoryItem, error) {
	out := make([]types.MemoryItem, 0, len(m.items))
	for _, item := range m.items {
		if item.Scope == scope {
			out = append(out, item)
		}
	}
	return out, nil
}

func TestMemoryExplainAPI(t *testing.T) {
	observability.InitLogger("error")

	memStore := &fakeMemoryStore{
		items: map[string]types.MemoryItem{
			"m-1": {ID: "m-1", Scope: "global", Text: "hello world", Type: types.ItemTypeRaw},
		},
	}
	s := &Server{
		memoryStore:     memStore,
		memoryInspector: memStore,
		memoryRouter:    memory.NewRouter(memStore, &fakeEmbedder{}),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/memory/m-1/explain", nil)
	req = req.WithContext(SetScopeConstraint(req.Context(), "global"))
	rr := httptest.NewRecorder()

	s.handleMemoryDetail(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var out []types.ExplainedHit
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if len(out) != 1 || out[0].Item.ID != "m-1" {
		t.Fatalf("unexpected explain payload: %+v", out)
	}
}

func TestMemoryEvidenceAPI(t *testing.T) {
	observability.InitLogger("error")

	memStore := &fakeMemoryStore{
		items: map[string]types.MemoryItem{
			"root":   {ID: "root", Scope: "global", Type: types.ItemTypeDailySummary, DerivedFrom: []string{"leaf"}},
			"leaf":   {ID: "leaf", Scope: "global", Type: types.ItemTypeRaw, EvidenceOf: []string{"root"}},
			"ignore": {ID: "ignore", Scope: "project:x", Type: types.ItemTypeRaw},
		},
	}
	s := &Server{
		memoryStore:     memStore,
		memoryInspector: memStore,
		memoryRouter:    memory.NewRouter(memStore, &fakeEmbedder{}),
	}

	req := httptest.NewRequest(http.MethodGet, "/api/memory/root/evidence", nil)
	req = req.WithContext(SetScopeConstraint(req.Context(), "global"))
	rr := httptest.NewRecorder()

	s.handleMemoryDetail(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON response: %v", err)
	}
	if out["item"] == nil {
		t.Fatalf("missing item in evidence payload: %+v", out)
	}
}
