package security

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"neuralclaw/internal/observability"
)

type AuditLogger struct {
	path string
	mu   sync.Mutex
}

func NewAuditLogger(path string) (*AuditLogger, error) {
	if path == "" {
		return nil, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return nil, fmt.Errorf("create audit log dir: %w", err)
	}
	return &AuditLogger{path: path}, nil
}

func (l *AuditLogger) Append(event SecurityEvent) {
	if l == nil || l.path == "" {
		return
	}
	if event.ID == "" {
		event.ID = uuid.New().String()
	}
	l.mu.Lock()
	defer l.mu.Unlock()

	file, err := os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		observability.Logger.Warn("Failed to open security audit log", zap.Error(err), zap.String("path", l.path))
		return
	}
	defer file.Close()

	if err := json.NewEncoder(file).Encode(event); err != nil {
		observability.Logger.Warn("Failed to append security audit log", zap.Error(err), zap.String("path", l.path))
	}
}

func (l *AuditLogger) List(scope string, limit int) ([]SecurityEvent, error) {
	if l == nil || l.path == "" {
		return nil, nil
	}
	file, err := os.Open(l.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer file.Close()

	events := make([]SecurityEvent, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var event SecurityEvent
		if err := json.Unmarshal(scanner.Bytes(), &event); err != nil {
			continue
		}
		if scope != "" && event.Scope != "" && event.Scope != scope {
			continue
		}
		events = append(events, event)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if limit > 0 && len(events) > limit {
		events = events[len(events)-limit:]
	}
	reverseSecurityEvents(events)
	return events, nil
}

func reverseSecurityEvents(events []SecurityEvent) {
	for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
		events[i], events[j] = events[j], events[i]
	}
}
