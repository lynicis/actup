package cmd

import (
	"testing"

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

func TestCheckFlagRegistered(t *testing.T) {
	f := rootCmd.Flags().Lookup("check")
	if f == nil {
		t.Fatal("--check flag not registered")
	}
	if f.Value.Type() != "bool" {
		t.Errorf("--check should be bool, got %s", f.Value.Type())
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
