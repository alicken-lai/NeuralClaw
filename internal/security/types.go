package security

import (
	"time"

	"neuralclaw/pkg/types"
)

type PromptRiskLevel string

const (
	RiskLow      PromptRiskLevel = "low"
	RiskMedium   PromptRiskLevel = "medium"
	RiskHigh     PromptRiskLevel = "high"
	RiskCritical PromptRiskLevel = "critical"
)

type PromptInspection struct {
	RiskLevel    PromptRiskLevel `json:"risk_level"`
	Score        int             `json:"score"`
	Reasons      []string        `json:"reasons,omitempty"`
	MatchedRules []string        `json:"matched_rules,omitempty"`
	Action       string          `json:"action"`
}

type ToolDecision string

const (
	ToolAllow           ToolDecision = "allow"
	ToolDeny            ToolDecision = "deny"
	ToolRequireApproval ToolDecision = "require_approval"
)

type ToolPolicyRule struct {
	ToolName                 string   `json:"tool_name"`
	AllowPatterns            []string `json:"allow_patterns,omitempty"`
	DenyPatterns             []string `json:"deny_patterns,omitempty"`
	RequireApprovalPatterns  []string `json:"require_approval_patterns,omitempty"`
}

type ToolPolicy struct {
	Default ToolDecision     `json:"default"`
	Rules   []ToolPolicyRule `json:"rules,omitempty"`
}

type ToolEvaluation struct {
	Decision    ToolDecision `json:"decision"`
	Reasons     []string     `json:"reasons,omitempty"`
	MatchedRule string       `json:"matched_rule,omitempty"`
}

type MemoryValidationResult struct {
	Allowed     bool            `json:"allowed"`
	RiskLevel   PromptRiskLevel `json:"risk_level"`
	Reasons     []string        `json:"reasons,omitempty"`
	Quarantined bool            `json:"quarantined"`
}

type SecurityEventType string

const (
	EventPromptInspected   SecurityEventType = "prompt_inspected"
	EventPromptBlocked     SecurityEventType = "prompt_blocked"
	EventToolEvaluated     SecurityEventType = "tool_evaluated"
	EventToolDenied        SecurityEventType = "tool_denied"
	EventApprovalRequired  SecurityEventType = "approval_required"
	EventMemoryQuarantined SecurityEventType = "memory_quarantined"
	EventCrossScopeAttempt SecurityEventType = "cross_scope_attempt"
)

type SecurityEvent struct {
	ID        string            `json:"id"`
	Time      time.Time         `json:"time"`
	Scope     string            `json:"scope"`
	EventType SecurityEventType `json:"event_type"`
	RiskLevel string            `json:"risk_level,omitempty"`
	Actor     string            `json:"actor"`
	Target    string            `json:"target,omitempty"`
	Summary   string            `json:"summary"`
	Details   map[string]any    `json:"details,omitempty"`
}

type ApprovalStatus string

const (
	ApprovalPending  ApprovalStatus = "pending"
	ApprovalApproved ApprovalStatus = "approved"
	ApprovalRejected ApprovalStatus = "rejected"
)

type ApprovalRequest struct {
	ID         string         `json:"id"`
	Time       time.Time      `json:"time"`
	Scope      string         `json:"scope"`
	Source     string         `json:"source"`
	Kind       string         `json:"kind"`
	TargetID   string         `json:"target_id,omitempty"`
	TargetType string         `json:"target_type,omitempty"`
	Payload    string         `json:"payload"`
	Reason     []string       `json:"reason,omitempty"`
	Status     ApprovalStatus `json:"status"`
}

type QuarantineRecord struct {
	ID         string                 `json:"id"`
	Time       time.Time              `json:"time"`
	Scope      string                 `json:"scope"`
	Source     string                 `json:"source"`
	Reasons    []string               `json:"reasons,omitempty"`
	RiskLevel  PromptRiskLevel        `json:"risk_level"`
	Provenance types.Provenance       `json:"provenance"`
	Item       types.MemoryItem       `json:"item"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}
