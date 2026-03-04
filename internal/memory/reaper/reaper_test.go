package reaper_test

import (
	"context"
	"testing"
	"time"

	"neuralclaw/internal/config"
	"neuralclaw/internal/memory/reaper"
	"neuralclaw/internal/observability"
	"neuralclaw/pkg/types"
)

// mockMemoryStore implements types.MemoryStore for testing
type mockMemoryStore struct {
	items         []types.MemoryItem
	deletedCounts int
}

func (m *mockMemoryStore) Upsert(ctx context.Context, items []types.MemoryItem) error {
	m.items = append(m.items, items...)
	return nil
}

func (m *mockMemoryStore) Query(ctx context.Context, q types.Query) (types.QueryResult, error) {
	// Simple mock: return all items that match the scope
	var matched []types.MemoryItem
	for _, item := range m.items {
		if item.Scope == q.Scope {
			matched = append(matched, item)
		}
	}
	return types.QueryResult{
		Items:      matched,
		TotalFound: len(matched),
	}, nil
}

func (m *mockMemoryStore) Delete(ctx context.Context, filter types.Filter) error {
	m.deletedCounts++
	return nil
}

func (m *mockMemoryStore) Health(ctx context.Context) error {
	return nil
}

func TestEffectiveTTLDays(t *testing.T) {
	policy := config.DefaultRetentionPolicy()

	tests := []struct {
		itemType types.ItemType
		override *int
		expected int
	}{
		{types.ItemTypeRaw, nil, 90},
		{types.ItemTypeDailySummary, nil, 730},
		{types.ItemTypeWeeklySummary, nil, 1825},
		{types.ItemTypeMonthlySummary, nil, 3650},
		{types.ItemTypeConceptEdges, nil, 1825},
		{types.ItemTypeAnomalies, nil, 730},
		{types.ItemTypeRaw, ptr(10), 10}, // testing override
	}

	for _, tt := range tests {
		item := types.MemoryItem{
			Type:    tt.itemType,
			TTLDays: tt.override,
		}
		got := item.EffectiveTTLDays(policy)
		if got != tt.expected {
			t.Errorf("EffectiveTTLDays(%s, curr=%v) = %d; want %d", tt.itemType, tt.override, got, tt.expected)
		}
	}
}

func TestReaperRunCutoffsAndScopeIsolation(t *testing.T) {
	observability.InitLogger("error")

	policy := config.DefaultRetentionPolicy()
	store := &mockMemoryStore{}

	now := time.Now()

	// Add items to Scope A
	store.Upsert(context.Background(), []types.MemoryItem{
		{ID: "1", Scope: "project:A", Type: types.ItemTypeRaw, CreatedAt: now.Add(-100 * 24 * time.Hour)},          // Expired (TTL 90)
		{ID: "2", Scope: "project:A", Type: types.ItemTypeRaw, CreatedAt: now.Add(-50 * 24 * time.Hour)},           // Kept
		{ID: "3", Scope: "project:A", Type: types.ItemTypeDailySummary, CreatedAt: now.Add(-800 * 24 * time.Hour)}, // Expired (TTL 730)
	})

	// Add items to Scope B (these should NOT be touched when reaping Scope A)
	store.Upsert(context.Background(), []types.MemoryItem{
		{ID: "4", Scope: "project:B", Type: types.ItemTypeRaw, CreatedAt: now.Add(-100 * 24 * time.Hour)}, // Expired if reaped, but wrong scope
	})

	r := reaper.NewReaper(store, policy)

	// Test 1: Dry Run on Scope A
	report, err := r.Run(context.Background(), "project:A", now, true)
	if err != nil {
		t.Fatalf("Reaper dry-run failed: %v", err)
	}
	if report.TotalDeleted != 2 {
		t.Errorf("Dry run expected to find 2 expired items, got %d", report.TotalDeleted)
	}
	if store.deletedCounts != 0 {
		t.Errorf("Dry run should not call Delete, but it was called %d times", store.deletedCounts)
	}

	// Test 2: Actual Run on Scope A
	report, err = r.Run(context.Background(), "project:A", now, false)
	if err != nil {
		t.Fatalf("Reaper run failed: %v", err)
	}
	if report.TotalDeleted != 2 {
		t.Errorf("Run expected to delete 2 items, got %d", report.TotalDeleted)
	}
	if store.deletedCounts != 2 {
		t.Errorf("Expected store.Delete to be called 2 times, got %d", store.deletedCounts)
	}
	if report.DeletedByType[types.ItemTypeRaw] != 1 {
		t.Errorf("Expected 1 raw item deleted")
	}
	if report.DeletedByType[types.ItemTypeDailySummary] != 1 {
		t.Errorf("Expected 1 daily summary item deleted")
	}
}

// helper
func ptr(i int) *int {
	return &i
}
