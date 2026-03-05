package observability

import (
	"encoding/json"
	"os"
	"sync"
	"time"
)

// TokenUsageLog represents a single token consumption event.
type TokenUsageLog struct {
	Timestamp    time.Time `json:"ts"`
	Source       string    `json:"source"` // e.g. "TaskRun:<RunID>", "Cron:DMN", "Ingest:Embed"
	Model        string    `json:"model"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	TotalTokens  int       `json:"total_tokens"`
}

// TokenTracker is a thread-safe, append-only JSONL logger for token usage.
type TokenTracker struct {
	mu       sync.Mutex
	filePath string
	file     *os.File
	encoder  *json.Encoder
	// In-memory buffer for fast queries (kept for the current session).
	logs []TokenUsageLog
}

// NewTokenTracker creates a new tracker that writes to the given JSONL file path.
func NewTokenTracker(filePath string) (*TokenTracker, error) {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}

	tt := &TokenTracker{
		filePath: filePath,
		file:     f,
		encoder:  json.NewEncoder(f),
	}

	// Load existing logs into memory for querying.
	tt.loadExisting()

	return tt, nil
}

// Record appends a new usage event to disk and in-memory buffer.
func (tt *TokenTracker) Record(source, model string, input, output, total int) {
	entry := TokenUsageLog{
		Timestamp:    time.Now(),
		Source:       source,
		Model:        model,
		InputTokens:  input,
		OutputTokens: output,
		TotalTokens:  total,
	}

	tt.mu.Lock()
	defer tt.mu.Unlock()

	tt.logs = append(tt.logs, entry)
	_ = tt.encoder.Encode(entry) // Best-effort write to disk
}

// GetLogs returns all logs in the current session and loaded history.
func (tt *TokenTracker) GetLogs() []TokenUsageLog {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	copied := make([]TokenUsageLog, len(tt.logs))
	copy(copied, tt.logs)
	return copied
}

// DailySummary represents aggregated token usage for one day + model.
type DailySummary struct {
	Date         string `json:"date"`
	Model        string `json:"model"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	TotalTokens  int    `json:"total_tokens"`
	CallCount    int    `json:"call_count"`
}

// GetDailySummaries returns the last N days of daily summaries grouped by model.
func (tt *TokenTracker) GetDailySummaries(days int) []DailySummary {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -days)
	buckets := make(map[string]*DailySummary) // key: "2026-03-05|model-name"

	for _, l := range tt.logs {
		if l.Timestamp.Before(cutoff) {
			continue
		}
		dateStr := l.Timestamp.Format("2006-01-02")
		key := dateStr + "|" + l.Model
		b, ok := buckets[key]
		if !ok {
			b = &DailySummary{Date: dateStr, Model: l.Model}
			buckets[key] = b
		}
		b.InputTokens += l.InputTokens
		b.OutputTokens += l.OutputTokens
		b.TotalTokens += l.TotalTokens
		b.CallCount++
	}

	result := make([]DailySummary, 0, len(buckets))
	for _, v := range buckets {
		result = append(result, *v)
	}
	return result
}

// SourceSummary represents aggregated token usage grouped by source prefix.
type SourceSummary struct {
	Source       string `json:"source"`
	InputTokens  int    `json:"input_tokens"`
	OutputTokens int    `json:"output_tokens"`
	TotalTokens  int    `json:"total_tokens"`
	CallCount    int    `json:"call_count"`
}

// GetSourceSummaries returns aggregated usage per source (TaskRun vs Cron vs Ingest).
func (tt *TokenTracker) GetSourceSummaries() []SourceSummary {
	tt.mu.Lock()
	defer tt.mu.Unlock()

	buckets := make(map[string]*SourceSummary)
	for _, l := range tt.logs {
		b, ok := buckets[l.Source]
		if !ok {
			b = &SourceSummary{Source: l.Source}
			buckets[l.Source] = b
		}
		b.InputTokens += l.InputTokens
		b.OutputTokens += l.OutputTokens
		b.TotalTokens += l.TotalTokens
		b.CallCount++
	}

	result := make([]SourceSummary, 0, len(buckets))
	for _, v := range buckets {
		result = append(result, *v)
	}
	return result
}

// Close flushes and closes the underlying file.
func (tt *TokenTracker) Close() error {
	tt.mu.Lock()
	defer tt.mu.Unlock()
	return tt.file.Close()
}

// loadExisting reads the JSONL file and populates the in-memory buffer.
func (tt *TokenTracker) loadExisting() {
	f, err := os.Open(tt.filePath)
	if err != nil {
		return // File doesn't exist or is unreadable — start fresh
	}
	defer f.Close()

	decoder := json.NewDecoder(f)
	for decoder.More() {
		var entry TokenUsageLog
		if err := decoder.Decode(&entry); err != nil {
			break
		}
		tt.logs = append(tt.logs, entry)
	}
}
