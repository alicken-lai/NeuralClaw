package cmd

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"
	"go.yaml.in/yaml/v3"

	"neuralclaw/internal/config"
	"neuralclaw/internal/eval"
	"neuralclaw/internal/memory"
	"neuralclaw/internal/memory/store"
	"neuralclaw/pkg/types"
)

type retrievalSearcher interface {
	SearchExplain(ctx context.Context, text string, scope string, topK int, explain bool) (types.QueryResult, error)
}

type retrievalEvalOptions struct {
	GoldenPath string
	K          int
	OutputPath string
}

var newRetrievalSearcher = func() (retrievalSearcher, error) {
	embedder := store.NewEmbedder(config.GlobalConfig.Memory.Embedding)
	memClient, err := store.NewJSONStore(
		config.GlobalConfig.Memory.DBPath,
		config.GlobalConfig.Memory.Embedding.Dimensions,
		embedder,
		config.GlobalConfig.Memory.Retrieval,
	)
	if err != nil {
		return nil, err
	}
	return memory.NewRouter(memClient, embedder), nil
}

var evalCmd = &cobra.Command{
	Use:   "eval",
	Short: "Evaluation toolset",
}

func newEvalRetrievalCmd() *cobra.Command {
	opts := retrievalEvalOptions{}

	cmd := &cobra.Command{
		Use:   "retrieval",
		Short: "Evaluate retrieval metrics against golden queries",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRetrievalEval(cmd, opts)
		},
	}
	cmd.Flags().StringVar(&opts.GoldenPath, "golden", "", "Path to golden query YAML")
	cmd.Flags().IntVar(&opts.K, "k", 10, "Top-K results to evaluate")
	cmd.Flags().StringVar(&opts.OutputPath, "output", "", "Optional JSON report output path")
	_ = cmd.MarkFlagRequired("golden")
	return cmd
}

func runRetrievalEval(cmd *cobra.Command, opts retrievalEvalOptions) error {
	if opts.K <= 0 {
		return errors.New("--k must be greater than 0")
	}

	payload, err := os.ReadFile(opts.GoldenPath)
	if err != nil {
		return fmt.Errorf("failed to read golden query file: %w", err)
	}

	var golden []eval.GoldenQuery
	if err := yaml.Unmarshal(payload, &golden); err != nil {
		return fmt.Errorf("failed to parse golden query YAML: %w", err)
	}
	if len(golden) == 0 {
		return errors.New("golden query file is empty")
	}

	searcher, err := newRetrievalSearcher()
	if err != nil {
		return fmt.Errorf("failed to initialize retrieval searcher: %w", err)
	}

	results := make([]eval.EvalResult, 0, len(golden))
	for _, gq := range golden {
		scope := gq.Scope
		if scope == "" {
			scope = "global"
		}
		res, err := searcher.SearchExplain(cmd.Context(), gq.Text, scope, opts.K, true)
		if err != nil {
			return fmt.Errorf("query failed for %s: %w", gq.ID, err)
		}
		results = append(results, eval.Evaluate(gq, res.Items, opts.K))
	}

	suite := eval.Aggregate(results)
	renderEvalTable(cmd, suite)

	if opts.OutputPath != "" {
		body, err := json.MarshalIndent(suite, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to serialize output JSON: %w", err)
		}
		if err := os.WriteFile(opts.OutputPath, body, 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "\nJSON report written to %s\n", opts.OutputPath)
	}

	return nil
}

func renderEvalTable(cmd *cobra.Command, suite eval.SuiteMetrics) {
	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "QueryID\tRecall@K\tMRR@K\tNDCG@K\tHits\tMissed")
	for _, row := range suite.Results {
		fmt.Fprintf(w, "%s\t%.4f\t%.4f\t%.4f\t%d\t%d\n",
			row.QueryID, row.RecallK, row.MRRK, row.NDCGK, row.Hits, len(row.Missed))
	}
	fmt.Fprintf(w, "AVG\t%.4f\t%.4f\t%.4f\t-\t-\n", suite.AvgRecall, suite.AvgMRR, suite.AvgNDCG)
	_ = w.Flush()
}

func init() {
	evalCmd.AddCommand(newEvalRetrievalCmd())
	rootCmd.AddCommand(evalCmd)
}
