# actup — Brainstorming: Improvements & New Features

> Generated: 2026-11-11
> Purpose: Standalone backlog of existing feature improvements and new feature candidates for deep-dive sessions.

---

## Part 1: Improvements to Existing Features

### 1a. `.actup.yaml` Lifecycle: `init` vs. `upgrade` / `outdated` Conflict

- **Where:**
  - `internal/github/client.go:159-191` (`ResolveVersion`)
  - `cmd/init.go`
  - `cmd/root.go` (`run`, `runNoTUI`, `runCheck`)
  - `internal/tui/tui.go`
- **Problem:**
  1. `actup init` writes exact versions currently in workflows to `.actup.yaml` (e.g., `actions/checkout: v4`).
  2. `github.ResolveVersion` treats any non-numeric, non-`skip` entry in `.actup.yaml` as an `exactVersion` override and returns it verbatim without querying the GitHub API.
  3. Consequently, running `actup outdated`, `actup ls --outdated`, or `actup` after `actup init` reports **all actions are already up to date** forever.
  4. When upgrades are applied via `actup` (TUI or `--no-tui`), `.actup.yaml` is never updated to reflect the newly upgraded versions, causing the manifest and workflows to drift apart.
- **Deep-Dive Questions / Options:**
  - Should `ResolveVersion` in `outdated` / `upgrade` ignore exact version strings in `.actup.yaml` (only honoring numeric major constraints like `4` and `skip`), while `actup sync` applies exact versions verbatim?
  - Should `actup` (TUI and `--no-tui`) automatically update `.actup.yaml` when an action is upgraded if `.actup.yaml` exists (matching `actup use` behavior)?

---

### 1b. Atomic File Writes: Preserve Permissions & Batch Per File

- **Where:**
  - `internal/upgrader/upgrader.go:29-83` (`replaceInFile`)
  - `internal/upgrader/upgrader.go:95-115` (`ApplyAllUpgrades`)
- **Problem:**
  1. `replaceInFile` creates a temp file with `os.CreateTemp`, which defaults to mode `0600`, and renames it over the target workflow file. This silently changes workflow files from `0644` to `0600`.
  2. `ApplyAllUpgrades` groups upgrades by file (`byFile`), but then calls `replaceInFile` inside the inner loop once per `Upgrade`—reading the file, writing a temp file, and renaming it $N$ times for a file with $N$ action references.
- **Proposed Fix:**
  - Stat the original file before editing to capture its `os.FileMode` and apply `os.Chmod(tmpPath, mode)` before `os.Rename`.
  - Refactor `replaceInFile` → `replaceAllInFile(file string, upgrades []Upgrade)` so each workflow file is read, modified in memory across all target lines, and atomically written once.

---

### 1c. Breaking-Change Registry Coverage & `with:` Input Inspection

- **Where:**
  - `internal/breakingchanges/registry.yaml`
  - `internal/breakingchanges/registry.go`
  - `internal/parser/parser.go`
- **Problem:**
  1. `registry.yaml` currently contains only a single entry (`actions/checkout` v2 → v3).
  2. `BreakingChange` structs already encode `type` (`renamed`, `deprecated`, `removed`), `input`, and `replacement`, but `actup` warns on every major jump in the registry even when the workflow step does not pass the affected `with:` input.
- **Proposed Improvements:**
  - Expand `registry.yaml` to cover common real-world breaking upgrades:
    - `actions/upload-artifact` & `actions/download-artifact` (v3 → v4 immutable artifact names, `merge-multiple` changes)
    - `actions/cache` (v3 → v4)
    - `actions/setup-node`, `actions/setup-go`, `actions/setup-python`, `actions/setup-java`
    - `golangci/golangci-lint-action`
    - `docker/build-push-action`, `docker/login-action`, `docker/setup-buildx-action`
    - `goreleaser/goreleaser-action`
  - Optionally parse the step's `with:` keys in `parser.ActionRef` so warnings can distinguish *"general major-version note"* vs. *"your workflow uses deprecated input `github_token` on line 42"*.

---

### 1d. Mixed-Version Handling in `runNoTUI` and TUI

- **Where:**
  - `cmd/root.go:165-166, 193-195`
  - `internal/tui/tui.go`
  - `cmd/common.go:154-174` (`confirmBreakingAll`)
- **Problem:**
  - `cmd/use.go` and `cmd/sync.go` use `confirmBreakingAll` to check breaking changes across every distinct version of an action found across workflows.
  - `runNoTUI` in `cmd/root.go` only inspects `current := acts[0].Current`. If `ci.yml` uses `v4` and `release.yml` uses `v2`, `runNoTUI` misses breaking changes for `v2 → v4` if `acts[0]` happens to be `v4`.
- **Proposed Fix:**
  - Unify `runNoTUI` and TUI on `confirmBreakingAll` / distinct current versions per action group.

---

## Part 2: New Features

### 2a. Commit SHA Pinning (`--sha`) & Comment Version Parsing

- **Where:**
  - `internal/parser/parser.go:25, 93-95`
  - `internal/github/client.go`
  - `internal/upgrader/upgrader.go`
  - `cmd/` flags & `.actup.yaml`
- **Motivation:**
  - Security standards (GitHub Security Hardening, OpenSSF Scorecard, zizmor, StepSecurity) require pinning third-party actions to full 40-character commit SHAs with a trailing human-readable tag comment:
    ```yaml
    uses: actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683 # v4.2.2
    ```
  - Currently, `parser.go:93` skips 40-char hex SHAs (`if len(ref) == 40 && isHex(ref) { return nil, nil }`), making `actup` blind to SHA-pinned workflows.
- **Proposed Scope:**
  1. **Parse SHA + trailing comment:** If `ref` is a 40-char SHA and the line has a `# vX.Y.Z` (or `# vX`) comment, extract both the SHA and the semantic version comment so `ls`, `outdated`, `audit`, and `upgrade` work on SHA-pinned repositories.
  2. **Resolve & write SHA pins (`--sha` / `sha: true`):** Resolve the target tag's commit SHA via the GitHub API (handling both lightweight and annotated tags) and rewrite the line as `<owner>/<repo>@<sha> # <tag>`.

---

### 2b. `actup audit --fix`

- **Where:**
  - `cmd/audit.go`
  - `internal/vulnscanner/vulnscanner.go`
  - `internal/upgrader/upgrader.go`
- **Motivation:**
  - `actup audit` already identifies vulnerable action refs and knows the `FixedVersion` from OSV.dev advisories (`vulnscanner.Finding`).
  - Users currently have to read the `actup audit` output and manually run `actup use <action>@<fixed>` for each vulnerable action.
- **Proposed Scope:**
  - Add `--fix` (and `--dry-run`, `--force`) to `actup audit`.
  - For each vulnerable action, compute the minimum version that resolves all advisories affecting that action (or latest safe tag), check for breaking changes, rewrite workflows, and update `.actup.yaml` if present.

---

### 2c. Composite Actions & Reusable Workflow Discovery

- **Where:**
  - `internal/scanner/scanner.go`
- **Motivation:**
  - Many repositories define local composite actions in `.github/actions/*/action.yml` (or root `action.yml` / `action.yaml`) which contain `runs.steps[].uses:` references identical to workflow files.
  - Today `scanner.DiscoverWorkflows` only scans `.github/workflows` by default.
- **Proposed Scope:**
  - When using default discovery paths, also scan `.github/actions` (for `action.yml`/`action.yaml`) and root `action.yml`/`action.yaml` if present.

---

### 2d. Dependabot-Style Grouped / Filtered Upgrades

- **Where:**
  - `cmd/root.go`, `cmd/outdated.go`
  - `internal/config/config.go`
- **Motivation:**
  - Listed on the README roadmap: allow filtering upgrades by glob/prefix on the CLI (e.g., `actup actions/*` or `actup docker/*`) or defining groups in `.actup.yaml` so teams can upgrade subsets of actions together.

---

### 2e. GitHub Actions Lockfile (`.github/workflows/actions.lock`) Integration

- **Where:**
  - `internal/parser/parser.go` (`$/…` same-repo local action prefix)
  - `internal/lockfile/` (new package)
  - `cmd/common.go`, `cmd/root.go`, `cmd/use.go`, `cmd/sync.go`, `cmd/audit.go`, `cmd/ls.go`, `internal/tui/tui.go`
- **Motivation:**
  - GitHub Actions workflow lockfiles (`.github/workflows/actions.lock` via `gh actions-lock`) are now GA.
  - Detailed design: `docs/plans/2026-11-12-actions-lock-and-gh-extension-design.md`.

---

### 2f. GitHub CLI Extension (`gh actup`) Support

- **Where:**
  - `.goreleaser.yaml`
  - `cmd/root.go`
  - `README.md`
- **Motivation:**
  - Ship `gh-actup-<os>-<arch>[.exe]` release binaries and `gh actup` usage detection so `actup` works as a native `gh` CLI extension alongside `gh actions-lock`.
  - Detailed design: `docs/plans/2026-11-12-actions-lock-and-gh-extension-design.md`.

---

## Suggested Order of Attack

1. **Wave 1 — Core Reliability & Manifest Consistency:** `1a` (`.actup.yaml` lifecycle), `1b` (atomic write permissions + batching), `1d` (mixed-version breaking-change check).
2. **Wave 2 — Flagship Security / UX Feature:** `2a` (SHA pinning + comment parsing) or `2b` (`actup audit --fix`).
3. **Wave 3 — Coverage & Ecosystem:** `2c` (composite actions discovery) + `1c` (expanded breaking-change registry).
