package security

import (
	"regexp"
	"sort"
	"strings"
)

type heuristicRule struct {
	name    string
	pattern *regexp.Regexp
	score   int
	reason  string
}

var promptRules = []heuristicRule{
	newRule("ignore_previous_instructions", `ignore\s+previous\s+instructions`, 35, "attempts to override previous instructions"),
	newRule("reveal_system_prompt", `reveal\s+(the\s+)?system\s+prompt`, 40, "attempts to reveal system prompt"),
	newRule("show_secrets", `show\s+secrets?`, 30, "requests sensitive secrets"),
	newRule("print_api_key", `print\s+(the\s+)?api\s+key`, 40, "requests API keys"),
	newRule("disable_safety", `disable\s+safety|disable\s+guard|disable\s+policy`, 35, "attempts to disable safety controls"),
	newRule("delete_all_files", `delete\s+all\s+files`, 45, "requests destructive file deletion"),
	newRule("rm_rf", `rm\s+-rf`, 50, "contains destructive shell pattern"),
	newRule("sudo_usage", `\bsudo\s+`, 20, "requests privileged command execution"),
	newRule("self_modify", `self-modify|rewrite\s+your\s+rules`, 35, "attempts self-modification or policy rewrite"),
	newRule("exfiltrate", `exfiltrate|send\s+all\s+logs|upload\s+memory|dump\s+config`, 40, "attempts data exfiltration"),
	newRule("bypass_policy", `bypass|override\s+policy`, 30, "attempts to bypass policy"),
	newRule("download_execute", `download\s+and\s+execute|curl.+\|\s*sh|powershell.+iex`, 50, "contains download-and-execute pattern"),
	newRule("ssh_scp", `\bssh\s+|\bscp\s+`, 25, "requests remote shell or copy"),
}

func newRule(name, pattern string, score int, reason string) heuristicRule {
	return heuristicRule{
		name:    name,
		pattern: regexp.MustCompile(`(?i)` + pattern),
		score:   score,
		reason:  reason,
	}
}

func InspectPrompt(scope, input string) PromptInspection {
	normalized := strings.ToLower(strings.TrimSpace(input))
	if normalized == "" {
		return PromptInspection{
			RiskLevel: RiskLow,
			Action:    "allow",
		}
	}

	score := 0
	reasons := make([]string, 0)
	matched := make([]string, 0)
	for _, rule := range promptRules {
		if rule.pattern.MatchString(normalized) {
			score += rule.score
			reasons = append(reasons, rule.reason)
			matched = append(matched, rule.name)
		}
	}

	if looksCredentialLike(normalized) {
		score += 30
		reasons = append(reasons, "contains credential-like material")
		matched = append(matched, "credential_like_text")
	}

	if scope != "" && strings.Contains(normalized, "scope:") && !strings.Contains(normalized, strings.ToLower(scope)) {
		score += 20
		reasons = append(reasons, "references a different scope than the active request")
		matched = append(matched, "cross_scope_reference")
	}

	sort.Strings(matched)
	risk, action := classifyPromptRisk(score)
	return PromptInspection{
		RiskLevel:    risk,
		Score:        score,
		Reasons:      dedupeStrings(reasons),
		MatchedRules: dedupeStrings(matched),
		Action:       action,
	}
}

func classifyPromptRisk(score int) (PromptRiskLevel, string) {
	switch {
	case score >= 75:
		return RiskCritical, "block"
	case score >= 45:
		return RiskHigh, "require_approval"
	case score >= 20:
		return RiskMedium, "warn"
	default:
		return RiskLow, "allow"
	}
}

func looksCredentialLike(text string) bool {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(api[_ -]?key|secret|token|password)\s*[:=]\s*["']?[a-z0-9_\-]{8,}`),
		regexp.MustCompile(`(?i)sk-[a-z0-9]{10,}`),
		regexp.MustCompile(`(?i)jina_[a-z0-9]{10,}`),
	}
	for _, pattern := range patterns {
		if pattern.MatchString(text) {
			return true
		}
	}
	return false
}

func dedupeStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}
