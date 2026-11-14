# actup — Deep-Dive Analysis & Implementation Spec

> Source: `docs/plans/brainstorming-ideas.md` items `1a`–`1d`, `2a`–`2d`.
> Read-only investigation performed against the working tree; no product code changed.
> Every file:line below was verified by reading the cited code. Claims marked `[INFERENCE]`
> were not executed.

## Verification notes that affect the specs

- `go.mod:10` uses `github.com/google/go-github/v89 v89.0.0`. `AGENTS.md` still says `v62`; the spec targets **v89**.
- GitHub tag behavior was checked against the live API:
  - `GET /repos/{owner}/{repo}/tags` returns the **peeled commit SHA** even for annotated tags (verified with `pnpm/action-setup@v4`, `Swatinem/rust-cache@v2`, `sigstore/cosign-installer@v3`).
  - `GET /repos/{owner}/{repo}/commits/{ref}` (`Repositories.GetCommitSHA1`) resolves lightweight tags, annotated tags (peeled), and branches to a commit SHA in **one** call. This removes the need for `Git.GetTag` dereferencing in 2a.
- `goccy/go-yaml` behavior verified with a throwaway program:
  - `pins: {}` unmarshals to a **non-nil empty** `map[string]string` — usable as a schema-mode marker.
  - Unquoted `actions/checkout: 4` coerces to the string `"4"`, so `map[string]string` is safe despite `schema.json` allowing integers.

## Summary

| Item | Effort | Risk | Wave | Primary packages |
|---|---|---|---|---|
| 1a manifest current-state vs constraint | M | Medium | 1 | `internal/config`, `internal/github`, `cmd`, `internal/tui`, `schema.json` |
| 1b atomic batch writes + permissions | S | Low | 1 | `internal/upgrader` |
| 1d mixed-version breaking checks | M | Low–Medium | 1 | `cmd`, `internal/tui`, `internal/parser` |
| 2a commit SHA pinning + comment parsing | L | Medium–High | 2 | `internal/parser`, `internal/github`, `internal/upgrader`, `cmd` |
| 2b `actup audit --fix` | M | Medium | 2 | `cmd`, `internal/vulnscanner`, `internal/upgrader` |
| 1c registry coverage + `with:` inspection | M | Low | 3 | `internal/parser`, `internal/breakingchanges` |
| 2c composite action discovery | S | Low | 3 | `internal/scanner` |
| 2d grouped / filtered upgrades | M | Low–Medium | 3 | `cmd`, `internal/config`, `schema.json` |

Recommended implementation order: **1b → 1a → 1d → 2a → 2b → 2c → 1c → 2d**.
`1b` is isolated. `1a` and `1d` both touch `runNoTUI` and `tui.loadActions`, so do them consecutively.
`2a` changes `ActionRef` semantics used by every later item, so it precedes `2b`/`1c`.

---

## 1a. `.actup.yaml` lifecycle: recorded state vs. version constraint

### 1. Exact files & symbols

| File | Symbols / lines | Role |
|---|---|---|
| `internal/config/config.go` | `Config` `:17-20`; `LoadDefault` `:22-33`; `Load` `:36-53`; `SaveDefault` `:56-65`; `Save` `:68-76`; `Render` `:78-99`; `formatScalar` `:103-118` | schema, load/save, deterministic render |
| `internal/github/client.go` | `ResolveVersion` `:159-193`; `ResolveAll` `:196-231` | exact-value short-circuit at `:186-188` |
| `cmd/root.go` | config load `:119-132`; resolution `:151-154`; `runCheck` resolution `:246-252` | bare `actup`, `outdated` |
| `internal/tui/tui.go` | `loadActions` `:263-327`; `Current` `:302`; `UpToDate` `:308-314`; breaking check `:322-323` | TUI resolution |
| `cmd/ls.go` | config load `:58-68`; `runLsDetail` `:73-97`; `runLsTable` `:100-158`; `resolveLatestForAll` `:185-196` | `ls`, `ls --outdated` |
| `cmd/init.go` | `doInit` `:46-92`; write `:74`; `SaveDefault` `:85` | bootstrap writer |
| `cmd/common.go` | `savePin` `:109-122`; `mostCommonVersion` `:90-106`; `confirmBreakingAll` `:157-175` | pin writer / helper |
| `cmd/sync.go` | `doSync` `:61-180`; keys `:83`; pin read `:89,111`; numeric resolve `:120-124`; apply `:167` | manifest applier |
| `cmd/pin.go` | `runPin`, `runUnpin` | manifest-only commands |
| `schema.json` | `properties` `:5-38`; `additionalProperties: false` `:37` | editor validation |

Call graph (verified):

- `ResolveVersion` callers: `ResolveAll` (`client.go:216`), `cmd/ls.go:191`. `ResolveAll` callers: `cmd/root.go:154` (`runNoTUI`), `cmd/root.go:252` (`runCheck`), `internal/tui/tui.go:290`.
- `LoadDefault` callers: `cmd/root.go:119`, `cmd/common.go:112` (`savePin`), `cmd/sync.go:62`, `cmd/outdated.go:48`, `cmd/ls.go:58`, `cmd/init.go:51`, `cmd/pin.go` (`runUnpin`).
- `SaveDefault` callers: `cmd/init.go:85`, `cmd/common.go:121`, `cmd/pin.go` (`runUnpin`). **No upgrade path writes the manifest.**

### 2. Data model / signature changes

The manifest currently holds three meanings in one map: exact pin, numeric major constraint, and `skip`. The minimal separation that keeps every existing test green is a second map plus a legacy marker:

```go
// internal/config/config.go
type Config struct {
	Major   *int              `yaml:"major,omitempty"`
	Actions map[string]string `yaml:"actions,omitempty"` // recorded workflow state (lock)
	// Pins holds user-authored constraints for managed manifests.
	// nil means a legacy manifest: every Actions entry is a constraint.
	Pins map[string]string `yaml:"pins,omitempty"`
}

func (c *Config) IsManaged() bool { return c != nil && c.Pins != nil }

// EffectivePins returns explicit constraints, or Actions for legacy manifests.
// Numeric majors and "skip" are always constraints, even when they appear in
// the recorded Actions section of a managed manifest.
func (c *Config) EffectivePins() map[string]string {
	if c == nil {
		return nil
	}
	if c.Pins == nil {
		return c.Actions
	}
	out := maps.Clone(c.Pins)
	for key, value := range c.Actions {
		if value != "skip" {
			if _, err := strconv.Atoi(value); err != nil {
				continue
			}
		}
		if _, exists := out[key]; !exists {
			out[key] = value
		}
	}
	return out
}
```

No change to `github.ResolveVersion` or `ResolveAll` signatures. Call sites pass `cfg.EffectivePins()` instead of `cfg.Actions`.

`Render` must be deterministic and preserve the managed marker:

```go
// after major:
if cfg.Pins != nil {
	if len(cfg.Pins) == 0 {
		b.WriteString("\npins: {}\n")
	} else {
		b.WriteString("\npins:\n")
		for _, key := range slices.Sorted(maps.Keys(cfg.Pins)) {
			fmt.Fprintf(&b, "  %s: %s\n", formatScalar(key), formatScalar(cfg.Pins[key]))
		}
	}
}
// then the existing sorted actions block
```

`pins: {}` must be emitted explicitly. `omitempty` would drop an empty map, reload it as `nil`, and silently flip a managed manifest back to legacy semantics.

`schema.json` gains a `pins` object with the same `additionalProperties` (`string` or `integer`) as `actions`. `additionalProperties: false` at `:37` means omitting it makes editors reject every managed manifest.

### 3. Command semantics after the change

| Command | Read | Write | Behavior |
|---|---|---|---|
| `init` | legacy config for merge | `Actions[key] = mostCommonVersion(refs)`; set `Pins = map[string]string{}` when nil | fresh file is managed; `--force` preserves existing `Pins` and `Actions` |
| `use` | `EffectivePins` for skip filtering | `Actions[key] = target`; if `cfg.IsManaged()`, `Pins[key] = target` | legacy stays legacy (all exact entries remain constraints) |
| `pin` | same | same as `use` | |
| `unpin` | same | delete key from `Actions` and `Pins` | |
| `sync` | union of `Actions` and `EffectivePins` | `Actions[key] = target` after a successful write | numeric pin → latest in major; exact pin → verbatim; snapshot-only key → verbatim; `skip` → skip |
| `outdated` / `ls --outdated` / TUI / `--no-tui` | `EffectivePins` | none for resolution | recorded exact `Actions` values are ignored; numeric and exact pins are honored |
| `ls` (offline) | `Actions` + `EffectivePins` | none | version-in-use from `Actions`; pin column from `EffectivePins` |
| upgrade (`runNoTUI`, TUI) | `EffectivePins` | `Actions[key] = latest` after success, one save per run | never creates a manifest; no-op for legacy or absent config |

`sync` key set becomes the union, not only `cfg.Actions` (`sync.go:83`), and the empty-manifest guard at `sync.go:66` must accept either map:

```go
pins := cfg.EffectivePins()
if len(cfg.Actions) == 0 && len(pins) == 0 {
	fmt.Println("No action pins found in .actup.yaml.")
	return nil
}
keys := slices.Sorted(maps.Keys(unionKeys(cfg.Actions, pins)))
```

For each key: `pin, pinned := pins[key]`; `snapshot := cfg.Actions[key]`. Target selection:

```go
switch {
case pin == "skip":           continue
case isNumeric(pin):          target = client.LatestTag(..., TagMode{Major: n})
case pinned:                  target = pin
case snapshot != "":          target = snapshot // restore lock
}
```

The existing `needsClient` scan (`sync.go:92-104`) must consider numeric pins **and** numeric snapshots? Snapshots are exact refs, so no API needed. Keep numeric-only.

Upgrade write-back helper in `cmd/common.go`:

```go
// recordUpgrades updates a managed manifest after successful rewrites.
// Legacy and absent manifests are left untouched.
func recordUpgrades(cfg *config.Config, upgraded map[string]string) error {
	if cfg == nil || !cfg.IsManaged() || len(upgraded) == 0 {
		return nil
	}
	for key, ref := range upgraded {
		cfg.Actions[key] = ref
	}
	return config.SaveDefault(cfg)
}
```

Call once at the end of `runNoTUI` (`cmd/root.go:139-235`) and from TUI `applyUpgrades` (`internal/tui/tui.go:353-417`) when `!dryRun` and `ApplyAllUpgrades` returned nil. `runNoTUI` already has per-key success/failure; collect `map[key]latest` only for successful non-dry-run keys.

### 4. Edge cases & failure modes found

- **Legacy ambiguity is unavoidable.** A file with `actions:` and no `pins:` may be an old `init` snapshot (bug case) or a deliberate exact pin. The design preserves old behavior until the file is regenerated. `actup init --force` on a legacy file initializes `Pins` empty, converting exact `Actions` entries into recorded state; document this one-way migration in README `:264-277` and the `init` help text.
- **Empty `pins` must round-trip.** Verified `pins: {}` → non-nil empty map. `Render` must write it; `TestRender_EmptyConfig` (`internal/config/config_test.go:91-96`) remains green because `Pins` is nil there.
- **`TestDoInit_CreatesManifestAndRefusesOverwrite`** (`cmd/versionmanager_test.go:234-277`) asserts `cfg.Actions == "v3"/"v4"` after init; keeping snapshot writes into `Actions` preserves it.
- **`TestDoSync_AppliesPins`** (`cmd/versionmanager_test.go:194-218`) writes a legacy manifest with only `actions:`. `EffectivePins()` falls back to `Actions`, so sync still applies it.
- **`TestDoUse_UpdatesWorkflowsAndSavesPin`** (`cmd/versionmanager_test.go:105-141`) asserts `cfg.Actions["actions/checkout"] == "v5"`. `savePin` must keep writing `Actions`.
- **`filterSkippedActions`** (`cmd/root.go:124-137`) and `runAudit` (`cmd/audit.go:75`) must use `EffectivePins()`; otherwise a managed manifest's `skip` in `Pins` is ignored.
- **`runUnpin`** (`cmd/pin.go:81-105`) guards on `cfg.Actions[spec.Key] == ""` and deletes only from `Actions`. In a managed manifest it must check `EffectivePins()` and delete the key from both maps, or `sync`/`outdated` keep honoring the removed pin.
- **`Save` writes `.actup.yaml` with mode `0600`** (`config.go:71`). Same class of permission bug as 1b. Out of scope for 1a unless fixed alongside; note in 1b.
- **`SaveDefault` may write `.actup.yml`** while `init` prints “wrote .actup.yaml” (`cmd/init.go:84`). Cosmetic; fix the message while touching init.
- **`runCheck` calls `os.Exit`** (`cmd/root.go:257,283`), so it cannot be extended with write-back. Keep it read-only.
- **Concurrent save risk** `[INFERENCE]`: TUI saves from a `tea.Cmd` goroutine; no other writer exists at that moment, so a single `SaveDefault` call is safe.
- **`--no-save`** updates workflows but not `Actions`; manifest drift is intentional and pre-existing.

### 5. Tests

Existing tests that pin behavior:

- `cmd/versionmanager_test.go:234-277` (`TestDoInit_*`), `:105-192` (`TestDoUse_*`), `:194-232` (`TestDoSync_*`), `:279-316` (`TestRunPinAndUnpin`).
- `internal/config/config_test.go:67-100` (`TestRender_*`), `:98-127` (`TestSaveLoadRoundtrip`), `:129-168` (`TestSaveDefault_*`).
- `cmd/root_test.go:10-27` (`TestFilterSkippedActions`) — only uses legacy `Actions`.

Add:

1. `internal/config`: `TestEffectivePins_LegacyFallsBackToActions`, `TestEffectivePins_ManagedPrefersPins`, `TestRender_ManagedEmptyPins` (exact `pins: {}` string), `TestRender_ManagedPinsSorted`, `TestLoadPinsEmptyIsNonNil`.
2. `cmd`: `TestInitCreatesManagedManifest` (Actions exact + Pins empty), `TestUseOnLegacyStaysLegacy`, `TestUseOnManagedWritesPin`, `TestUnpinRemovesBoth`, `TestSyncRestoresSnapshotOnlyKey`, `TestSyncPrefersPinOverSnapshot`, `TestRecordUpgrades_ManagedOnly`, `TestRecordUpgrades_NoManifestNoop`.
3. `internal/github`: table test for `ResolveVersion` is blocked by the concrete client. Extract the pin-precedence branch into a pure helper, e.g. `func resolvePin(cfgActions map[string]string, key string, defaultMajor int) (major int, exact string)`, and unit-test that no API is needed. Do not add an interface solely for this test.
4. `internal/tui`: no config tests exist; add a model-level test only if `loadActions` is refactored to accept a resolver seam. Otherwise rely on cmd tests.

### 6. Effort / risk

**Effort M (1.5–2 d). Risk medium** because the schema meaning changes and legacy files keep the old behavior until migrated. Main hazards: forgetting `EffectivePins()` at one resolution site, and the empty-`pins` round-trip. Mitigation: one method is the only source; grep for `cfg.Actions` after the change and require every read site to be either `EffectivePins()` (constraints) or an explicit snapshot/sync path.

---

## 1b. Atomic file writes: preserve permissions and batch per file

### 1. Exact files & symbols

| File | Symbols / lines | Defect |
|---|---|---|
| `internal/upgrader/upgrader.go` | `Upgrade` `:15-18`; `usesPath` `:20-27`; `replaceInFile` `:29-88`; `showDryRunDiff` `:85-93`; `ApplyAllUpgrades` `:95-116` | temp file `0600`; N read/write/rename cycles per file |
| `internal/upgrader/upgrader_test.go` | `TestReplaceInFile` `:120-162`; `TestApplyAllUpgrades` `:264-304`; `TestApplyAllUpgradesContinuesPastErrors` `:384-428`; `TestApplyUpgradesConcurrent` `:430-…` | no mode assertions; per-upgrade failure semantics |

Verified behavior:

- `os.CreateTemp` at `upgrader.go:64` creates `0600`; `os.Rename` at `:84` replaces the inode, so `0644` workflows become `0600` on POSIX. Ownership/ACLs/xattrs are not preserved.
- `ApplyAllUpgrades` groups by file (`:96-99`) but still loops upgrades and calls `replaceInFile` per upgrade (`:102-109`). For N refs in one file: N reads, N temp writes, N renames.
- Line numbers stay valid across passes because each replacement changes one line in place and `Split`/`Join` preserves the line count. This is accidental, not guaranteed by an invariant.
- A failure after some passes leaves a partially rewritten file. `errors.Join` is returned only after all passes (`:115`).
- Dry-run order is map-iteration order (`byFile`), so multi-file output is nondeterministic.

### 2. Data model / signature changes

```go
// replaceAllInFile applies every upgrade for one file in a single
// read-modify-write and preserves the original permission bits.
func replaceAllInFile(file string, upgrades []Upgrade) error

// replaceInFile remains as a thin wrapper so existing tests keep compiling.
func replaceInFile(file string, action parser.ActionRef, newTag string) error {
	return replaceAllInFile(file, []Upgrade{{Action: action, NewTag: newTag}})
}
```

`Upgrade` is unchanged for 1b (2a adds `NewSHA`/comment fields).

Implementation constraints:

1. `os.Stat(file)` before reading; capture `info.Mode().Perm()`.
2. Build an index `map[int]Upgrade` keyed by `Action.Line`. If two upgrades target the same line:
   - identical `NewTag` → dedupe;
   - different `NewTag` → record an error for that line and apply the first.
3. Read the file once, split, then for each upgrade locate `lines[line-1]` and apply the **existing** regex semantics from `:46-48`:
   `\b<usesPath>@<Current>` → `<usesPath>@<NewTag>`, leaving trailing comments untouched.
   Keep the exact error text `could not find expected uses line at line %d` (`:61`).
4. Collect per-upgrade failures; still apply upgrades that matched, then write the file once. Return `errors.Join(errs...)` at the end. This preserves `TestApplyAllUpgradesContinuesPastErrors`: the good checkout upgrade lands, the bad cache line errors.
5. Write temp in `filepath.Dir(file)`, `os.Chmod(tmpPath, mode)` **before** `os.Rename`, then close and rename. Keep existing `%w` wrapping prefixes (`read file`, `create temp file`, `write temp file`, `close temp file`, `rename temp file`).
6. `ApplyAllUpgrades`:
   - sort file keys with `slices.Sorted(maps.Keys(byFile))`;
   - sort upgrades within a file by `Action.Line` for deterministic diffs;
   - dry-run: print all diffs for a file, then continue;
   - non-dry-run: one `replaceAllInFile` call per file;
   - collect file-level errors with `fmt.Errorf("%s: %w", file, err)`.
7. `showDryRunDiff` currently fabricates `uses: path@ref` without indentation. For 1b, leave rendering alone (scope); 2a must extend it for SHA comments.

Cross-platform notes:

- POSIX: `Chmod` restores the mode. Ownership cannot be preserved without privileges; do not attempt `Chown`.
- Windows: `os.Rename` uses `MoveFileEx(MOVEFILE_REPLACE_EXISTING)` and replaces the destination. `os.Chmod` only toggles the read-only attribute; ACLs are not preserved. Document as a platform limit.
- Symlinked workflow files: `os.Rename` replaces the symlink itself. Existing behavior; not changed here.

### 3. Edge cases & failure modes

- File unreadable/stat fails: return the wrapped error before creating a temp file.
- Target line missing: file is still written with the other successful replacements; caller sees the joined error.
- Duplicate line entries: conflict error, first wins.
- `NewTag` empty: current code would write `path@`; add a guard returning `invalid empty tag` (defensive; callers never pass empty today).
- Temp cleanup: existing deferred `os.Remove` + `Close` (`:70-73`) must remain.
- Concurrent processes editing the same file: last rename wins. File locking is out of scope.
- CRLF files: `strings.Split(content, "\n")` leaves `\r` at line ends. The current regex matches before the comment/trailing space, so it works; do not normalize line endings.

### 4. Tests

Existing tests that must stay green: `TestApplyUpgrades` (`:12`), `TestApplyUpgradesMultilineSteps` (`:52`), `TestApplyUpgradesDryRun` (`:94`), `TestReplaceInFile` (`:133`), `TestReplaceInFileMultilineSteps` (`:174`), `TestApplyAllUpgrades` (`:216`), `TestReplaceInFileSubpathAction` (`:263`), `TestReplaceInFileTabsAndComment` (`:305`), `TestApplyAllUpgradesContinuesPastErrors` (`:339`), `TestApplyUpgradesConcurrent` (`:387`).

Add:

1. `TestApplyAllUpgradesPreservesMode` — write `0644`, apply, assert `Mode().Perm() == 0644` (skip mode assertion on Windows; assert read-only bit only).
2. `TestReplaceAllInFileSingleWrite` — two distinct lines; assert both changed and no `actup-*.tmp` remains in the directory.
3. `TestReplaceAllInFilePartialMissStillAppliesOthers` — batch analogue of the existing continues-past-errors test.
4. `TestReplaceAllInFileDuplicateLineConflict` — two upgrades same line, different tags → error, first applied.
5. `TestApplyAllUpgradesDryRunDeterministic` — two files, capture stdout, run twice, assert identical order (sort files/lines).

### 5. Effort / risk

**Effort S (0.5 d). Risk low.** The package is isolated and its tests already pin the replacement semantics. The only behavioral change beyond mode preservation is that a partial failure now writes once instead of after each successful pass; the existing test explicitly requires the successful upgrade to land.

---

## 1d. Mixed-version handling in `runNoTUI` and the TUI

### 1. Exact files & symbols

| File | Symbols / lines | Defect |
|---|---|---|
| `cmd/root.go` | `runNoTUI` `:139-235`; `acts := grouped[key]` `:165`; `current := acts[0].Current` `:166`; `needsUpgrade` `:180-185`; breaking call `:193-195` | checks only the first current version |
| `internal/tui/tui.go` | `ActionItem` `:44-63`; `loadActions` `:263-327`; `Current` `:302`; `UpToDate` `:308-314`; `BreakingChanges` `:322-323`; `applyUpgrades` `:353-417`; `viewChecklist` `:490`; `viewDetail` `:552` | first-version-only `UpToDate` can skip mixed groups entirely |
| `cmd/common.go` | `confirmBreakingAll` `:157-175`; `confirmBreaking` `:126-155` | already correct; not used by `runNoTUI`/TUI |
| `cmd/use.go` | `checkUseBreaking` `:174-177` | uses `confirmBreakingAll` |
| `cmd/sync.go` | `confirmBreakingAll` call `:136-143` | uses `confirmBreakingAll` |

Concrete failures:

1. **`runNoTUI`:** `ci.yml` uses `checkout@v4` and `release.yml` uses `checkout@v2`. Group order puts `v4` first. `needsUpgrade` contains the `v2` ref, so it is rewritten, but `breakingChangesFor(key,"v4","v4")` returns nothing (`registry.go:60-69`). The `v2 → v4` warning is never shown. If `v2` is first, the warning still does not appear for the current registry: `Registry.Check("v2","v4")` drops the checkout 2→3 entry because `toMajor (4) > entry.ToMajor (3)` (`registry.go:57-59`). Only a `v2 → v3` target returns it. Both the call-site and the registry-range logic must change; see 1c.
2. **TUI:** `UpToDate` is `r.Latest == acts[0].Current` (`:308-314`). With `[v4, v2]` and latest `v4`, the item is marked up-to-date and never selected (`:315-317`), so `applyUpgrades` skips it (`:356-358`) and the `v2` reference is **never upgraded**. `BreakingChanges` is also computed only from `item.Current` (`:322-323`).
3. **TUI display:** `viewChecklist` prints one version (`:490`); `viewDetail` prints `item.Current → item.Latest` (`:552`), so a mixed group renders as `v4 → v4` or just `v4`.

### 2. Data model / signature changes

Add a shared helper in `internal/parser`:

```go
// DistinctVersions returns distinct version strings in first-seen order.
func DistinctVersions(refs []ActionRef) []string
```

TUI model:

```go
type ActionItem struct {
	Owner           string
	Repo            string
	Current         string   // first-seen version, retained for compatibility/display
	Currents        []string // all distinct versions, first-seen order
	Latest          string
	FileCount       int
	Selected        bool
	UpToDate        bool
	// ... existing fields
}
```

`loadActions` changes:

```go
currents := parser.DistinctVersions(acts)
item := ActionItem{
	Owner:     owner,
	Repo:      repo,
	Current:   currents[0],
	Currents:  currents,
	FileCount: len(acts),
}
// ...
if r.Latest == item.Current {
	item.UpToDate = true
}
```

Replace with the same predicate `runNoTUI` uses:

```go
stale := 0
for _, act := range acts {
	if act.Current != r.Latest {
		stale++
	}
}
item.UpToDate = stale == 0
```

Breaking changes become a union over distinct currents:

```go
if registry != nil && !item.UpToDate && !item.APIError {
	seen := map[string]bool{}
	for _, current := range item.Currents {
		for _, bc := range registry.Check(key, current, item.Latest) {
			id := bc.Type + "\x00" + bc.Input + "\x00" + bc.Message
			if seen[id] {
				continue
			}
			seen[id] = true
			item.BreakingChanges = append(item.BreakingChanges, bc)
		}
	}
}
```

`BreakingChange` has no ID (`registry.go:14-20`); dedupe on `(Type, Input, Message)`.

Display:

- `viewChecklist`: if `len(item.Currents) > 1`, render a compact summary such as `v2 ×1, v4 ×1` instead of `item.Current`. Reuse `summarizeActionVersions` logic, but it lives in `cmd/ls.go:161-181` and is unexported; either duplicate three lines in TUI or move it to `parser`/a shared internal package. Prefer a local TUI helper to avoid a `cmd`→`tui` dependency.
- `viewDetail`: list each affected transition (`v2 → v5`, `v4 → v5`) above the breaking-change messages.

`runNoTUI` change (`cmd/root.go:193-200`):

```go
proceed, wasSkipped := confirmBreakingAll(key, acts, r.Latest, registry, dryRun, force, reader, isTTY)
breakingSkipped += wasSkipped
if !proceed {
	continue
}
```

`current` remains only for progress/dry-run text (`:224,235`); derive a display string from `parser.DistinctVersions(acts)` or keep the first for brevity. `confirmBreakingAll` already dedupes and only prints when `Check` returns changes, so `v4 → v4` produces no prompt.

No change to `tui.Run(actions, token, dryRun, semverMode, majorVer, cfg)`.

### 3. Edge cases & failure modes

- All refs at latest → `UpToDate` true, no selection (unchanged).
- Mixed group, some refs at latest → `UpToDate` false; `applyUpgrades` rewrites every ref matching the key, including ones already at latest. The upgrader replaces same-value refs harmlessly, but to avoid noise skip `act.Current == item.Latest` when building `upgrades` (`tui.go:360-368`).
- Same action with different subpaths: grouping is by `owner/repo` (`parser.go:115-121`); the TUI applies one latest tag to all subpaths. Existing behavior; note it.
- Different currents that map to the same breaking change: dedupe prevents repeated messages.
- **`Registry.Check` drops intermediate entries on multi-major jumps.** With the current conditions (`registry.go:52-67`), `Check("v2","v4")` skips a 2→3 entry because `toMajor > entry.ToMajor`. `confirmBreakingAll` alone therefore cannot fix the v2/v4 scenario. The range test must be “upgrade crosses this boundary”: include an entry when `fromMajor < entry.ToMajor && toMajor >= entry.ToMajor` (for `to_major != 0`), and when `fromMajor < entry.FromMajor && toMajor >= entry.FromMajor` for `to_major: 0`. Add table tests for 1→4, 2→4, 3→4, 2→3. This is implemented in 1c; 1d depends on it.
- Bare SHA refs (after 2a): `Current` differs per SHA; `UpToDate` comparison against a tag is always false. 2a changes comparison to `Version()`; do 2a before relying on this in SHA-pinned repos.
- `confirmBreakingAll` prints one prompt per distinct version on a TTY. For a group with three majors this is three prompts; acceptable, and matches `use`/`sync`.
- Non-TTY: the first breaking version causes `confirmBreaking` to skip with the existing message (`common.go:145-148`); `confirmBreakingAll` returns `false` immediately, so later versions are not reported. Existing semantics; if complete reporting is desired, change `confirmBreakingAll` to continue and return `proceed=false` at the end. Recommend keeping current behavior for scope.

### 4. Tests

Existing tests: `internal/tui/tui_test.go` covers rendering/selection only (`TestUpdateLoadingTransitionsToChecklist:36`, `TestViewChecklistRendering:155`, `TestViewChecklistVulnerableBadge:285`, `TestViewDetailVulnerabilities:308`); there is no `loadActions` or mixed-version test. `cmd/root_test.go:10` covers only `filterSkippedActions`.

Add:

1. `internal/parser`: `TestDistinctVersions` (dedupe, first-seen order, empty input).
2. `internal/tui`: pure helper test for `UpToDate` given `[v4,v2]` + latest `v4` → false; `[v4,v4]` → true. Extract the predicate into `func allAtLatest(acts []parser.ActionRef, latest string) bool` so it is testable without network.
3. `internal/tui`: `TestLoadActionsMixedVersions` is network-dependent; instead test the pure breaking-union helper `func breakingForAll(registry, key, currents, latest) []BreakingChange`. If a seam is added, use the embedded registry with the known checkout entry.
4. `cmd`: table test for `confirmBreakingAll` via a small constructor for `breakingchanges.Registry` (currently `entries` is unexported, `registry.go:30-32`). Add `func New(entries []Entry) *Registry` for tests; or test through the embedded registry with checkout `v2 → v3`.
5. Integration: two workflow files with checkout `v4`/`v2`, run `actup --no-tui --dry-run` with a fake resolver `[INFERENCE]`; assert the `v2 → v4` warning appears. The resolver is concrete (`github.NewClient`), so this needs a seam or a live API; prefer the pure helper tests above.

### 5. Effort / risk

**Effort M (1–1.5 d, including the registry range fix). Risk low–medium.** The `runNoTUI` change is two lines, but the scenario only produces a warning after `Registry.Check` handles multi-major jumps (the small subset of 1c below). The TUI change also fixes an upgrade-skipping bug that is worse than the missing warning. Main hazard: display regressions in existing rendering tests; keep `Current` populated.

---

## 2a. Commit SHA pinning (`--sha`) and comment version parsing

### 1. Exact files & symbols

| File | Symbols / lines | Change |
|---|---|---|
| `internal/parser/parser.go` | `ActionRef` `:14-22`; `usesRegex` `:24-25`; `extractFromFile` `:41-74`; `parseActionRef` `:76-107`; SHA skip `:93-95`; `isHex` `:109-112`; `GroupActions` `:115-121` | parse SHA + trailing comment; stop dropping SHAs |
| `internal/parser/parser_test.go` | `TestExtractActionsStripsTrailingComment` `:125`; `TestParseActionRef` `:151`; `TestParseActionRef_CommitSHASkipped` `:200` | invert SHA test; add comment cases |
| `internal/github/client.go` | `Client` `:33-36`; `LatestTag` `:57-75`; `fetchAllSemverTags` `:77-115`; `resolveLatestTag` `:117-157`; cache key `:29-31` | retain tag→SHA; add `ResolveTagSHA` |
| `internal/upgrader/upgrader.go` | `Upgrade` `:15-18`; `replaceInFile` `:29-88`; regex `:46-48`; `showDryRunDiff` `:85-93` | SHA-aware rewrite + comment replacement |
| `internal/vulnscanner/vulnscanner.go` | `Scan` match `:197`; `Finding.Ref` `:214` | compare `Version()` instead of raw SHA |
| `internal/breakingchanges/registry.go` | `Check` `:42-67`; `parseMajor` `:70-80` | callers pass version, not SHA |
| `cmd/common.go` | `mostCommonVersion` `:90-106`; `savePin` `:109-122` | version-aware |
| `cmd/root.go` | flags `:60-68`; `runNoTUI` comparisons `:180-185`; `runCheck` `:263-266` | `--sha`, version comparisons |
| `cmd/use.go` | flags `:52-58`; `clientForSpecs` `:154-168`; `applyUseToFileRefs` `:181-230` | SHA output, preserve existing SHA pins |
| `cmd/sync.go` | flags `:45-47`; target resolution `:118-124`; compare `:149-154` | SHA output |
| `internal/tui/tui.go` | `Run` signature `:103`; `loadActions` `:263-327`; `applyUpgrades` `:353-417` | SHA mode plumbing |
| `internal/config/config.go` | `Config` `:17-20`; `Render` `:78-99` | `sha` flag |
| `schema.json` | top-level properties `:5-38` | add `sha` boolean |
| `cmd/audit_test.go` | `TestAuditSkipsSHARefs` `:32-81` | replace: bare SHA now parses |

### 2. Data model / signature changes

Parser:

```go
type ActionRef struct {
	Owner   string
	Repo    string
	Path    string // full action path incl. subpath
	Current string // ref token exactly as written (tag, branch, or SHA)
	SHA     string // lowercase 40-hex commit SHA, "" otherwise
	Tag     string // version parsed from the trailing comment, "" if none
	Line    int
	File    string
}

func (a ActionRef) Pinned() bool { return a.SHA != "" }

// Version is the comparison value for OSV, breaking changes, outdated, ls,
// init, and the TUI. A SHA pin carries its version in the trailing comment.
func (a ActionRef) Version() string {
	if a.SHA != "" {
		return a.Tag
	}
	return a.Current
}
```

Parser regex (capture the comment instead of discarding it):

```go
var usesRegex = regexp.MustCompile(`^\s*(?:-\s+)?uses:\s*(.+?)(?:\s+#\s*(.*?))?\s*$`)
var shaRegex = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)
var commentVersion = regexp.MustCompile(`^(v?\d+(?:\.\d+){0,2})\b`)
```

`extractFromFile` attaches the comment only when `ref.SHA != ""`:

```go
usesValue := strings.TrimSpace(matches[1])
usesValue = trimMatchingQuotes(usesValue) // strip one pair of ' or "
ref, err := parseActionRef(usesValue)
// ...
if len(matches) > 2 && ref.SHA != "" {
	if m := commentVersion.FindStringSubmatch(strings.TrimSpace(matches[2])); m != nil {
		ref.Tag = m[1]
	}
}
```

`parseActionRef` replacement for the skip at `:93-95`:

```go
refObj := &ActionRef{Owner: owner, Repo: repo, Path: actionPath, Current: ref}
if shaRegex.MatchString(ref) {
	refObj.SHA = strings.ToLower(ref)
}
return refObj, nil
```

Keep the existing `nil` returns for `./`, `docker://`, wrong `@` count, and paths with fewer than two segments. Add skips for `ref == ""` and `strings.Contains(ref, "${{")` (matrix expressions must not be rewritten). Quote stripping is needed because the current parser produces owner `"actions` for `uses: "actions/checkout@v4"`.

Upgrader:

```go
type Upgrade struct {
	Action parser.ActionRef
	NewTag string // human-readable tag, always set
	NewSHA string // non-empty => write "@<sha> # <tag>"
}
```

If `NewSHA == ""`, keep the current byte-preserving behavior (including arbitrary trailing comments; `TestReplaceInFileTabsAndComment`). If `NewSHA != ""`, replace the ref **and** normalize the trailing comment:

```go
pattern := regexp.MustCompile(
	`(` + regexp.QuoteMeta(usesPath(a)+"@"+a.Current) + `)(?:[ \t]+#.*)?$`,
)
replacement := usesPath(a) + "@" + u.NewSHA + " # " + u.NewTag
```

This avoids the `# newtag # oldtag` corruption that a naive append would create. `showDryRunDiff` must render the full new line including the comment.

GitHub client:

```go
// ResolveTagSHA resolves any tag/branch ref to its peeled commit SHA.
// One API call per (owner, repo, ref); cached in the existing sync.Map.
func (c *Client) ResolveTagSHA(ctx context.Context, owner, repo, ref string) (string, error)
```

Implementation: `c.client.Repositories.GetCommitSHA1(ctx, owner, repo, ref, "")` (`repos_commits.go:213`). Verified to peel annotated tags. Cache key `"sha/"+owner+"/"+repo+"/"+ref`. Validate the returned string with `shaRegex`.

Zero-cost path for tags already in the list: change `fetchAllSemverTags` to return `[]TagInfo` (`Name`, `SHA` from `tag.Commit.SHA`, `repos.go:993-996`) and cache the slice per repo, not per mode. `LatestTag` keeps returning only the name. Add:

```go
func (c *Client) LatestTagInfo(ctx context.Context, owner, repo string, mode TagMode) (TagInfo, error)
```

Callers that need SHA use `LatestTagInfo` when the tag came from the list, falling back to `ResolveTagSHA` for verbatim refs outside the fetched 300 tags. Cost: 1 call per unique repo/tag for verbatim targets; 3 tag-list pages per repo reused by all modes.

Config and CLI:

```go
type Config struct {
	Major   *int              `yaml:"major,omitempty"`
	SHA     bool              `yaml:"sha,omitempty"` // write refs as SHA + "# tag"
	Actions map[string]string `yaml:"actions,omitempty"`
	Pins    map[string]string `yaml:"pins,omitempty"` // from 1a
}
```

- Flags: `--sha` on root, `use`, and `sync`. Config `sha: true` is the default; `--sha=false` overrides when the flag is explicitly changed (`cmd.Flags().Changed("sha")`). `pin` records a logical tag; no `--sha` there. `outdated`/`ls` need no flag because they read `Tag`.
- `use --sha` writes the pin as the **tag** (not the SHA) and sets `cfg.SHA = true` so a later `sync` keeps the SHA form. A SHA in `.actup.yaml` is not enough to reconstruct the tag comment.
- `sync` with `cfg.SHA` resolves each target tag to SHA and writes `@sha # tag`.

Invariant: **an existing SHA-pinned ref is always rewritten as a SHA pin, even without `--sha`.** Otherwise a routine upgrade silently de-pins the repository. `--sha` only converts tag-pinned refs.

Comparison updates:

- `outdated`/TUI `UpToDate`: compare `ref.Version()` to the latest **tag**.
- `use`/`sync` skip checks: in SHA mode compare `ref.SHA == newSHA`; in tag mode compare `ref.Current == target`. Otherwise SHA lines are rewritten on every run.
- `applyUseToFileRefs` (`cmd/use.go:181-230`) currently dedupes on `ref.Current` (`:182-186`); use `ref.Version()` when deciding “already at target”, or a SHA pin is always treated as a different version. The same applies to the `sync` loop at `cmd/sync.go:149-154`.
- `mostCommonVersion`: use `Version()` and ignore empty (bare SHA without comment). If all versions are empty, `init` should skip the key with a warning instead of writing a SHA as a version.
- `vulnscanner`: use `site.Version()` for `affects` and keep `Finding.Ref = site.Current` (raw) plus a new `Finding.Version` if display needs it. A bare SHA has `Version() == ""`; OSV cannot match it — skip with a one-line warning rather than reporting “no vulnerabilities”.
- `breakingchanges`: callers pass `Version()`; empty means “unknown, cannot check”.
- `init`: `mostCommonVersion(refs)` uses `Version()`; bare SHA groups are skipped with `ⓘ <key>: SHA-pinned without a version comment`.

### 3. Edge cases & failure modes

- **Bare SHA, no comment:** parse succeeds with `Tag == ""`. `ls` shows the SHA; `outdated`/`audit`/breaking checks skip with an explicit notice. Do not silently treat it as up-to-date.
- **Comment forms:** `# v4` works for major parsing; `# v4.2.2` is exact; `# pinned to v4.2.2` is rejected by the anchored `commentVersion` regex. Decide: strict `# <version>` only (recommended; predictable) and document. Arbitrary text is preserved in tag mode but replaced in SHA mode.
- **Annotated tags:** handled by `GetCommitSHA1` and by `ListTags` (peeled). No `Object.Type == "tag"` branch is needed.
- **Subpath actions:** `Path` includes the subpath; `SHA` is owner/repo-level. The rewrite uses `usesPath` (`upgrader.go:20-27`), already covered by `TestReplaceInFileSubpathAction`.
- **Mixed tag + SHA for the same action:** grouping is owner/repo; keep per-site style: SHA sites get `NewSHA`, tag sites get `NewTag` unless `--sha`.
- **Matrix expressions:** `@${{ matrix.version }}` currently parses as a ref. Skip it in `parseActionRef`; otherwise it may be rewritten.
- **Quoted values:** strip one matching pair of quotes before splitting; the upgrader regex then matches the unquoted ref inside the quotes.
- **Force-moved tags:** the comment version is trusted; only a fresh SHA resolution detects the move. Document as a known limit.
- **40 non-hex chars:** treated as a branch/tag, not a SHA.
- **Rate limits:** 60/hr unauthenticated. Reuse the per-repo tag cache; one `GetCommitSHA1` per unique repo/tag. `use --sha` must create a client even for a verbatim ref (`clientForSpecs` currently returns nil for verbatim refs, `cmd/use.go:152-168`).
- **`.actup.yaml` with a raw SHA:** reject with “pin a tag and use `sha: true`” — a SHA cannot produce the human-readable comment.
- **Existing comments in tag mode:** preserve byte-for-byte. SHA mode normalizes to `# <tag>`.
- **Windows/dry-run:** unchanged from 1b.

### 4. Tests

Existing behavior to change:

| Test | File:line | Action |
|---|---|---|
| `TestParseActionRef_CommitSHASkipped` | `parser_test.go:200` | invert: expect `SHA` set and `Current` = SHA |
| `TestExtractActionsStripsTrailingComment` | `parser_test.go:125` | keep `Current=="v3"`, assert `Tag==""`; add SHA+comment case |
| `TestAuditSkipsSHARefs` | `cmd/audit_test.go:32` | replace: bare SHA parses; audit with zero network must not claim “no actions”; expect the explicit “version unknown” skip path |
| `TestReplaceInFileTabsAndComment` | `upgrader_test.go:305` | keep for tag mode; add SHA-mode comment rewrite |
| `TestReplaceInFileSubpathAction` | `upgrader_test.go:263` | keep; add SHA+subpath |
| `TestDoUse_*`, `TestDoSync_*`, `TestDoInit_*`, `TestRunPinAndUnpin` | `cmd/versionmanager_test.go:105-330` | add `--sha` / `sha: true` variants |
| `TestScan*` | `internal/vulnscanner/vulnscanner_test.go:85-345` | add `Tag`-based matching and bare-SHA skip |

Add:

1. Parser matrix: bare SHA; SHA + `# v4.2.2`; SHA + `# v4`; SHA + arbitrary comment; SHA + subpath; quoted ref; matrix expression; tabs; uppercase SHA normalization.
2. `internal/github`: `ResolveTagSHA` with `httptest` requires an injectable base URL. `NewClient` hardcodes `github.NewClient` (`client.go:40-54`); add an unexported `newClientWithBaseURL` using `github.WithEnterpriseURLs` for tests. Cases: lightweight tag, annotated tag (fixture returns the peeled SHA), 404, rate-limit 403 with `Rate.Remaining == 0`.
3. `internal/github`: `fetchAllSemverTags` retains `Commit.SHA`; `LatestTagInfo` returns the matching SHA; cache hit avoids a second HTTP call.
4. Upgrader: SHA rewrite strips the old comment; tag rewrite preserves it; mixed SHA/tag sites in one file; dry-run shows the comment.
5. `cmd`: `use` on a SHA-pinned workflow without `--sha` keeps SHA form; `use --sha` converts tag form; `sync` with `sha: true`; `outdated` on SHA+comment compares `Tag`; `init` skips bare SHA with a warning.
6. Vuln: SHA+comment matches OSV; bare SHA produces no finding and a warning; breaking registry uses `Tag`.

### 5. Effort / risk

**Effort L (4–5 d). Risk medium–high.** The parser/`Version()` read-site sweep is broad and easy to miss one comparison. The upgrader comment handling is the main data-corruption risk; the single-write refactor from 1b makes it testable. Rate limits are manageable with the tag cache but must be stated in help text. Sequence strictly: parser + `Version()` + all call sites → upgrader → `ResolveTagSHA` → CLI/config → TUI → docs. `AGENTS.md:17-18` must be updated (still says SHAs are skipped and cites v62).

---

## 2b. `actup audit --fix`

### 1. Exact files & symbols

| File | Symbols / lines | Change |
|---|---|---|
| `cmd/audit.go` | `auditOpts` `:15-18`; flags `:39-42`; `runAudit` `:62-127`; text output `:101-118`; `os.Exit(1)` `:125`; `outputAuditJSON` `:131-168`; `os.Exit(1)` `:165` | add `--fix`, `--dry-run`, `--force`, `--semver`, `--token`; plan/apply path |
| `internal/vulnscanner/vulnscanner.go` | `Vulnerability` `:53-60`; `Finding` `:62-70`; `Scan` `:123-223`; `Fixed: v.firstFixed()` `:204`; `firstFixed` `:410-424`; `buildIntervals` `:287-328`; `eventsAffect` `:262-277` | per-ref fix computation |
| `cmd/common.go` | `savePin` `:109-122`; `confirmBreakingAll` `:157-175`; `scanVulns` `:233-251` | batched pin save; reuse breaking gate |
| `internal/upgrader/upgrader.go` | `ApplyAllUpgrades` `:95-116`; `Upgrade` `:15-18` | apply batch after 1b |
| `cmd/use.go` / `cmd/sync.go` | `resolveSpecRef` pattern `common.go:70-87`; `ApplyAllUpgrades` call `use.go:215`; `sync.go:167` | reference implementation for fix application |
| `cmd/audit_test.go` | `TestAuditJSONOutputShape` `:14`; `TestAuditSkipsSHARefs` `:32` | keep JSON envelope; add fix tests |
| `internal/vulnscanner/vulnscanner_test.go` | `TestScanVersionRangeMatching` `:175`; `TestScanMultipleIntervals` `:204`; `TestScanLastAffected` `:252` | add fix-version tests |

### 2. Data model / signature changes

**Prerequisite bug in `Fixed`.** `firstFixed()` returns the earliest `fixed` event across every interval, not the fix for the interval that contains the current ref. With intervals `[0,2)` and `[5,5.3)`, ref `v5.2` gets `Fixed == "2"`; `audit --fix` would downgrade. Replace it:

```go
// fixedFor returns the end of the vulnerable interval containing ref for this
// advisory, or "" when the ref is only listed in Versions (no fixed event).
func (v *osvVuln) fixedFor(actionKey, ref string) string {
	var fix string
	for _, aff := range v.Affected {
		if !strings.EqualFold(aff.Package.Ecosystem, ecosystem) ||
			!strings.EqualFold(aff.Package.Name, actionKey) {
			continue
		}
		if len(aff.Versions) > 0 {
			for _, ver := range aff.Versions {
				if versionsEqual(ver, ref) {
					return ""
				}
			}
			continue
		}
		for _, r := range aff.Ranges {
			if r.Type != "ECOSYSTEM" {
				continue
			}
			for _, iv := range buildIntervals(r.Events) {
				if !inInterval(ref, iv) || iv.end == "" {
					continue
				}
				if fix == "" || compareVersions(iv.end, fix) > 0 {
					fix = iv.end
				}
			}
		}
	}
	return fix
}

func inInterval(ref string, iv interval) bool {
	norm := normalizeVersion(ref)
	return compareVersions(norm, iv.start) >= 0 &&
		(iv.end == "" || compareVersions(norm, iv.end) < 0)
}
```

`Scan` changes `Fixed: v.firstFixed()` (`:204`) to `Fixed: v.fixedFor(key, site.Current)`. This is a behavior fix for display too: `audit` now reports the fix relevant to the version in use. `firstFixed` can be deleted after all callers move.

`Fixed` is inclusive-safe: OSV intervals are `[introduced, fixed)` and `eventsAffect` uses `< end` (`vulnscanner.go:262-277`), so writing exactly `Fixed` is not vulnerable; `current >= Fixed` produces no finding.

Command options:

```go
var auditOpts struct {
	paths  []string
	json   bool
	fix    bool
	dryRun bool
	force  bool
	semver bool
	token  string
}
```

Flags: `--fix`, `--dry-run`, `--force/-f`, `--semver/-s`, `--token/-t`. Mirror `use`/`sync` flag help text. `--json` + `--fix` is rejected (`fmt.Errorf("--json cannot be combined with --fix")`).

Planning types/helpers (unexported, `cmd/audit.go`):

```go
type auditFix struct {
	Key      string
	Refs     []parser.ActionRef
	Current  string // max version among refs, for reporting
	MinFixed string // max of all applicable Fixed versions
	Target   string // tag to write
}

func planAuditFixes(findings []vulnscanner.Finding) (fixes []auditFix, unfixable []vulnscanner.Finding)
func minFixedFor(vulns []vulnscanner.Vulnerability) (string, bool)
func resolveFixTarget(ctx context.Context, client *github.Client, key, fixed string, semverMode bool) (string, error)
```

Rules:

1. Group findings by `owner/repo` (lowercased for matching, original key for display).
2. `MinFixed = max(v.Fixed)` across every advisory on every finding for that key. Each `Fixed` is a lower bound; the maximum is the smallest version that clears all advisories. Deduplicate advisories by ID first.
3. If any advisory has `Fixed == ""` → the key is **unfixable**; report every site and do not rewrite any of them. Never pin to another advisory's fix.
4. If `MinFixed <= current` (multi-interval or stale advisory), skip with `already past the reported fix` rather than downgrading.
5. Target resolution:
   - default mode: `client.LatestTag(ctx, owner, repo, github.TagMode{Major: parseMajor(MinFixed)})`. The highest tag in that major is ≥ the fixed version because `resolveLatestTag` sorts descending (`client.go:127-157`). If the resolved tag is `< MinFixed`, surface an error for that key.
   - Fix `resolveLatestTag`’s major branch first: it currently matches only `vN.` and misses a bare `vN` tag, and it compares raw tag names so a repo tagging `41.0.0` (no `v`) never matches `TagMode{Major: 41}`. Normalize the tag name for comparison (`v` prefix), accept `vN`, and return the original spelling. Add `TestResolveLatestTag_MajorPinBareTag` and `TestResolveLatestTag_MajorPinNoVPrefix`. OSV frequently writes fixed versions without a `v` (`41`), and some actions publish only a bare `v41` tag.
   - `--semver`: normalize the fixed version to a `v` prefix and write it verbatim. If the repository's tag is spelled differently, GitHub's tag resolution is case-sensitive; this is a documented risk. Optional follow-up: `ResolveTag` that compares normalized semver against fetched tags.
   - 2a SHA mode: if `cfg.SHA` or the site is already SHA-pinned, also call `ResolveTagSHA(Target)` and write `@sha # Target`.
   - API failure/rate limit: fall back to the normalized `MinFixed` ref and print `⚠ <key>: using <ref> (GitHub API unavailable)`. Never silently skip the fix.
6. Breaking-change gate: call `confirmBreakingAll(key, refs, Target, registry, dryRun, force, reader, isTTY)`. If declined, the key remains vulnerable and counts toward exit code 1.
7. Apply: build `[]upgrader.Upgrade` for refs whose `Current != Target` (or `SHA != newSHA` in SHA mode) and call `ApplyAllUpgrades` once per key, or once for all keys after 1b. Skip in `--dry-run`, print `would update <key>: <current> → <target> (<n> refs)`.
8. `.actup.yaml`: if `config.LoadDefault()` returned non-nil, record every applied target. Add a batched helper to avoid N read/write cycles:

```go
// savePins records multiple pins with one load/save.
func savePins(pins map[string]string) error
```

Use `EffectivePins()`/managed semantics from 1a inside the helper; `savePin` becomes a one-entry wrapper.
9. Do **not** re-scan OSV after fixing. Residual count = unfixable + declined + API/target failures. Re-scanning doubles traffic and can change results mid-run.

Exit codes:

| Situation | Code |
|---|---|
| OSV unreachable | 2 (unchanged, `audit.go:92`) |
| No findings | 0 |
| `--fix` fixed every finding | 0 |
| `--fix` left any finding (unfixable/declined/failed) | 1 |
| `--fix --dry-run` with actionable fixes | 1 (nothing was written) |
| `--json` without `--fix` | existing: 1 when findings |

`--json` output is unchanged; `--fix` is text-only in this iteration.

### 3. Edge cases & failure modes

- **Multi-interval advisory:** fixed by `fixedFor`; add the `v5.2 → 5.3` regression test before wiring `--fix`.
- **Multiple advisories with different fixes:** max wins. Example fixes `2` and `5.3` → write `5.3`.
- **One advisory with no fix:** entire key is unfixable; do not partially fix.
- **Different current versions per site:** compute the target once per key using the max fixed across all sites; apply to every site. A site already at the target is omitted.
- **Major-tag target is too old:** if the highest tag in the fixed major is below `MinFixed`, do not write it; report `no tag ≥ <fixed> in major vN`.
- **Current ref is a branch (`master`):** `normalizeVersion` returns `""`; OSV ranges do not match (pre-existing limitation), so no finding and no fix. Document.
- **SHA-pinned refs (2a):** `Version()` supplies the OSV comparison; write SHA+comment. Bare SHA has no version and is skipped by the scanner.
- **`skip` pins:** already removed by `filterSkippedActions` before scanning (`cmd/audit.go:75`).
- **`--fix` on a non-TTY without `--force`:** breaking-change gate declines; the key stays vulnerable and exit is 1. This is intentional.
- **Rate limits:** one `LatestTag` per fixed key (cached per repo/mode), plus one `ResolveTagSHA` per key in SHA mode. `--semver` needs no API calls.
- **`.actup.yaml` absent:** do not create one; only workflows are rewritten. `use` already behaves this way for `--no-save`, but `audit --fix` should not surprise users with a new manifest.
- **Dry-run:** no OSV re-scan, no workflow writes, no pin writes. Print the plan and the breaking warnings that would be shown.
- **Exit from `runAudit`:** prefer returning a sentinel error over `os.Exit` for the new path so tests can assert; the existing `os.Exit` calls at `:92/:125` can stay for compatibility. New code should return `errAuditFindings` and let `Execute` exit 1.

### 4. Tests

Existing: `internal/vulnscanner/vulnscanner_test.go:85-345` pins range semantics and `Fixed == "41"` (`:142`); `cmd/audit_test.go:14` pins JSON shape; no cmd-level OSV tests.

Add:

1. `vulnscanner`: `TestFixedFor_MultiInterval` (`v5.2` → `5.3`), `TestFixedFor_EarlierInterval` (`v1` → `2`), `TestFixedFor_VersionsOnly` → `""`, `TestFixedFor_NoFixedEvent` → `""`.
2. `vulnscanner`: `TestScanFixedIsRefSpecific` — same advisory, two sites `v1`/`v5.2`, `Fixed` differs (`2` vs `5.3`).
3. `cmd`: `TestMinFixedFor` (max wins; empty → not ok), `TestPlanAuditFixes_DedupesAcrossFiles`, `TestPlanAuditFixes_UnfixableSkipsAllSites`, `TestPlanAuditFixes_DifferentCurrents`.
4. `cmd`: `TestAuditFixDryRunWritesNothing`, `TestAuditFixWritesTargetAndPin` (inject a stub resolver or use a nil client with verbatim/fixed target), `TestAuditFixNoFixKnownExits1`, `TestAuditFixJSONRejected`, `TestAuditFixBreakingDeclinedExits1`.
5. `cmd`: `TestSavePinsBatched` (one save; managed vs legacy).
6. `internal/github`: `TestResolveLatestTag_MajorPinBareTag`, `TestResolveLatestTag_MajorPinNoVPrefix` (prerequisite for `resolveFixTarget`).

### 5. Effort / risk

**Effort M (1.5–2 d). Risk high if `fixedFor` is skipped** (silent downgrades), medium otherwise. The breaking-change gate can block fixes in CI without `--force`; that is correct but should be documented in the flag help. Do not ship `--fix` before the `fixedFor` regression test is green.

Order: (1) `fixedFor` + tests, (2) plan/min helpers, (3) target resolution + apply, (4) pins + exit codes, (5) JSON/flag conflicts and docs.

---

## 1c. Breaking-change registry coverage and `with:` input inspection

### 1. Exact files & symbols

| File | Symbols / lines | Change |
|---|---|---|
| `internal/breakingchanges/registry.go` | `BreakingChange` `:15-21`; `Entry` `:23-28`; `Check` `:42-67`; `parseMajor` `:70-80` | add `inputs` parameter and filter |
| `internal/breakingchanges/registry.yaml` | entire file (2 changes, checkout only) | replace fabricated entries; expand verified table |
| `internal/breakingchanges/registry_test.go` | `TestCheck_V2ToV3` `:5`; `TestCheck_V3ToV4_NoEntry` `:27`; `TestCheck_UnknownAction` `:39`; `TestCheck_SmokeTestSeededData` `:51`; `TestLoadRegistry_ParsesYAML` `:83` | update signature/expectations |
| `internal/parser/parser.go` | `ActionRef` `:14-22`; `extractFromFile` `:41-74`; `parseActionRef` `:76-107` | add `Inputs []string`; scan `with:` |
| `internal/parser/parser_test.go` | extraction tests `:9-124`; `TestParseActionRef` `:151`; `TestExtractActionsConcurrent` `:262` | add `with:` matrix |
| `cmd/common.go` | `confirmBreakingAll` `:157-175`; `breakingChangesFor` `:195-200` | pass inputs |
| `cmd/root.go` | `runNoTUI` breaking call `:193-195` | pass inputs |
| `internal/tui/tui.go` | `loadActions` breaking call `:322-323`; `viewDetail` `:530-556` | union inputs |
| `cmd/use.go` / `cmd/sync.go` | `checkUseBreaking` `:174-177`; `confirmBreakingAll` `:136` | flow through |

Callers of `Check`: `cmd/common.go:199`, `internal/tui/tui.go:323`. `breakingChangesFor` is the only wrapper; update it once.

### 2. Data model / signature changes

```go
// internal/parser/parser.go
type ActionRef struct {
	// ... existing fields
	Inputs []string // keys under the step's with: block, file order; nil if none
}
```

`[]string` (not `map[string]string`) because the registry only needs membership and deterministic order keeps tests stable. Values are never used.

```go
// internal/breakingchanges/registry.go
func (r *Registry) Check(action, fromVersion, toVersion string, inputs []string) []BreakingChange {
	fromMajor := parseMajor(fromVersion)
	toMajor := parseMajor(toVersion)
	if fromMajor < 0 || toMajor < 0 {
		return nil
	}

	var results []BreakingChange
	for _, entry := range r.entries {
		if entry.Action != action {
			continue
		}
		// An entry applies when the upgrade crosses its boundary.
		if entry.ToMajor != 0 {
			if fromMajor >= entry.ToMajor || toMajor < entry.ToMajor {
				continue
			}
		} else {
			// to_major: 0 means "introduced in FromMajor and still relevant".
			if fromMajor >= entry.FromMajor || toMajor < entry.FromMajor {
				continue
			}
		}
		for _, bc := range entry.Changes {
			if bc.Input != "" && !slices.Contains(inputs, bc.Input) {
				continue
			}
			results = append(results, bc)
		}
	}
	return results
}
```

The existing condition set at `registry.go:52-67` is wrong for multi-major jumps: it returns a 2→3 entry only when the target is exactly v3, so a v2→v4 upgrade silently loses it. The boundary test above returns 2→3 for v1→v4, v2→v4, and v2→v3, and does not return it for v3→v4. `FromMajor` remains informational for `to_major: 0` entries.

Semantics:

- `Input == ""` → general note, always shown for a matching major jump (runtime bumps, immutable artifacts, etc.).
- `Input != ""` → shown only when at least one site for that major actually passes the key. A step without the affected input is not warned.
- `nil`/empty inputs suppresses all input-specific warnings. This is the safe direction (no false positives), but it means `Check` called without inputs no longer reports the old checkout entries. That is intended; those entries are wrong (see below).

`breakingChangesFor` gains `inputs []string`; `confirmBreakingAll` builds a union of inputs per distinct current version:

```go
inputsByVersion := map[string][]string{}
for _, ref := range refs {
	inputsByVersion[ref.Current] = append(inputsByVersion[ref.Current], ref.Inputs...)
}
// dedupe inputs before passing to Check
```

TUI passes the union of `Inputs` across `m.actions` for the key, or per-current if 1d stores them.

### 3. Parsing `with:` without a YAML AST

Run after the `uses` line is parsed, in `extractFromFile` (`parser.go:52-69`), using the original `lines` slice and index `i`.

Algorithm:

1. Compute `keyIndent = strings.Index(line, "uses:")` — the column of the `uses` token. For `      - uses: x@v1` this points at `uses`, not the dash; sibling keys (`with:`, `env:`) align with `uses`. For `        uses: x@v1` under `- name:` it is the key column.
2. Find the step block bounds:
   - `stepStart`: walk backward from `i`; the first earlier non-blank line with `indent < keyIndent` is the step’s `- name:` line (or the parent key). Start scanning from that line.
   - `stepEnd`: walk forward from `i`; the first non-blank line with `indent < keyIndent` is the next list item or the parent key.
   - Scanning the **whole step block**, not just lines after `uses:`, is required because YAML permits `with:` before `uses:` (`- with: {...}\n  uses: ...`).
3. Scan `stepStart..stepEnd` for a line at `indent == keyIndent` matching `^with:\s*(.*)$`. Requiring the exact sibling indent prevents matching `with:` text inside `run: |` block scalars at deeper indentation. Stop at the first match.
4. Parse the remainder:
   - Empty → block map. Scan following lines while `indent > keyIndent`; stop at `indent <= keyIndent`. Match `^\s*([^:#][^:#]*?)\s*:`; trim whitespace and one pair of quotes; append.
   - Starts with `{` → flow map. Strip braces, split on `,`, split each entry on the first `:`, trim quotes/space; skip empty keys. Nested flow maps are a documented limitation.
   - `*anchor`, `null`, or a scalar → no keys (safe suppression).
5. Cap the scan at the step boundary; never read past EOF.
6. Keep `Inputs == nil` when there is no `with:`.

Limitations to document, not fix:

- Flow-style steps (`- {uses: x@v1, with: {...}}`) are already misparsed by `usesRegex`; out of scope.
- Keys containing `:` inside quotes are not handled.
- Values are ignored, so `${{ matrix.* }}` is irrelevant.
- `with:` under a job-level reusable-workflow `uses:` follows the same rule and works.

### 4. Registry data: replace the fabricated entry, add verified coverage

**Current entry is wrong.** `actions/checkout@v2`/`v3` `action.yml` has inputs `repository, ref, token, path, clean, lfs, submodules` — there is no `github_token` and no `app_url`. The existing two changes are fabricated: they trigger a false warning on every checkout v2→v3 upgrade and disappear no input. Remove them (or replace with the verified runtime note) and update `TestCheck_V2ToV3`/`TestCheck_SmokeTestSeededData`.

Runtime transitions below were verified from each repository's `action.yml` at the two tags (`runs.using`):

| # | action | from→to | verified runtime | type | input | replacement |
|---|---|---|---|---|---|---|
| 1 | actions/checkout | 2→3 | node12 → node16 | behavior | — | — |
| 2 | actions/checkout | 3→4 | node16 → node20 | behavior | — | — |
| 3 | actions/upload-artifact | 3→4 | node16 → node20 | behavior | — | — |
| 4 | actions/download-artifact | 3→4 | node16 → node20 | behavior | — | — |
| 5 | actions/cache | 3→4 | node16 → node20 | behavior | — | — |
| 6 | actions/setup-node | 3→4 | node16 → node20 | behavior | — | — |
| 7 | actions/setup-go | 4→5 | node16 → node20 | behavior | — | — |
| 8 | actions/setup-python | 4→5 | node16 → node20 | behavior | — | — |
| 9 | actions/setup-java | 3→4 | node16 → node20 | behavior | — | — |
| 10 | actions/github-script | 6→7 | node16 → node20 | behavior | — | — |
| 11 | docker/build-push-action | 4→5 | node16 → node20 | behavior | — | — |
| 12 | docker/login-action | 2→3 | node16 → node20 | behavior | — | — |
| 13 | docker/setup-buildx-action | 2→3 | node16 → node20 | behavior | — | — |
| 14 | goreleaser/goreleaser-action | 4→5 | node16 → node20 | behavior | — | — |
| 15 | golangci/golangci-lint-action | 3→4 | node16 → node20 | behavior | — | — |
| 16 | peter-evans/create-pull-request | 5→6 | node16 → node20 | behavior | — | — |
| 17 | hashicorp/setup-terraform | 2→3 | node16 → node20 | behavior | — | — |
| 18 | actions/stale | 8→9 | node16 → node20 | behavior | — | — |
| 19 | actions/upload-artifact | 3→4 | immutable artifact names; hidden files excluded | behavior | `name` | — |
| 20 | actions/download-artifact | 3→4 | `name` optional; omitted downloads all into per-artifact dirs | behavior | `name` | — |
| 21 | actions/cache | 3→4 | requires newer runner / cache service v2 | behavior | — | — |
| 22 | codecov/codecov-action | 3→4 | `file` deprecated in favor of `files` | deprecated | `file` | `files` |

Rows 19–22 are behavior/deprecation notes, not verified input removals: `upload-artifact@v4` still declares `name`, and `codecov-action@v4` still declares both `file` and `files`. Keep `Input` empty for general behavior notes; only row 22 uses an input-specific deprecation. Every message must name the runtime change and the required runner/tooling, e.g.:

```yaml
- action: "actions/checkout"
  from_major: 3
  to_major: 4
  changes:
    - type: behavior
      message: "checkout v4 runs on Node 20; self-hosted runners must be v2.308.0 or newer."
```

Do not add entries for `docker/build-push-action` 5→6 (verified node20 → node20 — not a runtime break) or any action whose metadata was not checked. `to_major: 0` already means “this and later” in `Check` (`registry.go:56-61`), so do not duplicate entries per destination major.

`BreakingChange.Type` is free-form; the TUI only prints `Message`. Adding `behavior` requires no renderer change. `URL`/`Notes` fields are optional; skip unless the team wants release links in the detail view.

### 5. Edge cases & failure modes

- `with:` can appear before or after `uses:` and in any order among `if:`, `env:`, `timeout-minutes:`; scanning the whole step block plus exact sibling-indent matching handles both orders.
- Quoted keys (`"github_token":`) are trimmed. Keys with an embedded colon are not supported.
- `with:` inside a `run: |` block is at a deeper indent and is ignored. This is the main false-attribution risk and is covered by a test.
- Multiple steps in one file: the scan stops at the next `- ` at the step indent, so inputs never leak between steps.
- Composite actions have nested steps with deeper indentation; each `uses` line gets its own scan, and the indent boundary still binds to that step.
- Job-level reusable workflows: `jobs.x.uses` + `with:` is parsed; inputs are job inputs, which is the right key set.
- Anchored/aliased `with:` (`with: *defaults`) yields no keys → input-specific warnings suppressed. Safe.
- A workflow using the affected input via `env:` indirection cannot be detected; document.
- Registry `Check` is used by `use`/`sync`/`outdated`/TUI; all paths must pass inputs or the warnings silently disappear. Update the single `breakingChangesFor` wrapper and all callers in the same commit.
- Removing the old checkout entries changes `TestCheck_V2ToV3`; that test currently asserts fabricated behavior. Rewrite it to assert the verified runtime note and add `TestCheck_FiltersUnusedInput`.

### 6. Tests

Existing to update: `registry_test.go:5-92` (all `Check` calls gain an inputs argument; v2→v3 expectations change; `TestCheck_V3ToV4_NoEntry` becomes a runtime-note assertion because the new registry has a checkout 3→4 entry), `parser_test.go` extraction tests must still pass with `Inputs == nil`.

Add:

1. Parser: `TestExtractActionsWithBareWith` (block map), `...WithBeforeUses` (backward scan), `...WithInlineMap`, `...WithQuotedKeys`, `...MultipleStepsIsolateWith`, `...NestedComposite`, `...RunBlockScalarDoesNotLeakWith`, `...NoWith` (nil).
2. Registry: `TestCheck_FiltersUnusedInput`, `TestCheck_GeneralNoteAlwaysShown`, `TestCheck_ToMajorZeroCoversLater`, `TestCheck_UnionInputsFromMultipleSites` (at the `confirmBreakingAll` level), `TestLoadRegistry_AllEntriesHaveMessage`.
3. TUI/rendering: no change if messages are only printed; add a test only if the detail view starts showing input names.
4. Integration: a checkout v2 step with `with: {github_token: x}` no longer warns after the fabricated entry is removed; a step without the codecov `file` input does not warn for codecov v4.

### 7. Effort / risk

**Effort M (1–1.5 d):** parser ~60 lines + tests, registry plumbing ~20 lines, data verification already done. **Risk low–medium.** Parser false attribution can drop warnings (safe) or attach a `with:` from a `run:` block (guarded by sibling indent). Registry accuracy is the real risk: wrong entries either over-warn or miss a break. Only ship verified metadata transitions; label behavior notes as such.

Order: (1) `ActionRef.Inputs` + parser + tests, (2) `Check` signature + all callers + test updates, (3) registry data replacement, (4) docs/README registry section.

---

## 2c. Composite action discovery

### 1. Exact files & symbols

| File | Symbols / lines | Change |
|---|---|---|
| `internal/scanner/scanner.go` | `defaultWorkflowDir` `:11`; `DiscoverWorkflows` `:13-54`; `discoverInDir` `:57-80`; `isWorkflowFile` `:83-86` | add `.github/actions` + root action files under default mode only |
| `internal/scanner/scanner_test.go` | `TestDiscoverWorkflows` `:9`; `TestIsWorkflowFile` `:42`; `TestDiscoverWorkflowsFilePath` `:66`; `TestDiscoverWorkflowsEmpty` `:90` | add composite/root cases |
| `internal/parser/parser.go` | `usesRegex` `:25`; `parseActionRef` `:76-107` | no change required |
| `internal/upgrader/upgrader.go` | `usesPath` `:20-27`; `replaceInFile` `:38-50` | no change; `Path` preserves subpaths |
| `cmd/common.go` | `loadWorkflowActions` `:25-39` | no change; error text “no workflow files” may become “no workflow or action files” |
| `internal/tui/tui.go` | file count `:428-432` | no change |

### 2. Data model / signature changes

No `ActionRef` change. Scanner diff:

```go
const (
	defaultWorkflowDir = ".github/workflows"
	defaultActionsDir  = ".github/actions"
)

// discoverInDir gains a predicate.
func discoverInDir(dir string, match func(string) bool) ([]string, error)

func isActionFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return base == "action.yml" || base == "action.yaml"
}
```

`DiscoverWorkflows`:

```go
if len(paths) == 0 {
	// Defaults only: workflows + composite actions + repo-root action file.
	if err := appendPath(&files, seen, defaultWorkflowDir, isWorkflowFile); err != nil { ... }
	if err := appendPath(&files, seen, defaultActionsDir, isActionFile); err != nil { ... }
	for _, f := range []string{"action.yml", "action.yaml"} {
		if err := appendPath(&files, seen, f, isActionFile); err != nil { ... }
	}
	return files, nil
}
// Explicit -p semantics unchanged: every path is scanned with isWorkflowFile.
```

Lift the body of the current `for _, p := range paths` loop into `appendPath(files *[]string, seen map[string]bool, p string, match func(string) bool) error` to avoid duplicating the stat/walk branches. Missing defaults (`os.IsNotExist`) are skipped, matching today (`scanner.go:29-32`). Dedup stays by absolute path (`seen`).

Do **not** append the defaults when `-p` is supplied. `actup -p ./fixtures` must not silently scan the real repo; that would break test/CI scoping. Custom `-p` keeps the current behavior (recursive `.yml`/`.yaml`), which already allows users to opt into composite files by passing the directory explicitly.

### 3. Why the parser works unchanged

- Composite `action.yml` has `runs:\n  using: composite\n  steps:\n    - uses: actions/checkout@v4`; the line regex (`parser.go:25`) matches indented `- uses:` exactly like a workflow.
- `inputs:`, `outputs:`, `branding:`, and `runs.pre/post-entrypoint` contain no `uses: <owner/repo@ref>` lines. A pathological input literally named `uses` with value `a/b@v1` would parse, but that is contrived and acceptable.
- Node/docker actions have no `uses:` under `runs` → zero refs.
- Local composite references (`uses: ./other`) are skipped by `parseActionRef` (`parser.go:77`); correct, there is no external version to bump.
- Reusable workflow **definitions** are already under `.github/workflows` and already discovered. Job-level calls `uses: owner/repo/.github/workflows/x.yml@v1` are already parsed by the same regex; `Path` preserves the subpath and the upgrader writes it back intact. No separate reusable-workflow discovery is needed; the item’s wording implies a file location GitHub does not support.
- Rewrites work because `usesPath` prefers `Path` (`upgrader.go:20-27`) and `ActionRef.File`/`Line` come from the actual discovered file.

### 4. Edge cases & failure modes

- **Non-action YAML under `.github/actions`** (e.g. `foo/config.yml`) must be excluded by `isActionFile`; do not use `isWorkflowFile` for that directory.
- **Root `action.yml` name collision:** only the repo-root basenames are added; `.github/workflows/action.yml` is already covered by the workflow walk.
- **Unreadable `.github/actions`:** `os.Stat` returns a non-`IsNotExist` error and the whole discovery fails, same as an unreadable `.github/workflows` today. Consistent; keep.
- **Walk errors inside the directory are swallowed** (`scanner.go:61-63`), so permission-denied subdirs silently produce fewer files. Pre-existing.
- **Symlinked duplicates:** dedup uses `filepath.Abs`, not `EvalSymlinks`; a symlink and its target are two entries. Pre-existing.
- **File ordering:** default groups are workflows, then `.github/actions`, then root action; within each, `WalkDir` lexical. `ApplyAllUpgrades` should sort files after 1b for deterministic output.
- **TUI file count** includes composite files; wording “actions found across N files” remains accurate.
- **Error text:** `loadWorkflowActions` returns `no workflow files found` when only composite actions exist but are missing; update to `no workflow or action files found` and adjust any test asserting the string (grep shows none).
- **`action.yaml` and `action.yml` both present:** both discovered; dedup does not collapse them (different paths). Acceptable.
- **No `runs.using` check:** we do not parse the action type. A malformed `action.yml` with a `uses:` line in metadata could produce a ref; the parser’s owner/repo/`@` requirements bound the damage.

### 5. Tests

Existing: `scanner_test.go:9` (explicit workflow dir → 2 files), `:42` (`isWorkflowFile`), `:66` (explicit file), `:90` (nonexistent path no error). Parser/upgrader tests already cover line-based extraction and subpath rewrites.

Add:

1. `TestDiscoverWorkflows_DefaultsIncludeCompositeAndRootAction` — temp repo with `.github/workflows/ci.yml`, `.github/actions/deploy/action.yml`, root `action.yaml`; expect 3 files.
2. `TestDiscoverWorkflows_DefaultsIgnoreNonActionYAMLInActionsDir` — `.github/actions/foo/config.yml` excluded.
3. `TestDiscoverWorkflows_DefaultsMissingActionsDir` — absent `.github/actions` is not an error.
4. `TestDiscoverWorkflows_ExplicitPathDoesNotAddDefaults` — `-p fixtureDir` returns only that dir, even when `.github/actions` exists.
5. `TestDiscoverWorkflows_Dedup` — explicit file and parent dir passed twice; one entry.
6. Parser: `TestExtractActionsCompositeAction` — fixture with `inputs`/`outputs`/`branding` and two `runs.steps[].uses`; exactly 2 refs with correct lines.
7. Parser: `TestExtractActionsNodeAction` — no `uses` → 0 refs.
8. Upgrader: rewrite `- uses: x@v1` inside `action.yml`; assert `runs.using: composite` untouched.

### 6. Effort / risk

**Effort S (~35 production lines + ~80 test lines). Risk low.** The parser and upgrader need no change; the only behavior expansion is default discovery. Main risk is over-broad file matching and accidentally scanning defaults when `-p` is used; both are addressed above. Depends on nothing. Do 2c before 2d.

README updates: discovery bullet `README.md:53`, roadmap `:353`.

---

## 2d. Grouped / filtered upgrades

### 1. Exact files & symbols

| File | Symbols / lines | Change |
|---|---|---|
| `cmd/root.go` | `rootCmd` `:34-53` (no `Args`); `run` `:71-122`; `filterSkippedActions` `:124-137`; flags `:60-68` | accept positional globs; `--group` |
| `cmd/outdated.go` | `outdatedCmd` `:20-32` (no `Args`); `runOutdated` `:43-62`; flags `:34-41` | accept globs; `--group` |
| `cmd/ls.go` | `Args: MaximumNArgs(1)` `:37`; exact lookup `:82-84` | intentionally unchanged |
| `cmd/common.go` | new `filterActionsByPatterns` | shared filter |
| `internal/config/config.go` | `Config` `:17-20`; `Render` `:78-99` | `Groups map[string][]string` |
| `schema.json` | `properties` `:5-37` | add `groups` |
| `cmd/root_test.go` | `TestFilterSkippedActions` `:10` | keep |
| `internal/config/config_test.go` | exact render strings `:67-100`; roundtrip `:98` | add groups tests |

Cobra behavior verified: `rootCmd` has subcommands and no `Args`, so Cobra v1.10.2 applies `legacyArgs`; `actup 'actions/*'` currently fails with `unknown command "actions/*"`. `outdatedCmd` has no subcommands, so extra positionals are silently ignored today. Both need explicit `cobra.ArbitraryArgs`.

### 2. Data model / signature changes

Filter helper in `cmd/common.go`:

```go
func filterActionsByPatterns(actions []parser.ActionRef, patterns []string) ([]parser.ActionRef, error) {
	if len(patterns) == 0 {
		return actions, nil
	}
	var out []parser.ActionRef
	for _, a := range actions {
		key := a.Owner + "/" + a.Repo
		for _, p := range patterns {
			ok, err := path.Match(p, key)
			if err != nil {
				return nil, fmt.Errorf("invalid filter %q: %w", p, err)
			}
			if ok {
				out = append(out, a)
				break
			}
		}
	}
	return out, nil
}
```

Use `path.Match`, not `filepath.Match`: keys are always `/`-separated and `filepath.Match` treats `\` as a separator on Windows. `*` does not cross `/`, so `actions/*` and `docker/*` work; a bare `*` does not match `actions/checkout` (document; “everything” is bare `actup`). Multiple patterns are a union; duplicate refs are impossible because each input ref is appended at most once.

Root wiring:

```go
rootCmd.Args = cobra.ArbitraryArgs
rootCmd.Use = "actup [filter...]"
rootCmd.Flags().StringArrayVar(&filterGroups, "group", nil,
	"Only actions in a .actup.yaml group (repeatable)")
```

In `run`, after config/major load (`root.go:96-104`) and before `filterSkippedActions` (`:106`):

```go
patterns := args
for _, g := range filterGroups {
	if cfg == nil {
		return fmt.Errorf("--group requires .actup.yaml")
	}
	pats, ok := cfg.Groups[g]
	if !ok {
		return fmt.Errorf("unknown group %q", g)
	}
	patterns = append(patterns, pats...)
}
actions, err = filterActionsByPatterns(actions, patterns)
if err != nil {
	return err
}
if len(patterns) > 0 && len(actions) == 0 {
	return fmt.Errorf("no actions match %s", strings.Join(patterns, ", "))
}
```

The no-match error must run **before** `printVulnWarnings`/`runNoTUI`/`tui.Run`, so a typo or empty selection never hits the GitHub API or starts an empty TUI. Keep `filterSkippedActions` after the pattern filter: patterns select, `skip` subtracts.

`runOutdated` mirrors the same block after config load (`cmd/outdated.go:48-58`), before `printVulnWarnings`/`runCheck`. Add `groups []string` to `outdatedOpts` and the flag.

`runCheck` itself stays unchanged: it receives the already-filtered action slice.

Config:

```go
type Config struct {
	Major   *int                `yaml:"major,omitempty"`
	SHA     bool                `yaml:"sha,omitempty"` // 2a
	Actions map[string]string   `yaml:"actions,omitempty"`
	Pins    map[string]string   `yaml:"pins,omitempty"` // 1a
	Groups  map[string][]string `yaml:"groups,omitempty"`
}
```

`Render` appends groups after pins/actions, sorted by group name and by pattern, so output is deterministic:

```yaml
groups:
  ci:
    - actions/*
    - docker/*
```

`schema.json` addition before `"additionalProperties": false`:

```json
"groups": {
  "type": "object",
  "description": "Named action-glob groups selectable with actup --group <name>.",
  "additionalProperties": { "type": "array", "items": { "type": "string" } },
  "examples": [{ "ci": ["actions/*", "docker/*"] }]
}
```

Why `--group` and not dynamic subcommands: subcommands would collide with the existing namespace and typo handling (`TestVersionManagerSubcommandsRegistered`, `cmd/versionmanager_test.go:317`). A repeatable flag reuses the same filter helper and composes with positional patterns.

### 3. Edge cases & failure modes

- **Typo behavior changes.** `cobra.ArbitraryArgs` makes `actup udpate` a filter instead of `unknown command`. The no-match error still exits non-zero before any network call, but the error text differs. Document; this is the main regression surface.
- **Shell expansion.** `actup actions/*` expands if an `actions/` directory exists. Docs must quote: `actup 'actions/*'`.
- **Filter + config:** patterns are selection only. Exact/numeric pins still override resolution (`ResolveAll`), and `skip` still subtracts. A group containing `actions/checkout` while config says `skip` yields no checkout; this intersection is correct.
- **Group composition:** positional + `--group` patterns are additive (union). Unknown group or `--group` without a manifest is an error.
- **No match:** error, not an empty successful run. For `outdated`, exit 1 via `Execute`; for root, same. This avoids a CI job silently passing after a typo.
- **TUI empty:** prevented by the no-match error.
- **`ls` unchanged:** it is an exact-key viewer (`ls.go:82-84`), not a filter target. Converting it to globs would change the `[action]` contract. Leave it; optional later: treat a pattern containing `*` as a list query.
- **Windows:** `path.Match` keeps `/` semantics; patterns are `owner/repo`-only.
- **Pattern character classes:** `path.Match` supports `[abc]`; malformed `[` returns `ErrBadPattern`, surfaced as an error. Good.
- **Patterns matching subpaths:** keys are owner/repo only, so `google/osv-scanner-action/*` cannot distinguish subpaths. Use `google/osv-scanner-action`.
- **Pins for actions not discovered:** unaffected; 1a keeps pinned-but-unused reporting in `ls`.
- **`--group` on `use`/`sync`/`pin`:** intentionally not added; those commands take explicit action operands.

### 4. Tests

Existing: `cmd/root_test.go:10` (`filterSkippedActions`), `cmd/versionmanager_test.go:317` (no new subcommands), `internal/config/config_test.go:67-100` (byte-exact render without groups), `cmd/audit_test.go:14` (subcommand resolution with `SetArgs`).

Add:

1. `TestFilterActionsByPatterns`: exact key; `actions/*` matches checkout, not `docker/login-action`; multiple patterns union; invalid `[` errors; empty patterns returns input unchanged.
2. `TestRunRejectsNoMatchBeforeNetwork` — temp repo with one workflow, `rootCmd.SetArgs([]string{"nonexistent/*", "--no-tui"})`; assert the no-match error and no API path.
3. `TestRootAcceptsPositionalFilter` — `actup 'actions/*' --no-tui --dry-run` no longer returns `unknown command`.
4. `TestOutdatedRejectsNoMatch` — same for `outdated`.
5. Config: `TestLoadGroups`, `TestRenderGroupsSorted`, `TestSaveLoadGroupsRoundtrip`; keep `TestRender_SortedAndQuoted` byte-identical when `Groups == nil`.
6. `TestGroupFlagUnknown` / `TestGroupFlagWithoutManifest`; `TestGroupPlusPositionalUnion`.

### 5. Effort / risk

**Effort S for positional filters (~25 lines), S–M with config groups + render + schema (~60 lines). Risk medium-low.** The root `Args` change is the only behavior regression surface (typos), mitigated by the pre-network no-match error. Glob/pin interaction is intentionally an intersection. No dependency on 2c; groups depend on deterministic `Render` tests.

Order: 2c → 2d positional filters → 2d `--group` + config → README/schema. Docs: `README.md:53` (discovery), `:158-171` (usage), `:264-288` (manifest), `:340-353` (roadmap), `schema.json:36`.

Ponytail note: skipped `--filter` flags, regex filters, per-action `group:` fields, and glob support in `ls`/`use`/`sync`; none are needed. Risk to know: `cobra.ArbitraryArgs` on root means typos now depend on the no-match guard, so add both in the same commit.

---

## Cross-cutting implementation plan

1. **1b — upgrader batch/mode** (S, isolated). Prerequisite for safe SHA comment rewrites and deterministic diffs.
2. **1a — manifest split** (M). Add `Config.Pins`/`EffectivePins`, render `pins: {}`, migrate every resolution call site, add upgrade write-back. Do not change `init`’s `Actions` output.
3. **1d + the registry-range subset of 1c** (M). `parser.DistinctVersions`, `runNoTUI` → `confirmBreakingAll`, TUI `Currents`/`UpToDate`/breaking union, and `Registry.Check`’s boundary test. The v2→v4 warning is not observable without the registry fix, so land them together. Adjacent to 1a’s TUI edits.
4. **2a — SHA pinning** (L). Parser fields + `Version()`, read-site sweep, SHA-aware upgrader, `ResolveTagSHA`, config/flags, TUI. This changes the comparison primitive for every later item.
5. **2b — audit --fix** (M). Depends on `fixedFor` and 1a’s batched pin save; uses 2a’s SHA writer for SHA-pinned sites.
6. **2c — composite discovery** (S). Independent; can be done in parallel with 2a/2b if needed.
7. **1c remainder — `with:` inputs + registry data** (M). Depends on 2a’s `ActionRef` additions and 1d’s multi-version plumbing; update all `Check` callers atomically and replace the fabricated checkout entries.
8. **2d — filters/groups** (M). Independent of 2a/2b; config `Groups` must render alongside 1a/2a fields.

Verification per wave: `go test ./...` is not enough for 2a/2b because the GitHub/OSV clients are concrete. Add the httptest seams described in each section. Do not run these specs’ commands as part of this read-only session.

## Global risks

- **Schema compatibility:** `schema.json` has `additionalProperties: false`; every new top-level field (`pins`, `sha`, `groups`) must ship in the same release as the Go change, or editor validation breaks before the CLI reads the file.
- **Legacy manifests:** `pins` absent means legacy semantics. Old `init` files keep the “always up to date” bug until `init --force` (1a). Document the migration.
- **Comparison primitive:** after 2a, every `Current` comparison must become `Version()` (OSV, breaking changes, outdated, ls, init, TUI). A single missed site silently misbehaves on SHA-pinned repos. Grep for `.Current` and justify each remaining use.
- **Registry accuracy:** only verified metadata transitions ship; the current checkout entries are fabricated and must be removed. Wrong behavior notes are worse than no notes.
- **Exit codes:** `runCheck` and `runAudit` call `os.Exit` inside `RunE` (`root.go:257,283`; `audit.go:92,125,165`). New `--fix` logic should return errors/sentinels instead, so tests can assert them; do not add more `os.Exit` calls.
