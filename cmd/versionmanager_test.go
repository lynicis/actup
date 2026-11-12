package cmd

import (
	"context"
	"io"
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

	err = doUse(context.Background(), []actionSpec{spec}, useOptions{noSave: false, force: true})
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
	if err := doUse(context.Background(), []actionSpec{spec}, useOptions{force: true}); err != nil {
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

	err := doSync(context.Background(), syncOptions{force: true})
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

	if err := doSync(context.Background(), syncOptions{force: true}); err != nil {
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

func TestInitCreatesManagedManifest(t *testing.T) {
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
		t.Fatal("expected managed manifest")
	}
	if cfg.Actions["actions/checkout"] != "v3" || cfg.Actions["actions/setup-go"] != "v4" {
		t.Errorf("unexpected recorded actions: %v", cfg.Actions)
	}
	if cfg.Pins == nil {
		t.Fatal("fresh init must produce a managed manifest (pins non-nil)")
	}
	if len(cfg.Pins) != 0 {
		t.Errorf("fresh init must not add constraints, got %v", cfg.Pins)
	}
}

func TestUseOnLegacyStaysLegacy(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeWorkflow(t, dir, "ci.yml", testWorkflow)
	if err := os.WriteFile(".actup.yaml", []byte("actions:\n  actions/checkout: v3\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	spec, _ := parseActionSpec("actions/checkout@v5")
	if err := doUse(context.Background(), []actionSpec{spec}, useOptions{force: true}); err != nil {
		t.Fatalf("doUse: %v", err)
	}

	cfg, err := config.LoadDefault()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Pins != nil {
		t.Errorf("legacy manifest must not become managed, got pins %v", cfg.Pins)
	}
	if cfg.Actions["actions/checkout"] != "v5" {
		t.Errorf("expected checkout=v5, got %q", cfg.Actions["actions/checkout"])
	}
}

func TestUseOnManagedWritesPin(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeWorkflow(t, dir, "ci.yml", testWorkflow)
	if err := os.WriteFile(".actup.yaml", []byte("pins: {}\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	spec, _ := parseActionSpec("actions/checkout@v5")
	if err := doUse(context.Background(), []actionSpec{spec}, useOptions{force: true}); err != nil {
		t.Fatalf("doUse: %v", err)
	}

	cfg, err := config.LoadDefault()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Actions["actions/checkout"] != "v5" {
		t.Errorf("expected recorded checkout=v5, got %q", cfg.Actions["actions/checkout"])
	}
	if cfg.Pins["actions/checkout"] != "v5" {
		t.Errorf("expected constraint checkout=v5, got %q", cfg.Pins["actions/checkout"])
	}
}

func TestUnpinRemovesBoth(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(".actup.yaml", []byte("pins:\n  actions/checkout: v5\nactions:\n  actions/checkout: v5\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := runUnpin(unpinCmd, []string{"actions/checkout"}); err != nil {
		t.Fatalf("runUnpin: %v", err)
	}

	cfg, err := config.LoadDefault()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if _, ok := cfg.Actions["actions/checkout"]; ok {
		t.Error("key should be removed from actions")
	}
	if _, ok := cfg.Pins["actions/checkout"]; ok {
		t.Error("key should be removed from pins")
	}
}

func TestSyncRestoresSnapshotOnlyKey(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := writeWorkflow(t, dir, "ci.yml", testWorkflow)
	if err := os.WriteFile(".actup.yaml", []byte("pins: {}\nactions:\n  actions/checkout: v1\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := doSync(context.Background(), syncOptions{force: true}); err != nil {
		t.Fatalf("doSync: %v", err)
	}

	content, _ := os.ReadFile(path)
	if strings.Count(string(content), "actions/checkout@v1") != 2 {
		t.Errorf("sync should restore the recorded snapshot v1, got:\n%s", content)
	}
}

func TestSyncPrefersPinOverSnapshot(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := writeWorkflow(t, dir, "ci.yml", testWorkflow)
	if err := os.WriteFile(".actup.yaml", []byte("pins:\n  actions/checkout: v2\nactions:\n  actions/checkout: v1\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := doSync(context.Background(), syncOptions{force: true}); err != nil {
		t.Fatalf("doSync: %v", err)
	}

	content, _ := os.ReadFile(path)
	if strings.Count(string(content), "actions/checkout@v2") != 2 {
		t.Errorf("sync should prefer the pin v2, got:\n%s", content)
	}
}

func TestSyncRecordsActionsAfterSuccess(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeWorkflow(t, dir, "ci.yml", testWorkflow)
	if err := os.WriteFile(".actup.yaml", []byte("pins:\n  actions/checkout: v5\nactions:\n  actions/checkout: v3\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	if err := doSync(context.Background(), syncOptions{force: true}); err != nil {
		t.Fatalf("doSync: %v", err)
	}

	cfg, err := config.LoadDefault()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if cfg.Actions["actions/checkout"] != "v5" {
		t.Errorf("expected recorded checkout=v5 after sync, got %q", cfg.Actions["actions/checkout"])
	}
}

func TestRecordUpgrades_ManagedOnly(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	cfg := &config.Config{Actions: map[string]string{"actions/checkout": "v1"}, Pins: map[string]string{}}
	if err := recordUpgrades(cfg, map[string]string{"actions/checkout": "v2"}); err != nil {
		t.Fatalf("recordUpgrades: %v", err)
	}

	loaded, err := config.LoadDefault()
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	if loaded.Actions["actions/checkout"] != "v2" {
		t.Errorf("expected saved action v2, got %q", loaded.Actions["actions/checkout"])
	}
	if loaded.Pins == nil {
		t.Error("managed marker must survive the save")
	}
}

func TestRecordUpgrades_LegacyNoop(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	cfg := &config.Config{Actions: map[string]string{"actions/checkout": "v1"}}
	if err := recordUpgrades(cfg, map[string]string{"actions/checkout": "v2"}); err != nil {
		t.Fatalf("recordUpgrades: %v", err)
	}
	if _, err := os.Stat(".actup.yaml"); err == nil {
		t.Error("legacy manifests must not be written by recordUpgrades")
	}
}

func TestRecordUpgrades_NoManifestNoop(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := recordUpgrades(nil, map[string]string{"actions/checkout": "v2"}); err != nil {
		t.Fatalf("recordUpgrades(nil) = %v, want nil", err)
	}
	if _, err := os.Stat(".actup.yaml"); err == nil {
		t.Error("recordUpgrades must not create a manifest")
	}
}

// captureStdout runs fn with os.Stdout redirected to a pipe and returns what
// was printed.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = old }()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close pipe writer: %v", err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close pipe reader: %v", err)
	}
	return string(out)
}

func TestUseAndSync_AutoRelockWhenActionsLockExists(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeWorkflow(t, dir, "ci.yml", testWorkflow)
	lockPath := filepath.Join(dir, ".github", "workflows", "actions.lock")
	if err := os.WriteFile(lockPath, []byte("version: 'v0.0.3'\nworkflows: {}\ndependencies: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	calls := 0
	origRelock := relockFunc
	relockFunc = func(path string) error {
		calls++
		return nil
	}
	defer func() { relockFunc = origRelock }()

	// Dry-run must NOT trigger relock
	if err := doUse(context.Background(), []actionSpec{{Key: "actions/checkout", Ref: "v5"}}, useOptions{dryRun: true, force: true}); err != nil {
		t.Fatalf("doUse dry-run: %v", err)
	}
	if calls != 0 {
		t.Fatalf("dry-run must not call relock, got %d calls", calls)
	}

	// Real write MUST trigger relock once
	if err := doUse(context.Background(), []actionSpec{{Key: "actions/checkout", Ref: "v5"}}, useOptions{force: true}); err != nil {
		t.Fatalf("doUse: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 relock call after doUse, got %d", calls)
	}
}

func TestLs_ShowsLockStatusWhenActionsLockPresent(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeWorkflow(t, dir, "ci.yml", testWorkflow)
	lockContent := `version: 'v0.0.3'
workflows:
  '.github/workflows/ci.yml':
    - 'actions/checkout@v3'
dependencies:
  'actions/checkout@v3':
    ref: 'v3.6.0'
    commit: 'sha1-1111111111111111111111111111111111111111'
`
	lockPath := filepath.Join(dir, ".github", "workflows", "actions.lock")
	if err := os.WriteFile(lockPath, []byte(lockContent), 0o644); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		lsOpts.paths = nil
		lsOpts.outdated = false
		if err := runLs(lsCmd, nil); err != nil {
			t.Fatalf("runLs: %v", err)
		}
	})
	var checkoutLine, setupLine string
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.Contains(line, "actions/checkout"):
			checkoutLine = line
		case strings.Contains(line, "actions/setup-go"):
			setupLine = line
		}
	}
	if !strings.Contains(checkoutLine, "[locked]") || strings.Contains(checkoutLine, "[lock stale]") {
		t.Errorf("expected actions/checkout to show [locked], got line %q (full output:\n%s)", checkoutLine, out)
	}
	if !strings.Contains(setupLine, "[lock stale]") || strings.Contains(setupLine, "[locked]") {
		t.Errorf("expected actions/setup-go to show [lock stale], got line %q (full output:\n%s)", setupLine, out)
	}
}
