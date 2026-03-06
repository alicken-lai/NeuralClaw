package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"neuralclaw/internal/agent/llm"
	"neuralclaw/internal/config"
	"neuralclaw/internal/observability"
	"neuralclaw/internal/security"
	"neuralclaw/pkg/types"
)

type DispatchRequest struct {
	TaskID   string
	Scope    string
	Prompt   string
	Tags     []string
	Priority int
	Actor    string
}

type Dispatcher struct {
	guard *security.Guard
}

func NewDispatcher(guard *security.Guard) *Dispatcher {
	return &Dispatcher{guard: guard}
}

func (d *Dispatcher) Dispatch(ctx context.Context, req DispatchRequest) (types.Run, error) {
	runID := uuid.New().String()
	run := types.Run{
		ID:        runID,
		TaskID:    req.TaskID,
		Scope:     req.Scope,
		StartedAt: time.Now(),
		Status:    types.TaskStatusRunning,
	}

	observability.Logger.Info("Dispatching task",
		zap.String("task_id", req.TaskID),
		zap.String("run_id", runID),
		zap.String("scope", req.Scope),
	)

		go func(runId, taskId, scope, prompt string) {
		defer observability.Logger.Info("Task execution context finished", zap.String("run_id", runId))

		cfg := config.GlobalConfig.Agent
		if cfg.APIKey == "" {
			observability.Logger.Error("No AGENT API Key configured. Task failed.", zap.String("run_id", runId))
			return
		}

		provider := llm.NewOpenAIProvider(cfg.BaseURL, cfg.APIKey)
		model := cfg.Model
		if model == "" {
			model = "gpt-4o-mini"
		}

		sysPrompt := "You are NeuralClaw, a highly capable autonomous agent.\n" +
			"You possess tools to read files and execute shell commands.\n" +
			"Your current task scope is: " + scope + "\n" +
			"Think step-by-step, use your tools to gather info, and complete the task."

		messages := []llm.ChatMessage{
			{Role: llm.RoleSystem, Content: sysPrompt},
			{Role: llm.RoleUser, Content: prompt},
		}

		activeTools := []Tool{
			&ShellRunnerTool{},
			&FileReaderTool{},
		}

		var activeToolSpecs []llm.ToolSpec
		for _, t := range activeTools {
			activeToolSpecs = append(activeToolSpecs, t.Spec())
		}

		agentCtx := context.Background()

		for i := 0; i < 15; i++ {
			observability.Logger.Info("Agent querying LLM...", zap.Int("iteration", i+1), zap.String("model", model))
			resp, err := provider.Chat(agentCtx, messages, activeToolSpecs, model, 0.2)
			if err != nil {
				observability.Logger.Error("LLM Provider error", zap.Error(err), zap.String("run_id", runId))
				return
			}

			// Record token usage
			if observability.Tracker != nil {
				observability.Tracker.Record(
					fmt.Sprintf("TaskRun:%s", runId),
					model,
					resp.Usage.InputTokens,
					resp.Usage.OutputTokens,
					resp.Usage.TotalTokens,
				)
			}

			if !resp.HasToolCalls() || resp.StopReason == llm.StopReasonEndTurn {
				observability.Logger.Info("Agent finished task successfully",
					zap.String("run_id", runId),
					zap.String("final_response", resp.Text))
				return
			}

			b, _ := json.Marshal(resp.ToolCalls)
			messages = append(messages, llm.ChatMessage{
				Role:    llm.RoleAssistant,
				Content: fmt.Sprintf("Tool Calls Requested:\n%s\n%s", string(b), resp.Text),
			})

			for _, tc := range resp.ToolCalls {
				observability.Logger.Info("Agent executing tool", zap.String("tool", tc.Name), zap.String("args", tc.Arguments))

				var toolResult string
				if d.guard != nil {
					eval, approval, evalErr := d.guard.EvaluateTool(scope, "agent", runId, tc.Name, tc.Arguments)
					if evalErr != nil {
						toolResult = fmt.Sprintf("Security guard failed: %v", evalErr)
					} else if eval.Decision == security.ToolDeny {
						toolResult = fmt.Sprintf("Tool denied by security policy: %s", strings.Join(eval.Reasons, "; "))
					} else if eval.Decision == security.ToolRequireApproval {
						if approval != nil {
							toolResult = fmt.Sprintf("Tool execution pending approval (%s): %s", approval.ID, strings.Join(eval.Reasons, "; "))
						} else {
							toolResult = fmt.Sprintf("Tool execution requires approval: %s", strings.Join(eval.Reasons, "; "))
						}
					}
				}
				if toolResult != "" {
					messages = append(messages, llm.ChatMessage{
						Role:       llm.RoleTool,
						Name:       tc.Name,
						ToolCallID: tc.ID,
						Content:    toolResult,
					})
					continue
				}

				found := false
				for _, t := range activeTools {
					if t.Name() == tc.Name {
						res, tErr := t.Execute(agentCtx, tc.Arguments)
						if tErr != nil {
							toolResult = fmt.Sprintf("Error: %v", tErr)
						} else {
							toolResult = res
						}
						found = true
						break
					}
				}

				if !found {
					toolResult = fmt.Sprintf("Error: tool '%s' not found", tc.Name)
				}

				messages = append(messages, llm.ChatMessage{
					Role:       llm.RoleTool,
					Name:       tc.Name,
					ToolCallID: tc.ID,
					Content:    toolResult,
				})
			}
		}

		observability.Logger.Warn("Agent loop terminated (max iterations reached)", zap.String("run_id", runId))
	}(runID, req.TaskID, req.Scope, req.Prompt)

	return run, nil
}
