package memory_test

import (
	"context"
	"testing"

	"neuralclaw/internal/agent"
	"neuralclaw/internal/memory"
	"neuralclaw/internal/observability"
	"neuralclaw/pkg/types"
)

// mockMemoryStore implements types.MemoryStore for testing
type mockMemoryStore struct {
	lastQuery types.Query
}

func (m *mockMemoryStore) Upsert(ctx context.Context, items []types.MemoryItem) error {
	return nil
}

func (m *mockMemoryStore) Query(ctx context.Context, q types.Query) (types.QueryResult, error) {
	m.lastQuery = q
	return types.QueryResult{
		Items:      []types.MemoryItem{},
		TotalFound: 0,
	}, nil
}

func (m *mockMemoryStore) Delete(ctx context.Context, filter types.Filter) error {
	return nil
}

func (m *mockMemoryStore) Health(ctx context.Context) error {
	return nil
}

func TestHybridRouterScopeIsolation(t *testing.T) {
	observability.InitLogger("error") // suppress logs for test

	store := &mockMemoryStore{}
	embedder := agent.NewDummyEmbedder(10)
	router := memory.NewRouter(store, embedder)

	ctx := context.Background()
	_, err := router.Search(ctx, "test query", "project:secret", 5)
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}

	if store.lastQuery.Scope != "project:secret" {
		t.Errorf("Expected scope 'project:secret', got %s", store.lastQuery.Scope)
	}

	if len(store.lastQuery.Vector) != 10 {
		t.Errorf("Expected vector of length 10, got %d", len(store.lastQuery.Vector))
	}
}
