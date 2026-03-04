package store

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"neuralclaw/internal/config"
)

type Embedder struct {
	client *http.Client
	config config.EmbeddingConfig
}

func NewEmbedder(cfg config.EmbeddingConfig) *Embedder {
	return &Embedder{
		client: &http.Client{Timeout: 30 * time.Second},
		config: cfg,
	}
}

// Ensure dimensions match config
func (e *Embedder) Dimensions() int {
	return e.config.Dimensions
}

type OpenAIEmbeddingRequest struct {
	Model string   `json:"model"`
	Input []string `json:"input"`
}

type OpenAIEmbeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
	} `json:"data"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

// EmbedQuery calls the OpenAI-compatible embedding API for a single text string.
// Provider nuances (like Ollama or Jina) are hidden under the compatible `/embeddings` endpoint.
func (e *Embedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	if text == "" {
		return make([]float32, e.config.Dimensions), nil // Return zero vector for empty
	}

	endpoint := strings.TrimRight(e.config.BaseURL, "/") + "/embeddings"

	reqBody := OpenAIEmbeddingRequest{
		Model: e.config.Model,
		Input: []string{text},
	}

	payload, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("failed to build embed request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if e.config.APIKey != "" {
		req.Header.Set("Authorization", "Bearer "+e.config.APIKey)
	}

	resp, err := e.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("embedding API call failed: %w", err)
	}
	defer resp.Body.Close()

	var apiResp OpenAIEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("failed to decode embedding response: %w", err)
	}

	if apiResp.Error != nil {
		return nil, fmt.Errorf("embedding API returned error: %s", apiResp.Error.Message)
	}

	if len(apiResp.Data) == 0 {
		return nil, fmt.Errorf("embedding API returned empty data array")
	}

	vec := apiResp.Data[0].Embedding
	if len(vec) != e.config.Dimensions {
		return nil, fmt.Errorf("embedding dimension mismatch: got %d, expected %d", len(vec), e.config.Dimensions)
	}

	return vec, nil
}

// Embed implements the types.Embedder interface for an array of strings.
func (e *Embedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	results := make([][]float32, len(texts))
	for i, text := range texts {
		vec, err := e.EmbedQuery(ctx, text)
		if err != nil {
			return nil, fmt.Errorf("failed to embed text at index %d: %w", i, err)
		}
		results[i] = vec
	}
	return results, nil
}
