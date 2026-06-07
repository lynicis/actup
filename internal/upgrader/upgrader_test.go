package upgrader

import (
	"os"
	"path/filepath"
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

	actions := []parser.ActionRef{
		{Owner: "actions", Repo: "checkout", Current: "v3", Line: 7, File: workflowPath},
	}

	results, err := ApplyUpgrades(actions, "v4", false)
	if err != nil {
		t.Fatalf("ApplyUpgrades failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if !results[0].Updated {
		t.Error("expected action to be updated")
	}

	updatedContent, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("failed to read updated workflow: %v", err)
	}

	expected := "uses: actions/checkout@v4"
	if !contains(string(updatedContent), expected) {
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

	actions := []parser.ActionRef{
		{Owner: "actions", Repo: "checkout", Current: "v3", Line: 8, File: workflowPath},
	}

	results, err := ApplyUpgrades(actions, "v4", false)
	if err != nil {
		t.Fatalf("ApplyUpgrades failed: %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	if !results[0].Updated {
		t.Error("expected action to be updated")
	}

	updatedContent, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("failed to read updated workflow: %v", err)
	}

	expected := "uses: actions/checkout@v4"
	if !contains(string(updatedContent), expected) {
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

	actions := []parser.ActionRef{
		{Owner: "actions", Repo: "checkout", Current: "v3", Line: 7, File: workflowPath},
	}

	_, err := ApplyUpgrades(actions, "v4", true)
	if err != nil {
		t.Fatalf("ApplyUpgrades dry-run failed: %v", err)
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
	if !contains(string(updated), expectedLine) {
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
	if !contains(string(updated), expectedLine) {
		t.Errorf("expected line %q in updated content, got:\n%s", expectedLine, string(updated))
	}
}

func TestGroupByFile(t *testing.T) {
	actions := []parser.ActionRef{
		{Owner: "actions", Repo: "checkout", File: "/a.yml"},
		{Owner: "actions", Repo: "setup-go", File: "/a.yml"},
		{Owner: "actions", Repo: "checkout", File: "/b.yml"},
	}

	grouped := groupByFile(actions)

	if len(grouped) != 2 {
		t.Errorf("expected 2 files, got %d", len(grouped))
	}

	if len(grouped["/a.yml"]) != 2 {
		t.Errorf("expected 2 actions in /a.yml, got %d", len(grouped["/a.yml"]))
	}

	if len(grouped["/b.yml"]) != 1 {
		t.Errorf("expected 1 action in /b.yml, got %d", len(grouped["/b.yml"]))
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

	if !contains(string(updatedContent), "actions/checkout@v4") {
		t.Error("checkout should be upgraded to v4")
	}

	if !contains(string(updatedContent), "actions/setup-go@v5") {
		t.Error("setup-go should be upgraded to v5")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
