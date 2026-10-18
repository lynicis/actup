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

	upgrades := []Upgrade{
		{
			Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Current: "v3", Line: 7, File: workflowPath},
			NewTag: "v4",
		},
	}

	if err := ApplyAllUpgrades(upgrades, false); err != nil {
		t.Fatalf("ApplyAllUpgrades failed: %v", err)
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

	upgrades := []Upgrade{
		{
			Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Current: "v3", Line: 8, File: workflowPath},
			NewTag: "v4",
		},
	}

	if err := ApplyAllUpgrades(upgrades, false); err != nil {
		t.Fatalf("ApplyAllUpgrades failed: %v", err)
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

	upgrades := []Upgrade{
		{
			Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Current: "v3", Line: 7, File: workflowPath},
			NewTag: "v4",
		},
	}

	err := ApplyAllUpgrades(upgrades, true)
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

	upgrades := []Upgrade{
		{
			Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Current: "v3", Line: 7, File: workflowPath},
			NewTag: "v4",
		},
		{
			Action: parser.ActionRef{Owner: "actions", Repo: "setup-go", Current: "v4", Line: 8, File: workflowPath},
			NewTag: "v5",
		},
	}

	if err := ApplyAllUpgrades(upgrades, false); err != nil {
		t.Fatalf("ApplyAllUpgrades failed: %v", err)
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

func TestReplaceInFileSubpathAction(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.yml")
	content := `name: Test
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - name: Run OSV-Scanner Action
        uses: google/osv-scanner-action/osv-scanner-action@a345acffa64b0eaede81a3d9aae6141214d9c8fc # v2.6.0
`

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	action := parser.ActionRef{
		Owner:   "google",
		Repo:    "osv-scanner-action",
		Path:    "google/osv-scanner-action/osv-scanner-action",
		Current: "a345acffa64b0eaede81a3d9aae6141214d9c8fc # v2.6.0",
		Line:    8,
		File:    workflowPath,
	}

	if err := replaceInFile(workflowPath, action, "v2.7.0"); err != nil {
		t.Fatalf("replaceInFile failed: %v", err)
	}

	updated, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("failed to read updated file: %v", err)
	}

	expectedLine := "uses: google/osv-scanner-action/osv-scanner-action@v2.7.0"
	if !strings.Contains(string(updated), expectedLine) {
		t.Errorf("expected line %q in updated content, got:\n%s", expectedLine, string(updated))
	}
}

func TestReplaceInFileTabsAndComment(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.yml")
	content := "name: Test\non: push\njobs:\n  test:\n    runs-on: ubuntu-latest\n    steps:\n      - name: Scan\n        uses:\t\tgoogle/osv-scanner-action/osv-scanner-action@v2.6.0    # v2.6.0 pinned\n"

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	action := parser.ActionRef{
		Owner:   "google",
		Repo:    "osv-scanner-action",
		Path:    "google/osv-scanner-action/osv-scanner-action",
		Current: "v2.6.0    # v2.6.0 pinned",
		Line:    8,
		File:    workflowPath,
	}

	if err := replaceInFile(workflowPath, action, "v2.7.0"); err != nil {
		t.Fatalf("replaceInFile failed: %v", err)
	}

	updated, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("failed to read updated file: %v", err)
	}

	if !strings.Contains(string(updated), "uses:\t\tgoogle/osv-scanner-action/osv-scanner-action@v2.7.0") {
		t.Errorf("expected tab-indented uses line upgraded in place, got:\n%s", string(updated))
	}
}

func TestApplyAllUpgradesContinuesPastErrors(t *testing.T) {
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

	upgrades := []Upgrade{
		{
			// Bad line number: replaceInFile will fail to find this uses line.
			Action: parser.ActionRef{Owner: "actions", Repo: "cache", Current: "v3", Line: 99, File: workflowPath},
			NewTag: "v4",
		},
		{
			Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Current: "v3", Line: 7, File: workflowPath},
			NewTag: "v4",
		},
	}

	err := ApplyAllUpgrades(upgrades, false)
	if err == nil {
		t.Fatal("expected an error for the unfindable uses line")
	}

	updatedContent, err2 := os.ReadFile(workflowPath)
	if err2 != nil {
		t.Fatalf("failed to read updated workflow: %v", err2)
	}

	if !strings.Contains(string(updatedContent), "uses: actions/checkout@v4") {
		t.Error("checkout should still be upgraded despite the failed upgrade")
	}
	if !strings.Contains(string(updatedContent), "uses: actions/setup-go@v4") {
		t.Error("setup-go should be untouched")
	}
}

func TestApplyUpgradesConcurrent(t *testing.T) {
	tmpDir := t.TempDir()

	numFiles := 10
	var upgrades []Upgrade
	var paths []string

	for i := 0; i < numFiles; i++ {
		path := filepath.Join(tmpDir, "test"+string(rune(i+'a'))+".yml")
		content := `name: CI
on: push
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write workflow: %v", err)
		}
		paths = append(paths, path)

		upgrades = append(upgrades, Upgrade{
			Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Current: "v3", Line: 7, File: path},
			NewTag: "v4",
		})
	}

	if err := ApplyAllUpgrades(upgrades, false); err != nil {
		t.Fatalf("ApplyAllUpgrades failed: %v", err)
	}

	for _, path := range paths {
		updatedContent, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("failed to read updated workflow: %v", err)
		}

		expected := "uses: actions/checkout@v4"
		if !strings.Contains(string(updatedContent), expected) {
			t.Errorf("expected workflow to contain %q, got:\n%s", expected, string(updatedContent))
		}
	}
}
