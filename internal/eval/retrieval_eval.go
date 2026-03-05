package eval

import (
	"math"

	"neuralclaw/pkg/types"
)

// GoldenQuery represents a test case for retrieval evaluation.
type GoldenQuery struct {
	ID            string   `yaml:"id"`
	Text          string   `yaml:"text"`
	ExpectedIDs   []string `yaml:"expected_ids"`
	ExpectedTypes []string `yaml:"expected_types,omitempty"`
	Scope         string   `yaml:"scope"`
}

// EvalResult holds the calculated metrics for a single query.
type EvalResult struct {
	QueryID string
	RecallK float64
	MRRK    float64
	NDCGK   float64
	Hits    int
	Missed  []string
}

// SuiteMetrics holds aggregated metrics for the entire evaluation run.
type SuiteMetrics struct {
	AvgRecall float64
	AvgMRR    float64
	AvgNDCG   float64
	Total     int
	Results   []EvalResult
}

// Evaluate performs metrics calculation for a given set of actual hits against golden expectations.
func Evaluate(golden GoldenQuery, actual []types.MemoryItem, k int) EvalResult {
	if len(actual) > k {
		actual = actual[:k]
	}

	expectedMap := make(map[string]bool)
	for _, id := range golden.ExpectedIDs {
		expectedMap[id] = true
	}

	var hits int
	var mrr float64
	var dcg float64
	var idcg float64

	foundMap := make(map[string]bool)

	for i, hit := range actual {
		rank := i + 1
		if expectedMap[hit.ID] {
			hits++
			foundMap[hit.ID] = true
			if mrr == 0 {
				mrr = 1.0 / float64(rank)
			}
			dcg += 1.0 / math.Log2(float64(rank+1))
		}
	}

	// Calculate IDCG (Ideal DCG)
	numExpectedInK := len(golden.ExpectedIDs)
	if numExpectedInK > k {
		numExpectedInK = k
	}
	for i := 0; i < numExpectedInK; i++ {
		idcg += 1.0 / math.Log2(float64(i+2))
	}

	var ndcg float64
	if idcg > 0 {
		ndcg = dcg / idcg
	}

	var missed []string
	for _, id := range golden.ExpectedIDs {
		if !foundMap[id] {
			missed = append(missed, id)
		}
	}

	return EvalResult{
		QueryID: golden.ID,
		RecallK: float64(hits) / float64(len(golden.ExpectedIDs)),
		MRRK:    mrr,
		NDCGK:   ndcg,
		Hits:    hits,
		Missed:  missed,
	}
}

// Aggregate computes the averages across all results.
func Aggregate(results []EvalResult) SuiteMetrics {
	if len(results) == 0 {
		return SuiteMetrics{}
	}

	var sumRecall, sumMRR, sumNDCG float64
	for _, r := range results {
		sumRecall += r.RecallK
		sumMRR += r.MRRK
		sumNDCG += r.NDCGK
	}

	n := float64(len(results))
	return SuiteMetrics{
		AvgRecall: sumRecall / n,
		AvgMRR:    sumMRR / n,
		AvgNDCG:   sumNDCG / n,
		Total:     len(results),
		Results:   results,
	}
}
