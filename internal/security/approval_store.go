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

type ApprovalStore struct {
	path  string
	mu    sync.RWMutex
	items map[string]ApprovalRequest
}

func NewApprovalStore(path string) (*ApprovalStore, error) {
	if path == "" {
		return nil, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create approval dir: %w", err)
	}
	store := &ApprovalStore{
		path:  path,
		items: make(map[string]ApprovalRequest),
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *ApprovalStore) load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var items []ApprovalRequest
	if err := json.Unmarshal(data, &items); err != nil {
		return err
	}
	for _, item := range items {
		s.items[item.ID] = item
	}
	return nil
}

func (s *ApprovalStore) saveLocked() error {
	items := make([]ApprovalRequest, 0, len(s.items))
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

func (s *ApprovalStore) Create(req ApprovalRequest) (ApprovalRequest, error) {
	if s == nil {
		return req, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if req.ID == "" {
		req.ID = uuid.New().String()
	}
	if req.Time.IsZero() {
		req.Time = time.Now()
	}
	if req.Status == "" {
		req.Status = ApprovalPending
	}
	s.items[req.ID] = req
	return req, s.saveLocked()
}

func (s *ApprovalStore) Get(id string) (ApprovalRequest, error) {
	if s == nil {
		return ApprovalRequest{}, fmt.Errorf("approval store unavailable")
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	item, ok := s.items[id]
	if !ok {
		return ApprovalRequest{}, fmt.Errorf("approval not found")
	}
	return item, nil
}

func (s *ApprovalStore) List(scope string) ([]ApprovalRequest, error) {
	if s == nil {
		return nil, nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]ApprovalRequest, 0, len(s.items))
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

func (s *ApprovalStore) UpdateStatus(id string, status ApprovalStatus) (ApprovalRequest, error) {
	if s == nil {
		return ApprovalRequest{}, fmt.Errorf("approval store unavailable")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.items[id]
	if !ok {
		return ApprovalRequest{}, fmt.Errorf("approval not found")
	}
	item.Status = status
	s.items[id] = item
	return item, s.saveLocked()
}
