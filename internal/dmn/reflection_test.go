package dmn_test

import (
	"context"
	"testing"

	"neuralclaw/internal/config"
	"neuralclaw/internal/dmn"
	"neuralclaw/internal/observability"
	"neuralclaw/pkg/types"
)

// mockMemoryStore implements types.MemoryStore for testing
type mockMemoryStore struct {
	upsertedItems []types.MemoryItem
	lastQuery     types.Query
}

func (m *mockMemoryStore) Upsert(ctx context.Context, items []types.MemoryItem) error {
	m.upsertedItems = append(m.upsertedItems, items...)
	return nil
}

func (m *mockMemoryStore) Query(ctx context.Context, q types.Query) (types.QueryResult, error) {
	m.lastQuery = q
	return types.QueryResult{
		Items:      []types.MemoryItem{},
		TotalFound: 5, // mock that we found some memories to reflect on
	}, nil
}

func (m *mockMemoryStore) Delete(ctx context.Context, filter types.Filter) error {
	return nil
}

func (m *mockMemoryStore) Health(ctx context.Context) error {
	return nil
}

func TestDMNPipelineRun(t *testing.T) {
	observability.InitLogger("error")

	mockStore := &mockMemoryStore{}
	pipeline := dmn.NewPipeline(mockStore, nil, config.RetrievalConfig{})

	ctx := context.Background()
	date := "2024-05-10"
	scope := "global"

	err := pipeline.Run(ctx, scope, date)
	if err != nil {
		t.Fatalf("DMN Run failed: %v", err)
	}

	// Verify query constraints
	if mockStore.lastQuery.TimeWindowStart == nil || mockStore.lastQuery.TimeWindowEnd == nil {
		t.Fatalf("Expected time window in query but got nil")
	}

	// Verify writebacks
	if len(mockStore.upsertedItems) != 2 {
		t.Errorf("Expected 2 items upserted, got %d", len(mockStore.upsertedItems))
	}

	hasSummary := false
	hasEdges := false
	for _, item := range mockStore.upsertedItems {
		if item.Type == types.ItemTypeDailySummary {
			hasSummary = true
		}
		if item.Type == types.ItemTypeConceptEdges {
			hasEdges = true
		}
		if item.Scope != scope {
			t.Errorf("Writeback item scope mismatch. Expected %s, got %s", scope, item.Scope)
		}
	}

	if !hasSummary || !hasEdges {
		t.Errorf("Expected DMN to write both daily_summary and concept_edges")
	}
}
