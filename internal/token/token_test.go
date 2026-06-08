package token

import (
	"os"
	"os/exec"
	"testing"
)

func TestResolver_Resolve_FlagValueTakesPriority(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "env-token")

	resolver := NewResolver()
	resolver.run = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("echo", "gh-token")
	}

	result := resolver.Resolve("flag-token")
	if result != "flag-token" {
		t.Errorf("expected flag-token, got %q", result)
	}
}

func TestResolver_Resolve_EnvVarUsedWhenFlagEmpty(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "env-token")

	resolver := NewResolver()
	resolver.run = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("echo", "gh-token")
	}

	result := resolver.Resolve("")
	if result != "env-token" {
		t.Errorf("expected env-token, got %q", result)
	}
}

func TestResolver_Resolve_GhTokenUsedWhenFlagAndEnvEmpty(t *testing.T) {
	_ = os.Unsetenv("GITHUB_TOKEN")

	resolver := NewResolver()
	resolver.run = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("echo", "gh-token")
	}

	result := resolver.Resolve("")
	if result != "gh-token" {
		t.Errorf("expected gh-token, got %q", result)
	}
}

func TestResolver_Resolve_GhNotFoundReturnsEmpty(t *testing.T) {
	_ = os.Unsetenv("GITHUB_TOKEN")

	resolver := NewResolver()
	resolver.run = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("nonexistent-command-gh-auth-token")
	}

	result := resolver.Resolve("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestResolver_Resolve_GhAuthErrorReturnsEmpty(t *testing.T) {
	_ = os.Unsetenv("GITHUB_TOKEN")

	resolver := NewResolver()
	resolver.run = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("false")
	}

	result := resolver.Resolve("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}
