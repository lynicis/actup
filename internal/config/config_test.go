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

func TestRender_SortedAndQuoted(t *testing.T) {
	major := 4
	cfg := &Config{
		Major: &major,
		Actions: map[string]string{
			"actions/setup-go": "v5",
			"actions/checkout": "v5",
			"some-org/action":  "4",
		},
	}

	got := Render(cfg)

	want := schemaHeader + "\n" +
		"\nmajor: 4\n" +
		"\nactions:\n" +
		"  actions/checkout: v5\n" +
		"  actions/setup-go: v5\n" +
		"  some-org/action: \"4\"\n"
	if got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestRender_EmptyConfig(t *testing.T) {
	got := Render(&Config{})
	if got != schemaHeader+"\n" {
		t.Errorf("Render(empty) = %q, want header only", got)
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".actup.yaml")

	major := 5
	cfg := &Config{
		Major: &major,
		Actions: map[string]string{
			"actions/checkout": "v5",
			"securego/gosec":   "master",
		},
	}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Major == nil || *loaded.Major != 5 {
		t.Errorf("major = %v, want 5", loaded.Major)
	}
	if loaded.Actions["actions/checkout"] != "v5" {
		t.Errorf("checkout pin = %q, want v5", loaded.Actions["actions/checkout"])
	}
	if loaded.Actions["securego/gosec"] != "master" {
		t.Errorf("gosec pin = %q, want master", loaded.Actions["securego/gosec"])
	}
}

func TestSaveDefault_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	cfg := &Config{Actions: map[string]string{"actions/checkout": "v5"}}
	if err := SaveDefault(cfg); err != nil {
		t.Fatalf("SaveDefault: %v", err)
	}

	loaded, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault: %v", err)
	}
	if loaded == nil || loaded.Actions["actions/checkout"] != "v5" {
		t.Fatalf("expected saved pin, got %v", loaded)
	}
}

func TestSaveDefault_ReusesExistingYml(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	if err := os.WriteFile(".actup.yml", []byte("actions:\n  a/b: v1\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg := &Config{Actions: map[string]string{"c/d": "v2"}}
	if err := SaveDefault(cfg); err != nil {
		t.Fatalf("SaveDefault: %v", err)
	}

	if _, err := os.Stat(".actup.yaml"); err == nil {
		t.Error("SaveDefault should reuse existing .actup.yml, not create .actup.yaml")
	}
	loaded, err := Load(".actup.yml")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded.Actions["c/d"] != "v2" {
		t.Errorf("c/d pin = %q, want v2", loaded.Actions["c/d"])
	}
}
