# actup Reference

Detailed reference for `actup` CLI flags, exit codes, `.actup.yaml` manifest semantics, MCP tool schemas, and `actions.lock` integration.

## CLI Command & Flag Reference

Every subcommand defines its own local flags. Replace `actup` with `gh actup` when installed as a GitHub CLI extension.

### Root Upgrade (`actup [filter...] [flags]`)

Upgrades discovered workflow actions. Always pass `--no-tui` in agent environments.

| Flag | Purpose |
|------|---------|
| `--no-tui` | Run non-interactively instead of opening the Bubble Tea TUI |
| `--dry-run` | Preview changes without writing files |
| `-f, --force` | Apply upgrades with known breaking changes without prompting |
| `--group <name>` | Limit to actions in a `.actup.yaml` group (repeatable; unioned with positional globs) |
| `-m, --major <int>` | Constrain upgrades to a specific major version (e.g. `4` for latest `v4.x.x`) |
| `-p, --path <path>` | Scan custom workflow file or directory (repeatable) |
| `-s, --semver` | Resolve full semver tags (`v5.3.1`) instead of moving major tags (`v5`) |
| `--sha` | Write full 40-char commit SHAs with `# <tag>` trailing comment |
| `-t, --token <pat>` | Explicit GitHub PAT (overrides `GITHUB_TOKEN` and `gh auth token`) |

Quote positional `path.Match` globs in shell invocations: `actup 'actions/*' --no-tui --dry-run`.

### Subcommands

| Command | Key flags | Notes |
|---------|-----------|-------|
| `actup init` | `--dry-run`, `-f/--force`, `-p/--path` | Generates `.actup.yaml` from current workflow versions. `--force` merges into an existing manifest and migrates legacy `actions:`-only manifests. |
| `actup use <action>[@<ref>]...` | `--dry-run`, `-f/--force`, `--no-save`, `-p/--path`, `-s/--semver`, `--sha`, `-t/--token` | Sets version across all workflows and pins in `.actup.yaml` (omit saving with `--no-save`). |
| `actup pin <action>[@<ref>]` | `--dry-run`, `-p/--path` | Records a constraint in `.actup.yaml` without editing workflows. Omitting `@<ref>` pins the version currently in use. Use `@skip` to exclude from upgrades. |
| `actup unpin <action>` (alias `rm`) | `--dry-run` | Removes an action's pin from `.actup.yaml`. |
| `actup sync` (alias `install`) | `--dry-run`, `-f/--force`, `-p/--path`, `--sha`, `-t/--token` | Applies all `.actup.yaml` pins to workflows. |
| `actup ls [action]` | `--outdated`, `-p/--path`, `-s/--semver`, `-t/--token` | Offline by default. `--outdated` queries GitHub for latest versions. Passing an action name lists every `file:line` site. |
| `actup outdated` | `--group`, `-m/--major`, `-p/--path`, `-s/--semver`, `-t/--token` | Prints outdated actions honoring `.actup.yaml` pins. |
| `actup audit` | `--dry-run`, `--fix`, `-f/--force`, `--json`, `-p/--path`, `-s/--semver`, `--sha`, `-t/--token` | Scans direct and `actions.lock`-transitive refs against OSV.dev. `--fix` upgrades direct workflow refs to patched versions. |
| `actup install-hooks` | `--dry-run`, `-f/--force`, `-p/--path`, `--uninstall` | Installs or removes a git `pre-commit` hook running `actup outdated`. |
| `actup mcp` | — | Runs the stdio MCP server from the repository root. |

## Exit Codes

| Command | `0` | `1` | `2` |
|---------|-----|-----|-----|
| `actup outdated` | All actions up to date | Outdated actions found | — |
| `actup audit` | No vulnerable actions found | Vulnerable actions found | OSV.dev unreachable |

## Ref Resolution (`use` and `upgrade_actions`)

| Spec | Resolution |
|------|------------|
| `@v5` | Written verbatim as a moving major tag |
| `@5` | Resolved via GitHub API to the latest tag in major `v5` (e.g. `v5.3.1`) |
| `@latest` or omitted | Resolved to the latest major tag (or full semver with `--semver` / `semver: true`) |
| `@v5.3.1`, `@master` | Written verbatim |

## `.actup.yaml` Manifest Semantics

```yaml
# yaml-language-server: $schema=https://raw.githubusercontent.com/lynicis/actup/main/schema.json
major: 4          # Optional global major cap (overridden by --semver)
sha: true         # Write @<sha> # <tag> refs by default

pins:             # Constraints you choose
  actions/checkout: 4           # latest v4.x.x at sync/upgrade time
  actions/setup-go: v5.3.0      # exact ref
  some-org/some-action: skip    # exclude from upgrades, sync, and audit

actions:          # Recorded workflow state (rewritten by actup after writes)
  actions/checkout: v4.2.2
  actions/setup-go: v5.3.0

groups:           # Named glob groups for --group <name>
  ci:
    - actions/*
    - docker/*
```

- **Precedence**: CLI flags > `.actup.yaml` > built-in defaults (`major` tag mode, non-SHA).
- **Legacy manifests**: A file with `actions:` and no `pins:` key treats every `actions:` entry as a pin. Run `actup init --force` to migrate once.
- **SHA comments**: A bare 40-char SHA without `# <tag>` has no resolvable version and is skipped by breaking-change and vulnerability checks.
- **Deterministic saves**: `actup` re-renders `.actup.yaml` on write, so hand-written comments inside `.actup.yaml` are not preserved.

## MCP Tool Schemas (`actup mcp`)

Run the MCP server from the repository root. GitHub auth is resolved automatically from `GITHUB_TOKEN` or `gh auth token` (never passed as a tool parameter). Unreadable `.actup.yaml` or `actions.lock` degrades gracefully with a stderr warning instead of failing tool calls.

| Tool | Input parameters | Key output / behavior |
|------|------------------|-----------------------|
| `list_actions` | `paths?: string[]` | Actions, versions, usage sites, pins, `lock_status` (`locked` / `stale` / `none`). Offline. |
| `check_outdated` | `paths?: string[]`, `actions?: string[]`, `semver?: bool`, `major?: int` | Outdated status, target versions, known breaking changes. Uses GitHub API. |
| `audit_vulnerabilities` | `paths?: string[]` | `findings`, `suggested_fixes`, `unfixable`, `unknown_version`, `osv_unreachable`. Includes transitive `actions.lock` entries. |
| `upgrade_actions` | `paths?: string[]`, `actions?: string[]`, `dry_run?: bool`, `force?: bool`, `semver?: bool`, `major?: int`, `sha?: bool` | `actions` accepts `owner/repo[@ref]` specs (empty = all actions honoring `.actup.yaml`). Skips breaking or vulnerable targets (`skipped_breaking`, `skipped_vulnerable`) unless `dry_run` or `force`. Returns `lockfile_note` when `actions.lock` exists. |
| `pin_action` | `action: string`, `paths?: string[]`, `dry_run?: bool` | Records `owner/repo[@ref]` in `.actup.yaml` (omitting `@ref` pins the most common version in use). |
| `unpin_action` | `action: string`, `dry_run?: bool` | Removes the pin from `.actup.yaml`. |
| `init_config` | `paths?: string[]`, `force?: bool`, `dry_run?: bool` | Creates or merges `.actup.yaml` and returns the rendered YAML. |

## `actions.lock` Integration

When `.github/workflows/actions.lock` is present:

- `actup ls` / `list_actions` annotates each action as `locked` or `stale`.
- `actup audit` / `audit_vulnerabilities` checks both direct and transitive lockfile dependencies (marked `(transitive)` in terminal output and `"transitive": true` in JSON).
- CLI write commands (`actup --no-tui`, `use`, `sync`, `audit --fix`) automatically run `gh actions-lock` (skipped on `--dry-run`).
- MCP `upgrade_actions` does **not** shell out to `gh actions-lock`; check `lockfile_note` and offer to run `gh actions-lock` after applying changes.

## Authentication & Installation

- **Token order**: `--token` → `GITHUB_TOKEN` → `gh auth token` (5,000 req/hr authenticated vs 60 req/hr unauthenticated).
- **Install & MCP setup**: Ask the user before installing, then follow `https://raw.githubusercontent.com/lynicis/actup/main/llms.txt` (covers `gh extension install lynicis/gh-actup`, `brew install lynicis/tap/actup`, Scoop, Go, `.deb`/`.rpm`, Docker, and `actup mcp` client configs).
