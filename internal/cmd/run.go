package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"neuralclaw/internal/agent"
	"neuralclaw/internal/config"
	"neuralclaw/internal/observability"
	"neuralclaw/pkg/types"

	"github.com/google/uuid"
)

var (
	runTask  string
	runScope string
)

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Dispatch an agent task from the command line",
	Run: func(cmd *cobra.Command, args []string) {
		if config.GlobalConfig.Agent.APIKey == "" {
			observability.Logger.Fatal("agent.api_key is empty in config. Cannot run agent without an LLM provider key.")
		}

		taskID := uuid.New().String()
		observability.Logger.Info("Dispatching agent run",
			zap.String("task_id", taskID),
			zap.String("scope", runScope),
			zap.String("task", runTask),
		)

		dispatcher := agent.NewDispatcher()
		run, err := dispatcher.Dispatch(context.Background(), agent.DispatchRequest{
			TaskID:   taskID,
			Scope:    runScope,
			Prompt:   runTask,
			Priority: 5,
		})
		if err != nil {
			observability.Logger.Fatal("Failed to dispatch task", zap.Error(err))
		}

		// Ensure the run actually includes a Status
		if run.Status != types.TaskStatusRunning {
			observability.Logger.Error("Unexpected run status after dispatch", zap.String("status", string(run.Status)))
			return
		}

		fmt.Printf("Task dispatched successfully!\n")
		fmt.Printf("  Run ID   : %s\n", run.ID)
		fmt.Printf("  Task ID  : %s\n", run.TaskID)
		fmt.Printf("  Scope    : %s\n", run.Scope)
		fmt.Printf("  Status   : %s\n", run.Status)
		fmt.Println("\nThe agent loop is executing in the background. Check logs for progress.")
	},
}

func init() {
	runCmd.Flags().StringVar(&runTask, "task", "", "Task description / prompt to send to the agent")
	runCmd.Flags().StringVar(&runScope, "scope", "global", "Memory scope for context retrieval")
	_ = runCmd.MarkFlagRequired("task")
	rootCmd.AddCommand(runCmd)
}
