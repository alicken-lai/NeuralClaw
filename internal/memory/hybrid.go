package memory

import (
	"context"

	"neuralclaw/internal/observability"
	"neuralclaw/pkg/types"

	"go.uber.org/zap"
)

// Router mediates queries between pure embedding searches, keyword (BM25) searches, and re-ranking.
type Router struct {
	store    types.MemoryStore
	embedder types.Embedder
}

func NewRouter(store types.MemoryStore, embedder types.Embedder) *Router {
	return &Router{
		store:    store,
		embedder: embedder,
	}
}

// Search performs a hybrid query.
func (r *Router) Search(ctx context.Context, text string, scope string, topK int) (types.QueryResult, error) {
	return r.SearchExplain(ctx, text, scope, topK, false)
}

// SearchExplain performs a hybrid query with optional score breakdown population.
func (r *Router) SearchExplain(ctx context.Context, text string, scope string, topK int, explain bool) (types.QueryResult, error) {
	observability.Logger.Info("Executing hybrid search",
		zap.String("query", text),
		zap.String("scope", scope),
	)

	// 1. Embed query text if vector is requested
	vectors, err := r.embedder.Embed(ctx, []string{text})
	var vector []float32
	if err == nil && len(vectors) > 0 {
		vector = vectors[0]
	}

	// 2. Build Query object enforcing multi-scope constraints
	q := types.Query{
		Text:    text,
		Vector:  vector,
		Scope:   scope, // Enforces memory isolation
		TopK:    topK,
		Explain: explain,
	}

	// 3. Dispatch to Memory Store
	res, err := r.store.Query(ctx, q)
	if err != nil {
		return types.QueryResult{}, err
	}

	observability.Logger.Info("Search complete", zap.Int("results_found", res.TotalFound))
	return res, nil
}
