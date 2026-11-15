<p align="center">
  <img src="logo.png" alt="actup logo" width="120">
</p>

<h1 align="center">actup</h1>

<p align="center">
  <strong>A version manager for GitHub Actions.</strong> Set, pin, sync, and upgrade
  <code>uses:</code> refs from one manifest, with breaking-change warnings and
  OSV.dev vulnerability checks interactively or in CI.
</p>

<p align="center">
  <a href="https://github.com/lynicis/actup/actions/workflows/ci.yml"><img src="https://img.shields.io/github/actions/workflow/status/lynicis/actup/ci.yml?style=for-the-badge&label=CI" alt="CI"></a>
  <a href="https://github.com/lynicis/actup/releases"><img src="https://img.shields.io/github/v/release/lynicis/actup?style=for-the-badge" alt="Release"></a>
  <a href="https://github.com/lynicis/gh-actup"><img src="https://img.shields.io/badge/gh%20extension-lynicis%2Fgh--actup-2ea44f?style=for-the-badge&logo=github" alt="GitHub CLI extension"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/github/go-mod/go-version/lynicis/actup?style=for-the-badge" alt="Go Version"></a>
  <a href="https://github.com/lynicis/actup/blob/main/LICENSE"><img src="https://img.shields.io/github/license/lynicis/actup.svg?style=for-the-badge" alt="License"></a>
</p>

<p align="center">
  <a href="https://github.com/lynicis/actup/issues/new?labels=bug&template=bug_report.md">Report Bug</a>
  &middot;
  <a href="https://github.com/lynicis/actup/issues/new?labels=enhancement&template=feature_request.md">Request Feature</a>
  &middot;
  <a href="https://github.com/lynicis/gh-actup">GitHub CLI Extension</a>
</p>

---

**Status:** actively maintained, pre-1.0. Flags and manifest schemas may change
between minor releases; breaking changes are documented in the
[changelog](CHANGELOG.md).

`actup` is a version manager for GitHub Actions, similar to `nvm` for Node or `fvm` for Flutter. Pin workflow versions in a `.actup.yaml` manifest, check for breaking changes and advisories, and upgrade across all workflows interactively or in CI. It is also available as a GitHub CLI extension via [`lynicis/gh-actup`](https://github.com/lynicis/gh-actup).

## Demo

![actup TUI: selecting upgrades, opening the breaking-change details for actions/cache, and applying the selected upgrades](docs/demo.gif)

The TUI lists each action with its current → latest version, flags `⚠ breaking changes` and `⚠ vulnerable` inline, opens details with `i`, and writes files only when you press `enter`.

## Why actup?

Action versions live inside every workflow file, making updates manual and uneven: teams rarely agree on when to bump, and major version jumps can alter workflow behavior. Dependabot and Renovate automate updates by opening scheduled pull requests. `actup` provides upgrade commands you run directly, locally or in CI: nothing changes until invoked, and each run produces a single reviewable diff.

| | `actup` | [Dependabot](https://docs.github.com/en/code-security/dependabot) | [Renovate](https://docs.renovatebot.com/) | [`gh actions-lock`](https://github.com/github/actions-lockfile) |
|---|---|---|---|---|
| What it does | Upgrades, pins, and syncs `uses:` refs from a manifest | Opens PRs when an action has a newer version | Opens PRs from configurable rules | Writes `actions.lock` for reproducible, transitive pinning |
| When it runs | On demand (locally or in CI) | Scheduled by GitHub | Scheduled or via webhook | On demand |
| Where versions live | `.actup.yaml` (source of truth) | `dependabot.yml` (update policy) | `renovate.json` (rules) | `actions.lock` (resolved state) |
| Breaking-change warnings | Embedded registry + OSV.dev advisories, checked before writing | Release notes in PR | Release notes in PR | None |

`gh actions-lock` and `actup` work together. `gh actions-lock` records resolved versions, including transitive dependencies, without modifying action references. `actup` upgrades direct workflow refs, then re-runs `gh actions-lock` to keep the lockfile current. `actup audit` checks direct and transitive lockfile dependencies against OSV.dev, while `actup ls` marks actions `[locked]` or `[lock stale]`. For scheduled automated PRs, use Dependabot or Renovate. Use `actup` when you want manual or CI-gated upgrades.

## Quick Start

Install `actup` first ([Installation](#installation) or `gh extension install lynicis/gh-actup`), then run these from your repository root:

```sh
# Bootstrap a manifest pinning the versions your workflows already use
actup init

# Set an action version across every workflow (and pin it in .actup.yaml)
actup use actions/checkout@v5

# Make workflows follow the manifest after editing it
actup sync

# Upgrade actions interactively in a TUI
actup

# Preview changes without touching files
actup --dry-run
```

## Installation

### 🤖 Install with AI

Copy and paste this prompt into your AI coding assistant (Claude Code, Cursor, GitHub Copilot, Antigravity, etc.) to automate installation and setup:

```text
Install actup on my system. First inspect my environment and ask me whether I want to install it as a GitHub CLI extension (`gh actup`) or as a standalone CLI (`actup`) (via Homebrew, Scoop, Go, or native packages). Once installed and verified, ask if I also want to configure the `actup mcp` server for my current AI assistant/editor, and if so, set up the appropriate MCP configuration. Refer to https://raw.githubusercontent.com/lynicis/actup/main/llms.txt for full instructions.
```

### GitHub CLI extension

Install the [`gh-actup`](https://github.com/lynicis/gh-actup) extension for GitHub CLI:

```sh
gh extension install lynicis/gh-actup
gh actup --help
```

Binaries are mirrored to the companion repository [lynicis/gh-actup](https://github.com/lynicis/gh-actup)
because `gh` only installs extensions from repositories named `gh-*`. Upgrade with
`gh extension upgrade actup` (32-bit Linux ARM is not supported by gh extensions; use the
`.deb` or `.rpm` packages instead).

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

### AI & Agent Setup

For LLMs and AI coding assistants automating the installation and configuration of `actup`, see [`llms.txt`](llms.txt) (or [`llm.txt`](llm.txt)).

## Before and after

Starting workflow:

```yaml
name: CI
on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v4
        with:
          go-version: '1.24'
      - uses: actions/cache@v3
        with:
          path: ~/.cache/go-build
          key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
      - uses: actions/upload-artifact@v3
        with:
          name: coverage
          path: coverage.out
      - run: go test ./...
```

`actup --no-tui` resolves the latest major tags and checks each jump against the breaking-change registry before writing:

```console
$ actup --no-tui
⚠ actions/cache: v3 → v6 has breaking changes:
  • cache v4 runs on Node 20; self-hosted runners must be v2.308.0 or newer.
  • cache v4 requires the cache service v2 backend.
  Apply anyway? [y/N] n
✓ actions/checkout: v4 → v7

⚠ actions/setup-go: v4 → v7 has breaking changes:
  • setup-go v5 runs on Node 20; self-hosted runners must be v2.308.0 or newer.
  Apply anyway? [y/N] n

⚠ actions/upload-artifact: v3 → v7 has breaking changes:
  • upload-artifact v4 runs on Node 20; self-hosted runners must be v2.308.0 or newer.
  • upload-artifact v4 artifact names are immutable, and hidden files are excluded by default.
  Apply anyway? [y/N] n

1 action upgraded, 0 skipped (up to date), 0 skipped (API error), 3 skipped (breaking changes)
```

Answering `y` applies that upgrade; answering `n` skips it. Running `actup --no-tui --force` applies remaining upgrades without prompting. The updated workflow preserves inputs, step order, and surrounding YAML:

```yaml
      - uses: actions/checkout@v7
      - uses: actions/setup-go@v7
      - uses: actions/cache@v6
      - uses: actions/upload-artifact@v7
```

`.actup.yaml` records the new versions, so `actup outdated`, `actup sync`, and the lockfile stay in step. Writes are atomic (temp file + rename), and `--dry-run` reports the same upgrades without touching anything.

## Features

- **Manifest-driven**: Declare versions in `.actup.yaml`; enforce them across workflows with `actup sync`.
- **Interactive TUI**: Review outdated actions, inspect breaking change notes, and select updates in a Bubble Tea interface.
- **Safety checks**: Embedded registry flags breaking changes before writing; OSV.dev integration reports known security advisories.
- **Flexible versioning**: Support for moving major tags (`v5`), exact semver (`v5.3.1`), and commit SHAs with inline comments (`@<sha> # v5`).
- **`actions.lock` support**: Audits direct and transitive dependencies, reports stale locks in `actup ls`, and auto-refreshes the lockfile.
- **CI and automation**: Headless `--no-tui` mode, `--dry-run` diffs, atomic writes via temporary files, and pre-commit hooks (`actup install-hooks`).
- **MCP server**: Built-in `actup mcp` exposes typed tools for AI coding assistants.

## Usage

Run `actup` from the root of any repository with GitHub Actions workflows:

```sh
# Interactive mode (default): opens TUI to select upgrades
actup

# Non-interactive mode: upgrades all actions automatically
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

- **Automatic re-lock**: When `actup`, `actup use`, `actup sync`, or `actup audit --fix` writes to workflow files, `actup` invokes `gh actions-lock` to refresh the lockfile. `--dry-run` skips re-locking. If `gh` or `gh-actions-lock` is missing, `actup` prints a warning and continues.
- **Transitive dependencies in `audit`**: `actup audit` also checks lockfile transitive dependencies alongside direct `uses:` refs. Terminal output marks them with `(transitive)`; `--json` sets `"transitive": true` on those findings. `audit --fix` updates direct workflow files and reports remaining transitive findings.
- **Lock status in `ls`**: `actup ls` shows `[locked]` when an action matches the lockfile and `[lock stale]` when the workflow and lockfile disagree.

## Version Manager

`actup` treats `.actup.yaml` as the manifest of action versions declared for your repository, similar to `.nvmrc` for Node.

| Command | nvm / fvm equivalent | What it does |
|---|---|---|
| `actup init` | `npm init` | Bootstrap `.actup.yaml` from the versions your workflows already use |
| `actup use <action>@<ref>` | `nvm use` / `fvm use` | Set an action's version across **all** workflows and pin it in `.actup.yaml` |
| `actup pin <action>[@<ref>]` | `nvm alias` | Record a pin in `.actup.yaml` without touching workflows |
| `actup unpin <action>` | (none) | Remove a pin from `.actup.yaml` |
| `actup sync` | `fvm install` | Apply all `.actup.yaml` pins to your workflows |
| `actup ls` | `nvm ls` | List actions in use, their versions, and pin status (`--outdated` adds latest versions) |
| `actup outdated` | `nvm outdated` | List actions that can be upgraded (exits non-zero if any) |
| `actup audit` | `npm audit` | Check action versions against known vulnerabilities (exits non-zero if any) |
| `actup` | `nvm upgrade` | Interactively upgrade actions to latest versions (TUI) |

### Ref resolution in `use`

| Argument | Behavior |
|---|---|
| `@v5` | Written verbatim as a moving major tag |
| `@5` | Resolved via GitHub API to the latest tag within major v5 (e.g. `v5.3.1`) |
| `@latest` / no `@ref` | Resolved to the latest tag (major tag, or full semver with `--semver`) |
| `@v5.3.1`, `@master` | Written verbatim |

`actup use` saves the pin to `.actup.yaml` by default. Pass `--no-save` to update workflows only, or `--dry-run` to preview changes.

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

## MCP

`actup mcp` runs a [Model Context Protocol](https://modelcontextprotocol.io) server over stdio, allowing AI coding assistants to inspect and update action versions through typed tools rather than CLI text scraping. Upgrades, pins, and config writes happen only when the assistant invokes the corresponding tool.

### Claude Code

```sh
claude mcp add actup -- actup mcp
```

### Codex

```sh
codex mcp add actup -- actup mcp
```

Or in `~/.codex/config.toml` (or `.codex/config.toml`):

```toml
[mcp_servers.actup]
command = "actup"
args = ["mcp"]
```

### Cursor, Gemini CLI, Antigravity, oh-my-pi

These clients share the same `mcpServers` JSON structure:

```json
{
  "mcpServers": {
    "actup": {
      "command": "actup",
      "args": ["mcp"]
    }
  }
}
```

|Client|Config file|
|---|---|
|Cursor|`~/.cursor/mcp.json` (project: `.cursor/mcp.json`)|
|Gemini CLI|`~/.gemini/settings.json` (project: `.gemini/settings.json`)|
|Antigravity CLI|`~/.gemini/config/mcp_config.json` (workspace: `.agents/mcp_config.json`)|
|oh-my-pi (`omp`)|`~/.omp/agent/mcp.json` (project: `.omp/mcp.json`), or `/mcp add` inside omp|

### OpenCode

`opencode.jsonc` (or `opencode mcp add actup -- actup mcp`):

```jsonc
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "servers": {
      "actup": {
        "type": "local",
        "command": ["actup", "mcp"]
      }
    }
  }
}
```

### VS Code (GitHub Copilot)

`.vscode/mcp.json`:

```json
{
  "servers": {
    "actup": {
      "type": "stdio",
      "command": "actup",
      "args": ["mcp"]
    }
  }
}
```

- Run the server from the repository root; tools discover `.github/workflows/`, `.actup.yaml`, and `actions.lock` relative to the working directory.
- The GitHub token is resolved automatically from `GITHUB_TOKEN` or `gh auth token` (never passed as a tool parameter).
- `upgrade_actions` skips actions with known breaking changes or vulnerable target versions unless called with `force: true`. Use `dry_run: true` to preview changes.
- MCP tools do not shell out to `gh actions-lock` directly; when `actions.lock` exists, results include a `lockfile_note` to refresh it.

|Tool|What it does|
|---|---|
|`list_actions`|List every action with versions, usage sites, pins, and lock status|
|`check_outdated`|Resolve latest versions and report outdated actions with breaking changes|
|`audit_vulnerabilities`|Audit direct and lockfile-transitive versions against OSV.dev|
|`upgrade_actions`|Upgrade workflow refs (specs or all, honoring pins) and record the result|
|`pin_action`|Pin an action to a version constraint in `.actup.yaml`|
|`unpin_action`|Remove an action's pin|
|`init_config`|Create `.actup.yaml` recording current versions|

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

`.actup.yaml` is the manifest of declared action versions for your workflows. `actup init` generates it, `actup use` and `actup pin` maintain it, and `actup outdated` checks against it. Editors supporting YAML language servers provide autocomplete and validation with the `$schema` directive:

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

`pins` defines target constraints; `actions` records current workflow state. Constraints:

- **Exact ref** (`v5`, `v5.3.1`, `master`): Applied verbatim.
- **Major number** (`4`): Resolved to the latest tag within that major via the GitHub API.
- **`skip`**: Excluded from upgrades and syncing.

With `sha: true`, `sync` and upgrades resolve each target tag to its commit SHA and write `owner/repo@<sha> # <tag>`. Pins still store the tag; a bare SHA without the trailing comment has no version and is skipped by vulnerability and breaking-change checks. `use --sha`, `sync --sha`, and `--sha=false` override the manifest setting.

A manifest without a `pins:` key is legacy: every `actions:` entry is treated as a constraint, exactly as before this split. Run `actup init --force` to migrate a legacy manifest once: its exact `actions:` entries become recorded state, and you can then add constraints under `pins:`.

**Precedence**: CLI flags > config file > built-in defaults.

Positional arguments are `path.Match` globs matched against `owner/repo` (union match; `skip` subtracts). Because `actup` accepts arbitrary positional filters, an unrecognized command argument like `actup udpate` is treated as a filter, failing immediately with a no-match error before network calls or TUI initialization. Quote globs in shell invocations: `actup 'actions/*'`.

## Pre-commit Hooks

Catch stale action versions before committing:

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
    rev: v0.10.1
    hooks:
      - id: actup-check
```

> The hook runs `actup outdated` and fails the commit if any actions can be upgraded.

## Breaking Changes

`actup` embeds a registry of known breaking changes between major versions. When an upgrade involves breaking changes:

- **TUI mode**: Displays a `⚠ breaking changes` badge (press `i` for details).
- **Non-interactive mode**: Prompts for confirmation on interactive terminals, or use `--force` to bypass prompts.

The registry lives in [`internal/breakingchanges/registry.yaml`](internal/breakingchanges/registry.yaml). If you encounter an unlisted breaking change, pull requests are welcome.

## Vulnerability Audit

`actup audit` checks every `uses:` reference in your workflows against the [OSV.dev](https://osv.dev) advisory database (GitHub Actions ecosystem). It reports the advisory ID, severity, and the first patched version:

```sh
actup audit            # human-readable summary
actup audit --json     # machine-readable output for CI
```

- Exits `1` when vulnerable action versions are found and `2` when the OSV database cannot be reached.
- Vulnerability matching runs client-side against OSV range data, matching major-tag refs (such as `@v4`) against semver advisories.
- Actions marked `skip` in `.actup.yaml` are excluded.
- The interactive TUI and `--no-tui` upgrade flows warn inline: the TUI shows a `⚠ vulnerable` badge (press `i` for details), and non-interactive runs print warnings before upgrading.
- `actup use`, `actup sync`, and `actup outdated` surface the same warnings. `use` and `sync` also verify target versions before writing, warning if the target version is affected by a known advisory.

## Roadmap

Planned features:
- [ ] Pre-upgrade hooks and custom validation scripts
- [ ] Automated pull request creation for CI workflows

See [open issues](https://github.com/lynicis/actup/issues) for planned work and feature discussions.

## Contributing

1. Fork the repository and create a branch.
2. Make your changes and add tests where appropriate.
3. Ensure tests and linter pass:
   ```sh
   make test && make lint
   ```
4. Submit a pull request.

## Related Repositories

- [lynicis/gh-actup](https://github.com/lynicis/gh-actup): GitHub CLI (`gh`) extension wrapper mirroring prebuilt binaries.
- [lynicis/homebrew-tap](https://github.com/lynicis/homebrew-tap): Homebrew tap formula repository (`brew install lynicis/tap/actup`).
- [lynicis/scoop-bucket](https://github.com/lynicis/scoop-bucket): Windows Scoop bucket manifest repository.

## License

Distributed under the MIT License. See [`LICENSE`](LICENSE) for details.

## Acknowledgments

Built with:
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) (TUI framework)
- [Cobra](https://github.com/spf13/cobra) (CLI framework)
- [go-github](https://github.com/google/go-github) (GitHub API client)
- [go-yaml](https://github.com/goccy/go-yaml) (YAML parser)
