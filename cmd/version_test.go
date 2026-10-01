package cmd

import (
	"bytes"
	"fmt"
	"runtime"
	"strings"
	"testing"
)

func TestVersionOutput(t *testing.T) {
	SetVersionInfo("1.2.3", "abcdef1", "2026-03-01T00:00:00Z")
	t.Cleanup(func() {
		SetVersionInfo("dev", "none", "unknown")
	})

	expectedFull := fmt.Sprintf("actup version 1.2.3 (commit: abcdef1, built at: 2026-03-01T00:00:00Z, %s/%s)\n", runtime.GOOS, runtime.GOARCH)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := buf.String(); got != expectedFull {
		t.Errorf("expected %q, got %q", expectedFull, got)
	}
}

func TestVersionShortOutput(t *testing.T) {
	SetVersionInfo("1.2.3", "abcdef1", "2026-03-01T00:00:00Z")
	t.Cleanup(func() {
		SetVersionInfo("dev", "none", "unknown")
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"version", "--short"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expectedShort := "1.2.3\n"
	if got := buf.String(); got != expectedShort {
		t.Errorf("expected %q, got %q", expectedShort, got)
	}
}

func TestRootVersionFlag(t *testing.T) {
	SetVersionInfo("1.2.3", "abcdef1", "2026-03-01T00:00:00Z")
	t.Cleanup(func() {
		SetVersionInfo("dev", "none", "unknown")
	})

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--version"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(buf.String(), "1.2.3") {
		t.Errorf("expected version output to contain 1.2.3, got %q", buf.String())
	}
}

func TestVersionSubcommandRegistered(t *testing.T) {
	for _, c := range rootCmd.Commands() {
		if c.Use == "version" {
			return
		}
	}
	t.Fatal("version subcommand not registered")
}
