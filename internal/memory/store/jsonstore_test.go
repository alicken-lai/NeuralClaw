package store_test

import (
	"context"
	"os"
	"testing"
	"time"

	"neuralclaw/internal/config"
	"neuralclaw/internal/memory/store"
	"neuralclaw/internal/observability"
	"neuralclaw/pkg/types"
)

func TestJSONStore_BasicInitAndUpsert(t *testing.T) {
	observability.InitLogger("error")

	tmpDir, err := os.MkdirTemp("", "jsonstore_test")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	embedder := store.NewEmbedder(config.EmbeddingConfig{
		Dimensions: 3,
	})
	db, err := store.NewJSONStore(tmpDir, 3, embedder, config.RetrievalConfig{})
	if err != nil {
		t.Fatalf("Failed to init store: %v", err)
	}

	ctx := context.Background()

	item := types.MemoryItem{
		ID:     "test-1",
		Text:   "This is a test memory",
		Scope:  "global",
		Type:   types.ItemTypeRaw,
		Vector: []float32{1.0, 0.0, 0.0},
	}

	if err := db.Upsert(ctx, []types.MemoryItem{item}); err != nil {
		t.Fatalf("Upsert failed: %v", err)
	}

	// Verify time-based query
	now := time.Now()
	start := now.Add(-1 * time.Hour)
	end := now.Add(1 * time.Hour)

	query := types.Query{
		Text:            "", // Force time-only fetch
		TimeWindowStart: &start,
		TimeWindowEnd:   &end,
		Scope:           "global",
	}

	res, err := db.Query(ctx, query)
	if err != nil {
		t.Fatalf("Query failed: %v", err)
	}

	if res.TotalFound != 1 || len(res.Items) != 1 {
		t.Fatalf("Expected 1 item, got %d", res.TotalFound)
	}
	if res.Items[0].ID != "test-1" {
		t.Errorf("Expected test-1, got %s", res.Items[0].ID)
	}
}

func TestCosineSimilarity(t *testing.T) {
	// The function is internal, relying on integration through hybrid search.
	// We will create a dummy embedder to bypass HTTP calls.
}
