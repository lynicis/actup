# Brag Plan: actup

## What is this app?
`actup` is a version manager for GitHub Actions — like `nvm` is for Node or `fvm` for Flutter — that sets, pins, syncs, and upgrades `uses:` refs from a single `.actup.yaml` manifest, with breaking-change warnings and OSV.dev vulnerability checks, interactively via a terminal UI or in CI.

## The angle
Every repo has `uses:` refs scattered across `.github/workflows/`, where nobody agrees on when to bump and a major jump can silently change behavior — while automated bots flood you with PRs. `actup` brings the upgrade machinery to one command you run yourself: `.actup.yaml` as the source of truth, a Bubble Tea TUI that flags `⚠ breaking changes` and `⚠ vulnerable` before touching a file, and atomic rewrites (with `@<sha> # <tag>` and `actions.lock` sync) only when you press `enter`.

## Hook (first 2-3 seconds)
"Action versions live inside every workflow file."
"Nothing is rewritten until you ask."
With a glowing `.github/workflows/ci.yml` diff pill (`actions/checkout@v4 → v7`) snapping into focus.

## Key moments (the middle)
- **Interactive TUI checklist (`$ actup`)** — The exact Bubble Tea checklist (`actup — 4 actions found across 2 files`) populates with `[✓] actions/checkout  v4 → v7 (2 files)`, `[✓] actions/setup-go  v4 → v7 ⚠ breaking changes`, `[✓] actions/cache  v3 → v6 ⚠ breaking changes`, and `[✓] actions/upload-artifact  v3 → v7 ⚠ vulnerable`.
- **Inspecting breaking changes & OSV advisories (`[i] info`)** — Pressing `i` opens the inline detail card showing exact warnings (`cache v4 runs on Node 20; self-hosted runners must be v2.308.0 or newer`, `cache v4 requires the cache service v2 backend`) and OSV.dev advisory status before any file is modified, followed by pressing `enter` (`✅ 4 actions upgraded across 2 files`).
- **One `.actup.yaml` manifest & `nvm`-style CLI** — Showing `.actup.yaml` (`sha: true`, `pins:`, `actions:`) paired with `actup init`, `actup use actions/checkout@v7`, `actup sync`, `actup outdated`, and `actup audit` (plus automatic `gh actions-lock` sync).

## Outro / punchline
"Every upgrade is one reviewable change."
`actup` — A version manager for GitHub Actions.
`brew install lynicis/tap/actup` · `gh extension install lynicis/gh-actup`

## User flow worth showing
1. **Entry:** Run `$ actup` in a repo with scattered workflow `uses:` refs; TUI loads `actup — 4 actions found across 2 files` with inline `⚠ breaking changes` and `⚠ vulnerable` badges.
2. **Key action:** Press `[i] info` to inspect the breaking-change & OSV.dev detail panel (`cache v4 runs on Node 20...`), then press `[enter] apply`.
3. **Result:** Atomic workflow rewrites (`✅ 4 actions upgraded across 2 files`), `.actup.yaml` updated with `sha: true` pins, and `actions.lock` synced.

## Tone
- Preset: `polished`
- Creative direction: Crisp, high-craft developer CLI & terminal product film
- Interpretation: Confident dark-mode terminal aesthetics, exact Lipgloss ANSI colors (`#ff5faf`, `#5fffff`, `#5fd700`, `#ffaf00`, `#ff0000`), generous breathing room on copy, snappy terminal transitions, and zero corporate SaaS fluff.

## Format: landscape — 1920x1080
## Duration: 21 seconds

## Visual identity (from the project)
- Background: `#0b0f17` (deep terminal slate) with `#111827` / `#161f30` elevated terminal window chrome
- Accent: `#5fffff` (Lipgloss ANSI 87 cyan cursor) and `#ff5faf` (Lipgloss ANSI 205 pink selection)
- Semantic accents: `#5fd700` (ANSI 76 green `✓`), `#ffaf00` (ANSI 214 amber `⚠ breaking changes`), `#ff5555` (high-contrast red `⚠ vulnerable` for WCAG compliance on dark backgrounds)
- Text: `#f0f6fc` (primary bright text) and `#94a3b8` (muted secondary text, WCAG AA compliant)
- Display font: `'Inter'`, `-apple-system, BlinkMacSystemFont, sans-serif`
- Body / Terminal font: `'JetBrains Mono'`, `'SFMono-Regular', Consolas, monospace`
- Strongest visual element: The Bubble Tea TUI checklist & `[i] info` detail view alongside the Gopher-hugging-Octocat `logo.png`

## Share copy (draft)
Introducing actup: a version manager for GitHub Actions, like nvm for Node or fvm for Flutter. Pin, sync, audit, and upgrade `uses:` refs across every workflow from one `.actup.yaml` manifest — with breaking-change warnings and OSV.dev checks before anything touches disk.

## Audio direction
- Role: Warm, steady electronic bed with crisp, low-risk terminal and UI accents
- Music: `happy-beats-business-moves-vol-12-by-ende-dot-app.mp3` (109.96 BPM)
- Music treatment: Starts at 0.0s at `data-volume="0.32"` with `data-fade-in="0.6"` and `data-fade-out="1.8"` under the final logo and install commands
- Music cue guidance: Precomputed preset read (`happy-beats-business-moves-vol-12-by-ende-dot-app.music-cues.md`, 109.96 BPM).
  - Strong cues targeted: `8.74s` (0.99, strong_beat — `[i] info` detail & `[enter] apply` payoff), `13.11s` (0.98, strong_beat — `.actup.yaml` manifest & CLI command suite reveal), `17.47s` (0.99, strong_beat — final Gopher+Octocat logo & install outro).
  - Beat-grid windows for sequential events: `4.39s, 4.91s, 5.34s, 6.00s` (TUI checklist rows appearing in Scene 2); `13.64s, 14.73s, 15.84s` (every-other-beat command pill highlights in Scene 4 so each command is comfortably readable).
- Audio-reactive treatment: Subtle; use pre-extracted music RMS and bass energy from `happy-beats-business-moves-vol-12-by-ende-dot-app.music-cues.json` to modulate the ambient cyan/pink radial glow opacity and terminal card border luminance. No waveform or equalizer bars.
- SFX posture: Sparse, motion-matched, low high-frequency risk picks from `sfx-analysis.md`
- Audio-coupled moments:
  - Scene 1: Soft impact reveal (`impactSoft_medium_001.ogg`) on hook headline; subtle click (`click_003.ogg`) as `uses:` tag badge flips
  - Scene 2: Terminal launch (`impactSoft_medium_002.ogg`) and sequential row ticks (`click_002.ogg`, `click_005.ogg`) aligned to beat grid
  - Scene 3: Keypress (`click_003.ogg`) on `[i] info` modal open at `8.74s`, resonant bell (`impactBell_heavy_000.ogg`) on `[enter] apply` → `✅ 4 actions upgraded across 2 files`
  - Scene 4: Card slide (`card-slide-1.ogg`) on `.actup.yaml` manifest reveal at `13.11s`
  - Scene 5: Warm bell payoff (`impactBell_heavy_003.ogg`) on final logo lockup at `17.47s`
- Restraint rule: Never stack harsh or high-HF clicks; keep SFX under 0.65 volume so the steady electronic pulse stays clean.

## Storyboard

### Scene 1 — Hook: The Workflow Drift Problem — 3.8s (0.0s – 3.8s)
On a deep slate canvas (`#0b0f17`), a sleek `.github/workflows/ci.yml` code card shows scattered `uses: actions/checkout@v4` and `uses: actions/cache@v3` lines. Large display type slams in:
"Action versions live inside every workflow file."
Then the sub-line settles underneath:
"Nothing is rewritten until you ask."
Sequential/interaction: yes — the `uses:` version tags pulse with an amber `⚠ major jump?` indicator before snapping to `actup`'s clean command prompt `$ actup`.
Audio intent: Immediate authority and precision.
Audio-coupled idea: `impactSoft_medium_001.ogg` on headline entry; `click_003.ogg` on prompt command.
Music: `happy-beats-business-moves-vol-12-by-ende-dot-app.mp3` (steady, clean pulse).
Transition mood: Clean slide + crossfade → Scene 2.

### Scene 2 — The Interactive TUI Checklist — 4.9s (3.8s – 8.7s)
Full recreation of the real `actup` Bubble Tea TUI checklist inside a crisp terminal window:
Header: `actup — 4 actions found across 2 files`
4 rows populate sequentially on the beat grid (`4.39s, 4.91s, 5.34s, 6.00s`) and hold settled for 2.7s so every line is easily read:
- `[✓] actions/checkout         v4 → v7  (2 files)`
- `[✓] actions/setup-go         v4 → v7  ⚠ breaking changes`
- `[✓] actions/cache            v3 → v6  ⚠ breaking changes`
- `[✓] actions/upload-artifact  v3 → v7  ⚠ vulnerable`
Footer: `[space] toggle  [a] all  [n] none  [i] info  [enter] apply  [q] quit`
Sequential/interaction: yes — 4 action rows appear on consecutive beats, then the cyan cursor highlights `actions/cache` and presses `[i] info`.
Audio intent: Tactile, responsive terminal rhythm.
Audio-coupled idea: Beat-grid ticks (`click_002.ogg`, `click_005.ogg`) as the checklist populates and cursor selects `[i] info`.
Transition mood: Snappy modal focus on strong cue `8.74s` → Scene 3.

### Scene 3 — Inspect Risk & Apply Atomically — 4.4s (8.7s – 13.1s)
Beat-locked to the strong cue at `8.74s`, the `[i] info` detail overlay opens for `actions/cache — Breaking Changes` (`v3 → v6`):
- `⚠ cache v4 runs on Node 20; self-hosted runners must be v2.308.0 or newer.`
- `⚠ cache v4 requires the cache service v2 backend.`
 Alongside an OSV.dev advisory badge (`OSV.dev advisory check passed`). Then simulated keypress `[enter] apply` triggers the green summary banner:
`✅ 4 actions upgraded across 2 files` + `actions.lock synced`.
Sequential/interaction: yes — simulated `[i] info` inspection followed by `[enter] apply` confirmation state change at `10.93s` strong beat.
Audio intent: Relief and confidence — catching breaking changes before they break CI.
Audio-coupled idea: `impactSoft_medium_004.ogg` at `8.74s` detail reveal; `impactBell_heavy_000.ogg` at `10.93s` when `✅ 4 actions upgraded across 2 files` lands.
Transition mood: Clean wipe on strong cue `13.11s` → Scene 4.

### Scene 4 — One Manifest, Full Version Manager — 4.3s (13.1s – 17.4s)
Beat-locked to `13.11s`, split view showing `.actup.yaml` ("One manifest as source of truth — like `.nvmrc` for Node") with real repo config (`sha: true`, `pins:`, `actions: actions/checkout: v7`) on the left, and the imperative command suite on the right appearing on every-other beat (`13.64s, 14.73s, 15.84s`) and holding:
- `actup use actions/checkout@v7` — Pin across all workflows
- `actup audit` — OSV.dev + transitive `actions.lock` scan
- `actup sync` & `actup outdated` — Enforce in CI & pre-commit
Sequential/interaction: yes — 3 command cards highlight every 2 beats (~1.1s apart) and stay on screen so all 3 remain readable.
Audio intent: Structured, complete toolkit energy.
Audio-coupled idea: `card-slide-1.ogg` on manifest entrance at `13.11s`; soft `drop_001.ogg` accents on command cards.
Transition mood: Smooth scale reveal on strong cue `17.47s` → Scene 5.

### Scene 5 — Outro: One Reviewable Change — 3.6s (17.4s – 21.0s)
Beat-locked to `17.47s`, the `actup` Gopher+Octocat logo (`logo.png`) and wordmark `actup` land center-stage with headline:
"Every upgrade is one reviewable change."
Subheadline: "A version manager for GitHub Actions."
Install bar at bottom:
`brew install lynicis/tap/actup`  ·  `gh extension install lynicis/gh-actup`  ·  `github.com/lynicis/actup`
Sequential/interaction: none — fast 0.4s entrance, then 3.2s settled hold for reading and poster capture.
Audio intent: Clean, memorable landing with music fading smoothly underneath.
Audio-coupled idea: `impactBell_heavy_003.ogg` on the `17.47s` strong beat as the logo and install commands lock in.
Transition mood: Hold to end (21.0s).

**Music mood for this video:** `polished` (steady, clean electronic pulse — `vol-12`)
**Audio summary:** A steady 110 BPM electronic bed with subtle RMS/bass glow reactivity and low-HF terminal clicks and bell payoffs locked to 3 strong musical cues (`8.74s`, `13.11s`, `17.47s`).
