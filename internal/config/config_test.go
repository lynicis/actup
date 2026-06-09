package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_MissingFile(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "nonexistent.yaml"))
	if err != nil {
		t.Fatalf("Load missing file: %v", err)
	}
	if cfg != nil {
		t.Fatal("expected nil for missing file")
	}
}

func TestLoad_ValidConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".actup.yaml")
	if err := os.WriteFile(path, []byte("major: 4\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load valid config: %v", err)
	}
	if cfg.Major == nil || *cfg.Major != 4 {
		t.Fatalf("expected major=4, got %v", cfg.Major)
	}
}

func TestLoad_WithActions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".actup.yaml")
	if err := os.WriteFile(path, []byte("actions:\n  actions/checkout: \"4\"\n  actions/setup-go: skip\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load with actions: %v", err)
	}
	if cfg.Actions["actions/checkout"] != "4" {
		t.Errorf("expected checkout=4, got %q", cfg.Actions["actions/checkout"])
	}
	if cfg.Actions["actions/setup-go"] != "skip" {
		t.Errorf("expected setup-go=skip, got %q", cfg.Actions["actions/setup-go"])
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".actup.yaml")
	if err := os.WriteFile(path, []byte("major: [invalid\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}
