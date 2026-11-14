# Design: GitHub Actions Lockfile (`actions.lock`) Integration & `gh actup` CLI Extension

> Date: 2026-11-12
> Status: Draft (Pending Approval)

---

## 1. Intent & Why

GitHub Actions workflow lockfiles (`.github/workflows/actions.lock`, managed by `gh actions-lock`) are now generally available. Repositories that onboard to `actions.lock` keep readable tags in `.github/workflows/*.yml` while locking exact commit SHAs for both direct and **transitive** action dependencies in `.github/workflows/actions.lock`.

This creates two opportunities and one hazard for `actup`:
1. **Hazard:** When `actup` (`actup`, `use`, `sync`, `audit --fix`) updates a `uses:` tag in a workflow file, an existing `.github/workflows/actions.lock` immediately becomes stale, causing GitHub Actions to reject workflow runs until `gh actions-lock` is re-run. Also, `gh actions-lock --migrate-local-actions` rewrites `./…` local actions to `$/…`, which `internal/parser` must skip.
2. **Opportunity — Transitive `audit` & `ls` visibility:** `.github/workflows/actions.lock` records resolved `ref` tags and `commit` SHAs for every transitive action dependency. Reading `actions.lock` lets `actup audit` scan transitive dependencies against OSV.dev with zero extra GitHub API calls, and lets `actup ls` surface lock status.
3. **Opportunity — `gh` CLI Extension (`gh actup`):** Since `gh actions-lock` is a `gh` extension and `actup` already integrates with `gh auth token`, shipping `gh-actup-<os>-<arch>[.exe]` release binaries and `gh actup` CLI ergonomics lets users install and run `actup` natively via `gh`.

---

## 2. Goals, Non-Goals, and Anti-Goals

### Goals
- Skip `$/…` same-repo local action references in `internal/parser`.
- Parse `.github/workflows/actions.lock` (`v0.0.3` format) in a new `internal/lockfile` package.
- Automatically invoke `gh actions-lock` after `actup` (`--no-tui` and TUI), `actup use`, `actup sync`, or `actup audit --fix` modifies workflow files when `actions.lock` is present (and not `--dry-run`), degrading gracefully with a warning if `gh` or the `gh-actions-lock` extension is not installed.
- Include transitive dependencies from `actions.lock` in `actup audit` (human-readable and `--json`), while keeping `actup audit --fix` safe (only rewriting direct workflow files, and reporting remaining transitive findings clearly).
- Show lockfile status (`locked` / `lock stale`) in `actup ls` when `actions.lock` is present.
- Support running `actup` as a `gh` CLI extension (`gh-actup` binary detection for `Use:` help text + GoReleaser raw binary assets `gh-actup-<os>-<arch>[.exe]`).

### Non-Goals
- Generating or modifying `.github/workflows/actions.lock` directly in Go without `gh actions-lock` (`actions.lock` requires GitHub's transitive graph resolution and numeric `owner_id`/`repo_id` provenance; `actions.lock` explicitly states `# Do not edit by hand; run gh actions-lock to update`).
- Adding new persistent root flags (root flags remain local per `AGENTS.md`).

### Anti-Goals
- Failing an otherwise successful workflow upgrade if `gh actions-lock` is not installed on the user's machine.
- Making network or `gh` subprocess calls during unit tests (all external execution must go through an injectable seam).

---

## 3. Approaches Considered

| Decision | Chosen Approach | Alternative Rejected | Why |
|---|---|---|---|
| **Lockfile integration** | **Approach A:** Automatic detection of `.github/workflows/actions.lock` across `audit`, `ls`, and write commands (`root`, `use`, `sync`, `audit --fix`) via `internal/lockfile`. | **Approach B:** Require an explicit `--lock` flag on every command. | If `actions.lock` is already committed in the repo, leaving it stale after an upgrade breaks GitHub Actions runs immediately. Zero-config detection matches how `.actup.yaml` is loaded. |
| **`gh` CLI extension packaging** | **Approach 1a:** Keep repository as `lynicis/actup`, add a `formats: [binary]` archive in `.goreleaser.yaml` emitting `gh-actup-<os>-<arch>[.exe]`, and detect `gh-actup` in `cmd/root.go`. | **Approach 2:** Replace GoReleaser with `cli/gh-extension-precompile`. | GoReleaser already builds Homebrew, Scoop, deb/rpm, and multi-arch Docker images; adding a second `archives` entry in `.goreleaser.yaml` produces the exact `gh-actup-*` assets `gh extension install` expects without breaking any existing package manager or `go install github.com/lynicis/actup@latest`. |

---

## 4. Technical Specification

### 4.1 `internal/parser`: Skip `$/…` Local Action References
- **File:** `internal/parser/parser.go` (`parseActionRef`)
- **Change:** Alongside `strings.HasPrefix(uses, "./")` and `strings.HasPrefix(uses, "docker://")`, return `nil, nil` when `strings.HasPrefix(uses, "$/")`.

### 4.2 New Package: `internal/lockfile`
- **Files:** `internal/lockfile/lockfile.go`, `internal/lockfile/lockfile_test.go`
- **Data Model:**
  ```go
  package lockfile

  const DefaultPath = ".github/workflows/actions.lock"

  type Lockfile struct {
      Path         string                  `yaml:"-"`
      Version      string                  `yaml:"version"`
      Workflows    map[string][]string     `yaml:"workflows"`
      Dependencies map[string]Dependency   `yaml:"dependencies"`
  }

  type Dependency struct {
      Ref     string   `yaml:"ref"`
      Commit  string   `yaml:"commit"`
      OwnerID int64    `yaml:"owner_id"`
      RepoID  int64    `yaml:"repo_id"`
      Uses    []string `yaml:"uses,omitempty"`
  }
  ```
- **Discovery & Loading (`LoadForPaths(paths []string) (*Lockfile, error)`):**
  - When `len(paths) == 0`: checks `DefaultPath` (`.github/workflows/actions.lock`). Returns `(nil, nil)` if the file does not exist.
  - When `len(paths) > 0`: checks `<dir>/actions.lock` for each directory path (or `<filepath.Dir(file)>/actions.lock` for file paths), returning the first existing lockfile, or `(nil, nil)` if none exists. This allows `t.TempDir()` tests and `-p` custom workflow directories to work naturally without touching the working directory.
- **Extracting Transitive Refs (`(l *Lockfile) TransitiveRefs(direct []parser.ActionRef) []parser.ActionRef`):**
  - Note from upstream `github/actions-lockfile`: pin keys in `actions.lock` are always repo-scoped (`owner/repo@ref` with `owner/repo` lowercased; subpaths like `actions/cache/save@v4` are normalized to `actions/cache@v4`).
  - Collects the set of direct `strings.ToLower(owner/repo) + "@" + ref.Current` keys from `direct` (and the union of entries listed in `l.Workflows`).
  - Iterates over `slices.Sorted(maps.Keys(l.Dependencies))`.
  - Parses each dependency key (`owner/repo@spec`).
  - Skips any entry present in the direct workflow set.
  - Constructs a `parser.ActionRef`:
    - `Owner`, `Repo`, `Path`: parsed from the dependency key before `@`.
    - If `dep.Ref` is a 40-hex SHA (or `@spec` is a 40-hex SHA and `dep.Ref` is also a SHA): sets `Current: spec`, `SHA: strings.ToLower(spec)`, `Tag: ""` (unless `dep.Ref` is a non-SHA tag, in which case `Tag: dep.Ref`).
    - Otherwise: sets `Current: dep.Ref` so `Version()` returns `dep.Ref`.
    - `File`: `l.Path`, `Line`: `0` (`Line == 0` distinguishes lockfile-sourced transitive refs from workflow line refs).
  - Deduplicates by `(strings.ToLower(Owner + "/" + Repo), ref.Version(), ref.Current)` so the same transitive action at the same version is only reported once.
- **Checking Lock Status (`(l *Lockfile) StatusForAction(key string, refs []parser.ActionRef) Status`):**
  - `Status` enum: `StatusNone`, `StatusLocked`, `StatusStale`.
  - For each `ref` in `refs`, computes canonical lock key `strings.ToLower(ref.Owner + "/" + ref.Repo) + "@" + ref.Current` (stripped of subpaths, matching `gh actions-lock`).
  - Returns `StatusLocked` if every `ref` in `refs` exists in `l.Dependencies` (case-insensitive on `owner/repo`), else `StatusStale`.
- **Refreshing Lockfile (`Sync(dir string) error`):**
  - Uses package-level `var execCommand = exec.Command` seam (mirroring `internal/token/token.go`).
  - Runs `gh actions-lock` (with `cmd.Dir = dir` if non-empty).
  - Returns `nil` on success, or a wrapped error with trimmed stderr/stdout on failure.

### 4.3 Auto-Refresh `actions.lock` on Workflow Writes
- **Files:** `cmd/common.go`, `cmd/root.go`, `cmd/use.go`, `cmd/sync.go`, `cmd/audit.go`, `internal/tui/tui.go`
- **Behavior:**
  - Add `syncLockfile(paths []string)` in `cmd/common.go`:
    - Calls `lf, err := lockfile.LoadForPaths(paths)`. If `lf == nil` (no lockfile in repo), does nothing.
    - Calls `lockfile.Sync(repoRoot)` (current working directory when using default paths, or the parent of `.github/workflows` when a custom path is inside `.github/workflows`).
    - On success: prints `✓ Updated .github/workflows/actions.lock` (or runs silently inside TUI `applyUpgrades`).
    - On error: prints `⚠ Failed to update actions.lock (run 'gh actions-lock' manually): <err>` to `os.Stderr` and does **not** fail the command exit code.
  - Called at the end of `runNoTUI` (`cmd/root.go`), `doUse` (`cmd/use.go`), `doSync` (`cmd/sync.go`), `runAuditFix` (`cmd/audit.go`), and `applyUpgrades` (`internal/tui/tui.go`) whenever workflow files were modified and `!dryRun`.

### 4.4 Transitive Vulnerability Scanning in `actup audit`
- **File:** `cmd/audit.go`
- **Behavior:**
  - In `runAudit`, after loading `actions`, load `lf, err := lockfile.LoadForPaths(auditOpts.paths)`.
  - If `lf != nil`, get `transitive := filterSkippedActions(lf.TransitiveRefs(actions), cfg)` and append to `checkable`.
  - **Text output (`runAudit`):**
    - When `f.Line == 0`, print `%s (transitive): %s@%s\n` instead of `%s:%d: %s@%s\n`.
  - **JSON output (`outputAuditJSON`):**
    - Add `Transitive bool` (`json:"transitive,omitempty"`) to `auditJSONFinding` (`true` when `f.Line == 0`).
  - **`actup audit --fix` (`runAuditFix`):**
    - Partition `findings` into direct findings (`f.Line > 0`) and transitive findings (`f.Line == 0`).
    - Run the existing fix planner on direct findings.
    - If direct fixes are applied and `actions.lock` is present, `syncLockfile` runs `gh actions-lock`, which may automatically resolve transitive findings too.
    - For any transitive finding whose parent was not upgraded (or if no direct findings were fixed), print:
      `✗ <owner>/<repo>@<ref>: transitive dependency in <lockfile> (upgrade parent action or run 'gh actions-lock --relock')` and count it in `residual` so exit code `1` is preserved.

### 4.5 Lockfile Status in `actup ls`
- **File:** `cmd/ls.go`
- **Behavior:**
  - Load `lf, _ := lockfile.LoadForPaths(lsOpts.paths)`.
  - If `lf != nil`:
    - In `runLsTable`: append `  [locked]` or `  [lock stale]` to each row based on `lf.StatusForAction(key, grouped[key])`.
    - In `runLsDetail`: include lock status in the header line, e.g. `actions/checkout (pin: v4, lock: locked)`.
  - If `lf == nil`: output is 100% unchanged (preserving all existing `ls` tests).

### 4.6 `gh` CLI Extension Support (`gh actup`)
- **Files:** `cmd/root.go`, `.goreleaser.yaml`, `README.md`
- **Binary / Help detection (`cmd/root.go`):**
  - Helper `invokedAsGHExtension() bool` checks ` strings.HasPrefix(filepath.Base(os.Args[0]), "gh-")` or `os.Getenv("GH_PATH") != ""`.
  - In `init()`, if `invokedAsGHExtension()`, set `rootCmd.Use = "gh actup"`.
- **Release Assets (`.goreleaser.yaml`):**
  - Keep existing `archives` entry (`id: default`) for `.tar.gz`/`.zip`.
  - Add second `archives` entry (`id: gh-extension`):
    ```yaml
    - id: gh-extension
      formats: [binary]
      name_template: "gh-actup-{{ .Os }}-{{ .Arch }}{{ if .Arm }}v{{ .Arm }}{{ end }}"
    ```
    (GoReleaser automatically appends `.exe` on Windows for `formats: [binary]`).
- **Documentation (`README.md`, `AGENTS.md`):**
  - Add `gh extension` installation instructions and document `.github/workflows/actions.lock` support.
