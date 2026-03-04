package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"neuralclaw/internal/config"
	"neuralclaw/internal/observability"
	"neuralclaw/pkg/types"

	"go.uber.org/zap"
)

// JSONStore is a pure Go thread-safe in-memory vector database backed by a JSON file.
// It loads all memories into RAM on startup. Searching 10,000 vectors in RAM in Go takes < 5ms.
type JSONStore struct {
	mu           sync.RWMutex
	filePath     string
	memories     map[string]types.MemoryItem
	needsPersist bool
	dimensions   int
	embedder     *Embedder
	cfg          config.RetrievalConfig
}

func NewJSONStore(dbDir string, dimensions int, embedder *Embedder, cfg config.RetrievalConfig) (*JSONStore, error) {
	if err := os.MkdirAll(dbDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create db directory: %w", err)
	}

	filePath := filepath.Join(dbDir, "memories.json")
	s := &JSONStore{
		filePath:   filePath,
		memories:   make(map[string]types.MemoryItem),
		dimensions: dimensions,
		embedder:   embedder,
		cfg:        cfg,
	}

	if err := s.load(); err != nil {
		return nil, err
	}

	// Start background persister
	go s.persisterLoop()

	return s, nil
}

func (s *JSONStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := os.ReadFile(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			observability.Logger.Info("Init new pure Go JSON store", zap.String("file", s.filePath))
			return nil
		}
		return fmt.Errorf("failed to read memory db: %w", err)
	}

	var items []types.MemoryItem
	if err := json.Unmarshal(b, &items); err != nil {
		return fmt.Errorf("corrupt memory db: %w", err)
	}

	// Rebuild map index
	for _, item := range items {
		s.memories[item.ID] = item
	}

	observability.Logger.Info("Loaded memories from pure Go JSON store", zap.Int("count", len(s.memories)))
	return nil
}

func (s *JSONStore) AddMemory(ctx context.Context, item types.MemoryItem) error {
	if item.ID == "" {
		return fmt.Errorf("item ID cannot be empty")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.memories[item.ID] = item
	s.needsPersist = true
	return nil
}

func (s *JSONStore) DeleteMemories(ctx context.Context, ids []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, id := range ids {
		delete(s.memories, id)
	}
	s.needsPersist = true
	return nil
}

// persisterLoop periodically flushes to disk if dirty
func (s *JSONStore) persisterLoop() {
	ticker := time.NewTicker(3 * time.Second)
	for range ticker.C {
		s.mu.RLock()
		dirty := s.needsPersist
		s.mu.RUnlock()

		if dirty {
			s.flush()
		}
	}
}

func (s *JSONStore) flush() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.needsPersist {
		return
	}

	items := make([]types.MemoryItem, 0, len(s.memories))
	for _, item := range s.memories {
		items = append(items, item)
	}

	b, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		observability.Logger.Error("Failed to serialize memory db", zap.Error(err))
		return
	}

	// Atomic write
	tmpFile := s.filePath + ".tmp"
	if err := os.WriteFile(tmpFile, b, 0644); err != nil {
		observability.Logger.Error("Failed to write memory db tmp", zap.Error(err))
		return
	}
	if err := os.Rename(tmpFile, s.filePath); err != nil {
		observability.Logger.Error("Failed to rename memory db", zap.Error(err))
		return
	}

	s.needsPersist = false
}

// MemoryStore Interface Compliance

func (s *JSONStore) Upsert(ctx context.Context, items []types.MemoryItem) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, item := range items {
		if item.ID == "" {
			return fmt.Errorf("item ID cannot be empty")
		}
		if item.Timestamp == 0 {
			item.Timestamp = time.Now().Unix()
		}
		s.memories[item.ID] = item
	}
	s.needsPersist = true
	return nil
}

func (s *JSONStore) Query(ctx context.Context, q types.Query) (types.QueryResult, error) {
	filters := make(map[string]string)
	if q.Scope != "" {
		filters["scope"] = q.Scope
	}
	// Note: We bypass full Query logic if Text is empty (e.g., date-based fetch in DMN)
	var finalItems []types.MemoryItem
	var finalScores []float32

	if q.Text == "" && q.TimeWindowStart != nil && q.TimeWindowEnd != nil {
		// Time window fetch only
		s.mu.RLock()
		for _, m := range s.memories {
			if q.Scope != "" && m.Scope != q.Scope {
				continue
			}
			t := time.Unix(m.Timestamp, 0)
			if t.After(*q.TimeWindowStart) && t.Before(*q.TimeWindowEnd) {
				finalItems = append(finalItems, m)
				finalScores = append(finalScores, 1.0)
			}
		}
		s.mu.RUnlock()
	} else {
		// Hybrid Search
		results, err := s.Search(ctx, s.embedder, q.Text, s.cfg, filters)
		if err != nil {
			return types.QueryResult{}, err
		}

		for _, r := range results {
			finalItems = append(finalItems, r.Item)
			finalScores = append(finalScores, float32(r.FinalScore))
		}
	}

	// Truncate to TopK
	if q.TopK > 0 && len(finalItems) > q.TopK {
		finalItems = finalItems[:q.TopK]
		finalScores = finalScores[:q.TopK]
	}

	return types.QueryResult{
		Items:      finalItems,
		Scores:     finalScores,
		TotalFound: len(finalItems),
	}, nil
}

func (s *JSONStore) Delete(ctx context.Context, filter types.Filter) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for id, m := range s.memories {
		match := true
		if filter.Project != nil && m.Project != *filter.Project {
			match = false
		}
		if filter.Type != nil && m.Type != *filter.Type {
			match = false
		}
		if match {
			delete(s.memories, id)
			s.needsPersist = true
		}
	}
	return nil
}

func (s *JSONStore) Health(ctx context.Context) error {
	return nil // Local JSON in RAM is always healthy
}
