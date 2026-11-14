# `actup` Launch Plan (`v0.9.1`)

**Date:** 2026-11-14
**Release:** `v0.9.1`
**Primary Goal:** Grow installs (`gh extension`, Homebrew, `go install`, Docker) and GitHub stars (`lynicis/actup`).
**Baseline:** ~1 GitHub star, `v0.9.1` released (`2026-11-13`), 27 total releases, 19 breaking-change entries in `internal/breakingchanges/registry.yaml`.

---

## 1. Readiness Gate (SLC)

| Gate | Status | Notes |
|---|---|---|
| Simple | Pass | One clear job: imperative `uses:` version manager backed by `.actup.yaml`. |
| Lovable | Pass | `nvm`-style CLI (`init`, `use`, `sync`, `ls`), Bubble Tea TUI with `i` detail view, inline `⚠` breaking-change & OSV warnings, `docs/demo.gif`. |
| Complete | Pass | Ships `--sha`, `audit --fix`, `actions.lock` integration, `gh actup`, Homebrew/Scoop/deb/rpm/Go/Docker, and pre-commit hook. |

---

## 2. Launch Schedule (`v0.9.1`)

`Tue 2026-12-01` is targeted for Product Hunt to avoid US Thanksgiving week (`2026-11-24`–`2026-11-27`) while giving two weeks to stand up the Website, run warm peer feedback, and build the PH gallery assets.

| Stage | Window | Release | Objective | Channels |
|---|---|---|---|---|
| **Stage 1: Warm Feedback & Prep** | **Mon 2026-11-16 – Wed 2026-11-25** | `v0.9.1` | Get real-repo feedback from peers, collect 2–3 quotes, stand up static Website, prep PH gallery | Ex-colleagues Discord, `r/github` (`Thu 2026-11-19`), GitHub Discussions |
| **Stage 2: Pre-PH Readiness Check** | **Thu 2026-11-26 – Mon 2026-11-30** | `v0.9.1` | Finalize Website QA, verify all install paths, lock PH listing & 5 gallery screenshots | Repo + Website QA |
| **Stage 3: Public Launch Day** | **Tue 2026-12-01 (12:01 AM PT)** | `v0.9.1` | Peak discovery & install/star conversion | **Product Hunt**, **Website**, `r/devops`, Ex-colleagues Discord |
| **Stage 4: Post-Launch Follow-up** | **Wed 2026-12-02 – Tue 2026-12-15** | `v0.9.1` | `r/golang` deep-dive (`Thu 2026-12-03`), fix reported workflow edge cases, publish comparison pages, solicit `registry.yaml` PRs | `r/golang`, GitHub Discussions, Website |

---

## 3. ORB Channel Strategy

| Layer | Channel | Role | Primary Conversion Action |
|---|---|---|---|
| **Owned** | **Website** | Scannable landing page: one-liner, `docs/demo.gif`, copyable install switcher, Dependabot/Renovate/`actions-lock` table, before/after YAML | Copy install command + Star `lynicis/actup` |
| **Owned** | **GitHub Repo + Discussions** | Releases, `schema.json`, issue templates, *Show & Tell* / *Breaking-Change Registry* discussions | Install + Star + `registry.yaml` contributions |
| **Rented** | **Product Hunt** | Main public launch spike (`Tue 2026-12-01` at `12:01 AM PT`) | Visit Website / install CLI / comment |
| **Rented** | **Reddit** (`r/github`, `r/devops`, `r/golang`) | Value-first technical posts tailored per subreddit (never identical cross-posts) | Install / Star / feedback on workflow edge cases |
| **Borrowed** | **Discord (ex-colleagues)** | Stage 1 real-repo feedback; Stage 3 day-1 comments (**never ask for PH upvotes** — against PH TOS) | Run `actup --dry-run` in Stage 1; join comment thread on launch day |

---

## 4. Pre-Launch Checklists

### Repo Readiness (by Mon 2026-11-30)
- [ ] Enable **GitHub Discussions** (`Show & Tell`, `Breaking-Change Registry`, `Q&A`).
- [ ] Link the new Website URL in `README.md`.
- [ ] Smoke-test all 4 primary `v0.9.1` install paths on clean machines:
  - `gh extension install lynicis/gh-actup`
  - `brew tap lynicis/tap && brew install actup`
  - `go install github.com/lynicis/actup@latest`
  - `docker run --rm ghcr.io/lynicis/actup:latest --help`
- [ ] Record 2–3 Stage 1 user quotes in `.agents/product-marketing.md`.

### Website Launch QA (by Wed 2026-11-25)
- [ ] Above-the-fold: one-liner, `docs/demo.gif`, copyable install tabs, GitHub Star link.
- [ ] Sections: comparison vs Dependabot / Renovate / `gh actions-lock`, before/after SHA+manifest YAML diff, OSV + `registry.yaml` + `actions.lock` safety checks, CI gates (`actup outdated`, `actup audit --json`, pre-commit).
- [ ] Technical QA:
  - [ ] Unique `<title>`, meta description, single `<h1>`, Open Graph & Twitter card image/tags verified.
  - [ ] HTTPS active, apex/subdomain redirects clean, no `noindex`.
  - [ ] Mobile layout & code-block horizontal scroll checked; `alt` text on `logo.png` and `docs/demo.gif`.
  - [ ] All outbound links to GitHub/releases return `200 OK`.

### Product Hunt Listing Specs (by Sun 2026-11-29)
- **Name:** `actup`
- **Tagline (47 chars, ≤60, no period):** `An nvm-style version manager for GitHub Actions`
  *(Alternative, 51 chars: `Pin, audit, and upgrade GitHub Actions from one CLI`)*
- **Description (491 chars, ≤500; first 260 previewed):** `actup is an nvm-style version manager for GitHub Actions. Pin, sync, audit, and upgrade uses: refs across workflows and composite actions from .actup.yaml. v0.9.1 separates user pins from recorded actions state and groups, pins 40-char commit SHAs (--sha), checks with: inputs against a breaking-change registry, matches moving major tags against OSV.dev with audit --fix, audits and refreshes actions.lock, and ships as a gh CLI extension, binary, pre-commit hook, and signed container.`
- **Topics (max 3):** `Developer Tools`, `Open Source`, `GitHub`
- **Gallery (8 images attached to PH draft `C7FOKWLDO2V44DFGB7EOHDLVHSPHCB5QCT` from `brag-output/` & repo):**
  1. **Hero (1280×720 animated GIF from `brag-output/brag.mp4`):** Full 21s product film showing hook, Bubble Tea TUI checklist, `[i] info` breaking-change & OSV.dev inspection, `.actup.yaml` manifest, and install outro.
  2. **TUI Checklist (1920×1080 from `brag.mp4` Scene 2):** `actup — 4 actions found across 2 files` with inline `⚠ breaking changes` and `⚠ vulnerable` badges.
  3. **Risk Inspection & Apply (1920×1080 from `brag.mp4` Scene 3):** `[i] info` detail overlay (`actions/cache` Node 20 + cache service v2 warnings + OSV.dev check) and `✅ 4 actions upgraded across 2 files`.
  4. **Manifest & CLI Suite (1920×1080 from `brag.mp4` Scene 4):** `.actup.yaml` (`sha: true`, `pins`, `actions`) alongside `actup use`, `actup audit`, `actup sync`, and `actup outdated`.
  5. **40-Char SHA Diff (1270×760 slide):** `@v4` → `@<40-char-sha> # v5` preserving `with:` inputs and trailing comments + `actions.lock` refresh.
  6. **Workflow Drift Hook (1920×1080 from `brag.mp4` Scene 1):** *"Action versions live inside every workflow file. Nothing is rewritten until you ask."*
  7. **Outro Poster (`brag-output/brag.jpg`, 1920×1080):** *"Every upgrade is one reviewable change."* with `logo.png` and `brew` / `gh extension` install commands.
  8. **Raw Terminal Recording (`docs/demo.gif`, 905×488):** Live TUI session recording.
- **Video Embed (Optional):** Upload `brag-output/brag.mp4` (21.0s, 1920×1080 H.264 + AAC with audio cues) to YouTube/Loom and paste the link into Product Hunt's `videoUrl` field (and use `brag.mp4` directly for Reddit, Discord, and Website embeds).

---

## 5. Launch Copy (`v0.9.1`)

### Discord (Ex-Colleagues) — Stage 1 (Mon 2026-11-16)
> Hey folks — I've been building **actup** (`v0.9.1`), an open-source `nvm`-style version manager for GitHub Actions.
>
> Instead of Dependabot/Renovate opening bot PRs on every release, you keep a `.actup.yaml` manifest (`pins`, `actions`, `groups`) and run `actup` locally (TUI) or in CI (`actup outdated`, `actup audit`). Before touching any YAML, it checks an embedded breaking-change registry (including `with:` inputs) and OSV.dev advisories, supports `--sha` (`@<sha> # <tag>`), and auto-runs `gh actions-lock`.
>
> Would love 5 minutes of honest feedback on `v0.9.1` ahead of putting it on Product Hunt on Dec 1:
> `gh extension install lynicis/gh-actup` (or `brew tap lynicis/tap && brew install actup`) → run `actup --dry-run` on a repo with workflows.
> Repo: https://github.com/lynicis/actup — curious where it breaks, feels rough, or misses a breaking change.

### Product Hunt Maker Comment — Stage 3 (Tue 2026-12-01, 12:01 AM PT)
> Hi Product Hunt,
>
> I built actup (v0.9.1), an MIT-licensed version manager for GitHub Actions modeled on nvm.
>
> In most repositories, action versions are scattered across workflow and composite action files. Major bumps can change the Node runtime or break deprecated `with:` inputs on self-hosted runners, and updating 40-character commit SHAs by hand gets old fast. Dependabot and Renovate handle this if you want scheduled bot PRs on every upstream release. I wanted a local CLI I could run when I was ready to upgrade, review in one diff, and gate in CI.
>
> What v0.9.1 includes:
> - `.actup.yaml` separates your version constraints (`pins`) from recorded workflow versions (`actions`) and named glob `groups`, managed with `actup init`, `use`, `pin`, and `sync`.
> - Default discovery scans `.github/workflows/`, composite actions under `.github/actions/`, and root `action.yml` files, with glob and `--group` filters.
> - Before rewriting a file, `actup` checks an embedded breaking-change registry (including workflow `with:` inputs across multi-major jumps) and matches versions client-side against OSV.dev, even for moving major tags like `@v4`.
> - `actup audit --fix` upgrades vulnerable direct refs to a patched release, and `--sha` writes `@<40-char-sha> # <tag>` while keeping existing YAML comments and file permissions intact.
> - It reads `.github/workflows/actions.lock` to audit transitive dependencies and runs `gh actions-lock` after edits so your lockfile stays in sync.
> - You can pick upgrades in the Bubble Tea TUI or run `--no-tui`, `--dry-run`, `--json`, `actup outdated`, and the pre-commit hook in CI. It installs via `gh extension install lynicis/gh-actup`, Homebrew, Scoop, Go, deb/rpm packages, or a cosign-signed `scratch` Docker image.
>
> If you try it on your workflows, I'd love to know how it handles your setup and which actions you'd want added to `registry.yaml` next.

### Reddit Posts (Staggered)
1. **Stage 1 — `r/github` (Thu 2026-11-19):**
   - **Title:** `Built an nvm-style CLI for GitHub Actions (.actup.yaml + TUI + OSV/breaking-change checks) — v0.9.1 feedback`
   - **Focus:** Drift across `.github/workflows/*.yml` + composite actions, 6-line `.actup.yaml` + `actup --dry-run` diff, honest positioning vs Dependabot/Renovate/`gh actions-lock`, ask for broken workflow patterns (`matrix`, reusable workflows, self-hosted runners).
2. **Stage 3 — `r/devops` (Tue 2026-12-01):**
   - **Title:** `actup (v0.9.1): imperative GitHub Actions version manager with --sha pinning, OSV checks, and actions.lock integration`
   - **Focus:** Supply-chain control & CI gates — client-side OSV.dev matching for moving `@v4` tags, `audit --fix`, `@<40-char-sha> # <tag>` rewrites preserving comments, `actions.lock` auto-relock, and exit codes (`1` findings, `2` OSV unreachable).
3. **Stage 4 — `r/golang` (Thu 2026-12-03):**
   - **Title:** `Building actup in Go: Bubble Tea TUI, atomic YAML rewrites, and client-side OSV matching for GitHub Actions`
   - **Focus:** Architecture — Bubble Tea TUI, `goccy/go-yaml`, `go-github/v89`, atomic temp-file/rename edits, GoReleaser + cosign + SBOM `scratch` image, and `internal/breakingchanges/registry.yaml`.

---

## 6. Launch Day Timeline (Tue 2026-12-01, PT) & Weeks 1–2 Follow-up

- **12:01 AM:** Product Hunt live → post Maker Comment immediately; verify PH link targets Website.
- **12:15 AM:** Activate Website launch banner pointing to GitHub / Product Hunt.
- **06:00 AM:** First PH comment sweep (reply to every comment with concrete answers + follow-up questions).
- **08:30 AM:** Post `r/devops` thread + share PH launch in ex-colleague Discord (*ask for honest feedback/comments only, never upvotes*).
- **12:00 PM / 03:00 PM / 06:00 PM:** Reply to all PH, Reddit, and GitHub comments within 1–2 hours.
- **2026-12-02 – 2026-12-15:** Ship fixes for any reported workflow edge cases, add comparison sections on Website (`actup vs Dependabot`, `actup vs Renovate`, `actup + gh actions-lock`), and invite community PRs to `internal/breakingchanges/registry.yaml`.
