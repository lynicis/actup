# AGENTS.md

`actup` is a version manager for GitHub Actions: scans workflow files and sets, pins, syncs, or upgrades `uses:` refs (TUI or non-interactive), warning on known breaking changes and OSV.dev advisories.

## Commands

| Task | Command |
|------|---------|
| Build | `make build` |
| All tests | `make test` (`go test -v -race ./...`) |
| One package | `go test -v ./internal/parser` |
| Lint | `make lint` (`golangci-lint run ./...`) |
| Security / vulns | `make sec`, `make vuln` |
| Format / tidy | `make fmt`, `make mod-tidy` |

Tests write to `t.TempDir()`; no network or disk fixtures. Mock `github`/`token` runners to keep API calls out of unit tests.

## Packages

| Path | Role |
|------|------|
| `cmd/` | Cobra root (scan + upgrade) plus subcommands (`use`, `pin`/`unpin`, `ls`, `sync`, `init`, `outdated`, `audit`, `install-hooks`, `mcp`); shared in `cmd/common.go` |
| `internal/scanner` | Finds `.yml`/`.yaml` under `.github/workflows/` and composite/root action files |
| `internal/parser` | Extracts `uses:` refs; skips `./local`, `docker://`, matrix expressions; keeps SHA + `# <tag>` comment |
| `internal/lockfile` | Parses `.github/workflows/actions.lock` (v0.0.3), extracts transitive ActionRefs (Line 0), computes lock status, runs `gh actions-lock` via injectable execCommand |
| `internal/github` | `go-github/v89` client: tag lookup (major or semver mode), `ResolveTagSHA`, in-memory cache; `NewClientWithBaseURL` points it at httptest server |
| `internal/token` | `--token` → `GITHUB_TOKEN` → `gh auth token` |
| `internal/config` | `.actup.yaml` manifest (`major`, `sha`, `pins`, `actions`, `groups`); `EffectivePins()` resolves constraints |
| `internal/tui` | Bubble Tea checklist with breaking-change / vulnerability detail views |
| `internal/upgrader` | Atomic edits via temp file + rename; single-tag or per-action maps |
| `internal/breakingchanges` | Embedded `registry.yaml` (goccy/go-yaml) |
| `internal/vulnscanner` | OSV.dev `querybatch` + parallel `/vulns/{id}`; version matching is client-side |

## Non-obvious rules

- **Root flags are local.** Each subcommand redeclares its flags. Never use `PersistentFlags()`: `version` defines `-s/--short`, colliding with root `-s/--semver` and panicking at runtime.
- **`ActionRef.Version()` is the comparison primitive** for OSV, breaking changes, outdated, `ls`, `init`, and the TUI. Compare through it, not raw strings.
- **Default tag mode is major** (`v5`); `--semver` opts into full semver. Tag lookup reads 3 pages × 100 and filters non-semver/SHA refs.
- **Config semantics:** `Pins == nil` means a legacy manifest where every `actions` entry is a constraint; always resolve via `EffectivePins()`. `Save` re-renders deterministically — user comments are lost; numeric-looking values are quoted.
- **`runCheck` in `root.go` calls `os.Exit(1)`** when outdated entries exist; it backs `outdated`, so it must not be reused where returning is expected.
- **Exit codes in `audit`:** 1 on findings, 2 when OSV is unreachable. Query `owner/repo` lowercased; subpaths stripped.
- Breaking-change prompts go through `confirmBreaking` (`cmd/common.go`): proceed on dry-run/force, y/N on a TTY, skip otherwise.
- **`mcp` serves tools over stdio** (`modelcontextprotocol/go-sdk`): `list_actions`, `check_outdated`, `audit_vulnerabilities`, `upgrade_actions`, `pin_action`, `unpin_action`, `init_config`. Write tools hold `mcpWriteMu`; tests swap the `mcpNewGHClient` / `mcpVulnScan` seams instead of touching the network.
- **MCP `upgrade_actions` gates harder than the CLI:** breaking or vulnerable targets return `skipped_breaking` / `skipped_vulnerable` unless the caller sets `dry_run` or `force`.
- **MCP tools degrade, never fail, on bad inputs:** an unreadable `.actup.yaml`/`actions.lock` logs a stderr warning and yields a nil config rather than an error, so tool calls still return structured output.
- TUI model updates are easy to race — keep updates pure and rerun `make test`.
- CGO is disabled; builds target linux/darwin/windows amd64+arm64 (no Windows arm64). API concurrency is capped at 5 via semaphore.
- The `github-actions-updater` skill under `skills/` is a distributed artifact, not a convention for this repo.

## External references

| Need | File |
|------|------|
| CLI usage, config, installation | `README.md` |
| Release history, security policy | `CHANGELOG.md`, `SECURITY.md` |
| Manifest schema | `schema.json` |
| Design notes | `docs/plans/` |
| Packaging & CI | `.goreleaser.yaml`, `Dockerfile`, `.github/workflows/` |
| Pre-commit (runs `actup outdated`) | `.pre-commit-config.yaml` |
