package parser

import (
	"os"
	"path/filepath"
	"testing"
)

func TestExtractActions(t *testing.T) {
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
      - uses: docker/login-action@v2
      - uses: ./local-action
      - uses: docker://nginx:latest
      - uses: actions/setup-node@v3.2.0
`

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	actions, err := ExtractActions(t.Context(), []string{workflowPath})
	if err != nil {
		t.Fatalf("ExtractActions failed: %v", err)
	}

	if len(actions) != 4 {
		t.Errorf("expected 4 actions (excluding local and docker), got %d", len(actions))
	}

	expected := []struct {
		owner, repo, ref string
	}{
		{"actions", "checkout", "v3"},
		{"actions", "setup-go", "v4"},
		{"docker", "login-action", "v2"},
		{"actions", "setup-node", "v3.2.0"},
	}

	for i, exp := range expected {
		if actions[i].Owner != exp.owner {
			t.Errorf("action[%d].Owner = %s, want %s", i, actions[i].Owner, exp.owner)
		}
		if actions[i].Repo != exp.repo {
			t.Errorf("action[%d].Repo = %s, want %s", i, actions[i].Repo, exp.repo)
		}
		if actions[i].Current != exp.ref {
			t.Errorf("action[%d].Current = %s, want %s", i, actions[i].Current, exp.ref)
		}
	}
}

func TestExtractActionsMultilineSteps(t *testing.T) {
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
      - name: Docker Login
        uses: docker/login-action@v2
      - name: Local Action
        uses: ./local-action
      - name: Docker Action
        uses: docker://nginx:latest
      - name: Setup Node
        uses: actions/setup-node@v3.2.0
`

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	actions, err := ExtractActions(t.Context(), []string{workflowPath})
	if err != nil {
		t.Fatalf("ExtractActions failed: %v", err)
	}

	if len(actions) != 4 {
		t.Errorf("expected 4 actions (excluding local and docker), got %d", len(actions))
	}

	expected := []struct {
		owner, repo, ref string
		line             int
	}{
		{"actions", "checkout", "v3", 8},
		{"actions", "setup-go", "v4", 10},
		{"docker", "login-action", "v2", 12},
		{"actions", "setup-node", "v3.2.0", 18},
	}

	for i, exp := range expected {
		if actions[i].Owner != exp.owner {
			t.Errorf("action[%d].Owner = %s, want %s", i, actions[i].Owner, exp.owner)
		}
		if actions[i].Repo != exp.repo {
			t.Errorf("action[%d].Repo = %s, want %s", i, actions[i].Repo, exp.repo)
		}
		if actions[i].Current != exp.ref {
			t.Errorf("action[%d].Current = %s, want %s", i, actions[i].Current, exp.ref)
		}
		if actions[i].Line != exp.line {
			t.Errorf("action[%d].Line = %d, want %d", i, actions[i].Line, exp.line)
		}
	}
}

func TestExtractActionsStripsTrailingComment(t *testing.T) {
	tmpDir := t.TempDir()

	workflowPath := filepath.Join(tmpDir, "test.yml")
	content := "name: CI\non: push\njobs:\n  build:\n    runs-on: ubuntu-latest\n    steps:\n      - uses:\t\tactions/checkout@v3    # pinned\n"

	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	actions, err := ExtractActions(t.Context(), []string{workflowPath})
	if err != nil {
		t.Fatalf("ExtractActions failed: %v", err)
	}

	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	if actions[0].Current != "v3" {
		t.Errorf("Current = %q, want %q", actions[0].Current, "v3")
	}
	if actions[0].Line != 7 {
		t.Errorf("Line = %d, want 7", actions[0].Line)
	}
	if actions[0].Tag != "" {
		t.Errorf("Tag = %q, want empty for a tag ref", actions[0].Tag)
	}
}

func TestParseActionRef(t *testing.T) {
	tests := []struct {
		uses      string
		wantNil   bool
		wantOwner string
		wantRepo  string
		wantRef   string
	}{
		{"actions/checkout@v3", false, "actions", "checkout", "v3"},
		{"actions/setup-go@v4.0.1", false, "actions", "setup-go", "v4.0.1"},
		{"./local-action", true, "", "", ""},
		{"docker://nginx", true, "", "", ""},
		{"actions/checkout@e7d184456a78df3f4a5c6d7e8f9a0b1c2d3e4f60", false, "actions", "checkout", "e7d184456a78df3f4a5c6d7e8f9a0b1c2d3e4f60"},
		{"", true, "", "", ""},
		{"actions/checkout@${{ matrix.version }}", true, "", "", ""},
		{"docker/login-action@v2", false, "docker", "login-action", "v2"},
		{"github/super-linter@v6", false, "github", "super-linter", "v6"},
		{"google/osv-scanner-action/osv-scanner-action@v2.6.0", false, "google", "osv-scanner-action", "v2.6.0"},
	}

	for _, tt := range tests {
		t.Run(tt.uses, func(t *testing.T) {
			result, err := parseActionRef(tt.uses)
			if err != nil {
				t.Fatalf("parseActionRef(%q) returned error: %v", tt.uses, err)
			}

			if tt.wantNil {
				if result != nil {
					t.Errorf("parseActionRef(%q) = %+v, want nil", tt.uses, result)
				}
				return
			}

			if result == nil {
				t.Fatalf("parseActionRef(%q) = nil, want non-nil", tt.uses)
			}

			if result.Owner != tt.wantOwner {
				t.Errorf("Owner = %s, want %s", result.Owner, tt.wantOwner)
			}
			if result.Repo != tt.wantRepo {
				t.Errorf("Repo = %s, want %s", result.Repo, tt.wantRepo)
			}
			if result.Current != tt.wantRef {
				t.Errorf("Current = %s, want %s", result.Current, tt.wantRef)
			}
		})
	}
}

func TestParseActionRef_CommitSHA(t *testing.T) {
	ref, err := parseActionRef("actions/checkout@dae09b1f80164e2d312981ce81045e7e1f4640d2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref == nil {
		t.Fatal("expected commit SHA ref to parse")
	}
	if ref.SHA != "dae09b1f80164e2d312981ce81045e7e1f4640d2" {
		t.Errorf("SHA = %q, want the ref", ref.SHA)
	}
	if ref.Current != "dae09b1f80164e2d312981ce81045e7e1f4640d2" {
		t.Errorf("Current = %q, want the ref", ref.Current)
	}
	if !ref.Pinned() {
		t.Error("Pinned() = false, want true")
	}
	if ref.Tag != "" {
		t.Errorf("Tag = %q, want empty", ref.Tag)
	}
}

func TestExtractActionsSHPinMatrix(t *testing.T) {
	tmpDir := t.TempDir()
	workflowPath := filepath.Join(tmpDir, "test.yml")
	content := `name: CI
on: push
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@dae09b1f80164e2d312981ce81045e7e1f4640d2
      - uses: actions/cache@8a6d4d4b0c5f7e9a1b2c3d4e5f60718293a4b5c6 # v4.2.2
      - uses: actions/setup-go@6f1e2d3c4b5a69788796a5b4c3d2e1f0a9b8c7d6 # v4
      - uses: actions/setup-node@0123456789abcdef0123456789abcdef01234567 # pinned to v4.2.2
      - uses: google/osv-scanner-action/osv-scanner-action@a345acffa64b0eaede81a3d9aae6141214d9c8fc # v2.6.0
      - uses: "actions/upload-artifact@v4"
      - uses: actions/checkout@${{ matrix.version }}
      - uses:		actions/github-script@ABCDEF0123456789ABCDEF0123456789ABCDEF01    # v7
`
	if err := os.WriteFile(workflowPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}

	actions, err := ExtractActions(t.Context(), []string{workflowPath})
	if err != nil {
		t.Fatalf("ExtractActions failed: %v", err)
	}

	want := []struct {
		repo, path, current, sha, tag string
	}{
		{"checkout", "actions/checkout", "dae09b1f80164e2d312981ce81045e7e1f4640d2", "dae09b1f80164e2d312981ce81045e7e1f4640d2", ""},
		{"cache", "actions/cache", "8a6d4d4b0c5f7e9a1b2c3d4e5f60718293a4b5c6", "8a6d4d4b0c5f7e9a1b2c3d4e5f60718293a4b5c6", "v4.2.2"},
		{"setup-go", "actions/setup-go", "6f1e2d3c4b5a69788796a5b4c3d2e1f0a9b8c7d6", "6f1e2d3c4b5a69788796a5b4c3d2e1f0a9b8c7d6", "v4"},
		{"setup-node", "actions/setup-node", "0123456789abcdef0123456789abcdef01234567", "0123456789abcdef0123456789abcdef01234567", ""},
		{"osv-scanner-action", "google/osv-scanner-action/osv-scanner-action", "a345acffa64b0eaede81a3d9aae6141214d9c8fc", "a345acffa64b0eaede81a3d9aae6141214d9c8fc", "v2.6.0"},
		{"upload-artifact", "actions/upload-artifact", "v4", "", ""},
		{"github-script", "actions/github-script", "ABCDEF0123456789ABCDEF0123456789ABCDEF01", "abcdef0123456789abcdef0123456789abcdef01", "v7"},
	}

	if len(actions) != len(want) {
		t.Fatalf("expected %d actions, got %d: %+v", len(want), len(actions), actions)
	}
	for i, exp := range want {
		a := actions[i]
		if a.Repo != exp.repo || a.Path != exp.path || a.Current != exp.current || a.SHA != exp.sha || a.Tag != exp.tag {
			t.Errorf("action[%d] = {Repo:%q Path:%q Current:%q SHA:%q Tag:%q}, want %+v", i, a.Repo, a.Path, a.Current, a.SHA, a.Tag, exp)
		}
		if got, wantV := a.Version(), exp.tag; exp.sha != "" {
			if got != wantV {
				t.Errorf("action[%d].Version() = %q, want %q", i, got, wantV)
			}
		} else if got != exp.current {
			t.Errorf("action[%d].Version() = %q, want Current %q", i, got, exp.current)
		}
	}
}

func TestGroupActions(t *testing.T) {
	actions := []ActionRef{
		{Owner: "actions", Repo: "checkout", Current: "v3", File: "/a.yml"},
		{Owner: "actions", Repo: "checkout", Current: "v3", File: "/b.yml"},
		{Owner: "actions", Repo: "setup-go", Current: "v4", File: "/a.yml"},
	}

	grouped := GroupActions(actions)

	if len(grouped) != 2 {
		t.Errorf("expected 2 groups, got %d", len(grouped))
	}

	checkoutGroup, ok := grouped["actions/checkout"]
	if !ok {
		t.Fatal("missing actions/checkout group")
	}
	if len(checkoutGroup) != 2 {
		t.Errorf("expected 2 actions in checkout group, got %d", len(checkoutGroup))
	}

	setupGoGroup, ok := grouped["actions/setup-go"]
	if !ok {
		t.Fatal("missing actions/setup-go group")
	}
	if len(setupGoGroup) != 1 {
		t.Errorf("expected 1 action in setup-go group, got %d", len(setupGoGroup))
	}
}

func TestExtractActionsConcurrent(t *testing.T) {
	tmpDir := t.TempDir()

	numFiles := 15
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
      - uses: actions/setup-go@v4
`
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write workflow: %v", err)
		}
		paths = append(paths, path)
	}

	actions, err := ExtractActions(t.Context(), paths)
	if err != nil {
		t.Fatalf("ExtractActions failed: %v", err)
	}

	if len(actions) != numFiles*2 {
		t.Errorf("expected %d actions, got %d", numFiles*2, len(actions))
	}

	checkoutCount := 0
	setupGoCount := 0
	for _, a := range actions {
		if a.Repo == "checkout" {
			checkoutCount++
		}
		if a.Repo == "setup-go" {
			setupGoCount++
		}
	}

	if checkoutCount != numFiles || setupGoCount != numFiles {
		t.Errorf("expected %d of each action type, got %d checkout, %d setup-go", numFiles, checkoutCount, setupGoCount)
	}
}

func TestDistinctVersions(t *testing.T) {
	refs := []ActionRef{
		{Current: "v4"},
		{Current: "v2"},
		{Current: "v4"},
	}
	got := DistinctVersions(refs)
	want := []string{"v4", "v2"}
	if len(got) != len(want) {
		t.Fatalf("DistinctVersions = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("DistinctVersions[%d] = %q, want %q", i, got[i], want[i])
		}
	}

	if versions := DistinctVersions(nil); len(versions) != 0 {
		t.Errorf("expected no versions for empty input, got %v", versions)
	}
}

func inputWorkflow(t *testing.T, content string) []ActionRef {
	t.Helper()
	path := filepath.Join(t.TempDir(), "workflow.yml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write workflow: %v", err)
	}
	actions, err := ExtractActions(t.Context(), []string{path})
	if err != nil {
		t.Fatalf("ExtractActions failed: %v", err)
	}
	return actions
}

func assertInputs(t *testing.T, actions []ActionRef, idx int, want []string) {
	t.Helper()
	if idx >= len(actions) {
		t.Fatalf("expected action at index %d, got %d actions", idx, len(actions))
	}
	got := actions[idx].Inputs
	if len(got) != len(want) {
		t.Fatalf("action %d Inputs = %v, want %v", idx, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("action %d Inputs[%d] = %q, want %q", idx, i, got[i], want[i])
		}
	}
}

func TestExtractActionsWithBareWith(t *testing.T) {
	actions := inputWorkflow(t, `jobs:
  build:
    steps:
      - uses: actions/checkout@v4
        with:
          repository: foo/bar
          token: ${{ secrets.GITHUB_TOKEN }}
`)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	assertInputs(t, actions, 0, []string{"repository", "token"})
}

func TestExtractActionsWithBeforeUses(t *testing.T) {
	actions := inputWorkflow(t, `jobs:
  build:
    steps:
      - with: {repository: foo/bar, ref: main}
        uses: actions/checkout@v4
`)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	assertInputs(t, actions, 0, []string{"repository", "ref"})
}

func TestExtractActionsWithInlineMap(t *testing.T) {
	actions := inputWorkflow(t, `jobs:
  build:
    steps:
      - uses: actions/checkout@v4
        with: {repository: foo/bar, token: "${{ secrets.TOKEN }}", ref: main}
`)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	assertInputs(t, actions, 0, []string{"repository", "token", "ref"})
}

func TestExtractActionsWithQuotedKeys(t *testing.T) {
	actions := inputWorkflow(t, `jobs:
  build:
    steps:
      - uses: actions/checkout@v4
        with:
          "repository": foo/bar
          'token': x
`)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	assertInputs(t, actions, 0, []string{"repository", "token"})
}

func TestExtractActionsMultipleStepsIsolateWith(t *testing.T) {
	actions := inputWorkflow(t, `jobs:
  build:
    steps:
      - uses: actions/setup-go@v5
      - uses: actions/checkout@v4
        with:
          repository: foo/bar
      - uses: actions/cache@v4
`)
	if len(actions) != 3 {
		t.Fatalf("expected 3 actions, got %d", len(actions))
	}
	assertInputs(t, actions, 0, nil)
	assertInputs(t, actions, 1, []string{"repository"})
	assertInputs(t, actions, 2, nil)
}

func TestExtractActionsNestedComposite(t *testing.T) {
	actions := inputWorkflow(t, `runs:
  using: composite
  steps:
    - name: Checkout
      uses: actions/checkout@v4
      with:
        token: x
    - name: Go
      uses: actions/setup-go@v5
      with:
        go-version: "1.22"
`)
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
	assertInputs(t, actions, 0, []string{"token"})
	assertInputs(t, actions, 1, []string{"go-version"})
}

func TestExtractActionsRunBlockScalarDoesNotLeakWith(t *testing.T) {
	actions := inputWorkflow(t, `jobs:
  build:
    steps:
      - name: script
        run: |
          with:
            fake: key
        uses: actions/setup-go@v5
`)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d", len(actions))
	}
	assertInputs(t, actions, 0, nil)
}

func TestExtractActionsNoWith(t *testing.T) {
	actions := inputWorkflow(t, `jobs:
  build:
    steps:
      - uses: actions/checkout@v4
      - name: Setup
        uses: actions/setup-go@v5
`)
	if len(actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(actions))
	}
	assertInputs(t, actions, 0, nil)
	assertInputs(t, actions, 1, nil)
}

func TestExtractActionsSkipsDollarLocalRefs(t *testing.T) {
	actions := inputWorkflow(t, `jobs:
  build:
    steps:
      - uses: $/.github/actions/my-action@main
      - uses: $/actions/local-helper
      - uses: actions/checkout@v4
`)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d: %+v", len(actions), actions)
	}
	if actions[0].Owner != "actions" || actions[0].Repo != "checkout" {
		t.Errorf("unexpected action: %+v", actions[0])
	}
}
