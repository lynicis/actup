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
		{"actions/checkout@e7d184456a78df3f4a5c6d7e8f9a0b1c2d3e4f60", true, "", "", ""},
		{"docker/login-action@v2", false, "docker", "login-action", "v2"},
		{"github/super-linter@v6", false, "github", "super-linter", "v6"},
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

func TestIsHex(t *testing.T) {
	tests := []struct {
		s        string
		expected bool
	}{
		{"abc123", true},
		{"ABCDEF", true},
		{"0123456789", true},
		{"abcdef1234567890", true},
		{"xyz", false},
		{"abcg", false},
		{"", true},
	}

	for _, tt := range tests {
		result := isHex(tt.s)
		if result != tt.expected {
			t.Errorf("isHex(%q) = %v, want %v", tt.s, result, tt.expected)
		}
	}
}
