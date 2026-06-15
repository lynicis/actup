---
name: github-actions-updater
description: "Helps users keep GitHub Actions workflow files up to date. Use when the user mentions GitHub Actions, workflow files, upgrading actions, bumping action versions, CI/CD maintenance, or deprecated actions."
license: MIT
---

# GitHub Actions Updater

Keep `.github/workflows/*.yml` and `.github/workflows/*.yaml` files current by scanning actions, resolving latest versions, and suggesting upgrades.

## Quick start

When the user asks something like:

> "Update my GitHub Actions"
> "Which actions are out of date?"
> "Bump actions in my workflows"

Follow the workflow below and return a Markdown report with diff-style suggestions.

## Workflow

1. **Scan workflows**
   - Find all `.github/workflows/*.yml` and `.github/workflows/*.yaml` files.
   - If none exist, tell the user and stop.

2. **Extract `uses:` refs**
   - Parse each workflow for `uses:` lines.
   - Skip:
     - Local paths: `./...`
     - Docker refs: `docker://...`
     - Full SHA commits: anything that looks like a 40-char hex SHA
   - Record: `action`, `current_version`, `workflow_file`, `line_number`.

3. **Resolve latest version**
   - Split each action into `owner/repo`.
   - Try tools in this order:
     1. `gh api repos/{owner}/{repo}/tags --jq '.[].name'` if `gh` is available.
     2. `fetch` on `https://api.github.com/repos/{owner}/{repo}/tags`.
     3. Brave Search / Google Search MCP for `"{owner}/{repo} latest release tag"`.
   - Filter tags to valid semver.
   - Prefer major tags (`v5`) over full semver (`v5.3.1`) by default.
   - If no major tag exists, fall back to the highest semver tag.

4. **Detect breaking changes**
   - If the latest version is a higher major than the current version, mark it as a breaking change.
   - Cross-check the embedded registry below for known warnings.

5. **Generate suggestion report**
   - Produce a Markdown report with:
     - Summary table: action | current | latest | breaking? | file
     - Diff blocks per action
     - Breaking-change warnings with `⚠`
   - Do **not** edit workflow files unless the user explicitly asks.

## Breaking-change registry

```yaml
- action: actions/checkout
  from_major: 3
  to_major: 4
  warning: "Node runtime upgraded from 16 to 20. Check custom runners and self-hosted setup."
- action: actions/setup-node
  from_major: 3
  to_major: 4
  warning: "Node 16 runtime deprecated; Node 20 is now used."
- action: actions/setup-go
  from_major: 4
  to_major: 5
  warning: "Node 20 runtime. Verify any custom cache keys or side effects."
```

## Output template

```markdown
## GitHub Actions update report

| Action | Current | Latest | Breaking? | File |
|--------|---------|--------|-----------|------|
| actions/checkout | v3 | v4 | ⚠ yes | .github/workflows/ci.yml |

### Suggested changes

`.github/workflows/ci.yml`:
```diff
- uses: actions/checkout@v3
+ uses: actions/checkout@v4
```

### Breaking-change warnings

- **actions/checkout@v3 → v4**: Node runtime upgraded from 16 to 20. Check custom runners and self-hosted setup.
```

## Rules

- Never silently write workflow files.
- Always show a diff-style preview before any edit.
- If GitHub API rate limits are hit, fall back to search MCPs or manual guidance.
- If an action has no semver tags, note it as "unable to determine latest version".
