package token

import (
	"os"
	"os/exec"
	"testing"
)

func TestResolve_FlagValueTakesPriority(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "env-token")
	result := Resolve("flag-token")
	if result != "flag-token" {
		t.Errorf("expected flag-token, got %q", result)
	}
}

func TestResolve_EnvVarUsedWhenFlagEmpty(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "env-token")
	result := Resolve("")
	if result != "env-token" {
		t.Errorf("expected env-token, got %q", result)
	}
}

func TestResolve_GhTokenUsedWhenFlagAndEnvEmpty(t *testing.T) {
	_ = os.Unsetenv("GITHUB_TOKEN")
	orig := execCommand
	defer func() { execCommand = orig }()
	execCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("echo", "gh-token")
	}

	result := Resolve("")
	if result != "gh-token" {
		t.Errorf("expected gh-token, got %q", result)
	}
}

func TestResolve_GhNotFoundReturnsEmpty(t *testing.T) {
	_ = os.Unsetenv("GITHUB_TOKEN")
	orig := execCommand
	defer func() { execCommand = orig }()
	execCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("nonexistent-command-gh-auth-token")
	}

	result := Resolve("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestResolve_GhAuthErrorReturnsEmpty(t *testing.T) {
	_ = os.Unsetenv("GITHUB_TOKEN")
	orig := execCommand
	defer func() { execCommand = orig }()
	execCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("false")
	}

	result := Resolve("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}
