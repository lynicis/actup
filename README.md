<p align="center">
  <img src="logo.png" alt="actup logo" width="120">
</p>

<h1 align="center">actup</h1>

<p align="center">
  <strong>A version manager for GitHub Actions.</strong> Set, pin, sync, and upgrade
  <code>uses:</code> refs from one manifest — with breaking-change warnings and
  OSV.dev vulnerability checks, interactively or in CI.
</p>

<p align="center">
  <a href="https://github.com/lynicis/actup/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/lynicis/actup/ci.yml?style=for-the-badge&label=CI" alt="CI"></a>
  <a href="https://github.com/lynicis/actup/releases"><img src="https://img.shields.io/github/v/release/lynicis/actup?style=for-the-badge" alt="Release"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/github/go-mod/go-version/lynicis/actup?style=for-the-badge" alt="Go Version"></a>
  <a href="https://github.com/lynicis/actup/blob/main/LICENSE"><img src="https://img.shields.io/github/license/lynicis/actup.svg?style=for-the-badge" alt="License"></a>
</p>

<p align="center">
  <a href="https://github.com/lynicis/actup/issues/new?labels=bug&template=bug_report.md">Report Bug</a>
  &middot;
  <a href="https://github.com/lynicis/actup/issues/new?labels=enhancement&template=feature_request.md">Request Feature</a>
</p>

---

**Status:** actively maintained, pre-1.0. Flags and the manifest schema may change
between minor releases; breaking changes are called out in the
[changelog](CHANGELOG.md).

`actup` is a **version manager for GitHub Actions** — like `nvm` is for Node or `fvm` for Flutter. Set and pin the versions your workflows use, keep a `.actup.yaml` manifest as the source of truth, and upgrade to the latest versions when you're ready — interactively via a terminal UI, or non-interactively in CI.

## Why actup?

- **Runs locally, on your schedule** — no bot to install and no PR queue. Unlike Dependabot or Renovate, `actup` rewrites workflow files only when you run it.
- **One manifest drives every workflow** — `.actup.yaml` is the source of truth, and `actup sync` applies it everywhere, with glob and group filters for scoped upgrades.
- **Checks before writes** — the embedded breaking-change registry and OSV.dev advisories are checked before anything is rewritten.
- **Complements `gh actions-lock`** — `actup` upgrades the refs; the lockfile integration re-locks, audits transitive dependencies, and shows lock status in `actup ls`.

## Quick Start

Install `actup` first ([Installation](#installation)), then run these from your repository root:

```sh
# Bootstrap a manifest pinning the versions your workflows already use
actup init

# Set an action version across every workflow (and pin it in .actup.yaml)
actup use actions/checkout@v5

# Make workflows follow the manifest after editing it
actup sync

# Upgrade everything to the latest versions — pick in a TUI
actup

# Preview changes without touching files
actup --dry-run
```

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

**Debian / Ubuntu**: download the `.deb` for your architecture from the [latest release](https://github.com/lynicis/actup/releases/latest), then:
```sh
sudo dpkg -i actup_*_linux_amd64.deb
```

**Fedora / RHEL / CentOS**: download the `.rpm` for your architecture from the [latest release](https://github.com/lynicis/actup/releases/latest), then:
```sh
sudo rpm -i actup_*_linux_amd64.rpm
```

> Replace `amd64` with `arm64` for ARM systems.

### Docker

Images are published to `ghcr.io/lynicis/actup` for `linux/amd64` and `linux/arm64`. The image runs as UID 65532; pass `--user "$(id -u):$(id -g)"` when it must write to a bind-mounted directory.

```sh
# Show help
docker run --rm ghcr.io/lynicis/actup:latest --help

# Scan current directory
docker run --rm --user "$(id -u):$(id -g)" \
  -v "$PWD:/workdir" -w /workdir ghcr.io/lynicis/actup:latest

# With GitHub token for higher rate limits
docker run --rm --user "$(id -u):$(id -g)" \
  -v "$PWD:/workdir" -w /workdir \
  -e GITHUB_TOKEN \
  ghcr.io/lynicis/actup:latest
```

### Build from Source

Requires Go 1.27 or later.

```sh
git clone https://github.com/lynicis/actup.git
cd actup
make build
make install   # optional, installs to $GOPATH/bin
```

### Pre-built Binaries

Download binaries for Linux, macOS, or Windows from the [Releases](https://github.com/lynicis/actup/releases) page.

## Features

- **Version manager commands** — `use`, `pin`/`unpin`, `ls`, `init`, `sync`, and `outdated` give you nvm-style control over action versions
- **Vulnerability audit** — `actup audit` checks every action version against the OSV.dev advisory database (GitHub Actions ecosystem); the TUI and non-interactive flows also warn inline
- **`actions.lock` integration** — reads `.github/workflows/actions.lock`, scans transitive dependencies in `actup audit`, shows `[locked]` / `[lock stale]` in `actup ls`, and re-locks automatically after workflow edits
- **Manifest as source of truth** — `.actup.yaml` declares your action versions; `sync` applies it to every workflow
- **Workflow discovery** — recursively finds `.yml` / `.yaml` files under `.github/workflows/`
- **Interactive TUI** — Bubble Tea–powered checklist with select-all, per-action toggle, and breaking-change detail view
- **Non-interactive mode** — `--no-tui` upgrades everything; prompts on TTY for breaking changes (override with `--force`)
- **Breaking-change detection** — embedded registry of known breaking changes between major versions
- **Major-tag or full semver** — default resolves `v5`-style tags; `--semver` opts into `v5.3.1` pinning
- **Commit SHA pinning** — `--sha` (or `sha: true` in the manifest) writes `@<sha> # <tag>`; existing SHA pins stay pinned
- **Dry-run** — `--dry-run` previews a diff without touching files
- **Atomic writes** — temp-file + rename prevents partial updates
- **Concurrent API** — up to 5 parallel GitHub requests with in-memory caching
- **GitHub CLI fallback** — auto-discovers tokens from `gh auth token` when no env var is set

## Demo

[![asciicast](https://asciinema.org/a/BoLQP1Gt9L0CXvUR.svg?size=big)](https://asciinema.org/a/BoLQP1Gt9L0CXvUR)

## Usage

Run `actup` from the root of any repository with GitHub Actions workflows:

```sh
# Interactive mode (default) — opens TUI to select upgrades
actup

# Non-interactive mode — upgrades all actions automatically
actup --no-tui

# Preview changes without writing files
actup --dry-run

# Upgrade only matching actions (quote globs so the shell does not expand them)
actup 'actions/*' --dry-run

# Upgrade only actions in a manifest group
actup --group ci

# Combine positional filters and groups (union)
actup 'docker/*' --group ci --no-tui

# Use full semver tags instead of major tags (e.g., v5.3.1 instead of v5)
actup --semver

# Write full commit SHAs with the tag in a trailing comment (@<sha> # v5)
actup --sha

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

### `actions.lock` Integration

If your repository uses [`actions-lock`](https://github.com/github/actions-lockfile) (the `gh actions-lock` extension) and has a `.github/workflows/actions.lock` file, `actup` integrates with it:

- **Automatic re-lock** — whenever `actup`, `actup use`, `actup sync`, or `actup audit --fix` writes to workflow files, `actup` runs `gh actions-lock` to refresh the lockfile. `--dry-run` never re-locks. If `gh` or the `gh-actions-lock` extension is missing, `actup` prints a warning and continues.
- **Transitive dependencies in `audit`** — `actup audit` also checks the lockfile's transitive dependencies (not just direct `uses:` refs). Human-readable output marks them with `(transitive)`; `--json` sets `"transitive": true` on those findings. `audit --fix` rewrites only direct workflow files and reports any remaining transitive findings.
- **Lock status in `ls`** — `actup ls` shows `[locked]` when an action matches the lockfile and `[lock stale]` when the workflow and lockfile disagree.

## Version Manager

`actup` treats `.actup.yaml` as the manifest of the action versions your repository declares — the same role `.nvmrc` plays for Node.

| Command | nvm / fvm equivalent | What it does |
|---|---|---|
| `actup init` | `npm init` | Bootstrap `.actup.yaml` from the versions your workflows already use |
| `actup use <action>@<ref>` | `nvm use` / `fvm use` | Set an action's version across **all** workflows and pin it in `.actup.yaml` |
| `actup pin <action>[@<ref>]` | `nvm alias` | Record a pin in `.actup.yaml` without touching workflows |
| `actup unpin <action>` | — | Remove a pin from `.actup.yaml` |
| `actup sync` | `fvm install` | Apply all `.actup.yaml` pins to your workflows |
| `actup ls` | `nvm ls` | List actions in use, their versions, and pin status (`--outdated` adds latest versions) |
| `actup outdated` | `nvm outdated` | List actions that can be upgraded (exits non-zero if any) |
| `actup audit` | `npm audit` | Check action versions against known vulnerabilities (exits non-zero if any) |
| `actup` | `nvm upgrade` | Interactively upgrade everything to the latest versions (TUI) |

### Ref resolution in `use`

| Argument | Behavior |
|---|---|
| `@v5` | Written verbatim — the standard moving major tag |
| `@5` | Resolved via the GitHub API to the latest tag within major v5 (e.g. `v5.3.1`) |
| `@latest` / no `@ref` | Resolved to the latest tag (major tag, or full semver with `--semver`) |
| `@v5.3.1`, `@master` | Written verbatim |

`use` saves the pin to `.actup.yaml` by default so your manifest stays the source of truth; pass `--no-save` to only update workflows, or `--dry-run` to change nothing.

### Typical workflow

```sh
actup init                        # 1. declare current versions
actup use actions/checkout@v5     # 2. switch an action everywhere
# 3. ...or edit .actup.yaml by hand, then apply it:
actup sync
actup ls --outdated               # 4. review what could be upgraded
actup                             # 5. upgrade everything interactively
```

## AI Skill

You can also use `actup`'s workflow-upgrading logic as an Agent Skill in compatible AI coding agents (Claude Code, OpenCode, Cursor, etc.).

### Install

Via `gh skill` (agentskills.io):

```sh
gh skill install lynicis/actup github-actions-updater
```

Via `npx skills` (skills.sh):

```sh
npx skills add lynicis/actup --skill github-actions-updater
```

### Usage

Once installed, ask your agent:

> "Update my GitHub Actions"
> "Which actions are out of date?"
> "Bump actions in my workflows"

The skill scans your `.github/workflows/*.yml` files, resolves the latest version of each action from GitHub, warns about known breaking changes, and produces a diff-style report with suggested upgrades. It does **not** edit files unless you explicitly ask.

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

`.actup.yaml` is your action version manifest — the declared source of truth that `actup sync` applies to your workflows. `actup init` generates it, `actup use` and `actup pin` keep it up to date, and `actup outdated` honors it. Editors supporting YAML language servers will provide autocomplete and validation if you add the `$schema` directive:

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/lynicis/actup/main/schema.json

# Global default major version (overridden by --semver flag)
major: 4

# Write refs as full commit SHAs with the tag in a trailing comment
sha: true

# Version constraints you choose (major number, exact ref, or skip)
pins:
  actions/checkout: 4           # latest v4.x.x (resolved at sync/upgrade time)
  actions/setup-go: v5.3.0      # exact version
  some-org/some-action: skip    # exclude from upgrades

# Recorded workflow state (the lock); actup rewrites this after upgrades
actions:
  actions/checkout: v4.2.2
  actions/setup-go: v5.3.0

# Optional named action-glob groups selectable with `actup --group <name>`
groups:
  ci:
    - actions/*
    - docker/*
  release:
    - softprops/action-gh-release
```

`pins` holds what you asked for; `actions` records what your workflows currently use. Constraints:

- **Exact ref** (`v5`, `v5.3.1`, `master`) — applied verbatim
- **Major number** (`4`) — resolved to the latest tag within that major via the GitHub API
- **`skip`** — the action is excluded from upgrades and syncing

With `sha: true`, `sync` and upgrades resolve each target tag to its commit SHA and write `owner/repo@<sha> # <tag>`. Pins still store the tag; a bare SHA without the trailing comment has no version and is skipped by vulnerability and breaking-change checks. `use --sha`, `sync --sha`, and `--sha=false` override the manifest setting.

A manifest without a `pins:` key is legacy: every `actions:` entry is treated as a constraint, exactly as before this split. Run `actup init --force` to migrate a legacy manifest once: its exact `actions:` entries become recorded state, and you can then add constraints under `pins:`.

**Precedence**: CLI flags > config file > built-in defaults.

Positional arguments are `path.Match` globs matched against `owner/repo` (the search is a union; `skip` still subtracts afterwards). Because `actup` accepts arbitrary positional filters, a typo such as `actup udpate` is treated as a filter rather than an unknown command — it fails with a no-match error before any network call or TUI start. Quote globs in your shell: `actup 'actions/*'`.

## Pre-commit Hooks

Catch stale action versions before they land in a commit — two options:

### `actup install-hooks` (built-in)

Installs a plain `pre-commit` hook that runs `actup outdated` on every commit:

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
    rev: v0.7.0
    hooks:
      - id: actup-check
```

> The hook runs `actup outdated` and fails the commit if any actions can be upgraded.

## Breaking Changes

`actup` includes an embedded registry of known breaking changes between major versions. When an upgrade involves breaking changes:

- **TUI mode**: Shows a `⚠ breaking changes` badge — press `i` for details
- **Non-interactive mode**: Prompts for confirmation on TTY, or use `--force` to skip prompts

The registry lives in [`internal/breakingchanges/registry.yaml`](internal/breakingchanges/registry.yaml). Contributions welcome — if you encounter a breaking change not in the registry, please open a PR.

## Vulnerability Audit

`actup audit` checks every `uses:` reference in your workflows against the [OSV.dev](https://osv.dev) advisory database (GitHub Actions ecosystem). It reports the advisory ID, severity, and the first fixed version:

```bash
actup audit            # human-readable output
actup audit --json     # machine-readable output for CI
```

- Exits `1` when vulnerable action versions are found and `2` when the OSV database cannot be reached.
- Vulnerability matching happens client-side against OSV range data, so major-tag refs (e.g. `@v4`) are matched correctly even though OSV only indexes full semver tags.
- Actions marked `skip` in `.actup.yaml` are excluded.
- The interactive TUI and `--no-tui` upgrade flows also warn inline: the TUI shows a `⚠ vulnerable` badge (press `i` for details), and non-interactive runs print warnings before upgrading.
- `actup use`, `actup sync`, and `actup outdated` surface the same warnings, and `use`/`sync` also check the **target version** you're about to pin: if that version is itself affected by a known advisory, you'll see it before anything is written (interactive `y/N` prompt on a TTY, notice + proceed in CI, auto-proceed with `--force`/`--dry-run`).

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
- [x] Vulnerability audit against OSV.dev (`actup audit`)
- [x] Grouped / filtered upgrades (`actup 'actions/*'`, `--group`)
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
