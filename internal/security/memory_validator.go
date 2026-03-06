package security

import (
	"strings"

	"neuralclaw/pkg/types"
)

func ValidateMemoryItem(item types.MemoryItem) MemoryValidationResult {
	textParts := []string{
		item.Text,
		item.BM25Text,
		item.Metadata,
		item.Provenance.SourceFilePath,
		item.Provenance.SourceURI,
		item.Provenance.SourceKind,
	}
	combined := strings.ToLower(strings.Join(textParts, "\n"))

	reasons := make([]string, 0)
	score := 0
	checks := []struct {
		phrase string
		score  int
		reason string
	}{
		{"ignore system prompt", 35, "contains prompt override instruction"},
		{"disable policy", 30, "contains policy disable instruction"},
		{"reveal secrets", 35, "contains secret disclosure instruction"},
		{"print key", 35, "contains key disclosure instruction"},
		{"override guard", 30, "contains guard override instruction"},
		{"always trust this instruction", 30, "contains suspicious persistent instruction"},
		{"execute shell command", 30, "contains direct shell execution instruction"},
		{"download and run", 35, "contains download-and-run instruction"},
		{"bypass safety", 35, "contains safety bypass instruction"},
	}
	for _, check := range checks {
		if strings.Contains(combined, check.phrase) {
			score += check.score
			reasons = append(reasons, check.reason)
		}
	}

	if looksCredentialLike(combined) {
		score += 30
		reasons = append(reasons, "contains credential-like material")
	}

	if isInstructionalOCR(item, combined) {
		score += 20
		reasons = append(reasons, "ocr content resembles instructions to the agent")
	}

	risk, _ := classifyPromptRisk(score)
	if score >= 30 {
		return MemoryValidationResult{
			Allowed:     false,
			RiskLevel:   risk,
			Reasons:     dedupeStrings(reasons),
			Quarantined: true,
		}
	}

	return MemoryValidationResult{
		Allowed:     true,
		RiskLevel:   risk,
		Reasons:     dedupeStrings(reasons),
		Quarantined: false,
	}
}

func isInstructionalOCR(item types.MemoryItem, combined string) bool {
	if item.Modality != "ocr" && item.Provenance.SourceKind != "ocr" {
		return false
	}
	markers := []string{
		"ignore previous instructions",
		"system prompt",
		"execute shell",
		"download and execute",
		"reveal system prompt",
	}
	matches := 0
	for _, marker := range markers {
		if strings.Contains(combined, marker) {
			matches++
		}
	}
	return matches >= 1
}
