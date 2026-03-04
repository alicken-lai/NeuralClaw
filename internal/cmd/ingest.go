package cmd

import (
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"neuralclaw/internal/config"
	"neuralclaw/internal/ingest"
	ocrglm "neuralclaw/internal/ingest/ocr_glm"
	"neuralclaw/internal/memory/store"
	"neuralclaw/internal/observability"
)

var ingestInput string

var ingestCmd = &cobra.Command{
	Use:   "ingest",
	Short: "Ingest data into the memory store",
}

var ingestOcrCmd = &cobra.Command{
	Use:   "ocr",
	Short: "Ingest an image or PDF via OCR",
	Run: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()
		observability.Logger.Info("Starting OCR ingestion", zap.String("input", ingestInput))

		// Wire up dependencies
		ocrClient := ocrglm.NewClient()
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

		// Use the real embedder instead of the dummy one for pure Go indexing.
		pipeline := ingest.NewPipeline(ocrClient, memClient, embedder)

		// Scope handling would come from a global config flag, but defaulting to 'global'
		if err := pipeline.Process(ctx, ingestInput, "global"); err != nil {
			observability.Logger.Error("Ingestion failed", zap.Error(err))
		}
	},
}

func init() {
	ingestOcrCmd.Flags().StringVar(&ingestInput, "input", "", "File or directory to ingest")
	ingestOcrCmd.MarkFlagRequired("input")
	ingestCmd.AddCommand(ingestOcrCmd)

	rootCmd.AddCommand(ingestCmd)
}
