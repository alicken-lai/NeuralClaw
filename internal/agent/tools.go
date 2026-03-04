package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"

	"neuralclaw/internal/agent/llm"
)

// Tool represents a capability the agent can invoke.
type Tool interface {
	Name() string
	Description() string
	// Parameters returns the JSON Schema for the tool's arguments.
	Parameters() map[string]interface{}
	// Execute runs the tool with the given JSON arguments and returns the result string.
	Execute(ctx context.Context, arguments string) (string, error)
	// Spec returns the configured ToolSpec struct for inclusion in the LLM payload.
	Spec() llm.ToolSpec
}

// ShellRunnerTool allows the agent to execute shell commands natively.
type ShellRunnerTool struct{}

func (t *ShellRunnerTool) Name() string { return "shell_runner" }

func (t *ShellRunnerTool) Description() string {
	return "Execute a raw shell command on the host. Highly dangerous but powerful. Ensure safety."
}

func (t *ShellRunnerTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{
				"type":        "string",
				"description": "The shell command to execute.",
			},
		},
		"required": []string{"command"},
	}
}

func (t *ShellRunnerTool) Spec() llm.ToolSpec {
	return llm.ToolSpec{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  t.Parameters(),
	}
}

func (t *ShellRunnerTool) Execute(ctx context.Context, arguments string) (string, error) {
	var args struct {
		Command string `json:"command"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	cmd := exec.CommandContext(ctx, "powershell", "-c", args.Command)
	out, err := cmd.CombinedOutput()
	result := string(out)

	if err != nil {
		return fmt.Sprintf("Command failed with error: %v\nOutput:\n%s", err, result), nil
	}

	if result == "" {
		return "Command executed successfully with no output.", nil
	}
	return result, nil
}

// FileReaderTool allows the agent to read file contents.
type FileReaderTool struct{}

func (t *FileReaderTool) Name() string { return "file_reader" }

func (t *FileReaderTool) Description() string {
	return "Reads the absolute path of a file and returns its plaintext content."
}

func (t *FileReaderTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{
				"type":        "string",
				"description": "Absolute path to the file on disk.",
			},
		},
		"required": []string{"path"},
	}
}

func (t *FileReaderTool) Spec() llm.ToolSpec {
	return llm.ToolSpec{
		Name:        t.Name(),
		Description: t.Description(),
		Parameters:  t.Parameters(),
	}
}

func (t *FileReaderTool) Execute(ctx context.Context, arguments string) (string, error) {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("failed to parse tool arguments: %w", err)
	}

	b, err := os.ReadFile(args.Path)
	if err != nil {
		return fmt.Sprintf("Failed to read file '%s': %v", args.Path, err), nil
	}
	return string(b), nil
}
