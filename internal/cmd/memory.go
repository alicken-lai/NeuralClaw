package cmd

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"time"

	"neuralclaw/internal/config"
	"neuralclaw/internal/memory"
	"neuralclaw/internal/memory/reaper"
	"neuralclaw/internal/memory/store"
	"neuralclaw/internal/observability"
)

var (
	memoryScope   string
	memoryQuery   string
	memoryExplain bool
	memoryTopK    int
	reapScope     string
	reapDryRun    bool
	reapNowStr    string
)

var memoryCmd = &cobra.Command{
	Use:   "memory",
	Short: "Manage and query the memory store",
}

var memoryQueryCmd = &cobra.Command{
	Use:   "query",
	Short: "Query the memory store (hybrid vector + BM25)",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		observability.Logger.Info("Querying memory", zap.String("scope", memoryScope), zap.String("query", memoryQuery))

		embedder := store.NewEmbedder(config.GlobalConfig.Memory.Embedding)
		memClient, err := store.NewJSONStore(
			config.GlobalConfig.Memory.DBPath,
			config.GlobalConfig.Memory.Embedding.Dimensions,
			embedder,
			config.GlobalConfig.Memory.Retrieval,
		)
		if err != nil {
			observability.Logger.Fatal("Failed to init Memory Store", zap.Error(err))
		}

		router := memory.NewRouter(memClient, embedder)

		res, err := router.SearchExplain(ctx, memoryQuery, memoryScope, memoryTopK, memoryExplain)
		if err != nil {
			observability.Logger.Error("Memory query failed", zap.Error(err))
			return
		}

		observability.Logger.Info("Memory query returned",
			zap.Int("items_found", res.TotalFound),
			zap.Int64("took_ms", res.TookMillis),
		)

		if memoryExplain && len(res.ExplainedHits) > 0 {
			cmd.Printf("\n%-5s %-12s %-14s %7s %7s %7s %7s %7s %7s\n",
				"Rank", "ID", "Type", "Vec", "BM25", "Time", "Access", "Final", "Rerank")
			cmd.Println("-----+------------+--------------+-------+-------+-------+-------+-------+-------")
			for i, hit := range res.ExplainedHits {
				shortID := hit.Item.ID
				if len(shortID) > 12 {
					shortID = shortID[:12]
				}
				rerankStr := "  n/a"
				if hit.Score.RerankScore != nil {
					rerankStr = fmt.Sprintf("%7.4f", *hit.Score.RerankScore)
				}
				cmd.Printf("%-5d %-12s %-14s %7.4f %7.4f %7.4f %7.4f %7.4f %s\n",
					i+1,
					shortID,
					string(hit.Item.Type),
					hit.Score.VectorScore,
					hit.Score.BM25Score,
					hit.Score.TimeBoost,
					hit.Score.AccessBoost,
					hit.Score.FinalScore,
					rerankStr,
				)
				if len(hit.Score.Notes) > 0 {
					cmd.Printf("       Notes: %s\n", strings.Join(hit.Score.Notes, " | "))
				}
			}
		} else {
			for i, item := range res.Items {
				cmd.Printf("[%d] (%s) %s\n", i+1, item.Type, item.ID)
			}
		}
	},
}

var memoryPolicyShowCmd = &cobra.Command{
	Use:   "policy show",
	Short: "Show current time-aware retention policies",
	Run: func(cmd *cobra.Command, args []string) {
		observability.Logger.Info("Active Memory Retention Policy (Days)",
			zap.Int("raw", config.GlobalConfig.Retention.RawDays),
			zap.Int("daily_summary", config.GlobalConfig.Retention.DailySummaryDays),
			zap.Int("weekly_summary", config.GlobalConfig.Retention.WeeklySummaryDays),
			zap.Int("monthly_summary", config.GlobalConfig.Retention.MonthlySummaryDays),
			zap.Int("concept_edges", config.GlobalConfig.Retention.ConceptEdgesDays),
			zap.Int("anomalies", config.GlobalConfig.Retention.AnomaliesDays),
		)
	},
}

var memoryReapCmd = &cobra.Command{
	Use:   "reap",
	Short: "Run the retention enforcement job to reap expired memories",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		now := time.Now()
		if reapNowStr != "" {
			t, err := time.Parse(time.RFC3339, reapNowStr)
			if err != nil {
				observability.Logger.Fatal("Invalid time format for --now, must be RFC3339", zap.Error(err))
			}
			now = t
		}

		embedder := store.NewEmbedder(config.GlobalConfig.Memory.Embedding)
		memClient, err := store.NewJSONStore(
			config.GlobalConfig.Memory.DBPath,
			config.GlobalConfig.Memory.Embedding.Dimensions,
			embedder,
			config.GlobalConfig.Memory.Retrieval,
		)
		if err != nil {
			observability.Logger.Fatal("Failed to init Memory Store", zap.Error(err))
		}

		r := reaper.NewReaper(memClient, config.GlobalConfig.Retention)

		report, err := r.Run(ctx, reapScope, now, reapDryRun)
		if err != nil {
			observability.Logger.Fatal("Reaper failed", zap.Error(err))
		}

		observability.Logger.Info("Reaper report",
			zap.Int("total_deleted", report.TotalDeleted),
			zap.Any("deleted_by_type", report.DeletedByType),
		)
	},
}

func init() {
	memoryQueryCmd.Flags().StringVar(&memoryScope, "scope", "global", "Scope to query (e.g. global, project:x, user:y)")
	memoryQueryCmd.Flags().StringVar(&memoryQuery, "q", "", "Query string")
	memoryQueryCmd.Flags().BoolVar(&memoryExplain, "explain", false, "Print per-result score breakdown table")
	memoryQueryCmd.Flags().IntVar(&memoryTopK, "topk", 10, "Maximum number of results to return")
	memoryQueryCmd.MarkFlagRequired("q")
	memoryCmd.AddCommand(memoryQueryCmd)
	memoryCmd.AddCommand(memoryPolicyShowCmd)

	memoryReapCmd.Flags().StringVar(&reapScope, "scope", "", "Scope to reap (required)")
	memoryReapCmd.Flags().BoolVar(&reapDryRun, "dry-run", false, "Simulate reaper without deleting")
	memoryReapCmd.Flags().StringVar(&reapNowStr, "now", "", "Override current time (RFC3339)")
	memoryReapCmd.MarkFlagRequired("scope")
	memoryCmd.AddCommand(memoryReapCmd)

	rootCmd.AddCommand(memoryCmd)
}
