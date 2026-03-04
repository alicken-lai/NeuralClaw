package agent

import (
	"context"

	"neuralclaw/pkg/types"
)

// DummyEmbedder is used for testing and satisfying interface requirements when no remote is configured.
type DummyEmbedder struct {
	Dimensions int
}

func NewDummyEmbedder(dimensions int) *DummyEmbedder {
	return &DummyEmbedder{Dimensions: dimensions}
}

func (e *DummyEmbedder) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	// Returns a zero-vector for each string
	res := make([][]float32, len(texts))
	for i := range texts {
		res[i] = make([]float32, e.Dimensions)
		res[i][0] = 1.0 // just a stub value
	}
	return res, nil
}

// Ensure it implements Embedder
var _ types.Embedder = (*DummyEmbedder)(nil)
