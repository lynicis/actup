# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.10.0] - 2026-11-14

### Added
- **`actup mcp` — MCP server for AI coding agents.** Runs a [Model Context Protocol](https://modelcontextprotocol.io) server over stdio (official `modelcontextprotocol/go-sdk`), exposing the version manager as typed tools: `list_actions`, `check_outdated`, `audit_vulnerabilities`, `upgrade_actions`, `pin_action`, `unpin_action`, and `init_config`. Each call returns a one-line summary plus structured content, so agents get versions, usage sites, lock status, breaking changes, and OSV findings as data instead of parsing CLI text.
  - Tools discover `.github/workflows/`, `.actup.yaml`, and `actions.lock` relative to the working directory, exactly like the CLI; the GitHub token comes from `GITHUB_TOKEN` or `gh auth token` and is never a tool parameter.
  - `upgrade_actions` honors manifest pins and per-action targets, supports `dry_run`/`force`, and skips actions with known breaking changes or vulnerable targets unless one is set — stricter than the non-interactive CLI. Mutating tools are serialized; the server never shells out to `gh actions-lock` and returns a `lockfile_note` instead.
  - An unreadable manifest or lockfile degrades to a stderr warning so tool calls still return structured results, and a client disconnect is treated as a clean shutdown rather than an error.
  - README documents setup for Claude Code, Codex, Cursor, Gemini CLI, Antigravity, oh-my-pi (`omp`), OpenCode, and VS Code (GitHub Copilot).
- **GitHub CLI extension companion.** Prebuilt `gh-actup` binaries are mirrored to [`lynicis/gh-actup`](https://github.com/lynicis/gh-actup) on release for single-command `gh extension install lynicis/gh-actup` support.

## [0.9.1] - 2026-11-13

### Fixed
- The TUI counts distinct workflow files instead of usage sites in its checklist header and upgrade summary, and pluralizes `1 file` / `1 action` correctly.

## [0.9.0] - 2026-11-13

### Added
- **Managed manifests.** `.actup.yaml` now separates recorded workflow state (`actions`) from user-authored constraints (`pins`). Managed manifests resolve constraints through `EffectivePins()`, while legacy manifests without `pins` keep behaving exactly as before. Numeric majors and `skip` remain constraints even in the recorded section, and upgrades write new versions back deterministically.
- **Commit SHA pinning.** `--sha` (or `sha: true` in the manifest) writes `@<40-hex-sha> # <tag>`. Existing SHA pins with trailing comments are parsed, compared by their tag, and kept in the per-site style by `actup`, `use`, `sync`, `audit --fix`, and the TUI.
- **`actup audit --fix`.** Upgrades vulnerable direct refs to a version that clears every advisory affecting them, never downgrades, leaves advisories without a fixed version untouched, records the result in `.actup.yaml`, and reports remaining transitive findings. Supports `--dry-run`, `--force`, `--semver`, and `--sha`; `--json` cannot be combined with `--fix`.
- **Action filters.** Positional glob filters (`actup 'actions/*'`, using `path.Match`) and the repeatable `--group` flag select a subset of actions; the manifest's new `groups` section defines named glob sets. Filters combine as a union.
- **Composite and root actions.** Default discovery now includes `.github/actions/**/action.yml|yaml` and a repository-root `action.yml|yaml`; explicit `-p` paths still scan only what they name.
- **`actions.lock` integration.** After workflow writes, `actup` runs `gh actions-lock` to refresh `.github/workflows/actions.lock`; dry runs and no-op runs skip it, and a missing `gh` or extension only warns. `actup audit` also scans transitive lockfile dependencies, marking them `(transitive)` in text and `"transitive": true` in JSON, while `actup ls` shows `[locked]` / `[lock stale]`.
- **`gh` CLI extension packaging.** Releases ship `gh-actup-<os>-<arch>[.exe]` binaries, and the CLI detects `gh actup` invocations so help and usage use the right name.
- **Richer breaking-change detection.** `with:` inputs are parsed and matched against input-specific registry entries (runtime and behavior types), every distinct version in use is compared, and multi-major jumps surface all crossed entries.
- **Build and supply chain.** `make install` installs to `$GOBIN`, and developer tools run through Go's tool directive. CI adds hadolint, Trivy config/image, zizmor, gitleaks, dependency-review, and `actup audit` gates with SARIF upload; releases scan the image before publishing, generate SBOMs, sign images keylessly with cosign, and attest archive provenance. The Docker image now runs as UID 65532 with a version healthcheck.

### Changed
- Batch workflow rewrites now run once per file, preserve permission bits, and produce deterministic dry-run output.
- `schema.json` adds the `sha`, `pins`, and `groups` properties.

### Fixed
- OSV ranges that are open-ended in the advisory database are now bounded by `database_specific.last_known_affected_version_range`, so later major versions are no longer reported as vulnerable.
- `actup audit --json` exits with status 2 when OSV is unreachable, matching text mode.
- Lockfile integration recognizes narrowed refs in `actup ls`, finds `actions.lock` when `--path` points at `.github`, and keeps TUI relocking independent of manifest-save failures.
- `actup outdated` is used by the pre-commit hook; the removed root `--check` flag is no longer referenced.

### Security
- Hardened the `github-actions-updater` skill against indirect prompt injection (Snyk W011): version resolution only consumes `gh api` / `git ls-remote` output matching a strict semver pattern and selects a single version in the pipeline, so raw tag lists, JSON, and release-page text never reach agent context.

## [0.8.2] - 2026-10-23

### Fixed
- Trailing YAML comments on `uses:` lines (e.g. `uses: actions/checkout@v3    # pinned`) are stripped before parsing: rewrites now replace only the ref and preserve the comment, instead of capturing the comment as part of the ref and deleting it. Version listings, outdated checks, and vulnerability matching also see bare refs again; SHA pins with trailing comments are now skipped like bare SHA refs.
- Rewrites write the target ref verbatim, so refs containing `$` are no longer expanded as regexp capture references.

## [0.8.1] - 2026-10-18

### Fixed
- Upgrading subpath actions (e.g. `google/osv-scanner-action/osv-scanner-action@v2`) failed with "could not find expected uses line" because the upgrader matched `uses: owner/repo@tag`, dropping the subpath segment. `ActionRef` now carries the full action path and the upgrader, dry-run diffs, and `actup use` rewrite subpath references correctly.
- Batch upgrades no longer abort on the first per-action failure; `ApplyAllUpgrades` applies every upgrade it can and reports the combined errors.
- `uses:` refs with tabs or unusual spacing between the key and value (e.g. `uses:\t\taction@tag`) now match correctly; the rewrite is anchored to the `path@ref` segment instead of the literal `uses: ` prefix.

## [0.8.0] - 2026-10-18

### Added
- `actup audit` — vulnerability scanning for GitHub Actions workflows against the OSV.dev advisory database (GitHub Actions ecosystem):
  - Reports advisory ID, severity, summary, and first fixed version per usage site (`file:line`).
  - `--json` emits machine-readable output for CI pipelines.
  - Exits `1` when vulnerable versions are found, `2` when OSV is unreachable.
  - Client-side version-range matching (OSV does not match versions server-side for this ecosystem), with `v`-prefix normalization and support for `introduced`/`fixed`/`last_affected` events and multiple vulnerable intervals.
  - New `internal/vulnscanner` package: batch queries (`querybatch`), parallel advisory-detail fetches (capped at 5), case-insensitive package matching, and withdrawn-advisory filtering.
  - Inline warnings in the `--no-tui` upgrade flow and a `⚠ vulnerable` badge with detail view in the interactive TUI.
  - `actup use`, `actup sync`, and `actup outdated` print the same warnings before making changes; `use` and `sync` additionally check the target version itself and prompt (y/N on TTY) when the version being pinned is affected by a known advisory.
- Version-manager subcommands (nvm/fvm style):
  - `actup use <action>[@<ref>]` — sets an action's version across all workflows and records the pin in `.actup.yaml` by default (`--no-save` to skip, `--dry-run` to preview). Ref forms: `@vN` verbatim, `@N` resolved to latest in major, `@latest` per tag mode, others verbatim.
  - `actup pin <action>[@<ref>]` and `actup unpin <action>` — manage manifest pins without touching workflows; `pin` without a ref pins the version currently in use.
  - `actup ls [action]` — lists actions in use with versions and pin status; `--outdated` annotates latest versions via the API; per-action detail lists every `file:line`.
  - `actup sync` (alias `install`) — applies `.actup.yaml` pins to all workflows (exact refs verbatim, numeric majors resolved, `skip` ignored), with breaking-change warnings.
  - `actup init` — bootstraps `.actup.yaml` from the versions currently used in workflows (`--force` merges into an existing manifest).
  - `actup outdated` — subcommand form of the `--check` flag.
- `internal/config`: deterministic manifest writing via `Render`/`Save`/`SaveDefault` (sorted keys, schema header, type-safe quoting).

### Changed
- Root command help now presents actup as a GitHub Actions version manager; existing scan-and-upgrade behavior is unchanged.

## [0.7.7] - 2026-09-12

### Changed
- Flattened `internal/tui` package structure into a consolidated `tui.go` and `tui_test.go`.
- Simplified `internal/token` package by replacing `Resolver` struct and factory with a direct `Resolve` function.
- Simplified `internal/parser` and `internal/upgrader` to sequential execution.
- Replaced `golang.org/x/sync` dependency with standard library concurrency primitives.
- Cleaned up redundant code across internal packages and adopted standard library helper functions.

### Removed
- Eliminated `internal/checker` package by inlining check logic into `cmd/root.go`.
- Removed redundant `selectedSet` state tracking and custom signal handling from `internal/tui`.

## [0.7.6] - 2026-09-10

### Changed
- Upgraded Go version to `1.27` across CI workflows, build configurations, and documentation.
- Managed developer tools (`golangci-lint`, `gosec`, `govulncheck`) via Go tool directive in `go.mod` and updated Makefile targets.

### Removed
- Removed obsolete `install` target from Makefile.
- Removed legacy AI assistant configuration files and skills (`CLAUDE.md`, `.claude/skills`, `opencode.json`).

### Documentation
- Refined security policy and updated vulnerability reporting contact information in `SECURITY.md`.

## [0.7.5] - 2026-07-24

### Added
- Minimal multi-arch scratch Docker image support with GoReleaser.

### Changed
- Refactored internal packages to run concurrently using `errgroup`.
- Pinned `securego/gosec` action to `master` in CI.

### Fixed
- Fixed config file resolution and handling of loading errors in `--check` mode.

## [0.7.4] - 2026-07-23

### Added
- JSON schema for `.actup.yaml` configuration.
- Support for both `.actup.yaml` and `.actup.yml` config file extensions.
- Added OpenCode local MCP setup.

### Changed
- Upgraded to the latest `go-github` client.

## [0.7.3] - 2026-07-12

### Changed
- Bumped Go dependencies: `golang.org/x/mod`, `golang.org/x/term`, `golang.org/x/sys`, and `golang.org/x/text`.

### Documentation
- Corrected changelog versioning: removed the erroneous `1.0.0` entry and recorded the AI skill addition under `0.7.1`.

## [0.7.2] - 2026-07-02

### Added
- Static code analysis job with Gosec in CI workflow.
- Vulnerability check job with govulncheck in CI workflow.
- Makefile targets `sec` and `vuln` for security and vulnerability scans.

### Changed
- Upgraded GitHub checkout action to `v7` across workflows.
- Matrix builds for CI workflow on Ubuntu, macOS, and Windows.
- Refactored CLI and TUI to use unified `github.ResolveVersion` helper.
- Removed unused helper functions in breaking changes, config, and upgrader modules.

### Removed
- Chocolatey package publishing workflow and configuration.

### Fixed
- Fixed gosec security warnings for file permission settings and file paths.

## [0.7.1] - 2026-06-16

### Added
- GitHub Actions updater AI skill (`github-actions-updater`) for compatibility with AI coding agents (Claude Code, OpenCode, Cursor, etc.). Installable via `gh skill` or `npx skills`.

### Fixed
- Re-introduced reproducible builds by pinning `mod_timestamp` and build/commit dates in GoReleaser, ensuring the Windows ZIP checksum matches the GitHub release artifact and passes Chocolatey validation.

## [0.7.0] - 2026-06-16

### Added
- Chocolatey publishing pipeline on release tags.

### Fixed
- Explicit `url_template` configuration for the Chocolatey publisher in GoReleaser.

## [0.6.0] - 2026-06-14

### Fixed
- Removed faulty Chocolatey configuration that required a Windows runner in GoReleaser.

## [0.5.1] - 2026-06-10

### Documentation
- Updated pre-commit hook configuration examples and README guide.

## [0.5.0] - 2026-06-09

### Added
- New `--major` CLI flag to support major-version pinning (e.g., locking upgrades to major version like `v4` rather than specific semver).
- Configuration file support via `.actup.yaml` to define global defaults and per-action overrides or exclusions (e.g., `skip`).
- Wired `--major` flag into the TUI, non-interactive mode, and `--check` CLI pipelines.

## [0.4.0] - 2026-06-09

### Documentation
- Added pre-commit hooks section to the project roadmap and README.

## [0.3.1] - 2026-06-08

### Fixed
- Minor adjustments to Dockerfile to optimize GoReleaser builds.

## [0.3.0] - 2026-06-08

### Fixed
- Updated Dockerfile configuration for GoReleaser compatibility.

## [0.2.0] - 2026-06-08

### Documentation
- Updated `AGENTS.md` to reflect the latest repository state.

## [0.1.6] - 2026-06-08

### Fixed
- Changed Homebrew tap workflow trigger to run exclusively on pushing release tags.

## [0.1.5] - 2026-06-08

### Changed
- Migrated Homebrew tap publishing from direct GoReleaser brews to a proper custom formula repository.

## [0.1.4] - 2026-06-07

### Changed
- Minor updates and cleanups to GoReleaser configuration.

## [0.1.3] - 2026-06-07

### Added
- Highlight selected/focused items in the TUI checklist with the cursor text color.

## [0.1.2] - 2026-06-07

### Fixed
- Adjusted GoReleaser Scoop publishing settings.

## [0.1.1] - 2026-06-07

### Fixed
- Initial fixes for GoReleaser Scoop package publishing.

## [0.1.0] - 2026-06-07

### Added
- Initial release of the `actup` CLI tool.
- Workflow discovery scanner for recursively finding `.yml` and `.yaml` files under `.github/workflows/`.
- Interactive Bubble Tea TUI checklist with select-all, per-action toggles, and breaking-change detail view.
- Non-interactive mode (`--no-tui`) for automation and CI pipelines.
- Dry-run mode (`--dry-run`) to preview workflow changes without writing them to disk.
- Embedded breaking change registry with interactive warnings on upgrades that introduce breaking changes.
- Concurrent GitHub API client with tag caching and rate-limiting support.
- Falling back to the GitHub CLI (`gh`) token if no explicit token or env variable is set.

[Unreleased]: https://github.com/lynicis/actup/compare/v0.10.0...HEAD
[0.10.0]: https://github.com/lynicis/actup/releases/tag/v0.10.0
[0.9.1]: https://github.com/lynicis/actup/releases/tag/v0.9.1
[0.9.0]: https://github.com/lynicis/actup/releases/tag/v0.9.0
[0.8.2]: https://github.com/lynicis/actup/releases/tag/v0.8.2
[0.8.1]: https://github.com/lynicis/actup/releases/tag/v0.8.1
[0.8.0]: https://github.com/lynicis/actup/releases/tag/v0.8.0
[0.7.7]: https://github.com/lynicis/actup/releases/tag/v0.7.7
[0.7.6]: https://github.com/lynicis/actup/releases/tag/v0.7.6
[0.7.5]: https://github.com/lynicis/actup/releases/tag/v0.7.5
[0.7.4]: https://github.com/lynicis/actup/releases/tag/v0.7.4
[0.7.3]: https://github.com/lynicis/actup/releases/tag/v0.7.3
[0.7.2]: https://github.com/lynicis/actup/releases/tag/v0.7.2
[0.7.1]: https://github.com/lynicis/actup/releases/tag/v0.7.1
[0.7.0]: https://github.com/lynicis/actup/releases/tag/v0.7.0
[0.6.0]: https://github.com/lynicis/actup/releases/tag/v0.6.0
[0.5.1]: https://github.com/lynicis/actup/releases/tag/v0.5.1
[0.5.0]: https://github.com/lynicis/actup/releases/tag/v0.5.0
[0.4.0]: https://github.com/lynicis/actup/releases/tag/v0.4.0
[0.3.1]: https://github.com/lynicis/actup/releases/tag/v0.3.1
[0.3.0]: https://github.com/lynicis/actup/releases/tag/v0.3.0
[0.2.0]: https://github.com/lynicis/actup/releases/tag/v0.2.0
[0.1.6]: https://github.com/lynicis/actup/releases/tag/v0.1.6
[0.1.5]: https://github.com/lynicis/actup/releases/tag/v0.1.5
[0.1.4]: https://github.com/lynicis/actup/releases/tag/v0.1.4
[0.1.3]: https://github.com/lynicis/actup/releases/tag/v0.1.3
[0.1.2]: https://github.com/lynicis/actup/releases/tag/v0.1.2
[0.1.1]: https://github.com/lynicis/actup/releases/tag/v0.1.1
[0.1.0]: https://github.com/lynicis/actup/releases/tag/v0.1.0
