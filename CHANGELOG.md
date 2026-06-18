# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2026-06-15

### Added
- GitHub Actions updater AI skill (`github-actions-updater`) for compatibility with AI coding agents (Claude Code, OpenCode, Cursor, etc.). Installable via `gh skill` or `npx skills`.

## [0.7.1] - 2026-06-16

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

[1.0.0]: https://github.com/lynicis/actup/releases/tag/v1.0.0
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
