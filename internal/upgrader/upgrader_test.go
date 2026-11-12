package upgrader

import (
	"io"
	"os"
	"path/filepath"
	"runtime"
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
		Current: "a345acffa64b0eaede81a3d9aae6141214d9c8fc",
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
		Current: "v2.6.0",
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

	expectedLine := "uses:\t\tgoogle/osv-scanner-action/osv-scanner-action@v2.7.0    # v2.6.0 pinned"
	if !strings.Contains(string(updated), expectedLine) {
		t.Errorf("expected %q upgraded in place with comment preserved, got:\n%s", expectedLine, string(updated))
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

func TestApplyAllUpgradesPreservesMode(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.yml")
	content := "name: CI\non: push\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v3\n"
	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}
	if err := os.Chmod(workflowPath, 0644); err != nil {
		t.Fatalf("failed to chmod workflow: %v", err)
	}

	upgrades := []Upgrade{{
		Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Current: "v3", Line: 7, File: workflowPath},
		NewTag: "v4",
	}}
	if err := ApplyAllUpgrades(upgrades, false); err != nil {
		t.Fatalf("ApplyAllUpgrades failed: %v", err)
	}

	info, err := os.Stat(workflowPath)
	if err != nil {
		t.Fatalf("failed to stat workflow: %v", err)
	}

	if runtime.GOOS == "windows" {
		if info.Mode().Perm()&0200 == 0 {
			t.Errorf("expected writable file on Windows, got mode %v", info.Mode().Perm())
		}
		return
	}

	if got := info.Mode().Perm(); got != 0644 {
		t.Errorf("expected mode 0644 to be preserved, got %v", got)
	}
}

func TestReplaceAllInFileSingleWrite(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.yml")
	content := "name: CI\non: push\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v3\n      - uses: actions/setup-go@v4\n"
	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	upgrades := []Upgrade{
		{Action: parser.ActionRef{Owner: "actions", Repo: "setup-go", Current: "v4", Line: 8, File: workflowPath}, NewTag: "v5"},
		{Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Current: "v3", Line: 7, File: workflowPath}, NewTag: "v4"},
	}
	if err := replaceAllInFile(workflowPath, upgrades); err != nil {
		t.Fatalf("replaceAllInFile failed: %v", err)
	}

	updated, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("failed to read updated workflow: %v", err)
	}
	for _, expected := range []string{"uses: actions/checkout@v4", "uses: actions/setup-go@v5"} {
		if !strings.Contains(string(updated), expected) {
			t.Errorf("expected workflow to contain %q, got:\n%s", expected, string(updated))
		}
	}

	leftovers, err := filepath.Glob(filepath.Join(tmpDir, "actup-*.tmp"))
	if err != nil {
		t.Fatalf("failed to glob temp files: %v", err)
	}
	if len(leftovers) != 0 {
		t.Errorf("expected no temp files after atomic write, found %v", leftovers)
	}
}

func TestReplaceAllInFilePartialMissStillAppliesOthers(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.yml")
	content := "name: CI\non: push\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v3\n"
	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	upgrades := []Upgrade{
		{Action: parser.ActionRef{Owner: "actions", Repo: "cache", Current: "v3", Line: 99, File: workflowPath}, NewTag: "v4"},
		{Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Current: "v3", Line: 7, File: workflowPath}, NewTag: "v4"},
	}
	err := replaceAllInFile(workflowPath, upgrades)
	if err == nil {
		t.Fatal("expected an error for the missing uses line")
	}
	if !strings.Contains(err.Error(), "could not find expected uses line at line 99") {
		t.Errorf("expected missing-line error, got %v", err)
	}

	updated, err2 := os.ReadFile(workflowPath)
	if err2 != nil {
		t.Fatalf("failed to read updated workflow: %v", err2)
	}
	if !strings.Contains(string(updated), "uses: actions/checkout@v4") {
		t.Errorf("matched upgrade should still be applied, got:\n%s", string(updated))
	}
}

func TestReplaceAllInFileDuplicateLineConflict(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.yml")
	content := "name: CI\non: push\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v3\n"
	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	action := parser.ActionRef{Owner: "actions", Repo: "checkout", Current: "v3", Line: 7, File: workflowPath}
	upgrades := []Upgrade{
		{Action: action, NewTag: "v4"},
		{Action: action, NewTag: "v5"},
	}
	err := replaceAllInFile(workflowPath, upgrades)
	if err == nil {
		t.Fatal("expected a conflict error for duplicate line entries")
	}
	if !strings.Contains(err.Error(), "conflicting tags for line 7") {
		t.Errorf("expected conflict error, got %v", err)
	}

	updated, err2 := os.ReadFile(workflowPath)
	if err2 != nil {
		t.Fatalf("failed to read updated workflow: %v", err2)
	}
	if !strings.Contains(string(updated), "uses: actions/checkout@v4") {
		t.Errorf("first upgrade should win, got:\n%s", string(updated))
	}
	if strings.Contains(string(updated), "uses: actions/checkout@v5") {
		t.Error("conflicting second upgrade should not be applied")
	}
}

func TestApplyAllUpgradesDryRunDeterministic(t *testing.T) {
	tmpDir := t.TempDir()

	fileA := filepath.Join(tmpDir, "a.yml")
	fileB := filepath.Join(tmpDir, "b.yml")
	content := "name: CI\non: push\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v3\n      - uses: actions/setup-go@v4\n"
	for _, path := range []string{fileA, fileB} {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write workflow: %v", err)
		}
	}

	// Added out of order on purpose: b before a, line 8 before line 7.
	upgrades := []Upgrade{
		{Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Current: "v3", Line: 7, File: fileB}, NewTag: "v4"},
		{Action: parser.ActionRef{Owner: "actions", Repo: "setup-go", Current: "v4", Line: 8, File: fileA}, NewTag: "v5"},
		{Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Current: "v3", Line: 7, File: fileA}, NewTag: "v4"},
	}

	first := captureStdout(t, func() {
		if err := ApplyAllUpgrades(upgrades, true); err != nil {
			t.Fatalf("dry-run failed: %v", err)
		}
	})
	second := captureStdout(t, func() {
		if err := ApplyAllUpgrades(upgrades, true); err != nil {
			t.Fatalf("dry-run failed: %v", err)
		}
	})

	if first != second {
		t.Errorf("dry-run output is not deterministic:\nfirst:\n%s\nsecond:\n%s", first, second)
	}

	aHeader := "--- a/" + fileA
	bHeader := "--- a/" + fileB
	aLine7 := "@@ -7,1 +7,1 @@"
	aLine8 := "@@ -8,1 +8,1 @@"
	aPos, bPos := strings.Index(first, aHeader), strings.Index(first, bHeader)
	if aPos < 0 || bPos < 0 || aPos >= bPos {
		t.Errorf("expected files in sorted order, got:\n%s", first)
	}
	if aLine := strings.Index(first, aLine7); aLine < 0 {
		t.Errorf("expected a/ line 7 diff, got:\n%s", first)
	} else if aLine > strings.Index(first, aLine8) {
		t.Errorf("expected lines in sorted order within a file, got:\n%s", first)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}

	old := os.Stdout
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close pipe writer: %v", err)
	}

	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("failed to read captured output: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("failed to close pipe reader: %v", err)
	}

	return string(out)
}

func TestReplaceAllInFileSHAModeStripsOldComment(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.yml")
	content := "name: CI\non: push\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v3 # v3\n"
	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	const sha = "dae09b1f80164e2d312981ce81045e7e1f4640d2"
	upgrades := []Upgrade{{
		Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Path: "actions/checkout", Current: "v3", Line: 7, File: workflowPath},
		NewTag: "v4.2.2",
		NewSHA: sha,
	}}
	if err := replaceAllInFile(workflowPath, upgrades); err != nil {
		t.Fatalf("replaceAllInFile failed: %v", err)
	}

	updated, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("failed to read updated workflow: %v", err)
	}
	want := "- uses: actions/checkout@" + sha + " # v4.2.2"
	if !strings.Contains(string(updated), want) {
		t.Errorf("expected %q, got:\n%s", want, string(updated))
	}
	if strings.Contains(string(updated), "# v3") {
		t.Errorf("old comment should be stripped, got:\n%s", string(updated))
	}
}

func TestReplaceAllInFileMixedSHAAndTagSites(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.yml")
	const oldSHA = "8a6d4d4b0c5f7e9a1b2c3d4e5f60718293a4b5c6"
	content := "name: CI\non: push\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@" + oldSHA + " # v4.2.1\n      - uses: actions/setup-go@v4 # pinned\n"
	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	const newSHA = "dae09b1f80164e2d312981ce81045e7e1f4640d2"
	upgrades := []Upgrade{
		{
			Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Path: "actions/checkout", Current: oldSHA, Line: 7, File: workflowPath},
			NewTag: "v4.2.2",
			NewSHA: newSHA,
		},
		{
			Action: parser.ActionRef{Owner: "actions", Repo: "setup-go", Path: "actions/setup-go", Current: "v4", Line: 8, File: workflowPath},
			NewTag: "v5",
		},
	}
	if err := replaceAllInFile(workflowPath, upgrades); err != nil {
		t.Fatalf("replaceAllInFile failed: %v", err)
	}

	updated, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("failed to read updated workflow: %v", err)
	}
	for _, want := range []string{
		"- uses: actions/checkout@" + newSHA + " # v4.2.2",
		"- uses: actions/setup-go@v5 # pinned",
	} {
		if !strings.Contains(string(updated), want) {
			t.Errorf("expected %q, got:\n%s", want, string(updated))
		}
	}
}

func TestReplaceAllInFileDuplicateLineIncludesSHA(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.yml")
	content := "name: CI\non: push\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v3\n"
	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	action := parser.ActionRef{Owner: "actions", Repo: "checkout", Path: "actions/checkout", Current: "v3", Line: 7, File: workflowPath}
	upgrades := []Upgrade{
		{Action: action, NewTag: "v4", NewSHA: "dae09b1f80164e2d312981ce81045e7e1f4640d2"},
		{Action: action, NewTag: "v4", NewSHA: "8a6d4d4b0c5f7e9a1b2c3d4e5f60718293a4b5c6"},
	}

	err := replaceAllInFile(workflowPath, upgrades)
	if err == nil {
		t.Fatal("expected a conflict error for differing SHAs on the same line")
	}
	if !strings.Contains(err.Error(), "conflicting") {
		t.Errorf("error = %q, want a conflict", err)
	}
}

func TestApplyAllUpgradesDryRunSHAMode(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.yml")
	content := "name: CI\non: push\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v3\n"
	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	const sha = "dae09b1f80164e2d312981ce81045e7e1f4640d2"
	upgrades := []Upgrade{{
		Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Path: "actions/checkout", Current: "v3", Line: 7, File: workflowPath},
		NewTag: "v4.2.2",
		NewSHA: sha,
	}}

	out := captureStdout(t, func() {
		if err := ApplyAllUpgrades(upgrades, true); err != nil {
			t.Fatalf("dry-run failed: %v", err)
		}
	})
	want := "+uses: actions/checkout@" + sha + " # v4.2.2"
	if !strings.Contains(out, want) {
		t.Errorf("expected dry-run output to contain %q, got:\n%s", want, out)
	}
}

func TestReplaceAllInFileSHAModeCRLF(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.yml")
	content := "name: CI\r\non: push\r\njobs:\r\n  build:\r\n    runs-on: ubuntu-latest\r\n    steps:\r\n      - uses: actions/checkout@v3\r\n"
	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	const sha = "dae09b1f80164e2d312981ce81045e7e1f4640d2"
	upgrades := []Upgrade{{
		Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Path: "actions/checkout", Current: "v3", Line: 7, File: workflowPath},
		NewTag: "v4.2.2",
		NewSHA: sha,
	}}
	if err := replaceAllInFile(workflowPath, upgrades); err != nil {
		t.Fatalf("replaceAllInFile failed: %v", err)
	}

	updated, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("failed to read updated workflow: %v", err)
	}
	want := "- uses: actions/checkout@" + sha + " # v4.2.2\r\n"
	if !strings.Contains(string(updated), want) {
		t.Errorf("expected CRLF line %q, got %q", want, string(updated))
	}
	if strings.Count(string(updated), "\n") != strings.Count(string(updated), "\r\n") {
		t.Errorf("mixed line endings after rewrite: %q", string(updated))
	}
}

func TestReplaceAllInFileSHAModeQuotedRefs(t *testing.T) {
	const sha = "dae09b1f80164e2d312981ce81045e7e1f4640d2"

	tests := []struct {
		name string
		line string
		want string
	}{
		{
			name: "double quoted",
			line: "      - uses: \"actions/checkout@v3\"",
			want: "      - uses: \"actions/checkout@" + sha + "\" # v4.2.2",
		},
		{
			name: "single quoted",
			line: "      - uses: 'actions/checkout@v3'",
			want: "      - uses: 'actions/checkout@" + sha + "' # v4.2.2",
		},
		{
			name: "double quoted with old comment",
			line: "      - uses: \"actions/checkout@v3\" # v3",
			want: "      - uses: \"actions/checkout@" + sha + "\" # v4.2.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			workflowPath := filepath.Join(tmpDir, "test.yml")
			content := "name: CI\non: push\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n" + tt.line + "\n"
			if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
				t.Fatalf("failed to write workflow: %v", err)
			}

			upgrades := []Upgrade{{
				Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Path: "actions/checkout", Current: "v3", Line: 7, File: workflowPath},
				NewTag: "v4.2.2",
				NewSHA: sha,
			}}
			if err := replaceAllInFile(workflowPath, upgrades); err != nil {
				t.Fatalf("replaceAllInFile failed: %v", err)
			}

			updated, err := os.ReadFile(workflowPath)
			if err != nil {
				t.Fatalf("failed to read updated workflow: %v", err)
			}
			if !strings.Contains(string(updated), tt.want) {
				t.Errorf("expected %q, got %q", tt.want, string(updated))
			}
		})
	}
}
