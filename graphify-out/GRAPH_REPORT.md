# Graph Report - actup  (2026-10-01)

## Corpus Check
- Corpus is ~36,206 words - fits in a single context window. You may not need a graph.

## Summary
- 317 nodes · 623 edges · 14 communities (13 shown, 1 thin omitted)
- Extraction: 89% EXTRACTED · 11% INFERRED · 0% AMBIGUOUS · INFERRED: 70 edges (avg confidence: 0.86)
- Token cost: 0 input · 0 output

## Community Hubs (Navigation)
- Graphify Knowledge & Tooling
- Root Command & Dependencies
- Architecture & Component Specs
- CLI Execution & Tag Resolution
- Configuration & Project Governance
- CLI Command Testing
- Terminal User Interface
- Git Hook Management
- Workflow Configuration Schema
- Breaking Changes Registry
- Atomic File Upgrader
- Release & CI Pipelines
- Brand Assets & Visuals
- Go Module Root

## God Nodes (most connected - your core abstractions)
1. `model` - 23 edges
2. `ActionRef` - 15 edges
3. `ApplyAllUpgrades()` - 12 edges
4. `graphify Skill` - 11 edges
5. `run()` - 10 edges
6. `runNoTUI()` - 10 edges
7. `LoadRegistry()` - 9 edges
8. `resolveLatestTag()` - 9 edges
9. `Actup Project Documentation` - 9 edges
10. `actup CLI Architecture` - 9 edges

## Surprising Connections (you probably didn't know these)
- `graphify post-commit hook` --semantically_similar_to--> `pre-commit Configuration`  [INFERRED] [semantically similar]
  .agents/skills/graphify/references/hooks.md → .pre-commit-config.yaml
- `Atomic file upgrade with sibling temp file and os.Rename` --semantically_similar_to--> `Atomic file upgrade via temp-file and rename`  [INFERRED] [semantically similar]
  graphify-out/memory/query_20260912_120101_973593b2_explore_the_atomic_file_upgrade_mechanism_with_app.md → AGENTS.md
- `outdatedAction` --embeds--> `ActionRef`  [EXTRACTED]
  cmd/root.go → internal/parser/parser.go
- `run()` --calls--> `DiscoverWorkflows()`  [EXTRACTED]
  cmd/root.go → internal/scanner/scanner.go
- `run()` --calls--> `Resolve()`  [EXTRACTED]
  cmd/root.go → internal/token/token.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Continuous Integration Quality and Security Gates** — github_workflows_ci_job_test, github_workflows_ci_job_lint, github_workflows_ci_job_static_code_analysis, github_workflows_ci_job_vulnerability_check [EXTRACTED 1.00]
- **Multi-Platform Artifact Publishing Flow** — github_workflows_release_workflow, github_workflows_update_homebrew_tap_workflow, goreleaser_distribution_targets [INFERRED 0.85]
- **Major Version Pinning and Configuration Subsystem** — actup_config, docs_plans_2026_06_09_major_pin_and_config_file_feature_major_pin, docs_plans_2026_06_09_major_pin_and_config_file_config_architecture, readme_config_file [INFERRED 0.95]
- **Go Gopher and GitHub Octocat Branding** — logo_actup_logo, logo_go_gopher, logo_github_octocat [INFERRED 0.85]
- **Graphify Knowledge Pipeline Subsystem** — agents_skills_graphify_skill, agents_skills_graphify_skill_pipeline, agents_rules_graphify, agents_skills_graphify_references_extraction_spec, agents_skills_graphify_references_query, agents_skills_graphify_references_update [EXTRACTED 1.00]
- **Actup Core Upgrade Architecture** — agents_parser_component, agents_breakingchanges_component, agents_tui_component, agents_upgrader_component, agents_atomic_file_replacement [EXTRACTED 1.00]
- **Graphify Traversal and Memory Synthesis** — agents_skills_graphify_references_query, agents_skills_graphify_references_query_memory_recording, graphify_out_memory_query_20260912_115939_f107bfaa_why_does_actionref_connect_workflow_action_parser, graphify_out_memory_query_20260912_120020_ac0879f9_trace_path_from_actionref_to_breakingchange_and_th, graphify_out_memory_query_20260912_120101_973593b2_explore_the_atomic_file_upgrade_mechanism_with_app [INFERRED 0.85]

## Communities (14 total, 1 thin omitted)

### Community 0 - "Graphify Knowledge & Tooling"
Cohesion: 0.05
Nodes (31): Rule: graphify, Reference: add & watch, graphify add command, graphify watch daemon, Reference: exports & benchmark, graphify token reduction benchmark, graphify export falkordb, graphify MCP server (+23 more)

### Community 1 - "Root Command & Dependencies"
Cohesion: 0.09
Nodes (14): outdatedAction, discoverInDir(), DiscoverWorkflows(), isWorkflowFile(), TestDiscoverWorkflows(), TestDiscoverWorkflowsEmpty(), TestDiscoverWorkflowsFilePath(), TestIsWorkflowFile() (+6 more)

### Community 2 - "Architecture & Component Specs"
Cohesion: 0.07
Nodes (27): AGENTS.md Architecture Guidance, internal/breakingchanges Component, actup CLI Architecture, internal/github Component, internal/parser Component, internal/scanner Component, Reference: git hooks & Claude integration, graphify Claude Code CLAUDE.md integration (+19 more)

### Community 3 - "CLI Execution & Tag Resolution"
Cohesion: 0.13
Nodes (24): filterSkippedActions(), run(), runCheck(), runNoTUI(), Client, TagMode, Config, LoadDefault() (+16 more)

### Community 4 - "Configuration & Project Governance"
Cohesion: 0.08
Nodes (30): Actup Local Configuration, Pinned Action: securego/gosec, Contributor Covenant Code of Conduct, Major Pin & Config File Implementation Plan, Major-Version Pinning Feature (--major), TagMode Struct Architecture, Bug Report Template, Feature Request Template (+22 more)

### Community 5 - "CLI Command Testing"
Cohesion: 0.13
Nodes (24): TestCheckFlagRegistered(), TestFilterSkippedActions(), TestInstallHooksSubcommandRegistered(), Load(), TestLoad_InvalidYAML(), TestLoad_MissingFile(), TestLoad_ValidConfig(), TestLoad_WithActions() (+16 more)

### Community 6 - "Terminal User Interface"
Cohesion: 0.15
Nodes (7): ActionItem, actionsLoadedMsg, applyResult, model, progressItem, state, summaryResult

### Community 7 - "Git Hook Management"
Cohesion: 0.16
Nodes (13): hookFilePath(), installHook(), installHookDryRun(), runInstallHooks(), TestInstallHooks_CreatesHook(), TestInstallHooks_DryRun(), TestInstallHooks_ExistingHookForce(), TestInstallHooks_ExistingHookNoForce() (+5 more)

### Community 8 - "Workflow Configuration Schema"
Cohesion: 0.12
Nodes (16): additionalProperties, description, examples, type, additionalProperties, anyOf, description, $id (+8 more)

### Community 9 - "Breaking Changes Registry"
Cohesion: 0.21
Nodes (11): Entry, Registry, BreakingChange, LoadRegistry(), parseMajor(), TestCheck_SmokeTestSeededData(), TestCheck_UnknownAction(), TestCheck_V2ToV3() (+3 more)

### Community 10 - "Atomic File Upgrader"
Cohesion: 0.25
Nodes (12): ApplyAllUpgrades(), replaceInFile(), showDryRunDiff(), TestApplyAllUpgrades(), TestApplyUpgrades(), TestApplyUpgradesConcurrent(), TestApplyUpgradesDryRun(), TestApplyUpgradesMultilineSteps() (+4 more)

### Community 11 - "Release & CI Pipelines"
Cohesion: 0.33
Nodes (6): Automated Release & Docker Publish Job, Release Pipeline Workflow, Homebrew Formula Generator, Update Homebrew Tap Workflow, GoReleaser Packaging Configuration, Multi-Platform Packaging Targets

### Community 12 - "Brand Assets & Visuals"
Cohesion: 0.67
Nodes (4): actup Logo, actup Brand Identity, GitHub Octocat Mascot, Go Gopher Mascot

## Knowledge Gaps
- **58 isolated node(s):** `installHooksOptions`, `github.com/lynicis/actup`, `$schema`, `$id`, `title` (+53 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 85 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **1 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `model` connect `Terminal User Interface` to `CLI Execution & Tag Resolution`?**
  _High betweenness centrality (0.057) - this node is a cross-community bridge._
- **Why does `graphify Skill` connect `Graphify Knowledge & Tooling` to `Architecture & Component Specs`?**
  _High betweenness centrality (0.044) - this node is a cross-community bridge._
- **Why does `Reference: query, path & explain` connect `Architecture & Component Specs` to `Graphify Knowledge & Tooling`?**
  _High betweenness centrality (0.019) - this node is a cross-community bridge._
- **Are the 5 inferred relationships involving `ApplyAllUpgrades()` (e.g. with `TestApplyAllUpgrades()` and `TestApplyUpgrades()`) actually correct?**
  _`ApplyAllUpgrades()` has 5 INFERRED edges - model-reasoned connections that need verification._
- **What connects `installHooksOptions`, `github.com/lynicis/actup`, `$schema` to the rest of the system?**
  _58 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Graphify Knowledge & Tooling` be split into smaller, more focused modules?**
  _Cohesion score 0.047619047619047616 - nodes in this community are weakly interconnected._
- **Should `Root Command & Dependencies` be split into smaller, more focused modules?**
  _Cohesion score 0.09246088193456614 - nodes in this community are weakly interconnected._
