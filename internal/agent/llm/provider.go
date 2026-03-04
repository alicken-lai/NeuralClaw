package llm

import (
	"context"
)

// Role constants
const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

// ChatMessage represents a single message in a conversation.
type ChatMessage struct {
	Role       string `json:"role"`
	Content    string `json:"content"`
	Name       string `json:"name,omitempty"`         // Required for tool results in some APIs
	ToolCallID string `json:"tool_call_id,omitempty"` // ID of the tool call this message is responding to
}

// ToolCall represents a specific tool execution request from the LLM.
type ToolCall struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Arguments string `json:"arguments"` // JSON-encoded string of arguments
}

// TokenUsage tracks token consumption.
type TokenUsage struct {
	InputTokens  int `json:"prompt_tokens"`
	OutputTokens int `json:"completion_tokens"`
	TotalTokens  int `json:"total_tokens"`
}

// NormalizedStopReason provides a provider-agnostic stop condition.
type NormalizedStopReason string

const (
	StopReasonEndTurn  NormalizedStopReason = "end_turn"
	StopReasonToolCall NormalizedStopReason = "tool_call"
	StopReasonMaxToken NormalizedStopReason = "max_tokens"
	StopReasonUnknown  NormalizedStopReason = "unknown"
)

// ChatResponse is the payload returned by the Provider.
type ChatResponse struct {
	Text        string               `json:"text"`
	ToolCalls   []ToolCall           `json:"tool_calls"`
	Usage       TokenUsage           `json:"usage"`
	StopReason  NormalizedStopReason `json:"stop_reason"`
	RawResponse interface{}          `json:"-"` // Opaque access to the raw payload if needed for debugging
}

func (r *ChatResponse) HasToolCalls() bool {
	return len(r.ToolCalls) > 0
}

// ToolSpec represents the JSON Schema definition for a single tool.
type ToolSpec struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Provider defines the interface all LLM clients must implement.
type Provider interface {
	// Chat performs a stateless, synchronous turn, returning the generated response.
	// `tools` can be nil or empty if no tools are available.
	Chat(ctx context.Context, messages []ChatMessage, tools []ToolSpec, model string, temperature float64) (ChatResponse, error)
}
