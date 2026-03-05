package eval

import (
	"testing"

	"neuralclaw/pkg/types"
)

func TestEvaluateMetrics(t *testing.T) {
	golden := GoldenQuery{
		ID:          "q-1",
		Text:        "memory query",
		ExpectedIDs: []string{"a", "b"},
		Scope:       "global",
	}
	actual := []types.MemoryItem{
		{ID: "c"},
		{ID: "a"},
		{ID: "b"},
	}

	got := Evaluate(golden, actual, 3)
	if got.RecallK != 1.0 {
		t.Fatalf("Recall@K mismatch: got %v want 1.0", got.RecallK)
	}
	if got.MRRK != 0.5 {
		t.Fatalf("MRR@K mismatch: got %v want 0.5", got.MRRK)
	}
	if len(got.Missed) != 0 {
		t.Fatalf("Missed mismatch: got %v want empty", got.Missed)
	}
	if got.NDCGK <= 0 || got.NDCGK > 1 {
		t.Fatalf("NDCG@K out of range: %v", got.NDCGK)
	}
}

func TestAggregateMetrics(t *testing.T) {
	results := []EvalResult{
		{QueryID: "q1", RecallK: 1.0, MRRK: 0.5, NDCGK: 0.6},
		{QueryID: "q2", RecallK: 0.0, MRRK: 0.0, NDCGK: 0.0},
	}

	got := Aggregate(results)
	if got.Total != 2 {
		t.Fatalf("Total mismatch: got %d want 2", got.Total)
	}
	if got.AvgRecall != 0.5 {
		t.Fatalf("AvgRecall mismatch: got %v want 0.5", got.AvgRecall)
	}
	if got.AvgMRR != 0.25 {
		t.Fatalf("AvgMRR mismatch: got %v want 0.25", got.AvgMRR)
	}
	if got.AvgNDCG != 0.3 {
		t.Fatalf("AvgNDCG mismatch: got %v want 0.3", got.AvgNDCG)
	}
}
