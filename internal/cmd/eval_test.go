package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"neuralclaw/pkg/types"
)

type fakeSearcher struct{}

func (f *fakeSearcher) SearchExplain(ctx context.Context, text string, scope string, topK int, explain bool) (types.QueryResult, error) {
	return types.QueryResult{
		Items: []types.MemoryItem{
			{ID: "a"},
			{ID: "x"},
		},
		TotalFound: 2,
	}, nil
}

func TestEvalRetrievalCommand(t *testing.T) {
	origFactory := newRetrievalSearcher
	newRetrievalSearcher = func() (retrievalSearcher, error) {
		return &fakeSearcher{}, nil
	}
	defer func() {
		newRetrievalSearcher = origFactory
	}()

	dir := t.TempDir()
	goldenPath := filepath.Join(dir, "golden.yaml")
	yamlBody := `
- id: q1
  text: "hello"
  expected_ids: ["a"]
  scope: "global"
- id: q2
  text: "bye"
  expected_ids: ["b"]
  scope: "global"
`
	if err := os.WriteFile(goldenPath, []byte(yamlBody), 0644); err != nil {
		t.Fatalf("failed to write golden yaml: %v", err)
	}

	cmd := newEvalRetrievalCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"--golden", goldenPath, "--k", "2"})

	if err := cmd.ExecuteContext(context.Background()); err != nil {
		t.Fatalf("command failed: %v", err)
	}

	output := out.String()
	if !strings.Contains(output, "QueryID") {
		t.Fatalf("expected table header in output, got:\n%s", output)
	}
	if !strings.Contains(output, "AVG") {
		t.Fatalf("expected AVG row in output, got:\n%s", output)
	}
}

func TestRunRetrievalEvalFailsOnEmptyGolden(t *testing.T) {
	dir := t.TempDir()
	goldenPath := filepath.Join(dir, "empty.yaml")
	if err := os.WriteFile(goldenPath, []byte("[]"), 0644); err != nil {
		t.Fatalf("failed to write golden yaml: %v", err)
	}

	err := runRetrievalEval(newEvalRetrievalCmd(), retrievalEvalOptions{
		GoldenPath: goldenPath,
		K:          10,
	})
	if err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("expected empty golden error, got: %v", err)
	}
}
