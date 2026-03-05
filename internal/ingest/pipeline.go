package ingest

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"neuralclaw/internal/observability"
	"neuralclaw/pkg/types"
)

type Pipeline struct {
	ocr      types.OCREngine
	memory   types.MemoryStore
	embedder types.Embedder
}

func NewPipeline(ocr types.OCREngine, memory types.MemoryStore, embedder types.Embedder) *Pipeline {
	return &Pipeline{
		ocr:      ocr,
		memory:   memory,
		embedder: embedder,
	}
}

// Process handles File -> OCR -> Chunking -> Embedding -> Memory Store
func (p *Pipeline) Process(ctx context.Context, input string, targetScope string) error {
	observability.Logger.Info("Processing input for ingestion", zap.String("input", input))

	// 1. OCR Subprocess
	result, err := p.ocr.ExtractText(ctx, types.FileRef(input))
	if err != nil {
		return fmt.Errorf("OCR failed: %w", err)
	}

	// 2. Paragraph-aware Chunking
	if result.Text == "" {
		observability.Logger.Warn("OCR extraction returned empty text for image", zap.String("input", input))
		return nil
	}
	chunks := chunkText(result.Text, 2000)

	// 3. Embedding
	vectors, err := p.embedder.Embed(ctx, chunks)
	if err != nil {
		return fmt.Errorf("embedding failed: %w", err)
	}

	// 4. Create Memory Items
	var items []types.MemoryItem
	for i, chunk := range chunks {
		hash := fmt.Sprintf("%x", sha256.Sum256([]byte(chunk)))
		item := types.MemoryItem{
			ID:         uuid.New().String(),
			Type:       types.ItemTypeRaw,
			Scope:      targetScope,
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
			SourceTime: time.Now(), // Extracting valid metadata if possible
			Modality:   "ocr",
			Text:       chunk,
			BM25Text:   chunk,
			Vector:     vectors[i],
			Provenance: types.Provenance{
				SourceFilePath: input,
				SourceURI:      input, // Using input path as URI
				SourceKind:     "ocr",
				ChunkIndex:     ptrInt(i),
				Hash:           hash,
				ToolVersion:    "GLM-OCR v1",
			},
		}
		items = append(items, item)
	}

	// 5. Upsert
	if err := p.memory.Upsert(ctx, items); err != nil {
		return fmt.Errorf("memory upsert failed: %w", err)
	}

	observability.Logger.Info("Ingestion complete", zap.Int("items_upserted", len(items)))
	return nil
}

// chunkText splits text into segments of at most maxLen characters,
// preferring paragraph boundaries (\n\n) to keep semantic cohesion.
func chunkText(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var chunks []string
	start := 0
	for start < len(text) {
		end := start + maxLen
		if end >= len(text) {
			chunks = append(chunks, text[start:])
			break
		}

		// Look backwards for a paragraph boundary
		boundary := -1
		search := text[start:end]
		for i := len(search) - 1; i >= 0; i-- {
			if i > 1 && search[i] == '\n' && search[i-1] == '\n' {
				boundary = i
				break
			}
		}

		if boundary > 0 {
			chunks = append(chunks, text[start:start+boundary])
			start = start + boundary + 2
		} else {
			// Fall back: split a the last space
			lastSpace := -1
			for i := end - 1; i > start; i-- {
				if text[i] == ' ' {
					lastSpace = i
					break
				}
			}
			if lastSpace > start {
				chunks = append(chunks, text[start:lastSpace])
				start = lastSpace + 1
			} else {
				chunks = append(chunks, text[start:end])
				start = end
			}
		}
	}
	return chunks
}

func ptrInt(i int) *int {
	return &i
}
