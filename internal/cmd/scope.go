package cmd

import (
	"context"
	"fmt"
	"sort"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"neuralclaw/internal/config"
	"neuralclaw/internal/memory/store"
	"neuralclaw/internal/observability"
	"neuralclaw/pkg/types"
)

var scopeCmd = &cobra.Command{
	Use:   "scope",
	Short: "Manage query and memory scopes",
}

var scopeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all known scopes found in the memory store",
	Run: func(cmd *cobra.Command, args []string) {
		observability.Logger.Info("Listing scopes")

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

		result, err := memStore.Query(context.Background(), types.Query{
			Text: "*",
			TopK: 10000,
		})
		if err != nil {
			observability.Logger.Error("Failed to list memory items", zap.Error(err))
			return
		}

		// Collect unique scopes
		scopes := make(map[string]int)
		for _, item := range result.Items {
			scopes[item.Scope]++
		}

		if len(scopes) == 0 {
			fmt.Println("No scopes found. Ingest some files first.")
			return
		}

		keys := make([]string, 0, len(scopes))
		for k := range scopes {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		fmt.Printf("%-40s  %s\n", "SCOPE", "ITEMS")
		fmt.Printf("%-40s  %s\n", "-----", "-----")
		for _, k := range keys {
			fmt.Printf("%-40s  %d\n", k, scopes[k])
		}
	},
}

var scopeSetCmd = &cobra.Command{
	Use:   "check [scope-name]",
	Short: "Validate that a scope exists in the memory store",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		scope := args[0]
		observability.Logger.Info("Checking scope", zap.String("scope", scope))

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

		result, err := memStore.Query(context.Background(), types.Query{
			Text:  "*",
			Scope: scope,
			TopK:  1,
		})
		if err != nil {
			observability.Logger.Error("Failed to query scope", zap.Error(err))
			return
		}

		if len(result.Items) == 0 {
			fmt.Printf("Scope '%s' not found or has no memory items.\n", scope)
		} else {
			fmt.Printf("Scope '%s' exists and is accessible.\n", scope)
		}
	},
}

func init() {
	scopeCmd.AddCommand(scopeListCmd)
	scopeCmd.AddCommand(scopeSetCmd)
	rootCmd.AddCommand(scopeCmd)
}
