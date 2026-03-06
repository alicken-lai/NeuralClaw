package dmn

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"neuralclaw/internal/config"
	storePkg "neuralclaw/internal/memory/store"
	"neuralclaw/internal/observability"
	"neuralclaw/internal/security"
	"neuralclaw/pkg/types"
)

// Pipeline encapsulates the daily reflection and memory consolidation logic.
type Pipeline struct {
	store    types.MemoryStore
	embedder *storePkg.Embedder
	cfg      config.RetrievalConfig
	guard    *security.Guard
}

func NewPipeline(memStore types.MemoryStore, embedder *storePkg.Embedder, cfg config.RetrievalConfig, guard *security.Guard) *Pipeline {
	return &Pipeline{store: memStore, embedder: embedder, cfg: cfg, guard: guard}
}

// Run executes the DMN for a given scope and date.
func (p *Pipeline) Run(ctx context.Context, scope, date string) error {
	observability.Logger.Info("Starting DMN daily reflection",
		zap.String("scope", scope),
		zap.String("date", date),
	)

	// 1. Time-window fetch
	// Assuming date format is YYYY-MM-DD
	targetDate, err := time.Parse("2006-01-02", date)
	if err != nil {
		return fmt.Errorf("invalid date format: %w", err)
	}

	start := targetDate.Truncate(24 * time.Hour)
	end := start.Add(24 * time.Hour)

	// Simulate querying the store for memory items between start and end
	q := types.Query{
		Scope:           scope,
		TimeWindowStart: &start,
		TimeWindowEnd:   &end,
		Text:            "", // Empty text to pull all within time block
		TopK:            1000,
	}

	res, err := p.store.Query(ctx, q)
	if err != nil {
		return fmt.Errorf("DMN fetch failed: %w", err)
	}

	observability.Logger.Info("DMN fetched items", zap.Int("count", res.TotalFound))

	// Collect source memory IDs for causal linking (Memory DAG)
	sourceIDs := make([]string, 0, len(res.Items))
	for _, item := range res.Items {
		sourceIDs = append(sourceIDs, item.ID)
	}

	// 2. Cluster & Summarize (Stub: assuming we found items and summarized into one string)
	summaryText := fmt.Sprintf("Daily summary of %d items for %s", res.TotalFound, date)
	conceptLink := fmt.Sprintf("Concept node edges for %s", date)

	// 3. Writeback to MemoryStore using explicit ItemTypes
	summaryItem := types.MemoryItem{
		ID:          uuid.New().String(),
		Type:        types.ItemTypeDailySummary,
		Scope:       scope,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		SourceTime:  targetDate, // The day this summary represents
		Modality:    "text",
		BM25Text:    summaryText,
		CausalLinks: sourceIDs, // Living Memory: trace back to source memories
		DerivedFrom: sourceIDs, // [NEW] Evidence chain backlink
		// Vector: embedding of summaryText ...
	}

	edgesItem := types.MemoryItem{
		ID:          uuid.New().String(),
		Type:        types.ItemTypeConceptEdges,
		Scope:       scope,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		SourceTime:  targetDate,
		Modality:    "graph",
		BM25Text:    conceptLink,
		CausalLinks: sourceIDs, // Living Memory: trace back to source memories
		DerivedFrom: sourceIDs, // [NEW] Evidence chain backlink
	}

	itemsToUpsert := make([]types.MemoryItem, 0, 2)
	for _, item := range []types.MemoryItem{summaryItem, edgesItem} {
		if p.guard != nil {
			validation, _, err := p.guard.ValidateMemory(scope, "dmn", item)
			if err != nil {
				return fmt.Errorf("DMN security validation failed: %w", err)
			}
			if !validation.Allowed {
				observability.Logger.Warn("DMN writeback item quarantined",
					zap.String("memory_id", item.ID),
					zap.String("risk", string(validation.RiskLevel)),
				)
				continue
			}
		}
		itemsToUpsert = append(itemsToUpsert, item)
	}
	if len(itemsToUpsert) == 0 {
		observability.Logger.Warn("DMN produced no writeable memories after security validation", zap.String("scope", scope))
		return nil
	}

	err = p.store.Upsert(ctx, itemsToUpsert)
	if err != nil {
		return fmt.Errorf("DMN writeback failed: %w", err)
	}

	observability.Logger.Info("DMN consolidated and saved new memories.")
	return nil
}
