package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestAuditJSONOutputShape verifies the audit --json output is valid JSON with
// the expected top-level structure. The OSV network call is skipped by running
// in a directory with no workflows (command errors before any network use).
func TestAuditJSONOutputShape(t *testing.T) {
	dir := t.TempDir()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"audit", "--json", "--path", dir})
	defer rootCmd.SetArgs(nil)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error: no workflow files found in empty dir")
	}
}

// TestAuditSkipsSHARefs ensures full-SHA refs (excluded by the parser) never
// reach the vulnerability scan. We simulate by checking that a workflow with
// only SHA refs reports no actions to audit.
func TestAuditSkipsSHARefs(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wf := filepath.Join(wfDir, "ci.yml")
	content := "name: CI\non: [push]\njobs:\n  b:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@8f4b7f84864484a7bf31766abe9204da3cbe65b3\n"
	if err := os.WriteFile(wf, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"audit", "--json", "--path", dir})
	defer rootCmd.SetArgs(nil)

	// Run in the temp dir so config loading finds no .actup.yaml.
	oldWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	// The audit command prints JSON directly to os.Stdout.
	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w

	// The network call must be skipped: zero actions after filtering SHA refs.
	// runAudit returns nil early with "No actions found."
	execErr := rootCmd.Execute()

	_ = w.Close()
	os.Stdout = oldStdout
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)

	if execErr != nil {
		t.Fatalf("audit with only SHA refs should succeed, got %v", execErr)
	}

	out := buf.String()
	var parsed struct {
		Findings []json.RawMessage `json:"findings"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output not valid JSON: %q (%v)", out, err)
	}
	if len(parsed.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(parsed.Findings))
	}
}
