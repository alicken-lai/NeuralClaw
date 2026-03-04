package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"neuralclaw/internal/config"
	"neuralclaw/internal/dmn"
	"neuralclaw/internal/memory/store"
	"neuralclaw/internal/observability"
)

var (
	dmnScope string
	dmnDate  string
	dmnCron  string
)

var dmnCmd = &cobra.Command{
	Use:   "dmn",
	Short: "Manage the DMN reflection and consolidation process",
}

var dmnRunCmd = &cobra.Command{
	Use:   "run",
	Short: "Run DMN reflection once",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		observability.Logger.Info("Running DMN reflection", zap.String("scope", dmnScope), zap.String("date", dmnDate))

		embedder := store.NewEmbedder(config.GlobalConfig.Memory.Embedding)

		// Init pure go memory store
		memStore, err := store.NewJSONStore(
			config.GlobalConfig.Memory.DBPath,
			config.GlobalConfig.Memory.Embedding.Dimensions,
			embedder,
			config.GlobalConfig.Memory.Retrieval,
		)
		if err != nil {
			observability.Logger.Fatal("Failed to init Memory Store", zap.Error(err))
		}

		// Create the DMN pipeline with the new Store
		pipeline := dmn.NewPipeline(memStore, embedder, config.GlobalConfig.Memory.Retrieval)

		if err := pipeline.Run(ctx, dmnScope, dmnDate); err != nil {
			observability.Logger.Error("DMN run failed", zap.Error(err))
		}
	},
}

var dmnScheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Run DMN reflection on a continuous background schedule (interval in minutes)",
	Run: func(cmd *cobra.Command, args []string) {
		observability.Logger.Info("Starting DMN background scheduler", zap.String("interval", dmnCron))

		// Parse interval from cron field; for MVP use it as minute count
		// Full cron expression parsing requires an external library;
		// for MVP we treat the value as a numeric minute interval.
		var intervalMinutes int
		if _, err := fmt.Sscanf(dmnCron, "%d", &intervalMinutes); err != nil || intervalMinutes <= 0 {
			intervalMinutes = 60 // Default to hourly
		}

		observability.Logger.Info("DMN will run every N minutes", zap.Int("interval_minutes", intervalMinutes))

		ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
		defer ticker.Stop()

		embedder := store.NewEmbedder(config.GlobalConfig.Memory.Embedding)
		memStore, err := store.NewJSONStore(
			config.GlobalConfig.Memory.DBPath,
			config.GlobalConfig.Memory.Embedding.Dimensions,
			embedder,
			config.GlobalConfig.Memory.Retrieval,
		)
		if err != nil {
			observability.Logger.Fatal("Failed to init Memory Store", zap.Error(err))
		}

		runOnce := func() {
			p := dmn.NewPipeline(memStore, embedder, config.GlobalConfig.Memory.Retrieval)
			if err := p.Run(cmd.Context(), dmnScope, time.Now().Format("2006-01-02")); err != nil {
				observability.Logger.Error("Scheduled DMN run failed", zap.Error(err))
			} else {
				observability.Logger.Info("Scheduled DMN run completed")
			}
		}

		// Run immediately once on startup, then tick
		runOnce()
		for {
			select {
			case <-ticker.C:
				runOnce()
			case <-cmd.Context().Done():
				observability.Logger.Info("DMN scheduler shutting down")
				return
			}
		}
	},
}

func init() {
	dmnRunCmd.Flags().StringVar(&dmnScope, "scope", "global", "Scope to run DMN on")
	dmnRunCmd.Flags().StringVar(&dmnDate, "date", "", "Date to run DMN for (YYYY-MM-DD)")
	dmnRunCmd.MarkFlagRequired("date")
	dmnCmd.AddCommand(dmnRunCmd)

	dmnScheduleCmd.Flags().StringVar(&dmnCron, "cron", "0 3 * * *", "Cron expression for scheduling")
	dmnCmd.AddCommand(dmnScheduleCmd)

	rootCmd.AddCommand(dmnCmd)
}
