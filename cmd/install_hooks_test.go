package cmd

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallHooks_CreatesHook(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(filepath.Join(gitDir, "hooks"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := installHook(tmpDir, false); err != nil {
		t.Fatal(err)
	}

	installHookPath := filepath.Join(gitDir, "hooks", "pre-commit")
	data, err := os.ReadFile(installHookPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 {
		t.Error("hook file is empty")
	}

	info, err := os.Stat(installHookPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0111 == 0 {
		t.Error("hook should be executable")
	}
}

func TestInstallHooks_Uninstall(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(filepath.Join(gitDir, "hooks"), 0755); err != nil {
		t.Fatal(err)
	}

	hookPath := filepath.Join(gitDir, "hooks", "pre-commit")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\necho test\n"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := uninstallHook(tmpDir); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(hookPath); err == nil {
		t.Error("hook should be removed after uninstall")
	}
}

func TestInstallHooks_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(filepath.Join(gitDir, "hooks"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := installHookDryRun(tmpDir); err != nil {
		t.Fatal(err)
	}

	hookPath := filepath.Join(gitDir, "hooks", "pre-commit")
	if _, err := os.Stat(hookPath); err == nil {
		t.Error("hook should not be created in dry-run mode")
	}
}

func TestInstallHooks_ExistingHookNoForce(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(filepath.Join(gitDir, "hooks"), 0755); err != nil {
		t.Fatal(err)
	}

	hookPath := filepath.Join(gitDir, "hooks", "pre-commit")
	original := []byte("#!/bin/sh\necho original\n")
	if err := os.WriteFile(hookPath, original, 0755); err != nil {
		t.Fatal(err)
	}

	if err := installHook(tmpDir, false); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != string(original) {
		t.Error("existing hook should not be overwritten without --force")
	}
}

func TestInstallHooks_ExistingHookForce(t *testing.T) {
	tmpDir := t.TempDir()
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(filepath.Join(gitDir, "hooks"), 0755); err != nil {
		t.Fatal(err)
	}

	hookPath := filepath.Join(gitDir, "hooks", "pre-commit")
	if err := os.WriteFile(hookPath, []byte("#!/bin/sh\necho original\n"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := installHook(tmpDir, true); err != nil {
		t.Fatal(err)
	}

	data, err := os.ReadFile(hookPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) == "#!/bin/sh\necho original\n" {
		t.Error("existing hook should be overwritten with --force")
	}
}
