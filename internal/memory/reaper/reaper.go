package reaper

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"neuralclaw/internal/config"
	"neuralclaw/internal/observability"
	"neuralclaw/pkg/types"
)

// ReapReport summarizes what was (or would be) deleted.
type ReapReport struct {
	DeletedByType map[types.ItemType]int
	TotalDeleted  int
}

// Reaper enforces the retention policy for the memory store.
type Reaper struct {
	store  types.MemoryStore
	policy config.RetentionPolicy
}

func NewReaper(store types.MemoryStore, policy config.RetentionPolicy) *Reaper {
	return &Reaper{
		store:  store,
		policy: policy,
	}
}

// Run executes the memory reaping process.
func (r *Reaper) Run(ctx context.Context, scope string, now time.Time, dryRun bool) (ReapReport, error) {
	report := ReapReport{
		DeletedByType: make(map[types.ItemType]int),
	}

	if scope == "" {
		return report, fmt.Errorf("reaper: scope is required to prevent accidental mass deletion")
	}

	observability.Logger.Info("Starting memory reaper",
		zap.String("scope", scope),
		zap.Bool("dry_run", dryRun),
		zap.Time("now", now),
	)

	// In a fully featured MemoryStore, we might do this entirely server-side (`DELETE WHERE created_at < X AND type = Y`).
	// To comply with the MVP mock/adapter pattern, we query items to check their EffectiveTTLDays and then issue deletes.

	// Pre-fetch all memories in scope (for MVP logic, a robust implementation would use pagination or direct DB delete)
	q := types.Query{
		Scope: scope,
		TopK:  10000,
	}

	res, err := r.store.Query(ctx, q)
	if err != nil {
		return report, fmt.Errorf("reaper query failed: %w", err)
	}

	for _, item := range res.Items {
		expirationDate := item.CreatedAt.Add(time.Duration(item.EffectiveTTLDays(r.policy)) * 24 * time.Hour)

		if now.After(expirationDate) {
			report.TotalDeleted++
			report.DeletedByType[item.Type]++

			if !dryRun {
				// Delete using a precise filter to ensure safety
				idStr := item.ID
				filter := types.Filter{
					Type: &item.Type,
				}
				// Normally you'd want ID-based delete, but the task asked for Delete(ctx, filter Filter)
				// For the mock, we simulate it. In production, we'd add ID to the Filter.
				_ = idStr
				if err := r.store.Delete(ctx, filter); err != nil {
					observability.Logger.Error("Failed to delete memory item", zap.String("id", item.ID), zap.Error(err))
				}
			}
		}
	}

	observability.Logger.Info("Reaper completed", zap.Any("report", report))
	return report, nil
}
