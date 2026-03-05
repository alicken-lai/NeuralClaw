package cmd

import (
	"github.com/spf13/cobra"

	"neuralclaw/internal/config"
	"neuralclaw/internal/observability"
)

var rootCmd = &cobra.Command{
	Use:   "neuralclaw",
	Short: "NeuralClaw — Pure Go Autonomous Agent with a Default-Mode Brain",
	Long:  `NeuralClaw is a unified, production-grade AI agent framework written entirely in Go, featuring DMN-inspired background memory consolidation, hybrid retrieval, and explainable scoring.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&config.CfgFile, "config", "", "config file (default is configs/config.example.yaml)")
}

func initConfig() {
	config.Load()
	observability.InitLogger(config.GlobalConfig.Log.Level)
	observability.InitTokenTracker("logs")
}
