package cmd

import (
	"testing"
)

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
