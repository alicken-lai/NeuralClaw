package security

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
)

type QuarantineStore struct {
	path  string
	mu    sync.RWMutex
	items map[string]QuarantineRecord
}

func NewQuarantineStore(path string) (*QuarantineStore, error) {
	if path == "" {
		return nil, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create quarantine dir: %w", err)
	}
	store := &QuarantineStore{
		path:  path,
		items: make(map[string]QuarantineRecord),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *QuarantineStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var items []QuarantineRecord
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	for _, item := range items {
		s.items[item.ID] = item
	}
	return nil
}

func (s *QuarantineStore) saveLocked() error {
	items := make([]QuarantineRecord, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Time.After(items[j].Time)
	})
	data, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path, data, 0644)
}

func (s *QuarantineStore) Add(record QuarantineRecord) (QuarantineRecord, error) {
	if s == nil {
		return record, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	if record.Time.IsZero() {
		record.Time = time.Now()
	}
	s.items[record.ID] = record
	return record, s.saveLocked()
}

func (s *QuarantineStore) Get(id string) (QuarantineRecord, error) {
	if s == nil {
		return QuarantineRecord{}, fmt.Errorf("quarantine store unavailable")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[id]
	if !ok {
		return QuarantineRecord{}, fmt.Errorf("quarantine item not found")
	}
	return item, nil
}

func (s *QuarantineStore) List(scope string) ([]QuarantineRecord, error) {
	if s == nil {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]QuarantineRecord, 0, len(s.items))
	for _, item := range s.items {
		if scope != "" && item.Scope != scope {
			continue
		}
		items = append(items, item)
	}
	sort.Slice(items, func(i, j int) bool {
		return items[i].Time.After(items[j].Time)
	})
	return items, nil
}
