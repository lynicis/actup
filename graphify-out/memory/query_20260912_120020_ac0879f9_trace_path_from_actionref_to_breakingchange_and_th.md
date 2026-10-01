---
type: "path_query"
date: "2026-09-12T12:00:20.198786+00:00"
question: "Trace path from ActionRef to BreakingChange and the breaking change evaluation flow"
contributor: "graphify"
outcome: "useful"
source_nodes: ["ActionRef", "BreakingChange", "runNoTUI()", "Registry", "model", "ActionItem"]
---

# Q: Trace path from ActionRef to BreakingChange and the breaking change evaluation flow

## Answer

ActionRef flows to both runNoTUI() and tui.loadActions(), where grouped ActionRefs are checked against breakingchanges.Registry.Check(key, current, latest). If BreakingChange entries exist, TUI attaches them to ActionItem (showing a warning badge and detail view on 'i'), while runNoTUI checks TTY status to either prompt interactively [y/N] or skip unless --force/--dry-run is supplied before calling upgrader.ApplyAllUpgrades().

## Outcome

- Signal: useful

## Source Nodes

- ActionRef
- BreakingChange
- runNoTUI()
- Registry
- model
- ActionItem
