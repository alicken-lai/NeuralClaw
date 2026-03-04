package types

import "time"

type TaskStatus string

const (
	TaskStatusQueued    TaskStatus = "queued"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusSucceeded TaskStatus = "succeeded"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCanceled  TaskStatus = "canceled"
)

// Task represents a unit of work assigned to the agent.
type Task struct {
	ID           string     `json:"id"`
	Title        string     `json:"title"`
	Prompt       string     `json:"prompt"`
	Scope        string     `json:"scope"`
	Priority     int        `json:"priority"`
	Tags         []string   `json:"tags"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	ScheduledFor *time.Time `json:"scheduled_for,omitempty"`
	Status       TaskStatus `json:"status"`
	LastError    string     `json:"last_error,omitempty"`
	RunID        *string    `json:"run_id,omitempty"` // links to the active or latest Run
}

// Run represents an execution instance of a Task.
type Run struct {
	ID            string     `json:"id"`
	TaskID        string     `json:"task_id"`
	Scope         string     `json:"scope"`
	StartedAt     time.Time  `json:"started_at"`
	EndedAt       *time.Time `json:"ended_at,omitempty"`
	Status        TaskStatus `json:"status"`
	Command       string     `json:"command,omitempty"`        // optional free-form command
	OutputPreview string     `json:"output_preview,omitempty"` // last N lines
}
