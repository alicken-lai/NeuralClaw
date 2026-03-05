package cmd

import (
	"path/filepath"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"neuralclaw/internal/agent"
	"neuralclaw/internal/config"
	"neuralclaw/internal/memory/store"
	"neuralclaw/internal/observability"
	"neuralclaw/internal/taskstore"
	"neuralclaw/internal/web"
)

var (
	webAddr  string
	webScope string
)

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "Start the NeuralClaw Web GUI dispatcher",
	Run: func(cmd *cobra.Command, args []string) {
		addr := webAddr
		if addr == "" {
			addr = config.GlobalConfig.Web.Addr
			if addr == "" {
				addr = "127.0.0.1:8080"
			}
		}

		dataDir := filepath.Join(".", "data")
		taskStore, err := taskstore.NewJSONFileStore(dataDir)
		if err != nil {
			observability.Logger.Fatal("Failed to initialize TaskStore", zap.Error(err))
		}

		embedder := store.NewEmbedder(config.GlobalConfig.Memory.Embedding)
		memStore, err := store.NewJSONStore(
			config.GlobalConfig.Memory.DBPath,
			config.GlobalConfig.Memory.Embedding.Dimensions,
			embedder,
			config.GlobalConfig.Memory.Retrieval,
		)
		if err != nil {
			observability.Logger.Fatal("Failed to initialize Memory Store", zap.Error(err))
		}

		dispatcher := agent.NewDispatcher()
		server := web.NewServer(addr, config.GlobalConfig.Web.AuthToken, webScope, taskStore, memStore, embedder, dispatcher)

		if err := server.Start(); err != nil {
			observability.Logger.Fatal("Web GUI server failed", zap.Error(err))
		}
	},
}

func init() {
	webCmd.Flags().StringVar(&webAddr, "addr", "", "Address to bind the web server (e.g. 127.0.0.1:8080)")
	webCmd.Flags().StringVar(&webScope, "scope", "global", "The default scope this Web GUI controls")

	rootCmd.AddCommand(webCmd)
}
