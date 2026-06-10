# actup

<p align="center">
  <strong>Keep your GitHub Actions up to date — interactively, safely, and fast.</strong>
</p>

<p align="center">
  <a href="https://github.com/lynicis/actup/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/lynicis/actup/ci.yml?style=for-the-badge&label=CI" alt="CI"></a>
  <a href="https://github.com/lynicis/actup/releases"><img src="https://img.shields.io/github/v/release/lynicis/actup?style=for-the-badge" alt="Release"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/github/go-mod/go-version/lynicis/actup?style=for-the-badge" alt="Go Version"></a>
  <a href="https://github.com/lynicis/actup/blob/main/LICENSE"><img src="https://img.shields.io/github/license/lynicis/actup.svg?style=for-the-badge" alt="License"></a>
</p>

<p align="center">
  <a href="https://github.com/lynicis/actup/issues/new?labels=bug&template=bug-report---.md">Report Bug</a>
  &middot;
  <a href="https://github.com/lynicis/actup/issues/new?labels=enhancement&template=feature-request---.md">Request Feature</a>
</p>

---

`actup` scans your GitHub Actions workflows and upgrades action references to their latest versions. Pick upgrades interactively via a terminal UI, or run it non-interactively in CI. It detects known breaking changes between major versions and warns you before upgrading.

## Features

- **Workflow discovery** — recursively finds `.yml` / `.yaml` files under `.github/workflows/`
- **Interactive TUI** — Bubble Tea–powered checklist with select-all, per-action toggle, and breaking-change detail view
- **Non-interactive mode** — `--no-tui` upgrades everything; prompts on TTY for breaking changes (override with `--force`)
- **Breaking-change detection** — embedded registry of known breaking changes between major versions
- **Major-tag or full semver** — default resolves `v5`-style tags; `--semver` opts into `v5.3.1` pinning
- **Dry-run** — `--dry-run` previews a diff without touching files
- **Atomic writes** — temp-file + rename prevents partial updates
- **Concurrent API** — up to 5 parallel GitHub requests with in-memory caching
- **GitHub CLI fallback** — auto-discovers tokens from `gh auth token` when no env var is set
- **Config file** — optional `.actup.yaml` for per-action pins, overrides, and exclusions

## Demo

[![asciicast](https://asciinema.org/a/F2yUEbcAaNawOtHp.svg)](https://asciinema.org/a/F2yUEbcAaNawOtHp)

## Table of Contents

- [Installation](#installation)
- [Usage](#usage)
- [Configuration](#configuration)
- [Pre-commit Hooks](#pre-commit-hooks)
- [Breaking Changes](#breaking-changes)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)
- [Acknowledgments](#acknowledgments)

## Installation

### Package Managers

**Homebrew** (macOS & Linux):
```sh
brew tap lynicis/tap
brew install actup
```

**Scoop** (Windows):
```powershell
scoop bucket add actup https://github.com/lynicis/scoop-bucket.git
scoop install actup
```

**Go install**:
```sh
go install github.com/lynicis/actup@latest
```

**Debian / Ubuntu**:
```sh
curl -LO https://github.com/lynicis/actup/releases/latest/download/actup_latest_linux_amd64.deb
sudo dpkg -i actup_latest_linux_amd64.deb
```

**Fedora / RHEL / CentOS**:
```sh
curl -LO https://github.com/lynicis/actup/releases/latest/download/actup_latest_linux_amd64.rpm
sudo rpm -i actup_latest_linux_amd64.rpm
```

> Replace `amd64` with `arm64` for ARM systems.

### Docker

Images are published to `ghcr.io/lynicis/actup` for `linux/amd64` and `linux/arm64`:

```sh
# Show help
docker run --rm ghcr.io/lynicis/actup:latest --help

# Scan current directory
docker run --rm -v "$PWD:/workdir" -w /workdir ghcr.io/lynicis/actup:latest

# With GitHub token for higher rate limits
docker run --rm -v "$PWD:/workdir" -w /workdir \
  -e GITHUB_TOKEN \
  ghcr.io/lynicis/actup:latest
```

### Build from Source

Requires Go 1.26 or later.

```sh
git clone https://github.com/lynicis/actup.git
cd actup
make build
make install   # optional, installs to $GOPATH/bin
```

### Pre-built Binaries

Download binaries for Linux, macOS, or Windows from the [Releases](https://github.com/lynicis/actup/releases) page.

## Usage

Run `actup` from the root of any repository with GitHub Actions workflows:

```sh
# Interactive mode (default) — opens TUI to select upgrades
actup

# Non-interactive mode — upgrades all actions automatically
actup --no-tui

# Preview changes without writing files
actup --dry-run

# Use full semver tags instead of major tags (e.g., v5.3.1 instead of v5)
actup --semver

# Force upgrades past known breaking changes (non-interactive mode)
actup --no-tui --force

# Scan custom paths
actup -p ./my-workflows -p ./another-path

# Provide a GitHub token for higher rate limits (5,000 req/hr vs 60)
actup -t $GITHUB_TOKEN
# or set the environment variable
export GITHUB_TOKEN=ghp_xxx
actup
```

## Configuration

### GitHub Token Resolution

`actup` resolves GitHub tokens in this order:

1. `--token` flag
2. `GITHUB_TOKEN` environment variable
3. `gh auth token` from the [GitHub CLI](https://cli.github.com/)

Authenticated requests get **5,000 API calls/hour** vs 60 for unauthenticated. To set up `gh`:

```sh
gh auth login
```

### Config File (`.actup.yaml`)

Place an optional `.actup.yaml` in your project root for persistent overrides:

```yaml
# Global default major version (overridden by --semver flag)
major: 4

# Per-action overrides
actions:
  actions/checkout: 4           # pin to latest v4.x.x
  actions/setup-go: v5.3.0     # pin to exact version
  some-org/some-action: skip   # exclude from upgrades
```

**Precedence**: CLI flags > config file > built-in defaults.

## Pre-commit Hooks

Catch stale action versions before they land in a commit — two options:

### `actup install-hooks` (built-in)

Installs a plain `pre-commit` hook that runs `actup --check --no-tui` on every commit:

```sh
actup install-hooks              # install the hook
actup install-hooks --dry-run    # preview before installing
actup install-hooks -f           # overwrite existing hook
actup install-hooks --uninstall  # remove the hook
```

### pre-commit framework

Add to your `.pre-commit-config.yaml`:

```yaml
repos:
  - repo: https://github.com/lynicis/actup
    rev: v0.5.0
    hooks:
      - id: actup-check
```

> The hook runs `actup --check --no-tui` and fails the commit if any actions can be upgraded. Omit `rev` to track `main` for the latest version.

## Breaking Changes

`actup` includes an embedded registry of known breaking changes between major versions. When an upgrade involves breaking changes:

- **TUI mode**: Shows a `⚠ breaking changes` badge — press `i` for details
- **Non-interactive mode**: Prompts for confirmation on TTY, or use `--force` to skip prompts

The registry lives in [`internal/breakingchanges/registry.yaml`](internal/breakingchanges/registry.yaml). Contributions welcome — if you encounter a breaking change not in the registry, please open a PR.

## Roadmap

- [x] Interactive TUI with checklist selection
- [x] Non-interactive (`--no-tui`) mode
- [x] Dry-run support
- [x] Concurrent GitHub API calls with rate-limit awareness
- [x] Cross-platform builds (Linux, macOS, Windows)
- [x] Pre-commit hooks for CI integration
- [x] Major version pinning (`--major`)
- [x] Config file support (`.actup.yaml`)
- [x] Breaking-change detection with embedded registry
- [x] Full semver tag resolution (`--semver`)
- [ ] Integration with `dependabot`-style grouped updates
- [ ] Pre-upgrade hooks / custom validation

See [open issues](https://github.com/lynicis/actup/issues) for the full list of proposed features and known issues.

## Contributing

Contributions are welcome! Here's how to get started:

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Run tests and lint: `make test && make lint`
5. Commit with a descriptive message (`git commit -m 'feat: add amazing feature'`)
6. Push to your fork (`git push origin feature/amazing-feature`)
7. Open a pull request

Please ensure your code passes existing tests and follows the project's style conventions.

## License

Distributed under the MIT License. See [`LICENSE`](LICENSE) for details.

## Acknowledgments

Built with these excellent libraries:

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — TUI framework
- [Cobra](https://github.com/spf13/cobra) — CLI framework
- [go-github](https://github.com/google/go-github) — GitHub API client
- [go-yaml](https://github.com/goccy/go-yaml) — YAML parser
