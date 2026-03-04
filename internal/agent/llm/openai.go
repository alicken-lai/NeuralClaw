package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"neuralclaw/internal/observability"

	"go.uber.org/zap"
)

// OpenAIProvider implements the Provider interface for OpenAI-compatible REST APIs
// (including OpenAI, OpenRouter, Groq, Together, DeepSeek, etc).
type OpenAIProvider struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client
}

func NewOpenAIProvider(baseURL, apiKey string) *OpenAIProvider {
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	return &OpenAIProvider{
		BaseURL: baseURL,
		APIKey:  apiKey,
		HTTPClient: &http.Client{
			Timeout: 180 * time.Second,
		},
	}
}

// openaiChatRequest maps to the OpenAI JSON payload
type openaiChatRequest struct {
	Model       string                 `json:"model"`
	Messages    []openaiMessage        `json:"messages"`
	Temperature float64                `json:"temperature"`
	Tools       []openaiToolDefinition `json:"tools,omitempty"`
}

type openaiMessage struct {
	Role       string                   `json:"role"`
	Content    string                   `json:"content"`
	Name       string                   `json:"name,omitempty"`
	ToolCallID string                   `json:"tool_call_id,omitempty"`
	ToolCalls  []openaiToolCallResponse `json:"tool_calls,omitempty"`
}

type openaiToolDefinition struct {
	Type     string         `json:"type"`
	Function openaiFunction `json:"function"`
}

type openaiFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type openaiChatResponse struct {
	Choices []struct {
		Message struct {
			Role      string                   `json:"role"`
			Content   string                   `json:"content"`
			ToolCalls []openaiToolCallResponse `json:"tool_calls"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage TokenUsage `json:"usage"`
}

type openaiToolCallResponse struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

func mapToOpenAIMessages(in []ChatMessage) []openaiMessage {
	out := make([]openaiMessage, 0, len(in))
	for _, m := range in {
		out = append(out, openaiMessage{
			Role:       m.Role,
			Content:    m.Content,
			Name:       m.Name,
			ToolCallID: m.ToolCallID,
		})
	}
	return out
}

func mapToOpenAITools(in []ToolSpec) []openaiToolDefinition {
	if len(in) == 0 {
		return nil
	}
	out := make([]openaiToolDefinition, 0, len(in))
	for _, t := range in {
		out = append(out, openaiToolDefinition{
			Type: "function",
			Function: openaiFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		})
	}
	return out
}

func normalizeFinishReason(raw string) NormalizedStopReason {
	switch strings.ToLower(raw) {
	case "stop":
		return StopReasonEndTurn
	case "tool_calls", "function_call":
		return StopReasonToolCall
	case "length", "max_tokens":
		return StopReasonMaxToken
	default:
		return StopReasonUnknown
	}
}

// Chat executes the completion logic against the OpenAI-compatible endpoint.
func (p *OpenAIProvider) Chat(ctx context.Context, messages []ChatMessage, tools []ToolSpec, model string, temperature float64) (ChatResponse, error) {
	reqBody := openaiChatRequest{
		Model:       model,
		Messages:    mapToOpenAIMessages(messages),
		Temperature: temperature,
		Tools:       mapToOpenAITools(tools),
	}

	b, err := json.Marshal(reqBody)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := strings.TrimRight(p.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(b))
	if err != nil {
		return ChatResponse{}, fmt.Errorf("failed to create http request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	if p.APIKey != "" {
		httpReq.Header.Set("Authorization", "Bearer "+p.APIKey)
	}

	startTime := time.Now()
	resp, err := p.HTTPClient.Do(httpReq)
	if err != nil {
		return ChatResponse{}, fmt.Errorf("http execute failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		bodyStr, _ := io.ReadAll(resp.Body)
		return ChatResponse{}, fmt.Errorf("API returns error (Status %d): %s", resp.StatusCode, string(bodyStr))
	}

	var apiResp openaiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return ChatResponse{}, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResp.Choices) == 0 {
		return ChatResponse{}, fmt.Errorf("API returned 0 choices")
	}

	choice := apiResp.Choices[0]
	msg := choice.Message

	observability.Logger.Info("LLM Chat API completed",
		zap.String("model", model),
		zap.Duration("latency", time.Since(startTime)),
		zap.Int("input_tokens", apiResp.Usage.InputTokens),
		zap.Int("output_tokens", apiResp.Usage.OutputTokens),
		zap.String("finish_reason", choice.FinishReason),
	)

	// Map ToolCalls back out
	var parsedTools []ToolCall
	for _, tc := range msg.ToolCalls {
		if tc.Type == "function" {
			parsedTools = append(parsedTools, ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			})
		}
	}

	return ChatResponse{
		Text:       msg.Content,
		ToolCalls:  parsedTools,
		Usage:      apiResp.Usage,
		StopReason: normalizeFinishReason(choice.FinishReason),
	}, nil
}

// Ensure interface adherence
var _ Provider = (*OpenAIProvider)(nil)
