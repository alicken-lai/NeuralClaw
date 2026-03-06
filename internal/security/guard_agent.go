package security

import (
	"fmt"
	"time"

	"neuralclaw/internal/config"
	"neuralclaw/pkg/types"
)

type Guard struct {
	cfg        config.SecurityConfig
	toolPolicy ToolPolicy
	audit      *AuditLogger
	approvals  *ApprovalStore
	quarantine *QuarantineStore
}

func NewGuard(cfg config.SecurityConfig) (*Guard, error) {
	audit, err := NewAuditLogger(cfg.AuditLogPath)
	if err != nil {
		return nil, err
	}
	approvals, err := NewApprovalStore(cfg.ApprovalsStorePath)
	if err != nil {
		return nil, err
	}
	quarantine, err := NewQuarantineStore(cfg.QuarantineStorePath)
	if err != nil {
		return nil, err
	}
	return &Guard{
		cfg:        cfg,
		toolPolicy: ToolPolicyFromConfig(cfg.ToolPolicy),
		audit:      audit,
		approvals:  approvals,
		quarantine: quarantine,
	}, nil
}

func (g *Guard) Enabled() bool {
	return g != nil && g.cfg.Enabled
}

func (g *Guard) InspectPrompt(scope, actor, target, input string) (PromptInspection, *ApprovalRequest, error) {
	inspection := InspectPrompt(scope, input)
	if !g.Enabled() || !g.cfg.PromptFirewall {
		inspection = PromptInspection{RiskLevel: RiskLow, Action: "allow"}
	}

	g.appendEvent(SecurityEvent{
		Time:      time.Now(),
		Scope:     scope,
		EventType: EventPromptInspected,
		RiskLevel: string(inspection.RiskLevel),
		Actor:     actor,
		Target:    target,
		Summary:   fmt.Sprintf("prompt inspected with action %s", inspection.Action),
		Details: map[string]any{
			"score":         inspection.Score,
			"reasons":       inspection.Reasons,
			"matched_rules": inspection.MatchedRules,
		},
	})

	switch inspection.Action {
	case "block":
		g.appendEvent(SecurityEvent{
			Time:      time.Now(),
			Scope:     scope,
			EventType: EventPromptBlocked,
			RiskLevel: string(inspection.RiskLevel),
			Actor:     actor,
			Target:    target,
			Summary:   "prompt blocked by security guard",
			Details:   map[string]any{"reasons": inspection.Reasons},
		})
		return inspection, nil, nil
	case "require_approval":
		if !g.cfg.ApprovalMode {
			inspection.Action = "block"
			g.appendEvent(SecurityEvent{
				Time:      time.Now(),
				Scope:     scope,
				EventType: EventPromptBlocked,
				RiskLevel: string(inspection.RiskLevel),
				Actor:     actor,
				Target:    target,
				Summary:   "prompt blocked because approval mode is disabled",
				Details:   map[string]any{"reasons": inspection.Reasons},
			})
			return inspection, nil, nil
		}
		req, err := g.approvals.Create(ApprovalRequest{
			Scope:      scope,
			Source:     actor,
			Kind:       "prompt",
			TargetID:   target,
			TargetType: "task",
			Payload:    input,
			Reason:     inspection.Reasons,
			Status:     ApprovalPending,
		})
		if err != nil {
			return inspection, nil, err
		}
		g.appendEvent(SecurityEvent{
			Time:      time.Now(),
			Scope:     scope,
			EventType: EventApprovalRequired,
			RiskLevel: string(inspection.RiskLevel),
			Actor:     actor,
			Target:    target,
			Summary:   "prompt requires approval",
			Details: map[string]any{
				"approval_id": req.ID,
				"reasons":     inspection.Reasons,
			},
		})
		return inspection, &req, nil
	default:
		return inspection, nil, nil
	}
}

func (g *Guard) EvaluateTool(scope, actor, target, toolName, payload string) (ToolEvaluation, *ApprovalRequest, error) {
	if !g.Enabled() {
		return ToolEvaluation{Decision: ToolAllow}, nil, nil
	}
	eval := EvaluateToolCall(scope, toolName, payload, g.toolPolicy)
	g.appendEvent(SecurityEvent{
		Time:      time.Now(),
		Scope:     scope,
		EventType: EventToolEvaluated,
		Actor:     actor,
		Target:    target,
		Summary:   fmt.Sprintf("tool %s evaluated as %s", toolName, eval.Decision),
		Details: map[string]any{
			"tool_name":    toolName,
			"payload":      payload,
			"reasons":      eval.Reasons,
			"matched_rule": eval.MatchedRule,
		},
	})

	if eval.MatchedRule == "cross_scope_attempt" {
		g.appendEvent(SecurityEvent{
			Time:      time.Now(),
			Scope:     scope,
			EventType: EventCrossScopeAttempt,
			Actor:     actor,
			Target:    toolName,
			Summary:   "cross-scope tool attempt denied",
			Details:   map[string]any{"payload": payload, "reasons": eval.Reasons},
		})
	}

	switch eval.Decision {
	case ToolDeny:
		g.appendEvent(SecurityEvent{
			Time:      time.Now(),
			Scope:     scope,
			EventType: EventToolDenied,
			Actor:     actor,
			Target:    toolName,
			Summary:   "tool call denied by policy",
			Details:   map[string]any{"reasons": eval.Reasons, "payload": payload},
		})
		return eval, nil, nil
	case ToolRequireApproval:
		if !g.cfg.ApprovalMode {
			eval.Decision = ToolDeny
			eval.Reasons = append(eval.Reasons, "approval mode disabled")
			g.appendEvent(SecurityEvent{
				Time:      time.Now(),
				Scope:     scope,
				EventType: EventToolDenied,
				Actor:     actor,
				Target:    toolName,
				Summary:   "tool call denied because approval mode is disabled",
				Details:   map[string]any{"payload": payload, "reasons": eval.Reasons},
			})
			return eval, nil, nil
		}
		req, err := g.approvals.Create(ApprovalRequest{
			Scope:      scope,
			Source:     actor,
			Kind:       "tool",
			TargetID:   target,
			TargetType: "run",
			Payload:    fmt.Sprintf("%s %s", toolName, payload),
			Reason:     eval.Reasons,
			Status:     ApprovalPending,
		})
		if err != nil {
			return eval, nil, err
		}
		g.appendEvent(SecurityEvent{
			Time:      time.Now(),
			Scope:     scope,
			EventType: EventApprovalRequired,
			Actor:     actor,
			Target:    toolName,
			Summary:   "tool call requires approval",
			Details: map[string]any{
				"approval_id": req.ID,
				"payload":     payload,
				"reasons":     eval.Reasons,
			},
		})
		return eval, &req, nil
	default:
		return eval, nil, nil
	}
}

func (g *Guard) ValidateMemory(scope, actor string, item types.MemoryItem) (MemoryValidationResult, *QuarantineRecord, error) {
	result := ValidateMemoryItem(item)
	if !g.Enabled() {
		return MemoryValidationResult{Allowed: true, RiskLevel: RiskLow}, nil, nil
	}
	if result.Allowed || !result.Quarantined {
		return result, nil, nil
	}
	record, err := g.quarantine.Add(QuarantineRecord{
		Time:       time.Now(),
		Scope:      scope,
		Source:     actor,
		Reasons:    result.Reasons,
		RiskLevel:  result.RiskLevel,
		Provenance: item.Provenance,
		Item:       item,
	})
	if err != nil {
		return result, nil, err
	}
	g.appendEvent(SecurityEvent{
		Time:      time.Now(),
		Scope:     scope,
		EventType: EventMemoryQuarantined,
		RiskLevel: string(result.RiskLevel),
		Actor:     actor,
		Target:    item.ID,
		Summary:   "memory item quarantined",
		Details:   map[string]any{"reasons": result.Reasons, "quarantine_id": record.ID},
	})
	return result, &record, nil
}

func (g *Guard) ListApprovals(scope string) ([]ApprovalRequest, error) {
	if g == nil || g.approvals == nil {
		return nil, nil
	}
	return g.approvals.List(scope)
}

func (g *Guard) GetApproval(id string) (ApprovalRequest, error) {
	return g.approvals.Get(id)
}

func (g *Guard) UpdateApprovalStatus(id string, status ApprovalStatus) (ApprovalRequest, error) {
	return g.approvals.UpdateStatus(id, status)
}

func (g *Guard) ListQuarantine(scope string) ([]QuarantineRecord, error) {
	if g == nil || g.quarantine == nil {
		return nil, nil
	}
	return g.quarantine.List(scope)
}

func (g *Guard) GetQuarantine(id string) (QuarantineRecord, error) {
	return g.quarantine.Get(id)
}

func (g *Guard) ListEvents(scope string, limit int) ([]SecurityEvent, error) {
	if g == nil || g.audit == nil {
		return nil, nil
	}
	return g.audit.List(scope, limit)
}

func (g *Guard) appendEvent(event SecurityEvent) {
	if g == nil || g.audit == nil {
		return
	}
	g.audit.Append(event)
}
