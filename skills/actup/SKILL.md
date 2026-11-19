---
name: actup
description: Use when managing GitHub Actions versions with actup (MCP or CLI) to list, audit OSV.dev vulnerabilities, pin or sync .actup.yaml, switch to SHA or semver refs, and safely upgrade workflow uses: refs.
license: MIT
---

# actup — GitHub Actions Version Manager

Manage `uses:` references across `.github/workflows/` and composite actions through `actup` so `.actup.yaml`, SHA comments, breaking-change checks, OSV.dev audits, and `actions.lock` stay in sync.

## 1. Pick the Interface

1. **MCP first**: If `actup` MCP tools (`list_actions`, `check_outdated`, `audit_vulnerabilities`, `upgrade_actions`, `pin_action`, `unpin_action`, `init_config`) are loaded, use them.
2. **CLI second**: Otherwise detect `actup` or `gh actup` (`which actup` / `gh extension list`). Always pass non-interactive flags (`--no-tui` for root upgrades); never launch the interactive Bubble Tea TUI.
3. **Not installed**: Ask the user before installing, then follow `https://raw.githubusercontent.com/lynicis/actup/main/llms.txt` to install `gh actup` or standalone `actup` (and optionally configure `actup mcp`).

## 2. Task Routing

| Goal | MCP tool | CLI command |
|------|----------|-------------|
| List actions, pins & lock status | `list_actions` | `actup ls [action]` |
| Check outdated actions | `check_outdated` | `actup ls --outdated` or `actup outdated` |
| Audit OSV.dev vulnerabilities | `audit_vulnerabilities` | `actup audit --json` |
| Fix vulnerable actions | `upgrade_actions` (target fixed refs) | `actup audit --fix --dry-run` |
| Upgrade actions | `upgrade_actions` | `actup [filter...] --no-tui --dry-run` |
| Set one action everywhere | `upgrade_actions` (`actions: ["owner/repo@ref"]`) | `actup use <action>@<ref> --dry-run` |
| Bootstrap `.actup.yaml` | `init_config` | `actup init --dry-run` |
| Pin / unpin in `.actup.yaml` | `pin_action` / `unpin_action` | `actup pin <action>[@<ref>]` / `actup unpin <action>` |
| Apply `.actup.yaml` to workflows | `upgrade_actions` (empty `actions`) | `actup sync --dry-run` |

## 3. Safe Execution Workflow

1. **Inspect read-only first**: Run `list_actions` / `actup ls`, `check_outdated` / `actup ls --outdated`, or `audit_vulnerabilities` / `actup audit --json` before proposing edits.
2. **Preview before writing**: Run mutating operations with `dry_run: true` / `--dry-run` and show the planned changes unless the user explicitly asked to apply immediately.
3. **Honor safety gates**:
   - MCP `upgrade_actions` returns `skipped_breaking` or `skipped_vulnerable` unless `force: true`.
   - CLI non-interactive runs skip breaking upgrades without `--force`.
   - Summarize every breaking-change note and OSV advisory for the user and pass `force: true` / `--force` only after explicit confirmation.
4. **Apply via `actup`, not manual YAML edits**: Use `actup` tools or commands so atomic writes, `.actup.yaml` state, and `# <tag>` SHA comments remain consistent.
5. **Keep `actions.lock` current**: CLI write commands auto-run `gh actions-lock` when `.github/workflows/actions.lock` exists. MCP tools do not shell out—when a response includes `lockfile_note`, inform the user and offer to run `gh actions-lock`.
6. **Confirm completion**: Re-run `list_actions` / `actup ls` (or `audit_vulnerabilities` / `actup audit`) after writes to verify the final state.

## 4. Common Modifiers & Example

- **Commit SHA pinning**: pass `sha: true` / `--sha` to write `@<40-char-sha> # <tag>`.
- **Exact semver tags**: pass `semver: true` / `--semver` instead of default moving major tags (`v5`).
- **Scope filtering**: pass `actions` globs (MCP) or quoted globs / `--group <name>` (CLI).

```sh
# 1. Inspect and audit (read-only)
actup ls --outdated && actup audit --json

# 2. Preview an upgrade or pin change before writing
actup 'actions/*' --sha --no-tui --dry-run

# 3. Apply after user confirmation (auto-refreshes actions.lock)
actup 'actions/*' --sha --no-tui
```

## Reference

See [REFERENCE.md](REFERENCE.md) for full CLI flags, exit codes, ref resolution rules (`@v5` vs `@5` vs `@latest`), `.actup.yaml` manifest semantics, MCP tool parameters, and authentication/troubleshooting details.
