package cmd

import (
	"path/filepath"

	"github.com/spf13/cobra"

	"neuralclaw/internal/security"
	"neuralclaw/internal/taskstore"
	"neuralclaw/pkg/types"
)

var (
	securityScope string
	approvalID    string
	quarantineID  string
)

var securityCmd = &cobra.Command{
	Use:   "security",
	Short: "Inspect approvals, quarantine, and security events",
}

var securityApprovalsCmd = &cobra.Command{
	Use:   "approvals",
	Short: "Manage approval requests",
}

var securityApprovalsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List approval requests",
	RunE: func(cmd *cobra.Command, args []string) error {
		guard := newSecurityGuard()
		items, err := guard.ListApprovals(securityScope)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			cmd.Println("No approval requests found.")
			return nil
		}
		for _, item := range items {
			cmd.Printf("%s  %-9s %-6s %-8s %s\n", item.ID, item.Status, item.Source, item.Kind, item.Scope)
		}
		return nil
	},
}

var securityApprovalsApproveCmd = &cobra.Command{
	Use:   "approve",
	Short: "Approve a pending request",
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateApproval(cmd, security.ApprovalApproved)
	},
}

var securityApprovalsRejectCmd = &cobra.Command{
	Use:   "reject",
	Short: "Reject a pending request",
	RunE: func(cmd *cobra.Command, args []string) error {
		return updateApproval(cmd, security.ApprovalRejected)
	},
}

var securityQuarantineCmd = &cobra.Command{
	Use:   "quarantine",
	Short: "Inspect quarantined memory items",
}

var securityQuarantineListCmd = &cobra.Command{
	Use:   "list",
	Short: "List quarantined items",
	RunE: func(cmd *cobra.Command, args []string) error {
		guard := newSecurityGuard()
		items, err := guard.ListQuarantine(securityScope)
		if err != nil {
			return err
		}
		if len(items) == 0 {
			cmd.Println("No quarantined items found.")
			return nil
		}
		for _, item := range items {
			preview := item.Item.Text
			if preview == "" {
				preview = item.Item.BM25Text
			}
			if len(preview) > 80 {
				preview = preview[:80] + "..."
			}
			cmd.Printf("%s  %-8s %-8s %s\n", item.ID, item.RiskLevel, item.Scope, preview)
		}
		return nil
	},
}

var securityQuarantineReviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Show a quarantined item",
	RunE: func(cmd *cobra.Command, args []string) error {
		guard := newSecurityGuard()
		item, err := guard.GetQuarantine(quarantineID)
		if err != nil {
			return err
		}
		cmd.Printf("ID: %s\nScope: %s\nRisk: %s\nSource: %s\n", item.ID, item.Scope, item.RiskLevel, item.Source)
		cmd.Printf("Reasons:\n")
		for _, reason := range item.Reasons {
			cmd.Printf("  - %s\n", reason)
		}
		cmd.Printf("\nContent:\n%s\n", item.Item.Text)
		return nil
	},
}

func init() {
	securityApprovalsListCmd.Flags().StringVar(&securityScope, "scope", "", "Filter approvals by scope")
	securityApprovalsApproveCmd.Flags().StringVar(&approvalID, "id", "", "Approval request ID")
	securityApprovalsRejectCmd.Flags().StringVar(&approvalID, "id", "", "Approval request ID")
	_ = securityApprovalsApproveCmd.MarkFlagRequired("id")
	_ = securityApprovalsRejectCmd.MarkFlagRequired("id")
	securityApprovalsCmd.AddCommand(securityApprovalsListCmd, securityApprovalsApproveCmd, securityApprovalsRejectCmd)

	securityQuarantineListCmd.Flags().StringVar(&securityScope, "scope", "", "Filter quarantined items by scope")
	securityQuarantineReviewCmd.Flags().StringVar(&quarantineID, "id", "", "Quarantine record ID")
	_ = securityQuarantineReviewCmd.MarkFlagRequired("id")
	securityQuarantineCmd.AddCommand(securityQuarantineListCmd, securityQuarantineReviewCmd)

	securityCmd.AddCommand(securityApprovalsCmd, securityQuarantineCmd)
	rootCmd.AddCommand(securityCmd)
}

func updateApproval(cmd *cobra.Command, status security.ApprovalStatus) error {
	guard := newSecurityGuard()
	item, err := guard.UpdateApprovalStatus(approvalID, status)
	if err != nil {
		return err
	}
	if item.TargetID != "" && item.TargetType == "task" {
		taskStore, err := taskstore.NewJSONFileStore(filepath.Join(".", "data"))
		if err == nil {
			task, getErr := taskStore.GetTask(item.TargetID)
			if getErr == nil {
				switch status {
				case security.ApprovalApproved:
					task.Status = types.TaskStatusQueued
					task.LastError = ""
				case security.ApprovalRejected:
					task.Status = types.TaskStatusBlocked
					task.LastError = "Approval rejected"
				}
				_ = taskStore.SaveTask(task)
			}
		}
	}
	cmd.Printf("Approval %s marked as %s\n", item.ID, item.Status)
	return nil
}
