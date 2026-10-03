package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/parser"
)

// writeWorkflow writes a workflow file under <dir>/.github/workflows and
// returns its path.
func writeWorkflow(t *testing.T, dir, name, content string) string {
	t.Helper()
	workflowDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		t.Fatalf("mkdir workflows: %v", err)
	}
	path := filepath.Join(workflowDir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write workflow: %v", err)
	}
	return path
}

const testWorkflow = `name: CI
on: push
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: actions/setup-go@v4
      - uses: actions/checkout@v3
`

func TestParseActionSpec(t *testing.T) {
	tests := []struct {
		arg     string
		wantKey string
		wantRef string
		wantErr bool
	}{
		{arg: "actions/checkout@v5", wantKey: "actions/checkout", wantRef: "v5"},
		{arg: "actions/checkout", wantKey: "actions/checkout", wantRef: ""},
		{arg: "owner/repo/sub@v1", wantKey: "owner/repo", wantRef: "v1"},
		{arg: "checkout", wantErr: true},
		{arg: "@v5", wantErr: true},
		{arg: "owner/@v5", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.arg, func(t *testing.T) {
			spec, err := parseActionSpec(tt.arg)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseActionSpec(%q) expected error, got %+v", tt.arg, spec)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseActionSpec(%q) error: %v", tt.arg, err)
			}
			if spec.Key != tt.wantKey || spec.Ref != tt.wantRef {
				t.Errorf("parseActionSpec(%q) = %+v, want key=%s ref=%s", tt.arg, spec, tt.wantKey, tt.wantRef)
			}
		})
	}
}

func TestResolveSpecRef_Verbatim(t *testing.T) {
	specs := []struct {
		ref  string
		want string
	}{
		{ref: "v5.3.1", want: "v5.3.1"},
		{ref: "master", want: "master"},
	}
	for _, tt := range specs {
		got, err := resolveSpecRef(context.Background(), nil, actionSpec{Key: "a/b", Ref: tt.ref}, false)
		if err != nil {
			t.Fatalf("resolveSpecRef(%q): %v", tt.ref, err)
		}
		if got != tt.want {
			t.Errorf("resolveSpecRef(%q) = %q, want %q", tt.ref, got, tt.want)
		}
	}
}

func TestMostCommonVersion(t *testing.T) {
	refs := []parser.ActionRef{
		{Current: "v3"}, {Current: "v4"}, {Current: "v3"},
	}
	if got := mostCommonVersion(refs); got != "v3" {
		t.Errorf("mostCommonVersion = %q, want v3", got)
	}
	if got := mostCommonVersion(nil); got != "" {
		t.Errorf("mostCommonVersion(nil) = %q, want empty", got)
	}
}

func TestDoUse_UpdatesWorkflowsAndSavesPin(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := writeWorkflow(t, dir, "ci.yml", testWorkflow)

	spec, err := parseActionSpec("actions/checkout@v5")
	if err != nil {
		t.Fatalf("parseActionSpec: %v", err)
	}

	err = doUse(context.Background(), []actionSpec{spec}, useOptions{noSave: false})
	if err != nil {
		t.Fatalf("doUse: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read workflow: %v", err)
	}
	if strings.Contains(string(content), "actions/checkout@v3") {
		t.Error("workflow still contains actions/checkout@v3")
	}
	if strings.Count(string(content), "actions/checkout@v5") != 2 {
		t.Errorf("expected 2 checkout@v5 references, got:\n%s", content)
	}
	if strings.Contains(string(content), "actions/setup-go@v4") == false {
		t.Error("unrelated actions/setup-go@v4 should be untouched")
	}

	cfg, err := config.LoadDefault()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg == nil || cfg.Actions["actions/checkout"] != "v5" {
		t.Errorf("expected pin actions/checkout=v5 in config, got %v", cfg)
	}
}

func TestDoUse_NoSaveSkipsConfig(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeWorkflow(t, dir, "ci.yml", testWorkflow)

	spec, _ := parseActionSpec("actions/checkout@v5")
	if err := doUse(context.Background(), []actionSpec{spec}, useOptions{noSave: true}); err != nil {
		t.Fatalf("doUse: %v", err)
	}

	if _, err := os.Stat(".actup.yaml"); err == nil {
		t.Error("--no-save should not create .actup.yaml")
	}
}

func TestDoUse_DryRunLeavesEverythingUntouched(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := writeWorkflow(t, dir, "ci.yml", testWorkflow)
	before, _ := os.ReadFile(path)

	spec, _ := parseActionSpec("actions/checkout@v5")
	if err := doUse(context.Background(), []actionSpec{spec}, useOptions{dryRun: true}); err != nil {
		t.Fatalf("doUse: %v", err)
	}

	after, _ := os.ReadFile(path)
	if string(before) != string(after) {
		t.Error("--dry-run must not modify workflows")
	}
	if _, err := os.Stat(".actup.yaml"); err == nil {
		t.Error("--dry-run must not create .actup.yaml")
	}
}

func TestDoUse_AlreadyAtTarget(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := writeWorkflow(t, dir, "ci.yml", testWorkflow)

	spec, _ := parseActionSpec("actions/setup-go@v4")
	if err := doUse(context.Background(), []actionSpec{spec}, useOptions{}); err != nil {
		t.Fatalf("doUse: %v", err)
	}

	content, _ := os.ReadFile(path)
	if !strings.Contains(string(content), "actions/setup-go@v4") {
		t.Error("workflow should be unchanged")
	}
}

func TestDoSync_AppliesPins(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := writeWorkflow(t, dir, "ci.yml", testWorkflow)

	if err := os.WriteFile(".actup.yaml", []byte("actions:\n  actions/checkout: v5\n  actions/setup-go: skip\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	err := doSync(context.Background(), syncOptions{})
	if err != nil {
		t.Fatalf("doSync: %v", err)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read workflow: %v", err)
	}
	if strings.Count(string(content), "actions/checkout@v5") != 2 {
		t.Errorf("sync should pin checkout to v5, got:\n%s", content)
	}
	if !strings.Contains(string(content), "actions/setup-go@v4") {
		t.Error("skipped action must be untouched")
	}
}

func TestDoSync_ExactRefsNeedNoToken(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeWorkflow(t, dir, "ci.yml", testWorkflow)

	if err := os.WriteFile(".actup.yaml", []byte("actions:\n  actions/checkout: v4.2.2\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := doSync(context.Background(), syncOptions{}); err != nil {
		t.Fatalf("doSync with exact refs should not hit the API: %v", err)
	}
}

func TestDoInit_CreatesManifestAndRefusesOverwrite(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeWorkflow(t, dir, "ci.yml", testWorkflow)

	if err := doInit(context.Background(), initOptions{}); err != nil {
		t.Fatalf("doInit: %v", err)
	}

	cfg, err := config.LoadDefault()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected .actup.yaml to be created")
	}
	if cfg.Actions["actions/checkout"] != "v3" || cfg.Actions["actions/setup-go"] != "v4" {
		t.Errorf("unexpected pins: %v", cfg.Actions)
	}

	// Second run must refuse without --force.
	if err := doInit(context.Background(), initOptions{}); err == nil {
		t.Error("doInit should refuse to overwrite an existing config")
	}

	// With --force it merges and preserves existing pins.
	cfg.Actions["actions/checkout"] = "v5"
	if err := config.SaveDefault(cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if err := doInit(context.Background(), initOptions{force: true}); err != nil {
		t.Fatalf("doInit --force: %v", err)
	}
	cfg, err = config.LoadDefault()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Actions["actions/checkout"] != "v5" {
		t.Errorf("--force must preserve existing pins, got %v", cfg.Actions)
	}
	if cfg.Actions["actions/setup-go"] != "v4" {
		t.Errorf("--force must add missing pins, got %v", cfg.Actions)
	}
}

func TestRunPinAndUnpin(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeWorkflow(t, dir, "ci.yml", testWorkflow)

	// pin with explicit ref
	if err := runPin(pinCmd, []string{"actions/checkout@v5"}); err != nil {
		t.Fatalf("runPin: %v", err)
	}
	cfg, _ := config.LoadDefault()
	if cfg == nil || cfg.Actions["actions/checkout"] != "v5" {
		t.Fatalf("expected checkout pinned to v5, got %v", cfg)
	}

	// pin without ref uses the workflow version
	if err := runPin(pinCmd, []string{"actions/setup-go"}); err != nil {
		t.Fatalf("runPin: %v", err)
	}
	cfg, _ = config.LoadDefault()
	if cfg.Actions["actions/setup-go"] != "v4" {
		t.Errorf("expected setup-go pinned to v4, got %q", cfg.Actions["actions/setup-go"])
	}

	// unpin
	if err := runUnpin(unpinCmd, []string{"actions/checkout"}); err != nil {
		t.Fatalf("runUnpin: %v", err)
	}
	cfg, _ = config.LoadDefault()
	if _, ok := cfg.Actions["actions/checkout"]; ok {
		t.Error("expected checkout to be unpinned")
	}

	// unpin something absent is a no-op
	if err := runUnpin(unpinCmd, []string{"actions/checkout"}); err != nil {
		t.Fatalf("runUnpin (absent): %v", err)
	}
}

func TestVersionManagerSubcommandsRegistered(t *testing.T) {
	want := []string{"use", "pin", "unpin", "ls", "sync", "init", "outdated"}
	for _, name := range want {
		found := false
		for _, c := range rootCmd.Commands() {
			if c.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("subcommand %q not registered", name)
		}
	}
}

func TestSummarizeActionVersions(t *testing.T) {
	refs := []parser.ActionRef{
		{Current: "v4"}, {Current: "v4"}, {Current: "v3"},
	}
	got := summarizeActionVersions(refs)
	want := "v3 ×1, v4 ×2"
	if got != want {
		t.Errorf("summarizeActionVersions = %q, want %q", got, want)
	}
}
