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

func TestEffectivePins_LegacyFallsBackToActions(t *testing.T) {
	cfg := &Config{Actions: map[string]string{"actions/checkout": "v5", "actions/setup-go": "skip"}}

	got := cfg.EffectivePins()
	if got["actions/checkout"] != "v5" || got["actions/setup-go"] != "skip" {
		t.Errorf("EffectivePins() = %v, want Actions", got)
	}
	if cfg.IsManaged() {
		t.Error("Actions-only config must not be managed")
	}
}

func TestEffectivePins_ManagedPrefersPins(t *testing.T) {
	cfg := &Config{
		Actions: map[string]string{
			"actions/checkout": "v1",
			"actions/setup-go": "3",
			"some/action":      "skip",
			"other/action":     "v9",
		},
		Pins: map[string]string{"actions/checkout": "v5"},
	}

	got := cfg.EffectivePins()
	if got["actions/checkout"] != "v5" {
		t.Errorf("explicit pin should win, got %q", got["actions/checkout"])
	}
	if got["actions/setup-go"] != "3" {
		t.Errorf("numeric recorded value is always a constraint, got %q", got["actions/setup-go"])
	}
	if got["some/action"] != "skip" {
		t.Errorf("skip recorded value is always a constraint, got %q", got["some/action"])
	}
	if _, ok := got["other/action"]; ok {
		t.Error("exact recorded value without a pin must not become a constraint")
	}
	if !cfg.IsManaged() {
		t.Error("Pins != nil must be managed")
	}
}

func TestRender_ManagedEmptyPins(t *testing.T) {
	got := Render(&Config{Pins: map[string]string{}})
	want := schemaHeader + "\n" + "\npins: {}\n"
	if got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestRender_ManagedPinsSorted(t *testing.T) {
	cfg := &Config{
		Pins:    map[string]string{"z/z": "v2", "a/a": "4"},
		Actions: map[string]string{"a/a": "v1.0.0"},
	}
	got := Render(cfg)
	want := schemaHeader + "\n" +
		"\npins:\n" +
		"  a/a: \"4\"\n" +
		"  z/z: v2\n" +
		"\nactions:\n" +
		"  a/a: v1.0.0\n"
	if got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestLoadPinsEmptyIsNonNil(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".actup.yaml")
	if err := os.WriteFile(path, []byte("pins: {}\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Pins == nil {
		t.Fatal("pins: {} must load as a non-nil empty map")
	}
	if len(cfg.Pins) != 0 {
		t.Errorf("expected empty pins, got %v", cfg.Pins)
	}
	if !cfg.IsManaged() {
		t.Error("pins: {} must be a managed manifest")
	}
}

func TestRender_SHA(t *testing.T) {
	major := 4
	cfg := &Config{Major: &major, SHA: true, Pins: map[string]string{}}
	got := Render(cfg)
	want := schemaHeader + "\n" +
		"\nmajor: 4\n" +
		"\nsha: true\n" +
		"\npins: {}\n"
	if got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}

	if got := Render(&Config{}); got != schemaHeader+"\n" {
		t.Errorf("SHA false must not render, got %q", got)
	}
}

func TestSaveLoadRoundtripSHA(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".actup.yaml")

	cfg := &Config{SHA: true, Actions: map[string]string{"actions/checkout": "v5"}}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if loaded == nil || !loaded.SHA {
		t.Fatalf("SHA = %v, want true", loaded)
	}
}

func TestLoadGroups(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".actup.yaml")
	data := "groups:\n  ci:\n    - actions/*\n    - docker/*\n  release:\n    - softprops/action-gh-release\n"
	if err := os.WriteFile(path, []byte(data), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.Groups["ci"]) != 2 || cfg.Groups["ci"][0] != "actions/*" || cfg.Groups["ci"][1] != "docker/*" {
		t.Errorf("ci group = %v, want [actions/* docker/*]", cfg.Groups["ci"])
	}
	if len(cfg.Groups["release"]) != 1 || cfg.Groups["release"][0] != "softprops/action-gh-release" {
		t.Errorf("release group = %v", cfg.Groups["release"])
	}
}

func TestRenderGroupsSorted(t *testing.T) {
	cfg := &Config{Groups: map[string][]string{
		"release": {"softprops/action-gh-release"},
		"ci":      {"docker/*", "actions/*"},
	}}

	got := Render(cfg)

	want := schemaHeader + "\n" +
		"\ngroups:\n" +
		"  ci:\n" +
		"    - \"actions/*\"\n" +
		"    - \"docker/*\"\n" +
		"  release:\n" +
		"    - softprops/action-gh-release\n"
	if got != want {
		t.Errorf("Render() = %q, want %q", got, want)
	}
}

func TestSaveLoadGroupsRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".actup.yaml")

	cfg := &Config{Groups: map[string][]string{
		"ci":      {"docker/*", "actions/*"},
		"release": {"softprops/action-gh-release"},
	}}
	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(loaded.Groups) != 2 {
		t.Fatalf("groups = %v, want 2 entries", loaded.Groups)
	}
	if len(loaded.Groups["ci"]) != 2 || loaded.Groups["ci"][0] != "actions/*" || loaded.Groups["ci"][1] != "docker/*" {
		t.Errorf("ci group = %v", loaded.Groups["ci"])
	}
	if len(loaded.Groups["release"]) != 1 || loaded.Groups["release"][0] != "softprops/action-gh-release" {
		t.Errorf("release group = %v", loaded.Groups["release"])
	}
}
