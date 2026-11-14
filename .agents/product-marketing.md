# Product Marketing Context

**Document version:** v3
**Last updated:** 2026-11-14

## Product Overview
**One-liner:** A version manager for GitHub Actions — set, pin, sync, and upgrade `uses:` refs from one manifest, with breaking-change warnings and OSV.dev vulnerability checks.
**What it does:** `actup` treats `.actup.yaml` as the source of truth for GitHub Actions versions, playing the same role `.nvmrc` plays for Node. Developers can bootstrap a manifest (`init`), switch or pin versions (`use`, `pin`, `unpin`), sync workflows (`sync`), audit for known vulnerabilities (`audit` / `audit --fix`), and upgrade actions interactively via a Bubble Tea TUI or non-interactively in CI (`--no-tui`). It supports major tags (`v5`), full semver (`--semver`), full 40-char commit SHA pinning (`--sha`), and `.github/workflows/actions.lock` integration, and ships as a standalone CLI, `gh actup` extension, Docker image, pre-commit hook, and companion AI agent skill (`github-actions-updater`).
**Product category:** GitHub Actions version manager / CI dependency management CLI
**Product type:** Open-source CLI tool (Go) + `gh` CLI extension + pre-commit hook + distributed agent skill
**Business model:** Free, MIT-licensed open source (by Emre Sırmalı / `lynicis`); currently pre-1.0 (`v0.9.1`). [Needs your input: confirm if purely OSS or any future commercial plans]

## Target Audience
**Target companies:** Software teams, OSS maintainers, and security-conscious engineering orgs running GitHub Actions workflows (from solo repos to teams enforcing SHA pinning, `actions.lock`, pre-commit gates, and SARIF/supply-chain checks).
**Decision-makers:** Individual developers, DevOps / Platform / Release Engineers, Staff/Lead Engineers, AppSec / DevSecOps engineers.
**Primary use case:** Keeping GitHub Actions `uses:` references pinned, consistent, vulnerability-free, and up to date across workflows on your own schedule—without noisy bot PRs.
**Jobs to be done:**
- Standardize and pin action versions across workflows and composite actions from a single `.actup.yaml` manifest.
- Upgrade actions on demand with upfront warnings for known breaking changes (Node runner bumps, deprecated `with:` inputs, behavior changes) and OSV.dev advisories before modifying files.
- Gate CI and local commits against stale or vulnerable actions (`outdated`, `audit`, pre-commit hook) and remediate direct advisories (`audit --fix`).
- Keep SHA-pinned workflows (`@<sha> # <tag>`) and `.github/workflows/actions.lock` synchronized without manual lookups.
**Use cases:**
- Interactive local workflow upgrades via TUI (`actup`)
- Standardizing a single action across all workflows (`actup use actions/checkout@v5`)
- Supply-chain hardening with `--sha` and `actions.lock` auto-relocking
- CI & pre-commit checks (`actup outdated`, `actup audit --json`, `actup-check`)

## Personas
| Persona | Cares about | Challenge | Value we promise |
|---------|-------------|-----------|------------------|
| Maintainer / App Developer | Fast, painless CI maintenance without surprises | Action versions drift across workflow files; major bumps silently break jobs | `nvm`-style CLI (`use`, `sync`, `ls`) and TUI with inline breaking-change & OSV warnings |
| Platform / DevOps Engineer | Consistency across workflows, runner compatibility, pre-commit/CI gates | Self-hosted runner breakages (e.g., Node 16→20), stale `actions.lock`, noisy bot PRs | `.actup.yaml` (`pins`, `actions`, `groups`), `sync`, `outdated`, auto `gh actions-lock`, atomic writes |
| AppSec / DevSecOps Engineer | SHA pinning, CVE remediation, transitive risk | Manual SHA resolution; OSV doesn't match moving major tags (`@v4`) server-side | `--sha` (`@<sha> # <tag>`), client-side OSV matching (including transitive `actions.lock` deps), `audit --fix` |

## Problems & Pain Points
**Core problem:** Action versions live inside every workflow and composite action file, so upgrades are manual and uneven: nobody agrees on when to bump, major jumps can silently break jobs or runners, and SHA-pinned refs are tedious to maintain by hand.
**Why alternatives fall short:**
- Manual edits don't scale across multiple workflow/composite-action files and miss breaking changes or OSV advisories.
- Dependabot and Renovate are poll-based bots that open PRs on every upstream release, creating PR noise without a local imperative manifest or TUI review.
- `gh actions-lock` records resolved/transitive state for reproducibility, but never upgrades direct versions or audits OSV advisories.
**What it costs them:** Broken CI pipelines from unflagged major changes, unpatched vulnerable actions, version drift across workflows, and time lost triaging automated PRs or looking up commit SHAs.
**Emotional tension:** [Needs your input — e.g., PR fatigue from automated dependency bots vs. anxiety over silently breaking CI or failing security audits]

## Competitive Landscape
**Direct:** Dependabot & Renovate — automated PR bots driven by schedules/rules (`dependabot.yml`, `renovate.json`); fall short for developers wanting imperative, on-demand local or CI control, an `nvm`-style manifest (`use`/`sync`), a TUI checklist, and zero bot PR spam.
**Secondary:** `gh actions-lock` (complementary rather than competing—`actup` reads `actions.lock` for transitive `audit`/`ls` and automatically runs `gh actions-lock` after edits); manual `gh api`/`git ls-remote` scripts and standalone LLM prompts (lack deterministic manifest state, atomic multi-file rewrites, and client-side OSV matching).
**Indirect:** Freezing action refs indefinitely or ignoring updates — avoids immediate churn, but accumulates vulnerable actions, deprecated Node runtimes, and painful forced migrations.

## Differentiation
**Key differentiators:**
- `nvm`/`fvm`-style imperative workflow (`init`, `use`, `pin`, `unpin`, `sync`, `ls`, `outdated`, `audit`) with `.actup.yaml` separating user `pins` from recorded `actions` state and `groups`.
- Pre-write safety checks: embedded breaking-change registry (including `with:` input inspection and multi-major jumps) + client-side OSV.dev matching for moving major tags (`@v4`) and `audit --fix`.
- First-class commit SHA pinning (`@<40-hex-sha> # <tag>`) and automatic `.github/workflows/actions.lock` integration (transitive audits + auto re-lock).
- Zero-infra distribution: Bubble Tea TUI, `--no-tui`/`--dry-run`/`--json` for CI, `gh actup` CLI extension, Homebrew/Scoop/deb/rpm/Docker (`scratch`, UID 65532, cosign + SBOMs), and pre-commit hook.
**How we do it differently:** `actup` brings Dependabot/Renovate-grade upgrade machinery to a local/CI command you run yourself—nothing is rewritten until you ask, and every upgrade is checked for breaking changes and OSV advisories before atomic file writes.
**Why that's better:** One reviewable change instead of dozens of bot PRs, preserved file permissions and YAML comments, and safe coexistence with SHA pinning and `actions.lock`.
**Why customers choose us:** [Needs your input — verify why actual users picked `actup` over Renovate/Dependabot or manual bumps]

## Objections
| Objection | Response |
|-----------|----------|
| "We already use Dependabot or Renovate." | Keep them if you want scheduled bot PRs on every upstream release. `actup` is the imperative counterpart for teams who want local/CI control, `nvm`-style manifest pinning, and zero bot PR noise. |
| "We already use `gh actions-lock`." | They complement each other: `actions-lock` pins resolved/transitive state and never upgrades; `actup` upgrades direct refs, audits transitive lockfile deps, and automatically runs `gh actions-lock` afterward. |
| "Our security policy requires 40-char commit SHAs." | Use `--sha` (or `sha: true` in `.actup.yaml`) to write `@<sha> # <tag>`. `actup` also parses existing SHA+comment pins so `ls`, `outdated`, `audit`, and upgrades keep working. |
| "I don't want a CLI mangling my workflow YAML." | Rewrites target only the `path@ref` segment (preserving `with:` inputs, spacing, trailing comments, and file permissions) using atomic temp-file + rename, and `--dry-run` previews diffs without touching disk. |

**Anti-persona:** Teams that strictly want hands-off scheduled bot PRs for every release and never run local/CI CLI maintenance; teams not using GitHub Actions.

## Switching Dynamics
**Push:** Bot PR fatigue, uneven action versions across workflows, surprise CI failures on major bumps (Node 20 runner cutovers, artifact v4 changes), security alerts (zizmor, OpenSSF Scorecard, OSV) on unpinned or vulnerable actions.
**Pull:** Familiar `nvm`/`fvm` mental model, interactive Bubble Tea TUI with `i` detail view, inline breaking-change + OSV warnings, native `--sha` and `actions.lock` support.
**Habit:** Existing Dependabot/Renovate configs or ad-hoc copy-pasting versions from GitHub Marketplace when builds break.
**Anxiety:** Fear of unexpected YAML diffs, GitHub API rate limits (60/hr unauthenticated), or breaking `actions.lock` / SHA pins (mitigated by `--dry-run`, `gh auth token` auto-discovery for 5,000/hr, and auto re-lock). [Needs your input: any real user anxieties you've heard]

## Customer Language
**How they describe the problem:**
- "nobody agrees on when to bump, and a major jump can silently change behavior" (README)
- "Update my GitHub Actions" / "Which actions are out of date?" / "Bump actions in my workflows" (skill prompts)
- [Needs your input: verbatim quotes from GitHub issues, discussions, Reddit/HN, or users]
**How they describe us:**
- "version manager for GitHub Actions — like `nvm` is for Node or `fvm` for Flutter"
- "the imperative counterpart [to Dependabot/Renovate], built for reviewable upgrades and CI gates"
- [Needs your input: verbatim user descriptions]
**Words to use:** `version manager`, `uses:` refs, `manifest` (`.actup.yaml`), `pin`, `sync`, `outdated`, `audit`, `major tag`, `semver`, `commit SHA`, `reviewable change`, `imperative`, `dry-run`
**Words to avoid:** "platform", "all-in-one DevOps suite", "autonomous AI bot" (`actup` is deterministic CLI tooling, though it ships an optional companion skill). [Needs your input: any other banned words]
**Glossary:**
| Term | Meaning |
|------|---------|
| Manifest (`.actup.yaml`) | Declared source of truth separating user `pins` from recorded `actions` state and `groups` |
| Major tag | Moving major version tag like `v5` (`actup`'s default resolution mode) |
| Semver (`--semver`) | Full semantic version tag like `v5.3.1` |
| SHA pin (`--sha`) | Full 40-char commit SHA with trailing tag comment (`@<sha> # <tag>`) |
| `actions.lock` | `.github/workflows/actions.lock` managed by `gh actions-lock`; read by `actup` for transitive audits and lock status |

## Brand Voice
| Attribute | Means | Doesn't mean | Example line |
|-----------|-------|--------------|--------------|
| Plain-spoken & technical | Short, concrete developer prose with exact flags, files, and exit codes | Vague hype or dumbed-down summaries | "Nothing is rewritten until you ask, and every upgrade is one reviewable change." |
| Honest about boundaries | States pre-1.0 status, caveats, and when a competitor fits better | Apologetic or underselling | "If you want a bot that opens PRs on every upstream release, use Dependabot or Renovate — actup is the imperative counterpart..." |
| Calm & exact on risk | Clear `⚠` markers, exact exit codes, and actionable steps | Fear-mongering or alarmist | "Exits 1 when vulnerable action versions are found and 2 when the OSV database cannot be reached." |
| Familiar dev analogies | Maps concepts to tools developers already use (`nvm`, `fvm`, `npm audit`) | Forced jokes or cute metaphors | "actup is a version manager for GitHub Actions — like nvm is for Node or fvm for Flutter." |

**Tone by context:**
- Support: Direct, reproducible steps, zero corporate filler.
- Launch: Specific capabilities, real before/after YAML & CLI output, honest positioning vs Dependabot/Renovate/`actions-lock`.
- Errors: What happened, exit code, and exact next command or `y/N` prompt; no humor when blocked.
- Sales / Outreach: Peer-to-peer engineer tone (not applicable unless commercializing).
- Social: Builder-to-builder, terminal/TUI demos (`docs/demo.gif`), concrete CI pain solved.
**Terms we own:** `version manager for GitHub Actions`, `imperative counterpart`, `.actup.yaml` manifest, `pins` vs `actions`
**Never use (and why):** "AI-powered" / "revolutionary" (breaks plain-spoken dev credibility; core CLI is deterministic Go); "replaces `actions-lock`" (inaccurate—they complement each other).

## Proof Points
**Metrics:** 27 releases (`v0.1.0`–`v0.9.1`), 19 breaking-change entries across 18 actions in `internal/breakingchanges/registry.yaml`, ~1 GitHub star (2026-11-14 pre-launch baseline). [Needs your input: Homebrew/release downloads]
**Customers:** Dogfooded in `lynicis/actup` itself (`.actup.yaml` + `.github/workflows/ci.yml` using SHA-pinned refs). [Needs your input: notable external users/orgs]
**Testimonials:** [Needs your input]
**Value themes:**
| Theme | Proof |
|-------|-------|
| Safe pre-write upgrades | Embedded breaking-change registry (including `with:` input checks), OSV.dev warnings, `--dry-run`, atomic temp-file + rename preserving permissions |
| Supply-chain hardening | `--sha` (`@<sha> # <tag>`), transitive `actions.lock` auditing + auto re-lock, `scratch` UID 65532 Docker image with cosign signing & SBOMs |
| Fits any workflow | `gh actup` extension, Homebrew/Scoop/deb/rpm/Go/Docker, `pre-commit` hook, CI exit codes (`1` findings, `2` OSV unreachable) |

## Goals
**Business goal:** Grow OSS installs and GitHub stars by launching the current `v0.9.1` release as Early Access / Public Beta (warm Discord + `r/github` `2026-11-16`–`2026-11-27` → Website & Product Hunt launch `2026-12-01`, avoiding US Thanksgiving week) across Website, Product Hunt, Reddit, and ex-colleague Discord channels.
**Conversion action:** Install `actup` (`gh extension install lynicis/gh-actup`, `brew tap lynicis/tap && brew install actup`, `go install github.com/lynicis/actup@latest`, or Docker), run `actup init` / `actup --dry-run`, and star `lynicis/actup`.
**Current metrics:** ~1 GitHub star (`2026-11-14` baseline); no prior formal launch.

## Learnings
*Decisions and tested lessons. Newest first. Keep it to about 20 active entries.*
| Date | Learning | Evidence | Applies to | Status |
|------|----------|----------|------------|--------|
| 2026-11-14 | Launch targets current `v0.9.1` release directly (Early Access / Public Beta positioning, keeping `README.md:28` pre-1.0 honesty) with Product Hunt on `2026-12-01` (avoiding US Thanksgiving week) | User clarified `v0.9.1` is the release to promote directly | Launch & messaging | Active |

## Changelog
*Newest first. One line per revision: what changed and why.*
- v3 (2026-11-14) — Updated launch plan to promote current `v0.9.1` directly (Early Access framing) targeting `2026-12-01` Product Hunt launch.
- v2 (2026-11-14) — Added launch goals, baseline star metric (~1), channels (Website, Product Hunt, Reddit, Discord), and launch strategy decision.
- v1 (2026-11-14) — Initial context auto-drafted from repository codebase and docs.
