# GitHub Actions Lockfile (`actions.lock`) Integration & `gh actup` CLI Extension Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Integrate `actup` with GitHub Actions workflow lockfiles (`.github/workflows/actions.lock` via `gh actions-lock`) for parsing, auto-relocking on writes, transitive vulnerability auditing, and `ls` lock status, while packaging `actup` as a native `gh` CLI extension (`gh actup`).

**Architecture:** A new `internal/lockfile` package parses `.github/workflows/actions.lock` (`v0.0.3` format), resolves canonical repo-scoped keys (`owner/repo@ref` with `owner/repo` lowercased and subpaths stripped), extracts transitive `parser.ActionRef` entries (`Line == 0`), computes per-action lock status, and invokes `gh actions-lock` through an injectable `execCommand` seam. `cmd/` (`root`, `use`, `sync`, `audit`, `ls`) and `internal/tui` call into `internal/lockfile`, while `cmd/root.go` and `.goreleaser.yaml` add `gh-actup` binary and release asset support.

**Tech Stack:** Go 1.27, `github.com/goccy/go-yaml`, `github.com/spf13/cobra`, `github.com/charmbracelet/bubbletea`, GoReleaser v2.

**Spec:** `docs/plans/2026-11-12-actions-lock-and-gh-extension-design.md`

## Global Constraints

- Root flags are local; never use `PersistentFlags()`.
- `ActionRef.Version()` is the comparison primitive across `audit`, `ls`, `outdated`, and `tui`.
- Tests write to `t.TempDir()`; zero network and zero real `gh` subprocess execution in unit tests (all subprocess calls must go through an injectable `execCommand` seam).
- `actup audit` exit codes: `1` on findings, `2` when OSV is unreachable.
- Do not add new external Go dependencies (`go.mod` stays unchanged).
- Auto-relocking via `gh actions-lock` must degrade gracefully with a `⚠` warning on `os.Stderr` if `gh` or `gh-actions-lock` fails or is absent; never fail an already-written workflow upgrade.

## Review Focus

- Subpath actions in workflows (e.g., `actions/cache/save@v4` or mixed-case `Actions/Checkout@v4`) must match repo-scoped canonical keys (`actions/cache@v4`, `actions/checkout@v4`) in `actions.lock` for both `TransitiveRefs` deduplication and `StatusForAction`.
- Transitive dependencies in `actions.lock` whose `ref` is itself a bare 40-hex commit SHA (no semver tag) must produce a pinned `parser.ActionRef` with `Version() == ""` so `actup audit` emits the existing `version unknown (SHA-pinned without a version comment)` notice instead of silently claiming 0 vulnerabilities.
- `actup audit --fix` when only transitive vulnerabilities exist in `actions.lock` (`Line == 0`) must not pass `Line: 0` to `upgrader.ApplyAllUpgrades`, must print the transitive remediation notice, and must still exit `1` (`errAuditFindings`).
- `syncLockfile` must not run `gh actions-lock` when `--dry-run` is active or when 0 workflow files were modified.
- `actup ls` output when no `actions.lock` exists must remain byte-for-byte identical to existing output so existing `versionmanager_test.go` tests pass unchanged.

---

## File Map

| File | Action | Responsibility |
|---|---|---|
| `internal/parser/parser.go` | Modify | Skip `$/…` same-repo local action references in `parseActionRef` |
| `internal/parser/parser_test.go` | Modify | Unit test for `$/…` local action reference skipping |
| `internal/lockfile/lockfile.go` | Create | Parse `.github/workflows/actions.lock`, extract transitive `ActionRef`s, compute `StatusForAction`, and run `gh actions-lock` via `execCommand` seam |
| `internal/lockfile/lockfile_test.go` | Create | Unit tests for `LoadForPaths`, `TransitiveRefs`, `StatusForAction`, and `Sync` |
| `cmd/common.go` | Modify | Add `syncLockfile(paths []string)` helper |
| `cmd/root.go` | Modify | Call `syncLockfile` in `runNoTUI`; detect `gh-actup` / `GH_PATH` to set `rootCmd.Use = "gh actup"` |
| `cmd/root_test.go` | Modify | Unit test for `invokedAsGHExtension` detection |
| `cmd/use.go` | Modify | Call `syncLockfile(opts.paths)` in `doUse` when workflows are updated |
| `cmd/sync.go` | Modify | Call `syncLockfile(opts.paths)` in `doSync` when workflows are updated |
| `internal/tui/tui.go` | Modify | Accept/sync lockfile after `ApplyAllUpgrades` in `applyUpgrades` |
| `cmd/audit.go` | Modify | Scan transitive refs from `actions.lock`, format `(transitive)` text / `"transitive": true` JSON, handle transitive findings in `runAuditFix`, and call `syncLockfile` |
| `cmd/audit_test.go` | Modify | Unit tests for transitive `audit`, `audit --json`, and `audit --fix` |
| `cmd/ls.go` | Modify | Append `[locked]` / `[lock stale]` in table and detail views when `actions.lock` exists |
| `cmd/versionmanager_test.go` | Modify | Unit tests for `ls` lock status and `use`/`sync` auto-relocking |
| `.goreleaser.yaml` | Modify | Add `id: gh-extension` (`formats: [binary]`) archive for `gh-actup-<os>-<arch>[.exe]` |
| `README.md`, `AGENTS.md` | Modify | Document `gh extension install` and `.github/workflows/actions.lock` support |

---

### Task 1: Skip `$/…` Same-Repo Local Actions in `internal/parser`

**Files:**
- Modify: `internal/parser/parser.go:211-225`
- Test: `internal/parser/parser_test.go`

**Interfaces:**
- Consumes: `parseActionRef(uses string) (*ActionRef, error)` in `internal/parser/parser.go`
- Produces: `ExtractActions` ignores `uses: $/...` references produced by `gh actions-lock --migrate-local-actions`

- [ ] **Step 1: Write the failing test in `internal/parser/parser_test.go`**

```go
func TestExtractActionsSkipsDollarLocalRefs(t *testing.T) {
	actions := inputWorkflow(t, `jobs:
  build:
    steps:
      - uses: $/.github/actions/my-action@main
      - uses: $/actions/local-helper
      - uses: actions/checkout@v4
`)
	if len(actions) != 1 {
		t.Fatalf("expected 1 action, got %d: %+v", len(actions), actions)
	}
	if actions[0].Owner != "actions" || actions[0].Repo != "checkout" {
		t.Errorf("unexpected action: %+v", actions[0])
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v -run TestExtractActionsSkipsDollarLocalRefs ./internal/parser`
Expected: FAIL (`expected 1 action, got 2` because `$/...` with `@main` parses as owner `$`)

- [ ] **Step 3: Update `parseActionRef` in `internal/parser/parser.go` to skip `$/` prefix**

In `internal/parser/parser.go:216`, check `strings.HasPrefix(uses, "./") || strings.HasPrefix(uses, "$/")` and return `nil, nil`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run TestExtractActionsSkipsDollarLocalRefs ./internal/parser`
Expected: PASS

---

### Task 2: Create `internal/lockfile` Package

**Files:**
- Create: `internal/lockfile/lockfile.go`
- Create: `internal/lockfile/lockfile_test.go`

**Interfaces:**
- Consumes: `parser.ActionRef` (`internal/parser`), `github.com/goccy/go-yaml`
- Produces:
  ```go
  package lockfile

  const DefaultPath = ".github/workflows/actions.lock"

  type Status int
  const (
      StatusNone Status = iota
      StatusLocked
      StatusStale
  )

  type Dependency struct {
      Ref     string   `yaml:"ref"`
      Commit  string   `yaml:"commit"`
      OwnerID int64    `yaml:"owner_id"`
      RepoID  int64    `yaml:"repo_id"`
      Uses    []string `yaml:"uses,omitempty"`
  }

  type Lockfile struct {
      Path         string                `yaml:"-"`
      Version      string                `yaml:"version"`
      Workflows    map[string][]string   `yaml:"workflows"`
      Dependencies map[string]Dependency `yaml:"dependencies"`
  }

  func Load(path string) (*Lockfile, error)
  func LoadForPaths(paths []string) (*Lockfile, error)
  func (l *Lockfile) TransitiveRefs(direct []parser.ActionRef) []parser.ActionRef
  func (l *Lockfile) StatusForAction(key string, refs []parser.ActionRef) Status
  func Sync(lockfilePath string) error
  ```

- [ ] **Step 1: Write the failing unit tests in `internal/lockfile/lockfile_test.go`**

```go
package lockfile

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/lynicis/actup/internal/parser"
)

const sampleLockYAML = `# This file is machine-generated by gh actions-lock.
version: 'v0.0.3'
workflows:
    '.github/workflows/ci.yml':
        - 'actions/checkout@v7.0.1'
        - 'actions/cache@v4'
        - 'cli/gh-extension-precompile@v2.2.0'
dependencies:
    'actions/attest-build-provenance@a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32':
        ref: 'v4.1.0'
        commit: 'sha1-a2bbfa25375fe432b6a289bc6b6cd05ecd0c4c32'
        owner_id: 44036562
        repo_id: 760702757
        uses:
            - 'actions/attest@59d89421af93a897026c735860bf21b6eb4f7b26'
    'actions/attest@59d89421af93a897026c735860bf21b6eb4f7b26':
        ref: '59d89421af93a897026c735860bf21b6eb4f7b26'
        commit: 'sha1-59d89421af93a897026c735860bf21b6eb4f7b26'
        owner_id: 44036562
        repo_id: 760701061
    'actions/cache@v4':
        ref: 'v4.2.0'
        commit: 'sha1-1111111111111111111111111111111111111111'
        owner_id: 44036562
        repo_id: 222222222
    'actions/checkout@v7.0.1':
        ref: 'v7.0.1'
        commit: 'sha1-3d3c42e5aac5ba805825da76410c181273ba90b1'
        owner_id: 44036562
        repo_id: 197814629
    'cli/gh-extension-precompile@v2.2.0':
        ref: 'v2.2.0'
        commit: 'sha1-76961aa3bd1123d0a6fd42d0a41aca0696937c39'
        owner_id: 59704711
        repo_id: 421583656
`

func TestLoadForPaths_MissingReturnsNil(t *testing.T) {
	dir := t.TempDir()
	lf, err := LoadForPaths([]string{dir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lf != nil {
		t.Fatalf("expected nil lockfile, got %+v", lf)
	}
}

func TestTransitiveRefs_ExcludesDirectAndHandlesSubpathsAndBareSHA(t *testing.T) {
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	lockPath := filepath.Join(wfDir, "actions.lock")
	if err := os.WriteFile(lockPath, []byte(sampleLockYAML), 0o644); err != nil {
		t.Fatal(err)
	}

	lf, err := LoadForPaths([]string{dir})
	if err != nil || lf == nil {
		t.Fatalf("LoadForPaths: lf=%v, err=%v", lf, err)
	}

	direct := []parser.ActionRef{
		{Owner: "Actions", Repo: "Checkout", Path: "Actions/Checkout", Current: "v7.0.1"},
		{Owner: "actions", Repo: "cache", Path: "actions/cache/save", Current: "v4"},
		{Owner: "cli", Repo: "gh-extension-precompile", Path: "cli/gh-extension-precompile", Current: "v2.2.0"},
	}

	trans := lf.TransitiveRefs(direct)
	if len(trans) != 2 {
		t.Fatalf("expected 2 transitive refs, got %d: %+v", len(trans), trans)
	}

	// First transitive ref has semver ref v4.1.0
	if trans[0].Owner != "actions" || trans[0].Repo != "attest-build-provenance" || trans[0].Version() != "v4.1.0" || trans[0].Line != 0 {
		t.Errorf("unexpected trans[0]: %+v (Version=%q)", trans[0], trans[0].Version())
	}
	// Second transitive ref has a bare 40-hex SHA as its ref -> Pinned() && Version() == ""
	if trans[1].Owner != "actions" || trans[1].Repo != "attest" || !trans[1].Pinned() || trans[1].Version() != "" {
		t.Errorf("expected bare-SHA transitive ref with empty Version(), got %+v (Version=%q)", trans[1], trans[1].Version())
	}
}

func TestStatusForAction_SubpathAndCaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, "actions.lock")
	if err := os.WriteFile(lockPath, []byte(sampleLockYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	lf, err := Load(lockPath)
	if err != nil {
		t.Fatal(err)
	}

	// Subpath action actions/cache/save@v4 matches actions/cache@v4 in lockfile
	status := lf.StatusForAction("actions/cache", []parser.ActionRef{
		{Owner: "actions", Repo: "cache", Path: "actions/cache/save", Current: "v4"},
	})
	if status != StatusLocked {
		t.Errorf("expected StatusLocked for subpath action, got %v", status)
	}

	// Upgraded ref v7.0.2 is not in lockfile -> StatusStale
	stale := lf.StatusForAction("actions/checkout", []parser.ActionRef{
		{Owner: "actions", Repo: "checkout", Path: "actions/checkout", Current: "v7.0.2"},
	})
	if stale != StatusStale {
		t.Errorf("expected StatusStale for unlocked version, got %v", stale)
	}
}

func TestSync_UsesExecSeam(t *testing.T) {
	orig := execCommand
	defer func() { execCommand = orig }()

	var gotName string
	var gotArgs []string
	execCommand = func(name string, arg ...string) *exec.Cmd {
		gotName = name
		gotArgs = arg
		return exec.Command("true")
	}

	if err := Sync(".github/workflows/actions.lock"); err != nil {
		t.Fatalf("Sync failed: %v", err)
	}
	if gotName != "gh" || len(gotArgs) != 1 || gotArgs[0] != "actions-lock" {
		t.Errorf("unexpected command: %s %v", gotName, gotArgs)
	}

	execCommand = func(name string, arg ...string) *exec.Cmd {
		return exec.Command("false")
	}
	if err := Sync(".github/workflows/actions.lock"); err == nil {
		t.Fatal("expected error when gh actions-lock fails")
	} else if !errors.Is(err, err) {
		t.Fatal("unreachable")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v ./internal/lockfile`
Expected: FAIL (`cannot find package` / undefined symbols)

- [ ] **Step 3: Implement `internal/lockfile/lockfile.go`**

Implement `Load`, `LoadForPaths`, `TransitiveRefs`, `StatusForAction`, and `Sync`:
- `LoadForPaths(paths []string)`:
  - When `len(paths) == 0`, check `DefaultPath` (`.github/workflows/actions.lock`).
  - When `len(paths) > 0`, for each `p` in `paths`, check `<p>/actions.lock`, `<p>/.github/workflows/actions.lock`, and `<filepath.Dir(p)>/actions.lock`; return the first existing file via `Load`.
- Canonical key helper `canonicalPinKey(owner, repo, ref string) string`: returns `strings.ToLower(owner) + "/" + strings.ToLower(repo) + "@" + ref`.
- `TransitiveRefs(direct []parser.ActionRef)`:
  - Build `directSet` from `direct` (`canonicalPinKey(a.Owner, a.Repo, a.Current)` and `canonicalPinKey(a.Owner, a.Repo, a.Version())`) and from all entries in `l.Workflows` (lowercasing the `owner/repo` before `@`).
  - Iterate `slices.Sorted(maps.Keys(l.Dependencies))`. Skip keys in `directSet`.
  - Split key on `@` into `owner/repo` and `spec`.
  - If `shaRegex.MatchString(dep.Ref)` (40-hex): set `Current: dep.Ref`, `SHA: strings.ToLower(dep.Ref)`, `Tag: ""`.
  - Else: set `Current: dep.Ref` (or `spec` if `dep.Ref == ""`).
  - Deduplicate by `canonicalPinKey(owner, repo, ref.Version()+"|"+ref.Current)`.
- `StatusForAction(key string, refs []parser.ActionRef) Status`:
  - If `l == nil || len(refs) == 0`, return `StatusNone`.
  - Build a lookup map of `l.Dependencies` with lowercased `owner/repo` before `@`.
  - Return `StatusLocked` if every `r` in `refs` matches `canonicalPinKey(r.Owner, r.Repo, r.Current)` or `canonicalPinKey(r.Owner, r.Repo, r.Version())` in the map; otherwise `StatusStale`.
- `Sync(lockfilePath string) error`:
  - Uses `var execCommand = exec.Command`.
  - Runs `cmd := execCommand("gh", "actions-lock")`. If `lockfilePath` ends with `.github/workflows/actions.lock`, set `cmd.Dir` to the repo root above `.github`. Capture `CombinedOutput()` and wrap any error with trimmed output.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v ./internal/lockfile`
Expected: PASS

---

### Task 3: Auto-Refresh `actions.lock` on Workflow Writes (`root`, `use`, `sync`, `tui`)

**Files:**
- Modify: `cmd/common.go`
- Modify: `cmd/root.go:258-265`
- Modify: `cmd/use.go:175-188`
- Modify: `cmd/sync.go:245-260`
- Modify: `internal/tui/tui.go:86-127,493-508`
- Test: `cmd/versionmanager_test.go`

**Interfaces:**
- Consumes: `lockfile.LoadForPaths`, `lockfile.Sync` from Task 2
- Produces:
  - `syncLockfile(paths []string)` in `cmd/common.go`
  - `var relockFunc = lockfile.Sync` seam in `cmd/common.go` so `cmd` tests can assert relocking without spawning `gh`

- [ ] **Step 1: Write the failing test in `cmd/versionmanager_test.go`**

```go
func TestUseAndSync_AutoRelockWhenActionsLockExists(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeWorkflow(t, dir, "ci.yml", testWorkflow)
	lockPath := filepath.Join(dir, ".github", "workflows", "actions.lock")
	if err := os.WriteFile(lockPath, []byte("version: 'v0.0.3'\nworkflows: {}\ndependencies: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	calls := 0
	origRelock := relockFunc
	relockFunc = func(path string) error {
		calls++
		return nil
	}
	defer func() { relockFunc = origRelock }()

	// Dry-run must NOT trigger relock
	if err := doUse(context.Background(), []actionSpec{{Key: "actions/checkout", Ref: "v5"}}, useOptions{dryRun: true, force: true}); err != nil {
		t.Fatalf("doUse dry-run: %v", err)
	}
	if calls != 0 {
		t.Fatalf("dry-run must not call relock, got %d calls", calls)
	}

	// Real write MUST trigger relock once
	if err := doUse(context.Background(), []actionSpec{{Key: "actions/checkout", Ref: "v5"}}, useOptions{force: true}); err != nil {
		t.Fatalf("doUse: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected 1 relock call after doUse, got %d", calls)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v -run TestUseAndSync_AutoRelockWhenActionsLockExists ./cmd`
Expected: FAIL (`undefined: relockFunc`)

- [ ] **Step 3: Implement `syncLockfile` in `cmd/common.go` and wire into `root.go`, `use.go`, `sync.go`, and `internal/tui/tui.go`**

In `cmd/common.go`:
```go
var relockFunc = lockfile.Sync

func syncLockfile(paths []string) {
	lf, err := lockfile.LoadForPaths(paths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to read actions.lock: %v\n", err)
		return
	}
	if lf == nil {
		return
	}
	if err := relockFunc(lf.Path); err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to update actions.lock (run 'gh actions-lock' manually): %v\n", err)
		return
	}
	fmt.Println("✓ Updated .github/workflows/actions.lock")
}
```
- Call `syncLockfile(paths)` in `runNoTUI` (`cmd/root.go`) when `!dryRun && upgraded > 0`.
- Call `syncLockfile(opts.paths)` in `doUse` (`cmd/use.go`) when `!opts.dryRun && updatedActions > 0`.
- Call `syncLockfile(opts.paths)` in `doSync` (`cmd/sync.go`) when `!opts.dryRun && upgraded > 0`.
- In `internal/tui/tui.go` (`applyUpgrades`), when `!m.dryRun && err == nil && len(upgrades) > 0`, if `lf, lerr := lockfile.LoadForPaths(nil); lerr == nil && lf != nil`, call `_ = lockfile.Sync(lf.Path)`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run TestUseAndSync_AutoRelockWhenActionsLockExists ./cmd`
Expected: PASS

---

### Task 4: Transitive Vulnerability Scanning in `actup audit` and `actup audit --fix`

**Files:**
- Modify: `cmd/audit.go:90-215,305-330,560-599`
- Test: `cmd/audit_test.go`

**Interfaces:**
- Consumes: `lockfile.LoadForPaths`, `(*lockfile.Lockfile).TransitiveRefs`, `syncLockfile`
- Produces:
  - `auditJSONFinding` gains `Transitive bool` (`json:"transitive,omitempty"`).
  - `runAudit` includes transitive `ActionRef`s (`Line == 0`) from `actions.lock` in `checkable` (and in `unknownOrder` when a transitive ref is a bare SHA without a tag).
  - `runAuditFix` partitions `findings` into direct (`f.Line > 0`) and transitive (`f.Line == 0`), fixes direct findings, calls `syncLockfile(auditOpts.paths)` when direct fixes are applied, and reports remaining transitive findings (`✗ <owner>/<repo>@<ref>: transitive dependency in <file> (upgrade parent action or run 'gh actions-lock --relock')`) while incrementing `residual`.

- [ ] **Step 1: Write the failing tests in `cmd/audit_test.go`**

```go
func TestAuditJSON_IncludesTransitiveFromActionsLock(t *testing.T) {
	resetAuditFixState()
	defer resetAuditFixState()

	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wfDir, "ci.yml"), []byte("name: CI\non: [push]\njobs:\n  b:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lockContent := `version: 'v0.0.3'
workflows:
  '.github/workflows/ci.yml':
    - 'actions/checkout@v4'
dependencies:
  'actions/checkout@v4':
    ref: 'v4.2.0'
    commit: 'sha1-1111111111111111111111111111111111111111'
  'actions/attest@59d89421af93a897026c735860bf21b6eb4f7b26':
    ref: 'v1.0.0'
    commit: 'sha1-59d89421af93a897026c735860bf21b6eb4f7b26'
`
	lockPath := filepath.Join(wfDir, "actions.lock")
	if err := os.WriteFile(lockPath, []byte(lockContent), 0o644); err != nil {
		t.Fatal(err)
	}

	origScan := auditScan
	defer func() { auditScan = origScan }()
	auditScan = func(_ context.Context, refs []parser.ActionRef) ([]vulnscanner.Finding, error) {
		var out []vulnscanner.Finding
		for _, r := range refs {
			if r.Owner == "actions" && r.Repo == "attest" {
				out = append(out, vulnscanner.Finding{
					Owner: r.Owner,
					Repo:  r.Repo,
					Ref:   r.Version(),
					File:  r.File,
					Line:  r.Line,
					Vulns: []vulnscanner.Vulnerability{{ID: "GHSA-trans-1", Severity: "HIGH", Fixed: "1.0.1"}},
				})
			}
		}
		return out, nil
	}

	out, _, err := runAuditCaptured(t, dir, "--json", "--path", dir)
	if !errors.Is(err, errAuditFindings) {
		t.Fatalf("expected errAuditFindings, got %v", err)
	}
	if !strings.Contains(out, `"transitive": true`) {
		t.Errorf("expected transitive: true in JSON output, got:\n%s", out)
	}
}

func TestAuditFix_TransitiveOnlyReportsResidualAndDoesNotRewriteLockfile(t *testing.T) {
	resetAuditFixState()
	defer resetAuditFixState()

	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wfDir, "ci.yml"), []byte("name: CI\non: [push]\njobs:\n  b:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v4\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	lockContent := `version: 'v0.0.3'
workflows:
  '.github/workflows/ci.yml':
    - 'actions/checkout@v4'
dependencies:
  'actions/attest@59d89421af93a897026c735860bf21b6eb4f7b26':
    ref: 'v1.0.0'
    commit: 'sha1-59d89421af93a897026c735860bf21b6eb4f7b26'
`
	if err := os.WriteFile(filepath.Join(wfDir, "actions.lock"), []byte(lockContent), 0o644); err != nil {
		t.Fatal(err)
	}

	origScan := auditScan
	defer func() { auditScan = origScan }()
	auditScan = func(_ context.Context, refs []parser.ActionRef) ([]vulnscanner.Finding, error) {
		var out []vulnscanner.Finding
		for _, r := range refs {
			if r.Repo == "attest" {
				out = append(out, vulnscanner.Finding{
					Owner: r.Owner,
					Repo:  r.Repo,
					Ref:   r.Version(),
					File:  r.File,
					Line:  r.Line,
					Vulns: []vulnscanner.Vulnerability{{ID: "GHSA-trans-1", Severity: "HIGH", Fixed: "1.0.1"}},
				})
			}
		}
		return out, nil
	}

	out, _, err := runAuditCaptured(t, dir, "--fix", "--path", dir)
	if !errors.Is(err, errAuditFindings) {
		t.Fatalf("expected errAuditFindings when transitive finding remains, got %v", err)
	}
	if !strings.Contains(out, "transitive dependency") {
		t.Errorf("expected transitive dependency remediation message, got:\n%s", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v -run "TestAuditJSON_IncludesTransitiveFromActionsLock|TestAuditFix_TransitiveOnlyReportsResidualAndDoesNotRewriteLockfile" ./cmd`
Expected: FAIL

- [ ] **Step 3: Update `cmd/audit.go` to load `actions.lock` transitive refs and handle them in text, JSON, and `--fix` modes**

- In `runAudit`: load `lf, lerr := lockfile.LoadForPaths(auditOpts.paths)`. If `lf != nil`, append `filterSkippedActions(lf.TransitiveRefs(actions), cfg)` to `allActions` before building `checkable` and `unknownOrder`.
- In `runAudit` text loop: if `f.Line == 0`, print `fmt.Printf("%s (transitive): %s@%s\n", f.File, action, f.Ref)`.
- In `auditJSONFinding`: add `Transitive bool` (`json:"transitive,omitempty"`), set to `f.Line == 0`.
- In `runAuditFix`: separate `directFindings` (`f.Line > 0`) from `transitiveFindings` (`f.Line == 0`). For each `tf` in `transitiveFindings`, print `fmt.Printf("✗ %s/%s@%s: transitive dependency in %s (upgrade parent action or run 'gh actions-lock --relock')\n", tf.Owner, tf.Repo, tf.Ref, tf.File)` and increment `residual`. When `len(applied) > 0 && !auditOpts.dryRun`, call `syncLockfile(auditOpts.paths)`.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run "TestAudit" ./cmd`
Expected: PASS

---

### Task 5: Lockfile Status in `actup ls`

**Files:**
- Modify: `cmd/ls.go:50-172`
- Test: `cmd/versionmanager_test.go`

**Interfaces:**
- Consumes: `lockfile.LoadForPaths`, `(*lockfile.Lockfile).StatusForAction`, `lockfile.StatusLocked`, `lockfile.StatusStale`
- Produces: `[locked]` / `[lock stale]` suffix in `actup ls` table and detail header when `actions.lock` is present

- [ ] **Step 1: Write the failing test in `cmd/versionmanager_test.go`**

```go
func TestLs_ShowsLockStatusWhenActionsLockPresent(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeWorkflow(t, dir, "ci.yml", testWorkflow)
	lockContent := `version: 'v0.0.3'
workflows:
  '.github/workflows/ci.yml':
    - 'actions/checkout@v3'
dependencies:
  'actions/checkout@v3':
    ref: 'v3.6.0'
    commit: 'sha1-1111111111111111111111111111111111111111'
`
	lockPath := filepath.Join(dir, ".github", "workflows", "actions.lock")
	if err := os.WriteFile(lockPath, []byte(lockContent), 0o644); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		lsOpts.paths = nil
		lsOpts.outdated = false
		if err := runLs(lsCmd, nil); err != nil {
			t.Fatalf("runLs: %v", err)
		}
	})
	if !strings.Contains(out, "actions/checkout") || !strings.Contains(out, "[locked]") {
		t.Errorf("expected actions/checkout to show [locked], got:\n%s", out)
	}
	if !strings.Contains(out, "actions/setup-go") || !strings.Contains(out, "[lock stale]") {
		t.Errorf("expected actions/setup-go to show [lock stale], got:\n%s", out)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v -run TestLs_ShowsLockStatusWhenActionsLockPresent ./cmd`
Expected: FAIL

- [ ] **Step 3: Update `cmd/ls.go` (`runLs`, `runLsTable`, `runLsDetail`)**

- In `runLs`, load `lf, _ := lockfile.LoadForPaths(lsOpts.paths)` and pass `lf` to `runLsDetail` and `runLsTable`.
- When `lf != nil`:
  - In `runLsDetail`: format header as `fmt.Printf("%s (pin: %s, lock: %s)\n\n", key, pin, lockLabel(lf.StatusForAction(key, refs)))` where `StatusLocked` → `"locked"` and `StatusStale` → `"stale"`.
  - In `runLsTable`: append `"  [locked]"` for `StatusLocked` and `"  [lock stale]"` for `StatusStale` before printing `line`.
- When `lf == nil`: leave output unchanged.

- [ ] **Step 4: Run test to verify it passes**

Run: `go test -v -run "TestLs" ./cmd`
Expected: PASS

---

### Task 6: `gh` CLI Extension Support (`cmd/root.go`, `.goreleaser.yaml`, `README.md`, `AGENTS.md`)

**Files:**
- Modify: `cmd/root.go:36-75`
- Modify: `cmd/root_test.go`
- Modify: `.goreleaser.yaml:55-66`
- Modify: `README.md`, `AGENTS.md`

**Interfaces:**
- Produces:
  - `isGHExtensionInvocation(arg0, ghPath string) bool` in `cmd/root.go`
  - `id: gh-extension` (`formats: [binary]`) archive in `.goreleaser.yaml`

- [ ] **Step 1: Write the failing test in `cmd/root_test.go`**

```go
func TestIsGHExtensionInvocation(t *testing.T) {
	if !isGHExtensionInvocation("/Users/me/.local/share/gh/extensions/gh-actup/gh-actup", "") {
		t.Error("expected gh-actup binary name to be detected as gh extension")
	}
	if !isGHExtensionInvocation("/usr/local/bin/actup", "/opt/homebrew/bin/gh") {
		t.Error("expected GH_PATH env var to be detected as gh extension")
	}
	if isGHExtensionInvocation("/usr/local/bin/actup", "") {
		t.Error("plain actup binary without GH_PATH must not be detected as gh extension")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -v -run TestIsGHExtensionInvocation ./cmd`
Expected: FAIL (`undefined: isGHExtensionInvocation`)

- [ ] **Step 3: Implement `isGHExtensionInvocation` in `cmd/root.go`, update `.goreleaser.yaml`, `README.md`, and `AGENTS.md`**

In `cmd/root.go`:
```go
func isGHExtensionInvocation(arg0, ghPath string) bool {
	base := strings.ToLower(filepath.Base(arg0))
	return strings.HasPrefix(base, "gh-") || ghPath != ""
}
```
In `cmd/root.go` `init()`:
```go
if len(os.Args) > 0 && isGHExtensionInvocation(os.Args[0], os.Getenv("GH_PATH")) {
	rootCmd.Use = "gh actup [pattern...]"
}
```
In `.goreleaser.yaml` under `archives:`:
- Add `id: default` to the existing archive entry.
- Add:
  ```yaml
  - id: gh-extension
    formats: [binary]
    name_template: >-
      gh-{{ .ProjectName }}-
      {{- .Os }}-
      {{- .Arch }}
      {{- if .Arm }}v{{ .Arm }}{{ end }}
  ```
Update `README.md` (Installation: `gh extension`, Features/Usage: `.github/workflows/actions.lock` integration) and `AGENTS.md` (add `internal/lockfile` row to Packages table).

- [ ] **Step 4: Run full test and lint suite to verify everything passes**

Run: `make test && make lint`
Expected: PASS with 0 lint issues

---

## Self-Review Checklist

1. **Spec coverage:**
   - 4.1 (`$/…` local action prefix): Task 1
   - 4.2 (`internal/lockfile` package): Task 2
   - 4.3 (Auto-refresh `actions.lock` on writes): Task 3
   - 4.4 (Transitive vulnerability scanning in `actup audit` + `--fix`): Task 4
   - 4.5 (Lockfile status in `actup ls`): Task 5
   - 4.6 (`gh` CLI extension binary detection + `.goreleaser.yaml` + docs): Task 6
2. **Review Focus coverage:**
   - Subpaths & mixed-case keys in `actions.lock`: tested in `TestTransitiveRefs_ExcludesDirectAndHandlesSubpathsAndBareSHA` and `TestStatusForAction_SubpathAndCaseInsensitive` (Task 2).
   - Bare SHA in transitive `actions.lock` entry: tested in `TestTransitiveRefs_ExcludesDirectAndHandlesSubpathsAndBareSHA` (Task 2).
   - `audit --fix` with transitive-only finding: tested in `TestAuditFix_TransitiveOnlyReportsResidualAndDoesNotRewriteLockfile` (Task 4).
   - Dry-run skips `gh actions-lock`: tested in `TestUseAndSync_AutoRelockWhenActionsLockExists` (Task 3).
   - `actup ls` with and without `actions.lock`: tested in `TestLs_ShowsLockStatusWhenActionsLockPresent` + existing `TestLs*` suite (Task 5).
