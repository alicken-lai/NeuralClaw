package taskstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"neuralclaw/pkg/types"
)

type JSONFileStore struct {
	mu         sync.RWMutex
	tasksFile  string
	runsFile   string
	tasksCache map[string]types.Task
	runsCache  map[string]types.Run
}

func NewJSONFileStore(dataDir string) (*JSONFileStore, error) {
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data dir: %w", err)
	}

	store := &JSONFileStore{
		tasksFile:  filepath.Join(dataDir, "tasks.json"),
		runsFile:   filepath.Join(dataDir, "runs.json"),
		tasksCache: make(map[string]types.Task),
		runsCache:  make(map[string]types.Run),
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *JSONFileStore) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load Tasks
	if data, err := os.ReadFile(s.tasksFile); err == nil {
		var tasks []types.Task
		if err := json.Unmarshal(data, &tasks); err == nil {
			for _, t := range tasks {
				s.tasksCache[t.ID] = t
			}
		}
	}

	// Load Runs
	if data, err := os.ReadFile(s.runsFile); err == nil {
		var runs []types.Run
		if err := json.Unmarshal(data, &runs); err == nil {
			for _, r := range runs {
				s.runsCache[r.ID] = r
			}
		}
	}
	return nil
}

func (s *JSONFileStore) saveAll() error {
	tasks := make([]types.Task, 0, len(s.tasksCache))
	for _, t := range s.tasksCache {
		tasks = append(tasks, t)
	}
	tasksData, _ := json.MarshalIndent(tasks, "", "  ")
	if err := os.WriteFile(s.tasksFile, tasksData, 0644); err != nil {
		return err
	}

	runs := make([]types.Run, 0, len(s.runsCache))
	for _, r := range s.runsCache {
		runs = append(runs, r)
	}
	runsData, _ := json.MarshalIndent(runs, "", "  ")
	if err := os.WriteFile(s.runsFile, runsData, 0644); err != nil {
		return err
	}

	return nil
}

// Ensure interface compliance
var _ Store = (*JSONFileStore)(nil)

func (s *JSONFileStore) SaveTask(task types.Task) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	task.UpdatedAt = time.Now()
	s.tasksCache[task.ID] = task
	return s.saveAll()
}

func (s *JSONFileStore) GetTask(id string) (types.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if t, ok := s.tasksCache[id]; ok {
		return t, nil
	}
	return types.Task{}, fmt.Errorf("task not found")
}

func (s *JSONFileStore) ListTasks(scope string) ([]types.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var res []types.Task
	for _, t := range s.tasksCache {
		if t.Scope == scope {
			res = append(res, t)
		}
	}
	// Sort newest first
	return res, nil
}

func (s *JSONFileStore) SaveRun(run types.Run) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runsCache[run.ID] = run
	return s.saveAll()
}

func (s *JSONFileStore) GetRun(id string) (types.Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if r, ok := s.runsCache[id]; ok {
		return r, nil
	}
	return types.Run{}, fmt.Errorf("run not found")
}

func (s *JSONFileStore) ListRuns(scope string) ([]types.Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var res []types.Run
	for _, r := range s.runsCache {
		if r.Scope == scope {
			res = append(res, r)
		}
	}
	return res, nil
}

func (s *JSONFileStore) GetRunsByTask(taskID string) ([]types.Run, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var res []types.Run
	for _, r := range s.runsCache {
		if r.TaskID == taskID {
			res = append(res, r)
		}
	}
	return res, nil
}
