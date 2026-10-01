---
type: "query"
date: "2026-09-12T11:59:39.517944+00:00"
question: "Why does ActionRef connect Workflow Action Parser to TUI State & Action Items, CLI Orchestration & Execution, and Outdated Action Checker?"
contributor: "graphify"
outcome: "useful"
source_nodes: ["ActionRef", "model", "runNoTUI()", "runCheck()", "Checker", "ExtractActions()"]
---

# Q: Why does ActionRef connect Workflow Action Parser to TUI State & Action Items, CLI Orchestration & Execution, and Outdated Action Checker?

## Answer

Expanded from original query via vocab: [action, ref, parser, checker, outdated, workflow, tui, item, state]. ActionRef in internal/parser/parser.go is the core domain abstraction representing a parsed uses: ref. It connects Workflow Action Parser (where it is extracted by ExtractActions/parseActionRef), CLI Orchestration & Execution (where run, runNoTUI, and runCheck orchestrate pipeline inputs), Outdated Action Checker (where Checker.Run consumes []ActionRef to query GitHub tags), and TUI State & Action Items (where model and ActionItem wrap ActionRef to drive the interactive checklist and upgrader).

## Outcome

- Signal: useful

## Source Nodes

- ActionRef
- model
- runNoTUI()
- runCheck()
- Checker
- ExtractActions()
