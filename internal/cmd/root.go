package cmd

import (
	"github.com/spf13/cobra"

	"neuralclaw/internal/config"
	"neuralclaw/internal/observability"
)

var rootCmd = &cobra.Command{
	Use:   "zclaw",
	Short: "ZeroClaw + DMN Research Agent Architecture",
	Long:  `A unified Go monorepo orchestrating ZeroClaw agent runtime, memory-lancedb-pro, and GLM-OCR.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&config.CfgFile, "config", "", "config file (default is configs/config.example.yaml)")
	rootCmd.AddCommand(webCmd)
}

func initConfig() {
	config.Load()
	observability.InitLogger(config.GlobalConfig.Log.Level)
}
