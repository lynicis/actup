# Ponytail Codebase Simplification Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Simplify actup by eliminating over-engineered abstractions, speculative concurrency, redundant state, hand-rolled helpers, and the `golang.org/x/sync` dependency identified in the ponytail audit.

**Architecture:**
1. Consolidate `--check` logic into `cmd/root.go` using `github.ResolveVersion` directly, deleting the single-caller `internal/checker` package.
2. Replace hand-rolled hex checking and reflection sorting with Go standard library primitives (`encoding/hex`, `slices.SortFunc`).
3. Strip unnecessary concurrency (errgroup + mutex) from local file reading/writing in `internal/parser` and `internal/upgrader`.
4. Replace `errgroup` in `internal/tui/actions.go` with stdlib buffered channel semaphores, allowing `golang.org/x/sync` to be dropped from `go.mod`.
5. Remove redundant state (`selectedSet map[int]bool`) and custom SIGINT handling in `internal/tui`.

**Tech Stack:** Go 1.27, Bubble Tea, Lipgloss, Cobra, `goccy/go-yaml`, `go-github/v89`, standard library (`slices`, `encoding/hex`, `sync`).

**Spec:** Ponytail audit findings (`/ponytail:ponytail-audit`).

## Global Constraints

- Preserve identical CLI behavior and flags (`--path`, `--token`, `--dry-run`, `--no-tui`, `--semver`, `--major`, `--force`, `--check`, `install-hooks`).
- Preserve identical TUI keybindings, visuals, and breaking-change warning flows.
- All tests must pass: `go test -v -race ./...`.
- All linter checks must pass: `golangci-lint run ./...`.
- No new external dependencies. Eliminate `golang.org/x/sync`.

---

### Task 1: Simplify Standard Library Helpers & Prune Dead Scoping

**Files:**
- Modify: `internal/parser/parser.go:32,126-133`
- Modify: `internal/github/client.go:10,128-139`
- Modify: `internal/scanner/scanner.go:17-28`
- Modify: `internal/config/config.go:15-35`
- Modify: `internal/tui/actions.go:53`
- Test: `internal/parser/parser_test.go`
- Test: `internal/github/client_test.go`
- Test: `internal/config/config_test.go`
- Test: `internal/scanner/scanner_test.go`

**Interfaces:**
- Consumes: Go standard library `encoding/hex`, `slices`
- Produces: Cleaner `isHex` using `hex.DecodeString`, `slices.SortFunc` in `resolveLatestTag`, streamlined `LoadDefault()`, cleaned `DiscoverWorkflows`

- [x] **Step 1: Write test for `isHex` handling in `parser_test.go`**

Verify `parser_test.go` tests full 40-character hex commit hashes correctly:

```go
func TestParseActionRef_CommitSHASkipped(t *testing.T) {
	ref, err := parseActionRef("actions/checkout@dae09b1f80164e2d312981ce81045e7e1f4640d2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref != nil {
		t.Errorf("expected commit SHA ref to be skipped, got %+v", ref)
	}
}
```

- [x] **Step 2: Run test to verify it passes currently**

Run: `go test -v ./internal/parser`
Expected: PASS

- [x] **Step 3: Implement stdlib replacements and cleanups**

1. In `internal/parser/parser.go`:
Replace `isHex` with:
```go
import (
	"encoding/hex"
	...
)

func isHex(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
}
```
Remove redundant loop variable capture `file := file` at line 32.

2. In `internal/github/client.go`:
Replace `sort.Slice` in `resolveLatestTag` with `slices.SortFunc`:
```go
import (
	"slices"
	...
)

	slices.SortFunc(tags, func(a, b string) int {
		aC := a
		if !strings.HasPrefix(aC, "v") {
			aC = "v" + aC
		}
		bC := b
		if !strings.HasPrefix(bC, "v") {
			bC = "v" + bC
		}
		// descending order: highest semver first
		return semver.Compare(bC, aC)
	})
```

3. In `internal/tui/actions.go`:
Remove redundant loop variable capture `key := key` at line 53.

4. In `internal/scanner/scanner.go`:
Simplify default workflow path:
```go
	if len(paths) == 0 {
		paths = []string{defaultWorkflowDir}
	}
```
(`filepath.Abs(defaultWorkflowDir)` resolves relative to the current working directory automatically).

5. In `internal/config/config.go`:
Refactor `LoadDefault()` to call `Load(path)`:
```go
func LoadDefault() (*Config, error) {
	paths := []string{".actup.yaml", ".actup.yml"}
	for _, path := range paths {
		cfg, err := Load(path)
		if err != nil {
			return nil, err
		}
		if cfg != nil {
			return cfg, nil
		}
	}
	return nil, nil
}
```

- [x] **Step 4: Run tests across modified packages**

Run: `go test -v ./internal/parser ./internal/github ./internal/config ./internal/scanner`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add internal/parser/parser.go internal/parser/parser_test.go internal/github/client.go internal/tui/actions.go internal/scanner/scanner.go internal/config/config.go
git commit -m "refactor: use stdlib helpers and eliminate redundant code"
```

---

### Task 2: Simplify Token Resolution (`internal/token`)

**Files:**
- Modify: `internal/token/token.go`
- Modify: `internal/token/token_test.go`
- Modify: `cmd/root.go:64-67`

**Interfaces:**
- Consumes: `os.Getenv`, `exec.Command`
- Produces: Top-level `token.Resolve(flagValue string) string` function

- [x] **Step 1: Write test for top-level `token.Resolve`**

Update `internal/token/token_test.go` to test `Resolve` directly using package-level `execCommand`:

```go
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
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test -v ./internal/token`
Expected: FAIL (undefined `Resolve`, `execCommand`)

- [x] **Step 3: Implement minimal code in `internal/token/token.go` and `cmd/root.go`**

In `internal/token/token.go`:
```go
package token

import (
	"os"
	"os/exec"
	"strings"
)

var execCommand = exec.Command

func Resolve(flagValue string) string {
	if flagValue != "" {
		return flagValue
	}

	if envToken := os.Getenv("GITHUB_TOKEN"); envToken != "" {
		return envToken
	}

	cmd := execCommand("gh", "auth", "token")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}
```

In `cmd/root.go`:
Replace:
```go
	if githubToken == "" {
		resolver := token.NewResolver()
		githubToken = resolver.Resolve("")
	}
```
With:
```go
	if githubToken == "" {
		githubToken = token.Resolve("")
	}
```

- [x] **Step 4: Run token tests and full build**

Run: `go test -v ./internal/token` and `go test -v ./cmd`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add internal/token/token.go internal/token/token_test.go cmd/root.go
git commit -m "refactor(token): replace Resolver struct and factory with plain Resolve function"
```

---

### Task 3: Eliminate `internal/checker` Package & Deduplicate Skip Filtering

**Files:**
- Modify: `cmd/root.go:61-125,276-313`
- Modify: `cmd/root_test.go`
- Delete: `internal/checker/checker.go`
- Delete: `internal/checker/checker_test.go`

**Interfaces:**
- Consumes: `parser.GroupActions`, `github.ResolveVersion`
- Produces: Inlined `runCheck` in `cmd/root.go`, single skip-action filter in `run()`

- [x] **Step 1: Write integration test for `--check` in `cmd/root_test.go`**

Add tests for outdated detection logic in `cmd/root_test.go`:

```go
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
```

- [x] **Step 2: Run test to verify it fails**

Run: `go test -v ./cmd -run TestFilterSkippedActions`
Expected: FAIL (undefined `filterSkippedActions`)

- [x] **Step 3: Implement `filterSkippedActions` and inlined `runCheck`, delete `internal/checker`**

1. In `cmd/root.go`:
Remove `"github.com/lynicis/actup/internal/checker"` import.

Add helper:
```go
func filterSkippedActions(actions []parser.ActionRef, cfg *config.Config) []parser.ActionRef {
	if cfg == nil || len(cfg.Actions) == 0 {
		return actions
	}
	var filtered []parser.ActionRef
	for _, a := range actions {
		key := a.Owner + "/" + a.Repo
		if pin, ok := cfg.Actions[key]; ok && pin == "skip" {
			continue
		}
		filtered = append(filtered, a)
	}
	return filtered
}
```

In `run()` in `cmd/root.go`, apply `filterSkippedActions` once right after loading config:
```go
	actions = filterSkippedActions(actions, cfg)
	if len(actions) == 0 {
		fmt.Println("No actions to upgrade found.")
		return nil
	}
```
Remove redundant `if cfg != nil { ... }` filtering blocks from `runNoTUI`.

Refactor `runCheck`:
```go
type outdatedAction struct {
	parser.ActionRef
	Latest string
}

func runCheck(ctx context.Context, actions []parser.ActionRef, ghToken string, semverMode bool, majorVer int, cfg *config.Config) error {
	ghClient := github.NewClient(ghToken)
	grouped := parser.GroupActions(actions)

	var cfgActions map[string]string
	if cfg != nil {
		cfgActions = cfg.Actions
	}

	type checkResult struct {
		key    string
		latest string
		err    error
	}

	resultCh := make(chan checkResult, len(grouped))
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for key := range grouped {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			latest, err := github.ResolveVersion(ctx, ghClient, k, semverMode, majorVer, cfgActions)
			resultCh <- checkResult{key: k, latest: latest, err: err}
		}(key)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	results := make(map[string]string)
	for r := range resultCh {
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", r.err)
			os.Exit(2)
		}
		results[r.key] = r.latest
	}

	var outdated []outdatedAction
	for key, acts := range grouped {
		latest := results[key]
		for _, act := range acts {
			if act.Current != latest {
				outdated = append(outdated, outdatedAction{
					ActionRef: act,
					Latest:    latest,
				})
			}
		}
	}

	if len(outdated) == 0 {
		fmt.Println("All actions are up to date.")
		return nil
	}

	for _, oa := range outdated {
		fmt.Printf("%s:%d: %s/%s %s → %s\n", oa.File, oa.Line, oa.Owner, oa.Repo, oa.Current, oa.Latest)
	}
	fmt.Printf("\n%d action(s) can be upgraded.\n", len(outdated))
	os.Exit(1)
	return nil
}
```

2. Delete `internal/checker/checker.go` and `internal/checker/checker_test.go`.

- [x] **Step 4: Run tests to verify all pass**

Run: `go test -v ./...`
Expected: PASS (all packages compile and pass without `internal/checker`)

- [x] **Step 5: Commit**

```bash
git rm -r internal/checker
git add cmd/root.go cmd/root_test.go
git commit -m "refactor: inline check logic into cmd/root.go and eliminate internal/checker"
```

---

### Task 4: Remove Concurrency Boilerplate from Parser & Upgrader

**Files:**
- Modify: `internal/parser/parser.go:24-53`
- Modify: `internal/upgrader/upgrader.go:80-142`
- Test: `internal/parser/parser_test.go`
- Test: `internal/upgrader/upgrader_test.go`

**Interfaces:**
- Consumes: Standard sequential file reading and writing
- Produces: Non-concurrent `ExtractActions` and `ApplyAllUpgrades`, direct `fmt.Printf` in `showDryRunDiff`

- [x] **Step 1: Verify existing parser and upgrader tests pass**

Run: `go test -v ./internal/parser ./internal/upgrader`
Expected: PASS

- [x] **Step 2: Simplify `ExtractActions` in `internal/parser/parser.go`**

Remove `golang.org/x/sync/errgroup` and `sync.Mutex`. Replace `ExtractActions` with a simple sequential loop:

```go
func ExtractActions(_ context.Context, files []string) ([]ActionRef, error) {
	var actions []ActionRef

	for _, file := range files {
		fileActions, err := extractFromFile(file)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}
		actions = append(actions, fileActions...)
	}

	return actions, nil
}
```

- [x] **Step 3: Simplify `ApplyAllUpgrades` and `showDryRunDiff` in `internal/upgrader/upgrader.go`**

1. Replace `showDryRunDiff`:
```go
func showDryRunDiff(file string, action parser.ActionRef, newTag string) error {
	oldLine := fmt.Sprintf("uses: %s/%s@%s", action.Owner, action.Repo, action.Current)
	newLine := fmt.Sprintf("uses: %s/%s@%s", action.Owner, action.Repo, newTag)

	fmt.Printf("--- a/%s\n+++ b/%s\n@@ -%d,1 +%d,1 @@\n-%s\n+%s\n", file, file, action.Line, action.Line, oldLine, newLine)
	return nil
}
```

2. Replace `ApplyAllUpgrades`:
Remove `golang.org/x/sync/errgroup` and `sync.Mutex`. Use sequential file processing:
```go
func ApplyAllUpgrades(upgrades map[string]Upgrade, dryRun bool) (map[string][]Result, error) {
	results := make(map[string][]Result)

	byFile := make(map[string][]Upgrade)
	for _, u := range upgrades {
		byFile[u.Action.File] = append(byFile[u.Action.File], u)
	}

	for file, fileUpgrades := range byFile {
		var fileResults []Result

		for _, u := range fileUpgrades {
			result := Result{Action: u.Action, NewTag: u.NewTag}

			if dryRun {
				if err := showDryRunDiff(file, u.Action, u.NewTag); err != nil {
					result.Error = err
				} else {
					result.Updated = true
				}
			} else {
				if err := replaceInFile(file, u.Action, u.NewTag); err != nil {
					result.Error = err
				} else {
					result.Updated = true
				}
			}

			fileResults = append(fileResults, result)
		}

		results[file] = fileResults
	}

	return results, nil
}
```

- [x] **Step 4: Run tests to verify all pass**

Run: `go test -v ./internal/parser ./internal/upgrader`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add internal/parser/parser.go internal/upgrader/upgrader.go
git commit -m "refactor: simplify parser and upgrader to sequential execution"
```

---

### Task 5: Simplify TUI State & Signal Handling

**Files:**
- Modify: `internal/tui/model.go:26,36-37,52,57-60`
- Modify: `internal/tui/types.go:49`
- Modify: `internal/tui/update.go:16,34-65`
- Modify: `internal/tui/actions.go:77,108,123,131`
- Modify: `internal/tui/tui_test.go`
- Modify: `internal/tui/tui_render_test.go`

**Interfaces:**
- Consumes: Bubble Tea standard model loop
- Produces: Single source of truth for selection (`ActionItem.Selected`), default signal handling

- [x] **Step 1: Write test verifying selection toggling via `ActionItem.Selected`**

In `internal/tui/tui_test.go`, check that keypresses `'a'`, `'n'`, and `' '` update `item.Selected` properly without needing `selectedSet`:

```go
func TestUpdateChecklistTogglesSelection(t *testing.T) {
	m := model{
		state: stateChecklist,
		items: []ActionItem{
			{Owner: "actions", Repo: "checkout", Selected: true},
			{Owner: "actions", Repo: "setup-go", Selected: false},
		},
		cursor: 1,
	}

	// Toggle cursor item (index 1) with space
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	updated := newModel.(model)
	if !updated.items[1].Selected {
		t.Errorf("expected item 1 to be selected after space toggle")
	}

	// Select none with 'n'
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	updated = newModel.(model)
	if updated.items[0].Selected || updated.items[1].Selected {
		t.Errorf("expected all items to be unselected after 'n'")
	}
}
```

- [x] **Step 2: Run test to verify it compiles/runs**

Run: `go test -v ./internal/tui -run TestUpdateChecklistTogglesSelection`
Expected: PASS

- [x] **Step 3: Remove `selectedSet` and custom SIGINT handling**

1. In `internal/tui/model.go`:
- Remove `selectedSet map[int]bool` from `model` struct.
- In `Run`: remove `sigChan := make(chan os.Signal, 1)`, `signal.Notify`, and the goroutine calling `p.Quit()`. Bubble Tea already catches SIGINT by default.
- Remove `selectedSet: make(map[int]bool)` from `model` initialization.

2. In `internal/tui/types.go`:
- In `actionsLoadedMsg`, remove `selectedSet map[int]bool`.

3. In `internal/tui/actions.go`:
- In `loadActions`: remove `selectedSet := make(map[int]bool)`, remove `selectedSet[itemIndex] = true`, return `actionsLoadedMsg{items: items}`.
- In `applyUpgrades`: replace `if !m.selectedSet[i]` with `if !item.Selected`.

4. In `internal/tui/update.go`:
- In `updateLoading`: remove `m.selectedSet = msg.selectedSet`.
- In `updateChecklist`:
  - `case "a"`: set `item.Selected = true` directly.
  - `case "n"`: set `item.Selected = false` directly.
  - `case " "`: toggle `item.Selected = !item.Selected` directly.

5. In `internal/tui/tui_test.go` and `internal/tui/tui_render_test.go`:
- Remove `selectedSet` field assignments from test fixtures.

- [x] **Step 4: Run TUI tests**

Run: `go test -v ./internal/tui`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add internal/tui/
git commit -m "refactor(tui): eliminate redundant selectedSet and custom signal handler"
```

---

### Task 6: Prune `golang.org/x/sync` Dependency

**Files:**
- Modify: `internal/tui/actions.go:9,45-69`
- Modify: `go.mod`
- Modify: `go.sum`

**Interfaces:**
- Consumes: Go standard library `sync.WaitGroup` and buffered channel semaphore
- Produces: Zero usages of `golang.org/x/sync`, clean `go.mod`

- [x] **Step 1: Replace `errgroup` in `internal/tui/actions.go` with stdlib semaphore**

In `internal/tui/actions.go`:
Remove `"golang.org/x/sync/errgroup"`.
Replace lines 45-69 with standard library worker concurrency (matching `cmd/root.go`):

```go
	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for key := range grouped {
		if skipKeys[key] {
			continue
		}

		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var cfgActions map[string]string
			if m.cfg != nil {
				cfgActions = m.cfg.Actions
			}
			latest, err := github.ResolveVersion(context.Background(), client, k, m.semverMode, m.majorVer, cfgActions)
			resultCh <- fetchResult{k, latest, err}
		}(key)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()
```

- [x] **Step 2: Run tests in `internal/tui` to verify**

Run: `go test -v ./internal/tui`
Expected: PASS

- [x] **Step 3: Run `go mod tidy` and verify dependency elimination**

Run:
```bash
go mod tidy
```
Check `go.mod` to ensure `golang.org/x/sync` is no longer a required direct dependency.

- [x] **Step 4: Run all tests to verify clean compilation**

Run: `go test -v ./...`
Expected: PASS

- [x] **Step 5: Commit**

```bash
git add internal/tui/actions.go go.mod go.sum
git commit -m "chore: drop golang.org/x/sync dependency in favor of stdlib semaphore"
```

---

### Task 7: Full Verification Pass & Linter Proof

**Files:**
- Read-only check: entire repository

- [x] **Step 1: Run full test suite with race detector**

Run: `make test` (`go test -v -race ./...`)
Expected: PASS across all packages without race conditions.

- [x] **Step 2: Run linter**

Run: `make lint` (`golangci-lint run ./...`)
Expected: 0 errors/warnings.

- [x] **Step 3: Verify binary builds cleanly**

Run: `make build` (`go build -o actup .`)
Run: `./actup --help`
Expected: Binary builds and prints help text with all expected flags.

- [x] **Step 4: Verify `--check` flag works with the binary**

Run: `./actup --check`
Expected: Exits 0 or 1 with clear summary output, no panics.
Clean up binary: `rm -f actup`

- [x] **Step 5: Update graphify graph and commit any remaining adjustments**

Run: `graphify update .` (ast-only update per repo rules)
Run: `git status`
Expected: Working tree clean.
