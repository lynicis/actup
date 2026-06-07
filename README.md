# actup

<p align="center">Upgrade GitHub Actions versions interactively from your terminal</p>

<p align="center">
  <a href="https://github.com/lynicis/actup/graphs/contributors"><img src="https://img.shields.io/github/contributors/lynicis/actup.svg?style=for-the-badge" alt="Contributors"></a>
  <a href="https://github.com/lynicis/actup/network/members"><img src="https://img.shields.io/github/forks/lynicis/actup.svg?style=for-the-badge" alt="Forks"></a>
  <a href="https://github.com/lynicis/actup/stargazers"><img src="https://img.shields.io/github/stars/lynicis/actup.svg?style=for-the-badge" alt="Stargazers"></a>
  <a href="https://github.com/lynicis/actup/issues"><img src="https://img.shields.io/github/issues/lynicis/actup.svg?style=for-the-badge" alt="Issues"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/github/go-mod/go-version/lynicis/actup?style=for-the-badge" alt="Go Version"></a>
  <a href="https://github.com/lynicis/actup/blob/main/LICENSE"><img src="https://img.shields.io/github/license/lynicis/actup.svg?style=for-the-badge" alt="License"></a>
</p>

<p align="center">
  <a href="https://github.com/lynicis/actup"><strong>Explore the docs &raquo;</strong></a>
  <br />
  <a href="https://github.com/lynicis/actup/issues/new?labels=bug&template=bug-report---.md">Report Bug</a>
  &middot;
  <a href="https://github.com/lynicis/actup/issues/new?labels=enhancement&template=feature-request---.md">Request Feature</a>
</p>

<br />

`actup` is a CLI tool that scans your GitHub Actions workflow files (`.github/workflows/*.yml`) and upgrades action versions to their latest semver tags. It provides both an interactive terminal UI (TUI) powered by Bubble Tea for cherry-picking upgrades, and a non-interactive mode for CI/automation.

## Features

- **Automatic discovery** of workflow files in `.github/workflows/`
- **Interactive TUI** with a checkbox list to select which actions to upgrade
- **Non-interactive mode** (`--no-tui`) for automated upgrades
- **Dry-run support** (`--dry-run`) to preview changes without writing files
- **Atomic file edits** via temp-file + rename to prevent corruption
- **Semver-aware** — fetches and sorts tags by semantic versioning
- **Concurrent API requests** with built-in rate-limiting and caching

## Demo

[![asciicast](https://asciinema.org/a/MKNN1peBjDuV1Ujb.svg)](https://asciinema.org/a/MKNN1peBjDuV1Ujb)

## Table of Contents

- [Demo](#demo)
- [Installation](#installation)
- [Usage](#usage)
- [Roadmap](#roadmap)
- [Contributing](#contributing)
- [License](#license)
- [Acknowledgments](#acknowledgments)

## Installation

### Package managers

**go install:**
```sh
go install github.com/lynicis/actup@latest
```

**Homebrew (macOS & Linux):**
```sh
brew tap lynicis/tap
brew install actup
```

**Scoop (Windows):**
```powershell
scoop bucket add actup https://github.com/lynicis/scoop-bucket.git
scoop install actup
```

**Debian / Ubuntu:**
```sh
curl -LO https://github.com/lynicis/actup/releases/latest/download/actup_latest_linux_amd64.deb
sudo dpkg -i actup_latest_linux_amd64.deb
```

**Fedora / RHEL / CentOS:**
```sh
curl -LO https://github.com/lynicis/actup/releases/latest/download/actup_latest_linux_amd64.rpm
sudo rpm -i actup_latest_linux_amd64.rpm
```

> Replace `amd64` with `arm64` if you're on an ARM machine.

### Build from source

Requires Go 1.22 or later.

```sh
git clone https://github.com/lynicis/actup.git
cd actup
make build
make install   # optional, installs to $GOPATH/bin
```

### Manual binary

Grab a pre-built binary for Linux, macOS, or Windows from the [Releases](https://github.com/lynicis/actup/releases) page.

## Usage

Run `actup` from the root of any repository containing GitHub Actions workflows:

```sh
# Interactive mode (default) — opens a TUI to select upgrades
actup

# Non-interactive mode — upgrades everything automatically
actup --no-tui

# Preview changes without writing files
actup --dry-run

# Scan custom paths
actup -p ./my-workflows -p ./another-workflows

# Provide a GitHub token for higher rate limits
actup -t $GITHUB_TOKEN
# or
export GITHUB_TOKEN=ghp_xxx
actup
```

### Interactive TUI Controls

| Key | Action |
|-----|--------|
| `Space` | Toggle selection of an action |
| `a` | Select all upgradable actions |
| `n` | Deselect all |
| `Enter` | Apply selected upgrades |
| `q` / `Ctrl+C` | Quit |

### Example Output

```
  actup — 5 actions found across 3 files

  [✓] actions/checkout@v3       → v4 (3 files)
  [✓] actions/setup-go@v4       → v5 (2 files)
  [⏭] actions/cache@v4           (up to date)
  [✓] golangci/golangci-lint-action@v3 → v6 (1 file)
  [⚠] some-org/some-action@v1    (API error)

  [space] toggle  [a] all  [n] none  [enter] apply  [q] quit
```

## Roadmap

- [x] Interactive TUI with checklist selection
- [x] Non-interactive (`--no-tui`) mode
- [x] Dry-run support
- [x] Concurrent GitHub API calls with rate-limit awareness
- [x] Cross-platform builds (Linux, macOS, Windows)
- [ ] Add support for pinning to specific major versions (`--major`)
- [ ] Config file support (`.actup.yaml`)
- [ ] Integration with `dependabot`-style grouped updates
- [ ] Pre-upgrade hooks / custom validation

See the [open issues](https://github.com/lynicis/actup/issues) for a full list of proposed features and known issues.

## Contributing

1. Fork the Project
2. Create your Feature Branch (`git checkout -b feature/AmazingFeature`)
3. Commit your Changes (`git commit -m 'feat: add some AmazingFeature'`)
4. Push to the Branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

Please make sure your code passes the existing tests and lint checks:

```sh
make test
make lint
```

## License

Distributed under the MIT License. See `LICENSE` for more information.

## Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — the TUI framework that powers the interactive interface
- [Cobra](https://github.com/spf13/cobra) — CLI framework for Go
- [go-github](https://github.com/google/go-github) — GitHub API client library


