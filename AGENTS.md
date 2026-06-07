# AGENTS.md — actup

> Repo-specific guidance for AI agents working in this codebase.  
> Every entry should answer: *"Would an agent likely miss this without help?"*

## What this is

`actup` is a Go CLI that scans GitHub Actions workflow files and upgrades action versions interactively via a TUI, or non-interactively with `--no-tui`.

## Architecture

Standard Go CLI layout using Cobra:

- `main.go` → `cmd/root.go` orchestrates packages
- `internal/scanner` — discovers `.yml`/`.yaml` workflow files (default: `.github/workflows/`)
- `internal/parser` — extracts `uses:` refs; skips `./local`, `docker://`, and full SHA commits
- `internal/github` — GitHub API client (`go-github/v62`); fetches latest semver tags with in-memory cache
- `internal/tui` — Bubble Tea interactive checklist for selecting upgrades
- `internal/upgrader` — atomic file edits via temp-file + rename

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

# Install to GOPATH/bin
make install               # go install .
```

## Testing

- All tests use `t.TempDir()` and write temporary workflow files — safe to run in parallel, no external services needed.
- Run a single package: `go test -v ./internal/parser`
- The `github` package has no unit tests (it calls the live GitHub API).

## Toolchain quirks

- **Go version**: `go.mod` declares `go 1.26`, CI workflows pin `go-version: '1.26'`, and the README says "Go 1.22 or later". Trust the `go.mod` and CI as the executable source of truth; the project builds fine with Go 1.26+.
- **Codegen hook**: GoReleaser runs `go generate ./...` before builds. There are no `//go:generate` directives today, but any new ones will be executed during release.
- **Cross-platform**: CGO is disabled (`CGO_ENABLED=0`). Builds for linux/darwin/windows amd64+arm64 (Windows arm64 excluded).

## Runtime behavior

- GitHub token: `--token` flag or `GITHUB_TOKEN` env var. If missing, the CLI warns and proceeds with unauthenticated rate limits (60 req/hr).
- Default scan target is `.github/workflows` in the current working directory when no `--path` is given.
- `--dry-run` prints a diff-style preview without writing files.
- `--no-tui` upgrades everything non-interactively (respects `--dry-run`).
- API concurrency is capped at 5 parallel requests via a semaphore.
- Tag lookup fetches up to 300 tags (3 pages × 100) and filters for semver, skipping non-semver and SHA refs.

## CI / Release

- `.github/workflows/ci.yml` — builds, tests (`-race -coverprofile`), lints (`golangci-lint --timeout=5m`), and cross-compiles on macOS/Windows.
- `.github/workflows/release.yml` — triggered on `v*` tags, runs tests then GoReleaser.
- GoReleaser config in `.goreleaser.yaml` injects version metadata via ldflags (`main.version`, `main.commit`, `main.date`).

## Dependency notes

- `go-github/v62` is the API client. If upgrading to v63+, ensure the API signatures haven't changed.
- `golang.org/x/mod` is used for semver parsing and sorting.
- `golang.org/x/oauth2` is the OAuth2 transport for GitHub API authentication.
- Bubble Tea, Bubbles, and Lipgloss are the TUI stack. Be careful with TUI model updates to avoid race conditions.
