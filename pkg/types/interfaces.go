package types

import "context"

// MemoryStore defines the persistent storage layer for the agent memories.
type MemoryStore interface {
	Upsert(ctx context.Context, items []MemoryItem) error
	Query(ctx context.Context, q Query) (QueryResult, error)
	Delete(ctx context.Context, filter Filter) error
	Health(ctx context.Context) error
}

// FileRef points to a local or remote file to be processed.
type FileRef string

// OCRResult encapsulates text parsed by the OCR engine.
type OCRResult struct {
	Text  string
	Pages int
}

// OCREngine parses text from images or PDFs.
type OCREngine interface {
	ExtractText(ctx context.Context, input FileRef) (OCRResult, error)
}

// Embedder maps strings into vector space.
type Embedder interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}

// Scheduler handles periodic background tasks (e.g. DMN).
type Scheduler interface {
	Start(ctx context.Context)
	Stop(ctx context.Context)
}
