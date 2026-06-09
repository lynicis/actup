# Major-Version Pin (`--major`) & Config File (`.actup.yaml`) — Design & Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add (1) `--major` flag to pin all upgrades to a specific major version (e.g., `--major 4` → latest `v4.x.x`), and (2) `.actup.yaml` config file for per-action overrides and global defaults.

**Architecture:** Feature 1 introduces a `TagMode` struct through `internal/github` → `cmd/root.go` → `internal/tui` / `cmd/root.go runNoTUI` / `internal/checker`. Feature 2 introduces a new `internal/config` package loaded at startup in `cmd/root.go`, with precedence CLI > config > defaults.

**Tech Stack:** Go 1.26, Cobra, `goccy/go-yaml` (already in deps for B/C registry), no new dependencies.

---

## Feature 1 — `--major` Flag

### Semantics

```
actup --major 4        # upgrade all actions to latest v4.x.x
actup --major 4 -s     # same (--semver is irrelevant when --major is set)
actup -m 4             # short form
```

- `--major` takes an integer (`--major 4`), not `v4`.
- When set, `resolveLatestTag` filters tags to those matching `^v{M}\.\d+\.\d+$` and returns the highest.
- If no tag exists within the pinned major, the action is skipped with a warning.
- `--semver` is ignored when `--major` is set (major pinning is strictly more specific).
- Cache key in the GitHub client changes to include the major version: `"owner/repo/semver/major"`.

### Interface change to `internal/github`

Replace `semverMode bool` with a `TagMode` struct:

```go
type TagMode struct {
    Semver  bool
    Major   int   // 0 = no pin
}

func (c *Client) LatestTag(ctx context.Context, owner, repo string, mode TagMode) (string, error)
```

### Threading

- `cmd/root.go`: add `majorVer int` flag, pass to `runNoTUI`, `tui.Run`, `checker.New`.
- `internal/tui`: store `majorVer` in `model`, use in `loadActions`.
- `internal/checker`: add `majorVer` to `Checker` struct.
- Everywhere `LatestSemverTag` was used, replace with `LatestTag(ctx, owner, repo, TagMode{...})`.

---

## Feature 2 — Config File (`.actup.yaml`)

### Location

`$CWD/.actup.yaml` — project root only. Optional; if absent, everything works as before.

### Schema

```yaml
# Global default major version (overridden by CLI --major flag)
major: 4

# Per-action overrides / pins / exclusions
actions:
  actions/checkout: 4
  actions/setup-go: v5.3.0
  some-org/some-action: skip
```

- `4` (digit) → pin to that major version (latest `v4.x.x`)
- `v5.3.0` (full semver) → pin to exact version, no API call
- `skip` → exclude from upgrades entirely

### Precedence

CLI flags > config file > built-in defaults.

### New package: `internal/config`

```go
type Config struct {
    Major   *int              `yaml:"major,omitempty"`
    Actions map[string]string `yaml:"actions,omitempty"`
}

func Load(path string) (*Config, error)
func DefaultPath() string
```

- Returns `nil, nil` if file doesn't exist (optional config).
- Uses `goccy/go-yaml` (already a dependency).

---

## Implementation Tasks

### Task 1: Refactor `internal/github` — `TagMode` struct & major-pin resolution

**Files:**
- Modify: `internal/github/client.go`
- Modify: `internal/github/client_test.go`

**Step 1: Add `TagMode` struct**

```go
type TagMode struct {
    Semver bool
    Major  int // 0 = no pin; > 0 = pin to that major version
}

// cacheKey returns a unique cache key for this mode.
func (m TagMode) cacheKey(owner, repo string) string {
    return fmt.Sprintf("%s/%s/%v/%d", owner, repo, m.Semver, m.Major)
}
```

**Step 2: Update `LatestTag` signature and cache logic**

```go
func (c *Client) LatestTag(ctx context.Context, owner, repo string, mode TagMode) (string, error) {
    key := mode.cacheKey(owner, repo)
    if cached, ok := c.cache.Load(key); ok {
        return cached.(string), nil
    }

    tags, err := c.fetchAllSemverTags(ctx, owner, repo)
    if err != nil {
        return "", err
    }

    tagName := resolveLatestTag(tags, mode)
    if tagName == "" {
        return "", ErrNoSemverTags
    }

    c.cache.Store(key, tagName)
    return tagName, nil
}
```

**Step 3: Update `resolveLatestTag` for major pin**

```go
func resolveLatestTag(tags []string, mode TagMode) string {
    if len(tags) == 0 {
        return ""
    }

    sort.Slice(tags, func(i, j int) bool {
        iC, jC := tags[i], tags[j]
        if !strings.HasPrefix(iC, "v") { iC = "v" + iC }
        if !strings.HasPrefix(jC, "v") { jC = "v" + jC }
        return semver.Compare(iC, jC) > 0
    })

    if mode.Major > 0 {
        prefix := fmt.Sprintf("v%d.", mode.Major)
        for _, tag := range tags {
            if strings.HasPrefix(tag, prefix) {
                return tag
            }
        }
        // No tag found for this major — return empty, caller will skip
        return ""
    }

    if mode.Semver {
        return tags[0]
    }

    for _, tag := range tags {
        if majorTagRegex.MatchString(tag) {
            return tag
        }
    }
    return tags[0]
}
```

**Step 4: Update `LatestSemverTag` as convenience wrapper**

```go
func (c *Client) LatestSemverTag(ctx context.Context, owner, repo string) (string, error) {
    return c.LatestTag(ctx, owner, repo, TagMode{Semver: true})
}
```

**Step 5: Update tests**

Add to `client_test.go`:

```go
func TestResolveLatestTag_MajorPin(t *testing.T) {
    tags := []string{"v4.0.0", "v4.2.1", "v5.0.0", "v5.1.0"}
    got := resolveLatestTag(tags, TagMode{Major: 4})
    if got != "v4.2.1" {
        t.Errorf("major pin 4 = %q, want %q", got, "v4.2.1")
    }
}

func TestResolveLatestTag_MajorPinNoMatch(t *testing.T) {
    tags := []string{"v5.0.0", "v5.1.0"}
    got := resolveLatestTag(tags, TagMode{Major: 4})
    if got != "" {
        t.Errorf("major pin 4 = %q, want empty", got)
    }
}
```

**Step 6: Run tests**

Run: `go test -v -race ./internal/github`

**Step 7: Commit**

```bash
git add -A
git commit -m "feat(github): add TagMode with major-version pin support"
```

---

### Task 2: Add `--major` flag to CLI

**Files:**
- Modify: `cmd/root.go`
- Modify: `cmd/root_test.go`

**Step 1: Add flag variable and registration**

Add to `var` block:
```go
majorVer     int
```

Add to `init()`:
```go
rootCmd.Flags().IntVarP(&majorVer, "major", "m", 0, "Pin upgrades to a specific major version (e.g. 4 for latest v4.x.x)")
```

**Step 2: Thread `majorVer` through `run()`**

In `run()`, pass to subroutines:

```go
if noTUI {
    return runNoTUI(ctx, actions, githubToken, dryRun, semverMode, majorVer, force)
}
return tui.Run(ctx, actions, githubToken, dryRun, semverMode, majorVer)
```

**Step 3: Update `runNoTUI` signature**

```go
func runNoTUI(ctx context.Context, actions []parser.ActionRef, githubToken string, dryRun bool, semverMode bool, majorVer int, force bool) error {
```

Replace `LatestSemverTag` with:
```go
latest, err := ghClient.LatestTag(ctx, owner, repo, github.TagMode{Semver: semverMode, Major: majorVer})
```

**Step 4: Update `runCheck` call**

Pass `majorVer` to `checker.New(ghClient, semverMode, majorVer)`.

**Step 5: Run tests**

Run: `make test`

**Step 6: Commit**

```bash
git add -A
git commit -m "feat(cmd): add --major flag for major-version pinning"
```

---

### Task 3: Update `internal/tui` for `--major`

**Files:**
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/actions.go`

**Step 1: Update `model` struct and `Run` signature**

In `model.go`:
```go
type model struct {
    // ... existing fields
    majorVer    int
}

func Run(ctx context.Context, actions []parser.ActionRef, token string, dryRun bool, semverMode bool, majorVer int) error {
    m := model{
        // ...
        majorVer:  majorVer,
    }
}
```

**Step 2: Update `loadActions`**

In `actions.go`, replace `client.LatestSemverTag(...)` with:
```go
latest, err := client.LatestTag(context.Background(), owner, repo, github.TagMode{Semver: m.semverMode, Major: m.majorVer})
```

Also update the import to use `github.com/lynicis/actup/internal/github` (or ensure `github.TagMode` is accessible — if there's a naming conflict, import with alias).

**Step 3: Run tests**

Run: `go test -v -race ./internal/tui`

**Step 4: Commit**

```bash
git add -A
git commit -m "feat(tui): thread --major through TUI tag resolution"
```

---

### Task 4: Update `internal/checker` for `--major`

**Files:**
- Modify: `internal/checker/checker.go`
- Modify: `internal/checker/checker_test.go`

**Step 1: Update `Checker` struct and constructor**

```go
type Checker struct {
    client      githubClient
    semverMode  bool
    majorVer    int
}

func New(client githubClient, semverMode bool, majorVer int) *Checker {
    return &Checker{client: client, semverMode: semverMode, majorVer: majorVer}
}
```

**Step 2: Update `Run` method**

Replace `LatestTag(ctx, owner, repo, c.semverMode)` with:
```go
latest, err := c.client.LatestTag(ctx, owner, repo, github.TagMode{Semver: c.semverMode, Major: c.majorVer})
```

Wait — the checker uses a `githubClient` interface, not the concrete type. The interface needs updating:

```go
type githubClient interface {
    LatestTag(ctx context.Context, owner, repo string, mode github.TagMode) (string, error)
}
```

**Step 3: Run tests**

Run: `go test -v -race ./internal/checker`

**Step 4: Commit**

```bash
git add -A
git commit -m "feat(checker): support --major in check mode"
```

---

### Task 5: Create `internal/config` package

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`

**Step 1: Write `config.go`**

```go
package config

import (
    "os"

    "github.com/goccy/go-yaml"
)

type Config struct {
    Major   *int              `yaml:"major,omitempty"`
    Actions map[string]string `yaml:"actions,omitempty"`
}

func DefaultPath() string {
    return ".actup.yaml"
}

func Load(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        if os.IsNotExist(err) {
            return nil, nil // optional config
        }
        return nil, err
    }

    var cfg Config
    if err := yaml.Unmarshal(data, &cfg); err != nil {
        return nil, err
    }
    return &cfg, nil
}
```

**Step 2: Write `config_test.go`**

```go
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
    os.WriteFile(path, []byte("major: 4\n"), 0644)

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
    os.WriteFile(path, []byte("actions:\n  actions/checkout: \"4\"\n  actions/setup-go: skip\n"), 0644)

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
    os.WriteFile(path, []byte("::: invalid\n"), 0644)

    _, err := Load(path)
    if err == nil {
        t.Fatal("expected error for invalid YAML")
    }
}
```

**Step 3: Run tests**

Run: `go test -v -race ./internal/config`

**Step 4: Commit**

```bash
git add internal/config/
git commit -m "feat(config): add .actup.yaml config file support"
```

---

### Task 6: Wire config into `cmd/root.go`

**Files:**
- Modify: `cmd/root.go`

**Step 1: Load config and merge with CLI flags**

In `run()`, after flag parsing and before branching:

```go
cfg, _ := config.Load(config.DefaultPath())
if cfg != nil {
    if majorVer == 0 && cfg.Major != nil {
        majorVer = *cfg.Major
    }
}
```

**Step 2: Pass config to subroutines**

Pass `cfg` to `runNoTUI` and `tui.Run`. Add `cfg *config.Config` parameter to both.

**Step 3: Remove config-only test entries from runNoTUI/TUI**

In `runNoTUI`, before grouping:
```go
if cfg != nil {
    var filtered []parser.ActionRef
    for _, a := range actions {
        key := a.Owner + "/" + a.Repo
        if pin, ok := cfg.Actions[key]; ok && pin == "skip" {
            continue
        }
        filtered = append(filtered, a)
    }
    actions = filtered
}
```

**Step 4: Run tests**

Run: `make test`

**Step 5: Commit**

```bash
git add -A
git commit -m "feat(cmd): wire .actup.yaml config into CLI pipeline"
```

---

### Task 7: Per-action pins in `runNoTUI`

**Files:**
- Modify: `cmd/root.go`

**Step 1: Before resolving each group, check config for per-action pin**

In `runNoTUI`, inside the group loop:

```go
resolvedMajor := majorVer
exactVersion := ""
if cfg != nil {
    if pin, ok := cfg.Actions[key]; ok {
        if pin == "skip" {
            continue // already filtered, but safety check
        }
        if n, err := strconv.Atoi(pin); err == nil {
            resolvedMajor = n
        } else {
            exactVersion = pin // treat as exact version
        }
    }
}

var latest string
var err error
if exactVersion != "" {
    latest = exactVersion
} else {
    latest, err = ghClient.LatestTag(ctx, owner, repo, github.TagMode{Semver: semverMode, Major: resolvedMajor})
}
if err != nil {
    // ... existing error handling
}
if latest == "" {
    fmt.Fprintf(os.Stderr, "  ⏭ %s (no tags for major v%d)\n", key, resolvedMajor)
    skippedCount++
    continue
}
```

**Step 2: Run tests**

Run: `make test`

**Step 3: Commit**

```bash
git add -A
git commit -m "feat(cmd): respect per-action pins in non-interactive mode"
```

---

### Task 8: Per-action pins in TUI

**Files:**
- Modify: `internal/tui/actions.go`
- Modify: `internal/tui/model.go`

**Step 1: Add `cfg` to model**

In `model.go`:
```go
type model struct {
    // ...
    cfg         *config.Config
}
```

Update `Run` to accept `cfg *config.Config` and store it.

**Step 2: Respect per-action pins in `loadActions`**

In `actions.go`, add config checking logic similar to Task 7, before resolving latest tag for each group.

**Step 3: Filter skip actions from TUI list**

Before building `ActionItem`s, filter out any actions with `cfg.Actions[key] == "skip"`.

**Step 4: Run tests**

Run: `go test -v -race ./internal/tui`

**Step 5: Commit**

```bash
git add -A
git commit -m "feat(tui): respect per-action pins from config"
```

---

### Task 9: Update README

**Files:**
- Modify: `README.md`

**Step 1: Mark roadmap items as done**

```markdown
- [x] Add support for pinning to specific major versions (`--major`)
- [x] Config file support (`.actup.yaml`)
```

**Step 2: Add usage examples**

In the Usage section, add `--major` example:
```sh
# Pin all upgrades to major version 4
actup --major 4
```

Add config file section:
```sh
actup          # auto-discovers .actup.yaml in current directory
```

**Step 3: Commit**

```bash
git add -A
git commit -m "docs: update README roadmap with --major and .actup.yaml"
```

---

## Summary

| Task | Package | Description |
|------|---------|-------------|
| 1 | `internal/github` | `TagMode` struct, major-pin in `resolveLatestTag` |
| 2 | `cmd` | `--major` flag, thread through `run()`, `runNoTUI` |
| 3 | `internal/tui` | Thread `--major` through TUI |
| 4 | `internal/checker` | Thread `--major` through check mode |
| 5 | `internal/config` | New package for `.actup.yaml` |
| 6 | `cmd` | Wire config into CLI |
| 7 | `cmd` | Per-action pins in `runNoTUI` |
| 8 | `internal/tui` | Per-action pins in TUI |
| 9 | `README.md` | Documentation |
