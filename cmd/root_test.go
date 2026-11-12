package cmd

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lynicis/actup/internal/breakingchanges"
	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/parser"
)

func TestFilterSkippedActions(t *testing.T) {
	actions := []parser.ActionRef{
		{Owner: "actions", Repo: "checkout", Current: "v3"},
		{Owner: "actions", Repo: "setup-go", Current: "v4"},
	}
	cfg := &config.Config{
		Actions: map[string]string{
			"actions/checkout": "skip",
		},
	}
	filtered := filterSkippedActions(actions, cfg)
	if len(filtered) != 1 {
		t.Fatalf("expected 1 action after skip filter, got %d", len(filtered))
	}
	if filtered[0].Repo != "setup-go" {
		t.Errorf("expected setup-go, got %s", filtered[0].Repo)
	}
}

func TestNoTUIFlagRegistered(t *testing.T) {
	f := rootCmd.Flags().Lookup("no-tui")
	if f == nil {
		t.Fatal("--no-tui flag not registered")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("--no-tui should be bool, got %s", f.Value.Type())
	}
}

func TestInstallHooksSubcommandRegistered(t *testing.T) {
	for _, c := range rootCmd.Commands() {
		if c.Use == "install-hooks" {
			return
		}
	}
	t.Fatal("install-hooks subcommand not registered")
}

func TestConfirmBreakingAllMixedVersions(t *testing.T) {
	reg := breakingchanges.New([]breakingchanges.Entry{{
		Action:    "actions/checkout",
		FromMajor: 2,
		ToMajor:   3,
		Changes:   []breakingchanges.BreakingChange{{Type: "behavior", Message: "runs on Node 16"}},
	}})
	refs := []parser.ActionRef{{Current: "v4"}, {Current: "v2"}}
	reader := bufio.NewReader(strings.NewReader(""))

	proceed, skipped := confirmBreakingAll("actions/checkout", refs, "v4", reg, true, false, reader, false)
	if !proceed {
		t.Error("expected dry-run to proceed despite the v2 breaking change")
	}
	if skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", skipped)
	}
}

func TestConfirmBreakingAll_UnionInputsFromMultipleSites(t *testing.T) {
	reg := breakingchanges.New([]breakingchanges.Entry{{
		Action:    "codecov/codecov-action",
		FromMajor: 3,
		ToMajor:   4,
		Changes: []breakingchanges.BreakingChange{{
			Type:    "deprecated",
			Input:   "file",
			Message: "file is deprecated in favor of files",
		}},
	}})
	reader := bufio.NewReader(strings.NewReader(""))

	// Two sites of the same version: only the second passes the affected input.
	refs := []parser.ActionRef{
		{Current: "v3", Inputs: []string{"token"}},
		{Current: "v3", Inputs: []string{"file"}},
	}
	proceed, skipped := confirmBreakingAll("codecov/codecov-action", refs, "v4", reg, false, false, reader, false)
	if proceed || skipped != 1 {
		t.Errorf("expected the union to trigger the warning, got proceed=%v skipped=%d", proceed, skipped)
	}

	// No site passes the affected input: the warning is suppressed.
	refs = []parser.ActionRef{
		{Current: "v3", Inputs: []string{"token"}},
		{Current: "v3"},
	}
	proceed, skipped = confirmBreakingAll("codecov/codecov-action", refs, "v4", reg, false, false, reader, false)
	if !proceed || skipped != 0 {
		t.Errorf("expected suppression without the input, got proceed=%v skipped=%d", proceed, skipped)
	}
}

func resetFilterTestState(t *testing.T) {
	t.Helper()
	oldGroups, oldPaths := filterGroups, paths
	oldOutdatedPaths, oldOutdatedGroups := outdatedOpts.paths, outdatedOpts.groups
	oldToken := githubToken
	filterGroups = nil
	paths = nil
	outdatedOpts.paths = nil
	outdatedOpts.groups = nil
	githubToken = "test-token"
	t.Cleanup(func() {
		filterGroups = oldGroups
		paths = oldPaths
		outdatedOpts.paths = oldOutdatedPaths
		outdatedOpts.groups = oldOutdatedGroups
		githubToken = oldToken
		rootCmd.SetArgs(nil)
	})
}

func TestFilterActionsByPatterns(t *testing.T) {
	actions := []parser.ActionRef{
		{Owner: "actions", Repo: "checkout"},
		{Owner: "docker", Repo: "login-action"},
	}

	t.Run("exact key", func(t *testing.T) {
		got, err := filterActionsByPatterns(actions, []string{"docker/login-action"})
		if err != nil {
			t.Fatalf("filter: %v", err)
		}
		if len(got) != 1 || got[0].Repo != "login-action" {
			t.Fatalf("got %v, want only docker/login-action", got)
		}
	})

	t.Run("glob stays within owner", func(t *testing.T) {
		got, err := filterActionsByPatterns(actions, []string{"actions/*"})
		if err != nil {
			t.Fatalf("filter: %v", err)
		}
		if len(got) != 1 || got[0].Repo != "checkout" {
			t.Fatalf("got %v, want only actions/checkout", got)
		}
	})

	t.Run("multiple patterns union", func(t *testing.T) {
		got, err := filterActionsByPatterns(actions, []string{"actions/*", "docker/*"})
		if err != nil {
			t.Fatalf("filter: %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("got %d actions, want 2", len(got))
		}
	})

	t.Run("invalid pattern", func(t *testing.T) {
		_, err := filterActionsByPatterns(actions, []string{"["})
		if err == nil || !strings.Contains(err.Error(), "invalid filter") {
			t.Fatalf("expected invalid filter error, got %v", err)
		}
	})

	t.Run("empty patterns returns input", func(t *testing.T) {
		got, err := filterActionsByPatterns(actions, nil)
		if err != nil {
			t.Fatalf("filter: %v", err)
		}
		if len(got) != len(actions) || &got[0] != &actions[0] {
			t.Fatalf("expected the input slice unchanged")
		}
	})
}

func TestRunRejectsNoMatchBeforeNetwork(t *testing.T) {
	dir := t.TempDir()
	writeWorkflow(t, dir, "ci.yml", "steps:\n  - uses: actions/checkout@v5\n")
	t.Chdir(dir)
	resetFilterTestState(t)

	rootCmd.SetArgs([]string{"nonexistent/*", "--no-tui", "--dry-run", "--path", dir})
	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "no actions match") {
		t.Fatalf("expected no-match error, got %v", err)
	}
	if strings.Contains(err.Error(), "unknown command") {
		t.Fatalf("positional filter must not be treated as a command: %v", err)
	}
}

func TestRootAcceptsPositionalFilter(t *testing.T) {
	dir := t.TempDir()
	writeWorkflow(t, dir, "ci.yml", "steps:\n  - uses: actions/checkout@v5\n")
	if err := os.WriteFile(filepath.Join(dir, ".actup.yaml"), []byte("pins:\n  actions/checkout: skip\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	resetFilterTestState(t)

	rootCmd.SetArgs([]string{"actions/*", "--no-tui", "--dry-run", "--path", dir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("positional filter should be accepted, got %v", err)
	}
}

func TestOutdatedRejectsNoMatch(t *testing.T) {
	dir := t.TempDir()
	writeWorkflow(t, dir, "ci.yml", "steps:\n  - uses: actions/checkout@v5\n")
	t.Chdir(dir)
	resetFilterTestState(t)

	rootCmd.SetArgs([]string{"outdated", "nonexistent/*", "--path", dir})
	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "no actions match") {
		t.Fatalf("expected no-match error, got %v", err)
	}
}

func TestGroupFlagUnknown(t *testing.T) {
	dir := t.TempDir()
	writeWorkflow(t, dir, "ci.yml", "steps:\n  - uses: actions/checkout@v5\n")
	if err := os.WriteFile(filepath.Join(dir, ".actup.yaml"), []byte("groups:\n  ci:\n    - actions/*\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	resetFilterTestState(t)

	rootCmd.SetArgs([]string{"--group", "nope", "--no-tui", "--dry-run", "--path", dir})
	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "unknown group") {
		t.Fatalf("expected unknown group error, got %v", err)
	}
}

func TestGroupFlagWithoutManifest(t *testing.T) {
	dir := t.TempDir()
	writeWorkflow(t, dir, "ci.yml", "steps:\n  - uses: actions/checkout@v5\n")
	t.Chdir(dir)
	resetFilterTestState(t)

	rootCmd.SetArgs([]string{"--group", "ci", "--no-tui", "--dry-run", "--path", dir})
	err := rootCmd.Execute()
	if err == nil || !strings.Contains(err.Error(), "--group requires .actup.yaml") {
		t.Fatalf("expected missing manifest error, got %v", err)
	}
}

func TestGroupPlusPositionalUnion(t *testing.T) {
	actions := []parser.ActionRef{
		{Owner: "actions", Repo: "checkout"},
		{Owner: "docker", Repo: "login-action"},
		{Owner: "other", Repo: "action"},
	}
	cfg := &config.Config{Groups: map[string][]string{"ci": {"actions/*"}}}

	got, err := applyActionFilters(actions, []string{"docker/*"}, []string{"ci"}, cfg)
	if err != nil {
		t.Fatalf("applyActionFilters: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d actions, want 2", len(got))
	}
}

func TestIsGHExtensionInvocation(t *testing.T) {
	if !isGHExtensionInvocation("/Users/me/.local/share/gh/extensions/gh-actup/gh-actup", "") {
		t.Error("expected gh-actup binary name to be detected as gh extension")
	}
	if !isGHExtensionInvocation("/usr/local/bin/actup", "/opt/homebrew/bin/gh") {
		t.Error("expected GH_PATH env var to be detected as gh extension")
	}
	if isGHExtensionInvocation("/usr/local/bin/actup", "") {
		t.Error("plain actup binary without GH_PATH must not be detected as gh extension")
	}
}
