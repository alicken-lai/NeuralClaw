package security

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"neuralclaw/internal/config"
)

func ToolPolicyFromConfig(cfg config.ToolPolicyConfig) ToolPolicy {
	policy := ToolPolicy{
		Default: ToolDecision(strings.ToLower(strings.TrimSpace(cfg.Default))),
	}
	if policy.Default == "" {
		policy.Default = ToolAllow
	}
	for _, rule := range cfg.Rules {
		policy.Rules = append(policy.Rules, ToolPolicyRule{
			ToolName:                rule.ToolName,
			AllowPatterns:           rule.AllowPatterns,
			DenyPatterns:            rule.DenyPatterns,
			RequireApprovalPatterns: rule.RequireApprovalPatterns,
		})
	}
	return policy
}

func EvaluateToolCall(scope string, toolName string, payload string, policy ToolPolicy) ToolEvaluation {
	eval := ToolEvaluation{Decision: policy.Default}
	rule, ok := findToolRule(toolName, policy)
	if !ok {
		return eval
	}

	if crossScope, summary := detectCrossScopeAttempt(scope, payload); crossScope {
		return ToolEvaluation{
			Decision:    ToolDeny,
			Reasons:     []string{summary},
			MatchedRule: "cross_scope_attempt",
		}
	}

	if match := firstMatch(payload, rule.DenyPatterns); match != "" {
		return ToolEvaluation{
			Decision:    ToolDeny,
			Reasons:     []string{fmt.Sprintf("matches deny pattern %q", match)},
			MatchedRule: match,
		}
	}

	if match := firstMatch(payload, rule.RequireApprovalPatterns); match != "" {
		return ToolEvaluation{
			Decision:    ToolRequireApproval,
			Reasons:     []string{fmt.Sprintf("matches approval pattern %q", match)},
			MatchedRule: match,
		}
	}

	if len(rule.AllowPatterns) > 0 {
		if match := firstMatch(payload, rule.AllowPatterns); match != "" {
			return ToolEvaluation{
				Decision:    ToolAllow,
				Reasons:     []string{fmt.Sprintf("matches allow pattern %q", match)},
				MatchedRule: match,
			}
		}
		return ToolEvaluation{
			Decision:    policy.Default,
			Reasons:     []string{"did not match explicit allow patterns"},
			MatchedRule: rule.ToolName,
		}
	}

	return eval
}

func findToolRule(toolName string, policy ToolPolicy) (ToolPolicyRule, bool) {
	for _, rule := range policy.Rules {
		if strings.EqualFold(rule.ToolName, toolName) {
			return rule, true
		}
	}
	return ToolPolicyRule{}, false
}

func firstMatch(payload string, patterns []string) string {
	for _, pattern := range patterns {
		re, err := regexp.Compile("(?i)" + pattern)
		if err != nil {
			continue
		}
		if re.MatchString(payload) {
			return pattern
		}
	}
	return ""
}

func detectCrossScopeAttempt(scope, payload string) (bool, string) {
	if scope == "" || payload == "" {
		return false, ""
	}
	var decoded map[string]interface{}
	if err := json.Unmarshal([]byte(payload), &decoded); err == nil {
		if rawScope, ok := decoded["scope"].(string); ok && rawScope != "" && rawScope != scope {
			return true, fmt.Sprintf("tool payload targets scope %q while active scope is %q", rawScope, scope)
		}
	}

	scopePattern := regexp.MustCompile(`(?i)(global|project:[a-z0-9._\-]+|user:[a-z0-9._\-]+|session:[a-z0-9._\-]+)`)
	matches := scopePattern.FindAllString(payload, -1)
	for _, match := range matches {
		if !strings.EqualFold(match, scope) {
			return true, fmt.Sprintf("tool payload references scope %q outside active scope %q", match, scope)
		}
	}
	return false, ""
}
