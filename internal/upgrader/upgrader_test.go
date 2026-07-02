package upgrader

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lynicis/actup/internal/parser"
)

func TestApplyUpgrades(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.yml")
	content := `name: CI
on: push
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
`

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	upgrades := map[string]Upgrade{
		"actions/checkout": {
			Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Current: "v3", Line: 7, File: workflowPath},
			NewTag: "v4",
		},
	}

	results, err := ApplyAllUpgrades(upgrades, false)
	if err != nil {
		t.Fatalf("ApplyAllUpgrades failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected results for 1 file, got %d", len(results))
	}

	fileResults := results[workflowPath]
	if len(fileResults) != 1 {
		t.Fatalf("expected 1 result, got %d", len(fileResults))
	}

	if !fileResults[0].Updated {
		t.Error("expected action to be updated")
	}

	updatedContent, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("failed to read updated workflow: %v", err)
	}

	expected := "uses: actions/checkout@v4"
	if !strings.Contains(string(updatedContent), expected) {
		t.Errorf("expected workflow to contain %q, got:\n%s", expected, string(updatedContent))
	}
}

func TestApplyUpgradesMultilineSteps(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.yml")
	content := `name: CI
on: push
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3
      - name: Set up Go
        uses: actions/setup-go@v4
`

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	upgrades := map[string]Upgrade{
		"actions/checkout": {
			Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Current: "v3", Line: 8, File: workflowPath},
			NewTag: "v4",
		},
	}

	results, err := ApplyAllUpgrades(upgrades, false)
	if err != nil {
		t.Fatalf("ApplyAllUpgrades failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected results for 1 file, got %d", len(results))
	}

	fileResults := results[workflowPath]
	if len(fileResults) != 1 {
		t.Fatalf("expected 1 result, got %d", len(fileResults))
	}

	if !fileResults[0].Updated {
		t.Error("expected action to be updated")
	}

	updatedContent, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("failed to read updated workflow: %v", err)
	}

	expected := "uses: actions/checkout@v4"
	if !strings.Contains(string(updatedContent), expected) {
		t.Errorf("expected workflow to contain %q, got:\n%s", expected, string(updatedContent))
	}
}

func TestApplyUpgradesDryRun(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.yml")
	originalContent := `name: CI
on: push
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
`

	if err := os.WriteFile(workflowPath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	upgrades := map[string]Upgrade{
		"actions/checkout": {
			Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Current: "v3", Line: 7, File: workflowPath},
			NewTag: "v4",
		},
	}

	_, err := ApplyAllUpgrades(upgrades, true)
	if err != nil {
		t.Fatalf("ApplyAllUpgrades dry-run failed: %v", err)
	}

	unchangedContent, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("failed to read workflow: %v", err)
	}

	if string(unchangedContent) != originalContent {
		t.Error("dry-run should not modify the file")
	}
}

func TestReplaceInFile(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.yml")
	content := `name: Test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
`

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	action := parser.ActionRef{
		Owner:   "actions",
		Repo:    "checkout",
		Current: "v3",
		Line:    7,
		File:    workflowPath,
	}

	err := replaceInFile(workflowPath, action, "v4.1.0")
	if err != nil {
		t.Fatalf("replaceInFile failed: %v", err)
	}

	updated, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("failed to read updated file: %v", err)
	}

	expectedLine := "      - uses: actions/checkout@v4.1.0"
	if !strings.Contains(string(updated), expectedLine) {
		t.Errorf("expected line %q in updated content, got:\n%s", expectedLine, string(updated))
	}
}

func TestReplaceInFileMultilineSteps(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.yml")
	content := `name: Test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout
        uses: actions/checkout@v3
`

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	action := parser.ActionRef{
		Owner:   "actions",
		Repo:    "checkout",
		Current: "v3",
		Line:    8,
		File:    workflowPath,
	}

	err := replaceInFile(workflowPath, action, "v4.1.0")
	if err != nil {
		t.Fatalf("replaceInFile failed: %v", err)
	}

	updated, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("failed to read updated file: %v", err)
	}

	expectedLine := "        uses: actions/checkout@v4.1.0"
	if !strings.Contains(string(updated), expectedLine) {
		t.Errorf("expected line %q in updated content, got:\n%s", expectedLine, string(updated))
	}
}

func TestApplyAllUpgrades(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.yml")
	content := `name: CI
on: push
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
`

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	upgrades := map[string]Upgrade{
		"actions/checkout": {
			Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Current: "v3", Line: 7, File: workflowPath},
			NewTag: "v4",
		},
		"actions/setup-go": {
			Action: parser.ActionRef{Owner: "actions", Repo: "setup-go", Current: "v4", Line: 8, File: workflowPath},
			NewTag: "v5",
		},
	}

	results, err := ApplyAllUpgrades(upgrades, false)
	if err != nil {
		t.Fatalf("ApplyAllUpgrades failed: %v", err)
	}

	if len(results) != 1 {
		t.Errorf("expected 1 file result, got %d", len(results))
	}

	updatedContent, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("failed to read updated workflow: %v", err)
	}

	if !strings.Contains(string(updatedContent), "actions/checkout@v4") {
		t.Error("checkout should be upgraded to v4")
	}

	if !strings.Contains(string(updatedContent), "actions/setup-go@v5") {
		t.Error("setup-go should be upgraded to v5")
	}
}
