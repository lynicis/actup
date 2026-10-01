# Graph Report - actup  (2026-09-12)

## Corpus Check
- 53 files · ~36,082 words
- Verdict: corpus is large enough that graph structure adds value.

## Summary
- 313 nodes · 485 edges · 32 communities (22 shown, 7 thin omitted)
- Extraction: 86% EXTRACTED · 14% INFERRED · 0% AMBIGUOUS · INFERRED: 68 edges (avg confidence: 0.86)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `1adf785f`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Actup Project Documentation
- testing.T
- run
- ActionRef
- schema.json
- install_hooks.go
- LoadRegistry
- What You Must Do When Invoked
- github.com/charmbracelet/bubbletea.Msg
- model
- Resolve
- DiscoverWorkflows
- Release Pipeline Workflow
- model
- actup Logo
- github.com/lynicis/actup
- graphify reference: extra exports and benchmark
- graphify reference: query, path, explain
- Q: Why does ActionRef connect Workflow Action Parser to TUI State & Action Items, CLI Orchestration & Execution, and Outdated Action Checker?
- Q: Trace path from ActionRef to BreakingChange and the breaking change evaluation flow
- Q: Explore the atomic file upgrade mechanism with ApplyAllUpgrades
- graphify reference: add a URL and watch a folder
- graphify reference: commit hook and native CLAUDE.md integration
- graphify reference: incremental update and cluster-only
- graphify reference: GitHub clone and cross-repo merge
- graphify reference: transcribe video and audio
- rules/graphify.md
- extraction-spec.md
- workflows/graphify.md

## God Nodes (most connected - your core abstractions)
1. `ActionRef` - 15 edges
2. `ApplyAllUpgrades()` - 12 edges
3. `What You Must Do When Invoked` - 12 edges
4. `model` - 11 edges
5. `run()` - 10 edges
6. `runNoTUI()` - 10 edges
7. `/graphify` - 10 edges
8. `LoadRegistry()` - 9 edges
9. `resolveLatestTag()` - 9 edges
10. `Actup Project Documentation` - 9 edges

## Surprising Connections (you probably didn't know these)
- `Agent Skill Workflow Pipeline` --semantically_similar_to--> `Actup Agent Architecture Guide`  [INFERRED] [semantically similar]
  skills/github-actions-updater/SKILL.md → AGENTS.md
- `Security Policy & Reporting` --conceptually_related_to--> `Actup Project Documentation`  [INFERRED]
  SECURITY.md → README.md
- `run()` --calls--> `ExtractActions()`  [EXTRACTED]
  cmd/root.go → internal/parser/parser.go
- `run()` --calls--> `DiscoverWorkflows()`  [EXTRACTED]
  cmd/root.go → internal/scanner/scanner.go
- `run()` --calls--> `Resolve()`  [EXTRACTED]
  cmd/root.go → internal/token/token.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Continuous Integration Quality and Security Gates** — github_workflows_ci_job_test, github_workflows_ci_job_lint, github_workflows_ci_job_static_code_analysis, github_workflows_ci_job_vulnerability_check [EXTRACTED 1.00]
- **Go Gopher and GitHub Octocat Branding** — logo_actup_logo, logo_go_gopher, logo_github_octocat [INFERRED 0.85]
- **Multi-Platform Artifact Publishing Flow** — github_workflows_release_workflow, github_workflows_update_homebrew_tap_workflow, goreleaser_distribution_targets [INFERRED 0.85]
- **Major Version Pinning and Configuration Subsystem** — actup_config, docs_plans_2026_06_09_major_pin_and_config_file_feature_major_pin, docs_plans_2026_06_09_major_pin_and_config_file_config_architecture, readme_config_file [INFERRED 0.95]

## Communities (32 total, 7 thin omitted)

### Community 0 - "Actup Project Documentation"
Cohesion: 0.06
Nodes (41): Actup Local Configuration, Pinned Action: securego/gosec, Actup Agent Architecture Guide, Atomic File Upgrade Pattern, Developer Commands & Testing Guide, Major-Tag Default Resolution Mode, GitHub Token Resolution Strategy, Actup Version Changelog (+33 more)

### Community 1 - "testing.T"
Cohesion: 0.09
Nodes (36): TestCheckFlagRegistered(), TestFilterSkippedActions(), TestInstallHooksSubcommandRegistered(), testing.T, Load(), TestLoad_InvalidYAML(), TestLoad_MissingFile(), TestLoad_ValidConfig() (+28 more)

### Community 2 - "run"
Cohesion: 0.16
Nodes (16): Execute(), filterSkippedActions(), run(), runCheck(), runNoTUI(), Client, TagMode, context.Context (+8 more)

### Community 3 - "ActionRef"
Cohesion: 0.23
Nodes (14): outdatedAction, ActionRef, ApplyAllUpgrades(), replaceInFile(), showDryRunDiff(), TestApplyAllUpgrades(), TestApplyUpgrades(), TestApplyUpgradesConcurrent() (+6 more)

### Community 4 - "schema.json"
Cohesion: 0.12
Nodes (16): additionalProperties, description, examples, type, additionalProperties, anyOf, description, $id (+8 more)

### Community 5 - "install_hooks.go"
Cohesion: 0.23
Nodes (12): hookFilePath(), installHook(), installHookDryRun(), runInstallHooks(), TestInstallHooks_CreatesHook(), TestInstallHooks_DryRun(), TestInstallHooks_ExistingHookForce(), TestInstallHooks_ExistingHookNoForce() (+4 more)

### Community 6 - "LoadRegistry"
Cohesion: 0.25
Nodes (11): Entry, Registry, BreakingChange, LoadRegistry(), parseMajor(), TestCheck_SmokeTestSeededData(), TestCheck_UnknownAction(), TestCheck_V2ToV3() (+3 more)

### Community 7 - "What You Must Do When Invoked"
Cohesion: 0.08
Nodes (24): For /graphify add and --watch, For /graphify query, For the commit hook and native CLAUDE.md integration, For --update and --cluster-only, /graphify, Honesty Rules, Interpreter guard for subcommands, Part A - Structural extraction for code files (+16 more)

### Community 8 - "github.com/charmbracelet/bubbletea.Msg"
Cohesion: 0.38
Nodes (5): github.com/charmbracelet/bubbletea.Cmd, github.com/charmbracelet/bubbletea.Model, github.com/charmbracelet/bubbletea.Msg, model, model

### Community 9 - "model"
Cohesion: 0.22
Nodes (8): github.com/charmbracelet/bubbles/spinner.Model, model, ActionItem, actionsLoadedMsg, applyResult, progressItem, state, summaryResult

### Community 10 - "Resolve"
Cohesion: 0.39
Nodes (6): Resolve(), TestResolve_EnvVarUsedWhenFlagEmpty(), TestResolve_FlagValueTakesPriority(), TestResolve_GhAuthErrorReturnsEmpty(), TestResolve_GhNotFoundReturnsEmpty(), TestResolve_GhTokenUsedWhenFlagAndEnvEmpty()

### Community 11 - "DiscoverWorkflows"
Cohesion: 0.39
Nodes (7): discoverInDir(), DiscoverWorkflows(), isWorkflowFile(), TestDiscoverWorkflows(), TestDiscoverWorkflowsEmpty(), TestDiscoverWorkflowsFilePath(), TestIsWorkflowFile()

### Community 12 - "Release Pipeline Workflow"
Cohesion: 0.33
Nodes (7): Automated Release & Docker Publish Job, Release Pipeline Workflow, Homebrew Formula Generator, Update Homebrew Tap Workflow, GoReleaser Packaging Configuration, Multi-Platform Packaging Targets, Reproducible Builds Strategy

### Community 14 - "actup Logo"
Cohesion: 0.67
Nodes (4): actup Logo, actup Brand Identity, GitHub Octocat Mascot, Go Gopher Mascot

### Community 19 - "graphify reference: extra exports and benchmark"
Cohesion: 0.22
Nodes (8): graphify reference: extra exports and benchmark, Step 6b - Wiki (only if --wiki flag), Step 7 - Neo4j export (only if --neo4j or --neo4j-push flag), Step 7a - FalkorDB export (only if --falkordb or --falkordb-push flag), Step 7b - SVG export (only if --svg flag), Step 7c - GraphML export (only if --graphml flag), Step 7d - MCP server (only if --mcp flag), Step 8 - Token reduction benchmark (only if total_words > 5000)

### Community 20 - "graphify reference: query, path, explain"
Cohesion: 0.33
Nodes (5): For /graphify explain, For /graphify path, graphify reference: query, path, explain, Step 0 — Constrained query expansion (REQUIRED before traversal), Step 1 — Traversal

### Community 21 - "Q: Why does ActionRef connect Workflow Action Parser to TUI State & Action Items, CLI Orchestration & Execution, and Outdated Action Checker?"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: Why does ActionRef connect Workflow Action Parser to TUI State & Action Items, CLI Orchestration & Execution, and Outdated Action Checker?, Source Nodes

### Community 22 - "Q: Trace path from ActionRef to BreakingChange and the breaking change evaluation flow"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: Trace path from ActionRef to BreakingChange and the breaking change evaluation flow, Source Nodes

### Community 23 - "Q: Explore the atomic file upgrade mechanism with ApplyAllUpgrades"
Cohesion: 0.40
Nodes (4): Answer, Outcome, Q: Explore the atomic file upgrade mechanism with ApplyAllUpgrades, Source Nodes

### Community 24 - "graphify reference: add a URL and watch a folder"
Cohesion: 0.50
Nodes (3): For /graphify add, For --watch, graphify reference: add a URL and watch a folder

### Community 25 - "graphify reference: commit hook and native CLAUDE.md integration"
Cohesion: 0.50
Nodes (3): For git commit hook, For native CLAUDE.md integration, graphify reference: commit hook and native CLAUDE.md integration

### Community 26 - "graphify reference: incremental update and cluster-only"
Cohesion: 0.50
Nodes (3): For --cluster-only, For --update (incremental re-extraction), graphify reference: incremental update and cluster-only

## Knowledge Gaps
- **74 isolated node(s):** `installHooksOptions`, `github.com/lynicis/actup`, `$schema`, `$id`, `title` (+69 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 106 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **7 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `model` connect `model` to `github.com/charmbracelet/bubbletea.Msg`, `run`, `ActionRef`?**
  _High betweenness centrality (0.039) - this node is a cross-community bridge._
- **Why does `ActionRef` connect `ActionRef` to `testing.T`, `run`, `model`?**
  _High betweenness centrality (0.032) - this node is a cross-community bridge._
- **Why does `LoadRegistry()` connect `LoadRegistry` to `run`?**
  _High betweenness centrality (0.024) - this node is a cross-community bridge._
- **Are the 5 inferred relationships involving `ApplyAllUpgrades()` (e.g. with `TestApplyAllUpgrades()` and `TestApplyUpgrades()`) actually correct?**
  _`ApplyAllUpgrades()` has 5 INFERRED edges - model-reasoned connections that need verification._
- **What connects `installHooksOptions`, `github.com/lynicis/actup`, `$schema` to the rest of the system?**
  _74 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Actup Project Documentation` be split into smaller, more focused modules?**
  _Cohesion score 0.06219512195121951 - nodes in this community are weakly interconnected._
- **Should `testing.T` be split into smaller, more focused modules?**
  _Cohesion score 0.08970099667774087 - nodes in this community are weakly interconnected._
