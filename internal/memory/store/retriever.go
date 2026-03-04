package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"neuralclaw/internal/config"
	"neuralclaw/internal/observability"
	"neuralclaw/pkg/types"

	"go.uber.org/zap"
)

type RetrievalResult struct {
	Item         types.MemoryItem
	Score        float64
	VectorScore  float64
	KeywordScore float64
	FinalScore   float64
}

// Search executes a hybrid search over the JSON memory store.
func (s *JSONStore) Search(ctx context.Context, embedder *Embedder, query string, cfg config.RetrievalConfig, filters map[string]string) ([]RetrievalResult, error) {
	s.mu.RLock()
	// Copy references to allow parallel evaluation without holding lock long
	candidates := make([]types.MemoryItem, 0, len(s.memories))
	for _, m := range s.memories {
		// Apply hard filters first (e.g. scope)
		match := true
		for k, v := range filters {
			if k == "scope" && m.Scope != v {
				match = false
				break
			}
			if k == "category" && m.Category != v {
				match = false
				break
			}
		}
		if match {
			candidates = append(candidates, m)
		}
	}
	s.mu.RUnlock()

	if len(candidates) == 0 {
		return nil, nil
	}

	// 1. Vector Search
	queryVec, err := embedder.EmbedQuery(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed query failed: %w", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	vectorScores := make(map[string]float64)
	keywordScores := make(map[string]float64)

	// Simple TF-IDF / TF fallback for keyword match
	queryTokens := strings.Fields(strings.ToLower(query))

	// Parallel scoring for speed
	numWorkers := 4
	chunkSize := (len(candidates) + numWorkers - 1) / numWorkers

	for i := 0; i < numWorkers; i++ {
		start := i * chunkSize
		if start >= len(candidates) {
			break
		}
		end := start + chunkSize
		if end > len(candidates) {
			end = len(candidates)
		}

		wg.Add(1)
		go func(chunk []types.MemoryItem) {
			defer wg.Done()
			localVecScores := make(map[string]float64)
			localKeyScores := make(map[string]float64)

			for _, item := range chunk {
				// Cosine Similarity
				sim := cosineSimilarity(queryVec, item.Vector)
				localVecScores[item.ID] = sim

				// Simple Keyword match frequency
				keyScore := 0.0
				lowerText := strings.ToLower(item.Text)
				for _, token := range queryTokens {
					if strings.Contains(lowerText, token) {
						keyScore += 1.0
					}
				}
				if len(queryTokens) > 0 {
					keyScore = keyScore / float64(len(queryTokens))
				}
				localKeyScores[item.ID] = keyScore
			}

			mu.Lock()
			for k, v := range localVecScores {
				vectorScores[k] = v
			}
			for k, v := range localKeyScores {
				keywordScores[k] = v
			}
			mu.Unlock()
		}(candidates[start:end])
	}

	wg.Wait()

	// 2. Rank Fusion (RRF or Linear)
	results := make([]RetrievalResult, 0, len(candidates))
	for _, item := range candidates {
		vs := vectorScores[item.ID]
		// Map cosine (-1 to 1) to (0 to 1) roughly for fusion
		normalizedVs := (vs + 1.0) / 2.0
		if normalizedVs < 0 {
			normalizedVs = 0
		}

		ks := keywordScores[item.ID]

		// Linear combination based on config weights
		combined := (normalizedVs * cfg.VectorWeight) + (ks * cfg.BM25Weight)

		results = append(results, RetrievalResult{
			Item:         item,
			VectorScore:  normalizedVs,
			KeywordScore: ks,
			Score:        combined,
		})
	}

	// 3. Late Scoring Pipeline (Recency, Importance, Time Decay)
	now := time.Now()
	for i := range results {
		res := &results[i]

		// Time Decay
		res.FinalScore = res.Score

		if res.Item.Timestamp > 0 {
			ageDays := now.Sub(time.Unix(res.Item.Timestamp, 0)).Hours() / 24.0
			if ageDays > 0 {
				decayFactor := math.Pow(0.5, ageDays/cfg.TimeDecayHalfLifeDays)
				// Important memories decay slower
				importanceBuff := float64(res.Item.Importance) * 0.1
				decayFactor = decayFactor + importanceBuff
				if decayFactor > 1.0 {
					decayFactor = 1.0
				}
				res.FinalScore *= decayFactor
			}
		}

		// Length Normalization
		if cfg.LengthNormAnchor > 0 {
			wordCount := float64(len(strings.Fields(res.Item.Text)))
			lnFactor := 1.0 / (1.0 + math.Log10(1.0+wordCount/cfg.LengthNormAnchor))
			// Only penalize excessively long chunks slightly, don't crush them
			res.FinalScore = res.FinalScore * (0.8 + 0.2*lnFactor)
		}
	}

	// Sort desc
	sort.Slice(results, func(i, j int) bool {
		return results[i].FinalScore > results[j].FinalScore
	})

	// Top K cutoff
	if len(results) > cfg.CandidatePoolSize {
		results = results[:cfg.CandidatePoolSize]
	}

	// 4. Reranking (Cross-Encoder HTTP APIs)
	if cfg.RerankProvider != "" && cfg.RerankProvider != "none" {
		results, err = rerankResults(ctx, query, results, cfg)
		if err != nil {
			observability.Logger.Warn("Reranking failed, returning hybrid scores", zap.Error(err))
		}
	}

	// Final hard threshold
	finalList := make([]RetrievalResult, 0)
	for _, r := range results {
		if r.FinalScore >= cfg.HardMinScore {
			finalList = append(finalList, r)
		}
	}

	return finalList, nil
}

// cosineSimilarity computes pure math float32 similarity
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	var dotProduct, normA, normB float32
	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return float64(dotProduct / (float32(math.Sqrt(float64(normA))) * float32(math.Sqrt(float64(normB)))))
}

// Cross-Encoder API Caller
func rerankResults(ctx context.Context, query string, results []RetrievalResult, cfg config.RetrievalConfig) ([]RetrievalResult, error) {
	if len(results) == 0 {
		return results, nil
	}

	docs := make([]string, len(results))
	for i, r := range results {
		docs[i] = r.Item.Text
	}

	reqBody := map[string]interface{}{
		"model":     cfg.RerankModel,
		"query":     query,
		"documents": docs,
		"top_n":     len(results),
	}

	payload, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", cfg.RerankEndpoint, bytes.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	if cfg.RerankAPIKey != "" {
		req.Header.Set("Authorization", "Bearer "+cfg.RerankAPIKey)
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var apiResp struct {
		Results []struct {
			Index          int     `json:"index"`
			RelevanceScore float64 `json:"relevance_score"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, err
	}

	// Update scores based on Reranker
	for _, r := range apiResp.Results {
		if r.Index >= 0 && r.Index < len(results) {
			// Combine rerank absolute score with the original time-decayed score
			results[r.Index].FinalScore = r.RelevanceScore * results[r.Index].FinalScore
		}
	}

	// Resort after reranking
	sort.Slice(results, func(i, j int) bool {
		return results[i].FinalScore > results[j].FinalScore
	})

	return results, nil
}
