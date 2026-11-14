# Hyperframes Composition Brief: actup

## Objective
Create a short launch-style brag video for `actup`.

## Output
- Composition directory: `brag-output/composition/`
- Rendered video: `brag-output/brag.mp4`
- Format: landscape — 1920x1080
- Duration: 21 seconds

## Source Material
- Project root: `/Users/lynicis/Projects/actup`
- Primary files read: `README.md`, `internal/tui/tui.go`, `.actup.yaml`, `internal/breakingchanges/registry.yaml`, `logo.png`
- Product name: `actup`
- Tagline / strongest claim: "A version manager for GitHub Actions. Nothing is rewritten until you ask, and every upgrade is one reviewable change."
- Key UI or visual moment to recreate:
  - The Bubble Tea interactive TUI (`internal/tui/tui.go`) showing `actup — 4 actions found across 2 files`, per-action version jumps (`v4 → v7`), inline `⚠ breaking changes` and `⚠ vulnerable` badges, the `[i] info` breaking-change detail view (`cache v4 runs on Node 20; self-hosted runners must be v2.308.0 or newer`), and the apply summary (`✅ 4 actions upgraded across 2 files`).
  - The `.actup.yaml` manifest (`sha: true`, `pins`, `actions`) alongside `actup use`, `actup sync`, `actup outdated`, and `actup audit`.
- Copy that must appear verbatim:
  - `A version manager for GitHub Actions.`
  - `Action versions live inside every workflow file.`
  - `Nothing is rewritten until you ask.`
  - `actup — 4 actions found across 2 files`
  - `⚠ breaking changes` / `⚠ vulnerable`
  - `cache v4 runs on Node 20; self-hosted runners must be v2.308.0 or newer.`
  - `cache v4 requires the cache service v2 backend.`
  - `✅ 4 actions upgraded across 2 files`
  - `Every upgrade is one reviewable change.`
  - `brew install lynicis/tap/actup`
  - `gh extension install lynicis/gh-actup`

## Creative Direction
- Tone preset: `polished`
- Creative direction: Crisp, high-craft developer CLI & terminal product film
- Interpretation: Confident dark-mode terminal aesthetics using exact Lipgloss ANSI colors from `internal/tui/tui.go`, generous breathing room on copy, snappy terminal transitions, and zero corporate SaaS filler.
- Angle: Action versions live inside every workflow file, where major jumps silently break CI and bots spam you with PRs. `actup` gives you `nvm`-style control from one `.actup.yaml` manifest, with a Bubble Tea TUI that checks breaking changes and OSV.dev vulnerabilities before writing atomically.
- Hook: "Action versions live inside every workflow file. Nothing is rewritten until you ask." (with `.github/workflows/ci.yml` `uses:` refs snapping into `$ actup`).
- Outro / punchline: "Every upgrade is one reviewable change." + `actup` Gopher+Octocat logo and install commands.
- Avoid:
  - Generic SaaS language ("streamline your workflow", "all-in-one platform")
  - Abstract filler visuals
  - Low-contrast colors or `oklch()` colors that skip WCAG checks

## Visual Identity
- Background: `#0b0f17` (deep terminal slate), `#111827` / `#161f30` (elevated terminal panels)
- Text: `#f0f6fc` (primary bright white), `#cbd5e1` (secondary slate), `#94a3b8` (muted label, WCAG AA ≥ 4.5:1 on `#0b0f17` and `#111827`)
- Accent: `#5fffff` (Lipgloss ANSI 87 cyan cursor), `#ff5faf` (Lipgloss ANSI 205 pink selected state), `#38bdf8` (Gopher sky blue)
- Semantic accents: `#5fd700` (Lipgloss ANSI 76 green `✓`), `#ffaf00` (Lipgloss ANSI 214 amber `⚠ breaking changes`), `#ff6b6b` (WCAG-compliant bright coral red `⚠ vulnerable` on dark backgrounds)
- Display font: `'Inter'`, `-apple-system, BlinkMacSystemFont, sans-serif`
- Body / Terminal font: `'JetBrains Mono'`, `'SFMono-Regular', Menlo, Consolas, monospace`
- Visual references from the project: `internal/tui/tui.go` checklist/detail/summary views, `.actup.yaml`, `README.md` before/after workflow examples, `logo.png`

## Storyboard
Use the storyboard in `brag-output/brag-plan.md` as the creative contract.

Scene summary:
1. **Scene 1: Hook — The Workflow Drift Problem** — `3.8s` (`0.0s–3.8s`) — "Action versions live inside every workflow file. Nothing is rewritten until you ask." + `.github/workflows/ci.yml` card with `uses: actions/checkout@v4` and `$ actup` prompt.
2. **Scene 2: Interactive TUI Checklist** — `4.9s` (`3.8s–8.7s`) — Bubble Tea TUI window `actup — 4 actions found across 2 files`; 4 action rows (`actions/checkout`, `actions/setup-go`, `actions/cache`, `actions/upload-artifact`) reveal on beat grid (`4.39s, 4.91s, 5.34s, 6.00s`) with `⚠ breaking changes` and `⚠ vulnerable` badges, then cursor selects `actions/cache` and hits `[i] info`.
3. **Scene 3: Inspect Breaking Changes & Apply** — `4.4s` (`8.7s–13.1s`) — Beat-locked to `8.74s` strong cue: `actions/cache — Breaking Changes` detail panel (`cache v4 runs on Node 20...`, `cache v4 requires the cache service v2 backend.`) + OSV.dev advisory check, then `[enter] apply` at `10.93s` strong beat flips to `✅ 4 actions upgraded across 2 files` + `actions.lock synced`.
4. **Scene 4: One Manifest, Full Version Manager** — `4.3s` (`13.1s–17.4s`) — Beat-locked to `13.11s` strong cue: `.actup.yaml` (`sha: true`, `pins`, `actions`) on left + `nvm`-style CLI commands (`actup use`, `actup audit`, `actup sync` / `actup outdated`) highlighting at `13.64s, 14.73s, 15.84s`.
5. **Scene 5: Outro — One Reviewable Change** — `3.6s` (`17.4s–21.0s`) — Beat-locked to `17.47s` strong cue: `logo.png` (Gopher hugging Octocat) + `actup` + "Every upgrade is one reviewable change." + `brew install lynicis/tap/actup` & `gh extension install lynicis/gh-actup`.

## Audio
- Audio role: Warm, steady electronic bed with crisp, low-risk terminal and UI accents
- Audio arc: Builds from a clean hook pulse into rhythmic TUI checklist ticks, lands a resonant confirmation bell on `[enter] apply`, glides through the `.actup.yaml` manifest reveal, and resolves with a warm bell and smooth fade-out on the final logo.
- Music: `assets/music/happy-beats-business-moves-vol-12-by-ende-dot-app.mp3` (109.96 BPM)
- Music treatment: `data-start="0"` `data-duration="21"` `data-volume="0.32"` `data-fade-in="0.6"` `data-fade-out="1.8"`
- Music cue guidance: Precomputed preset `assets/music/happy-beats-business-moves-vol-12-by-ende-dot-app.music-cues.json`
  - Strong cues locked: `8.74s` (Scene 3 detail reveal), `10.93s` (Scene 3 apply payoff), `13.11s` (Scene 4 manifest reveal), `17.47s` (Scene 5 final logo reveal).
  - Beat grid locked: `4.39s, 4.91s, 5.34s, 6.00s` (Scene 2 TUI checklist rows); `13.64s, 14.73s, 15.84s` (Scene 4 CLI command cards).
- Audio-reactive treatment: Subtle; sample RMS and bass energy from the music cues envelope to modulate the ambient background glow opacity (`#glow-orb`) and terminal card border luminance synchronously on the GSAP timeline.
- Audio-coupled moments:
  - `0.15s` — Hook reveal (`assets/sfx/impact/impactSoft_medium_001.ogg`)
  - `2.19s` — Prompt `$ actup` key action (`assets/sfx/interface/click_003.ogg`)
  - `3.82s` — TUI window entrance (`assets/sfx/impact/impactSoft_medium_002.ogg`)
  - `4.39s, 4.91s, 5.34s, 6.00s` — Sequential TUI checklist items (`assets/sfx/interface/click_002.ogg`, `click_005.ogg`)
  - `8.74s` — `[i] info` breaking-change detail modal (`assets/sfx/impact/impactSoft_medium_004.ogg`)
  - `10.93s` — `[enter] apply` upgrade completion (`assets/sfx/impact/impactBell_heavy_000.ogg`)
  - `13.11s` — `.actup.yaml` manifest slide-in (`assets/sfx/casino/card-slide-1.ogg`)
  - `13.64s, 14.73s, 15.84s` — Command cards (`assets/sfx/interface/drop_001.ogg`, `drop_002.ogg`)
  - `17.47s` — Final logo & install bar lockup (`assets/sfx/impact/impactBell_heavy_003.ogg`)
- SFX selection guidance: Low high-frequency-risk files from `sfx-analysis.md`.
- Audio files: Copied into `brag-output/composition/assets/`.

## Hyperframes Instructions
- Build `brag-output/composition/index.html` using native Hyperframes conventions (`data-composition-id="root"`, `data-width="1920"`, `data-height="1080"`, `class="clip"` with `data-start` and `data-duration`, paused GSAP timeline registered at `window.__timelines["root"]`).
- Ensure all text has ≥ 4.5:1 WCAG contrast on its immediate background using sRGB hex colors (no `oklch()`), zero text overflow, and settled reading time floors.
- Run `npx hyperframes check` inside `brag-output/composition/` and resolve any issues before rendering `brag-output/brag.mp4`.
