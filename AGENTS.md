# AGENTS.md — actup

> Repo-specific guidance for AI agents working in this codebase.
> Every entry should answer: *"Would an agent likely miss this without help?"*

## What this is

`actup` is a Go CLI that scans GitHub Actions workflow files and upgrades action versions interactively via a TUI, or non-interactively with `--no-tui`. It detects known breaking changes between major versions and warns before upgrading.

## Architecture

Standard Go CLI layout using Cobra:

- `main.go` → `cmd/root.go` orchestrates packages
- `internal/scanner` — discovers `.yml`/`.yaml` workflow files (default: `.github/workflows/`)
- `internal/parser` — extracts `uses:` refs; skips `./local`, `docker://`, and full SHA commits
- `internal/github` — GitHub API client (`go-github/v62`); fetches latest tags with dual mode (major-tag or full semver) and in-memory cache
- `internal/token` — resolves GitHub token via flag → `GITHUB_TOKEN` env → `gh auth token` CLI fallback
- `internal/breakingchanges` — embedded YAML registry (`//go:embed registry.yaml`) for known breaking changes between major versions; uses `goccy/go-yaml` for parsing
- `internal/tui` — Bubble Tea interactive checklist with detail view for breaking changes
- `internal/upgrader` — atomic file edits via temp-file + rename; supports both single-tag and per-action upgrade maps

## Developer commands

```bash
# Build
make build                 # go build -o actup .

# Test (all packages, with race detector)
make test                  # go test -v -race ./...

# Lint (requires golangci-lint installed)
make lint                  # golangci-lint run ./...

# Format and tidy
make fmt                   # go fmt ./...
make mod-tidy              # go mod tidy

# Clean build artifacts
make clean                 # rm -f actup && go clean

# Install to GOPATH/bin
make install               # go install .
```

## Testing

- All tests use `t.TempDir()` and write temporary workflow files — safe to run in parallel, no external services needed.
- Run a single package: `go test -v ./internal/parser`
- The `github` package has unit tests for pure logic (`resolveLatestTag`, major-tag regex) — no API calls in tests.
- The `breakingchanges` package has 10 unit tests covering registry parsing, version checking, and decision helpers.
- The `token` package has 5 unit tests covering the resolution chain with mocked command runners.
- The `tui` package has both logic tests (`tui_test.go`) and render tests (`tui_render_test.go`) with an ANSI-stripping helper.

## Toolchain quirks

- **Go version**: `go.mod` declares `go 1.26`, CI workflows pin `go-version: '1.26'`, and the README says "Go 1.22 or later". Trust the `go.mod` and CI as the executable source of truth; the project builds fine with Go 1.26+.
- **GoReleaser hooks**: `before.hooks` runs `go mod tidy` (not `go generate`). No `//go:generate` directives exist today.
- **Cross-platform**: CGO is disabled (`CGO_ENABLED=0`). Builds for linux/darwin/windows amd64+arm64 (Windows arm64 excluded).
- **Distribution**: GoReleaser also produces `.deb`/`.rpm` packages (`nfpms`) and publishes to Scoop (`lynicis/scoop-bucket`) and Homebrew (`lynicis/homebrew-tap`).

## Runtime behavior

- GitHub token resolution order: `--token` flag → `GITHUB_TOKEN` env var → `gh auth token` CLI. If all fail, the CLI warns and proceeds with unauthenticated rate limits (60 req/hr).
- Default scan target is `.github/workflows` in the current working directory when no `--path` is given.
- `--dry-run` prints a diff-style preview without writing files.
- `--no-tui` upgrades everything non-interactively (respects `--dry-run`).
- `--semver` / `-s` opts into full semver tag resolution (e.g., `v5.3.1`). **Default is major-tag mode** (e.g., `v5`), which prefers `^v\d+$` tags and falls back to highest semver if none exist.
- `--force` / `-f` upgrades past known breaking changes without prompting in non-interactive mode.
- Breaking-change detection: both TUI and no-TUI modes check the embedded registry before upgrading. The TUI shows a `⚠ breaking changes` badge and a detail view (`i` key). No-TUI mode uses TTY detection (`golang.org/x/term`) for interactive y/N prompts.
- API concurrency is capped at 5 parallel requests via a semaphore.
- Tag lookup fetches up to 300 tags (3 pages × 100) and filters for semver, skipping non-semver and SHA refs.

## CI / Release

- `.github/workflows/ci.yml` — builds, tests (`-race -coverprofile`), lints (`golangci-lint --timeout=5m`), and cross-compiles on macOS/Windows.
- `.github/workflows/release.yml` — triggered on `v*` tags, runs tests then GoReleaser. Uses `HOMEBREW_TAP_GITHUB_TOKEN` for Scoop/Homebrew publishing. Sets up QEMU + Buildx and logs into `ghcr.io` for Docker image publishing.
- `.github/workflows/update-homebrew-tap.yaml` — triggered on `v*` tags, updates the Homebrew formula in `lynicis/homebrew-tap`.
- GoReleaser config in `.goreleaser.yaml` injects version metadata via ldflags (`main.version`, `main.commit`, `main.date`).
- **Reproducible builds**: both `.goreleaser.yaml` and `.goreleaser.choco.yml` use `metadata.mod_timestamp` and `builds.mod_timestamp` set to `{{ .CommitTimestamp }}`, and `main.date` is set from `{{ .CommitDate }}`. This ensures the Windows ZIP archive produced by the Ubuntu release job and the Windows Chocolatey job are byte-identical, so the checksum embedded in the Chocolatey package matches the GitHub release artifact.

## Docker

- `Dockerfile` — single-stage Alpine image with pre-built binary from GoReleaser. Produces a ~19 MB image with ca-certificates.
- `.dockerignore` — excludes `.git/`, `*.md`, `dist/`, `testdata/`, coverage artifacts.
- GoReleaser uses `dockers_v2` to build a multi-arch manifest (linux/amd64 + linux/arm64) tagged `vX.Y.Z` and `latest` on `ghcr.io/lynicis/actup`.
- Release workflow logs into `ghcr.io` using `GITHUB_TOKEN` (no extra secret needed).
- Local Docker build: `docker build -t actup .`

## Dependency notes

- `go-github/v62` is the API client. If upgrading to v63+, ensure the API signatures haven't changed.
- `golang.org/x/mod` is used for semver parsing and sorting.
- `golang.org/x/oauth2` is the OAuth2 transport for GitHub API authentication.
- `golang.org/x/term` is used for TTY detection in non-interactive mode.
- `goccy/go-yaml` parses the breaking-change registry (not `gopkg.in/yaml.v3`).
- Bubble Tea, Bubbles, and Lipgloss are the TUI stack. Be careful with TUI model updates to avoid race conditions.

<!-- gitnexus:start -->
# GitNexus — Code Intelligence

This project is indexed by GitNexus as **actup** (461 symbols, 1061 relationships, 26 execution flows). Use the GitNexus MCP tools to understand code, assess impact, and navigate safely.

> Index stale? Run `node .gitnexus/run.cjs analyze` from the project root — it auto-selects an available runner. No `.gitnexus/run.cjs` yet? `npx gitnexus analyze` (npm 11 crash → `npm i -g gitnexus`; #1939).

## Always Do

- **MUST run impact analysis before editing any symbol.** Before modifying a function, class, or method, run `impact({target: "symbolName", direction: "upstream"})` and report the blast radius (direct callers, affected processes, risk level) to the user.
- **MUST run `detect_changes()` before committing** to verify your changes only affect expected symbols and execution flows. For regression review, compare against the default branch: `detect_changes({scope: "compare", base_ref: "main"})`.
- **MUST warn the user** if impact analysis returns HIGH or CRITICAL risk before proceeding with edits.
- When exploring unfamiliar code, use `query({query: "concept"})` to find execution flows instead of grepping. It returns process-grouped results ranked by relevance.
- When you need full context on a specific symbol — callers, callees, which execution flows it participates in — use `context({name: "symbolName"})`.

## Never Do

- NEVER edit a function, class, or method without first running `impact` on it.
- NEVER ignore HIGH or CRITICAL risk warnings from impact analysis.
- NEVER rename symbols with find-and-replace — use `rename` which understands the call graph.
- NEVER commit changes without running `detect_changes()` to check affected scope.

## Resources

| Resource | Use for |
|----------|---------|
| `gitnexus://repo/actup/context` | Codebase overview, check index freshness |
| `gitnexus://repo/actup/clusters` | All functional areas |
| `gitnexus://repo/actup/processes` | All execution flows |
| `gitnexus://repo/actup/process/{name}` | Step-by-step execution trace |

## CLI

| Task | Read this skill file |
|------|---------------------|
| Understand architecture / "How does X work?" | `.claude/skills/gitnexus/gitnexus-exploring/SKILL.md` |
| Blast radius / "What breaks if I change X?" | `.claude/skills/gitnexus/gitnexus-impact-analysis/SKILL.md` |
| Trace bugs / "Why is X failing?" | `.claude/skills/gitnexus/gitnexus-debugging/SKILL.md` |
| Rename / extract / split / refactor | `.claude/skills/gitnexus/gitnexus-refactoring/SKILL.md` |
| Tools, resources, schema reference | `.claude/skills/gitnexus/gitnexus-guide/SKILL.md` |
| Index, status, clean, wiki CLI commands | `.claude/skills/gitnexus/gitnexus-cli/SKILL.md` |

<!-- gitnexus:end -->
