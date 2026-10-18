# Graph Report - feat-vuln-scanner  (2026-10-18)

## Corpus Check
- 50 files · ~37,071 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 6 file(s) not represented in the graph (top: (none) 6)

## Summary
- 631 nodes · 1404 edges · 49 communities (38 shown, 11 thin omitted)
- Extraction: 87% EXTRACTED · 13% INFERRED · 0% AMBIGUOUS · INFERRED: 178 edges (avg confidence: 0.88)
- Token cost: 0 input · 0 output

## Graph Freshness
- Built from commit: `a2738616`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- Graphify Knowledge & Tooling
- go_pkg_os
- Architecture & Component Specs
- root.go
- Actup Project Documentation
- testing.T
- Terminal User Interface
- Git Hook Management
- Workflow Configuration Schema
- common.go
- Atomic File Upgrader
- Release & CI Pipelines
- Brand Assets & Visuals
- Go Module Root
- vulnscanner.go
- vulnscanner_test.go
- Contributor Covenant Code of Conduct
- Changelog
- README.md
- bug_report.md
- Actup Local Configuration
- GitHub Actions Updater
- CI Multi-OS Build Job
- [0.7.2] - 2026-07-02
- feature_request.md
- PULL_REQUEST_TEMPLATE.md
- Installation
- Security Policy
- [0.7.5] - 2026-07-24
- [0.7.6] - 2026-09-10
- Contributing
- Pre-commit Hooks
- [0.7.0] - 2026-06-16
- [0.7.1] - 2026-06-16
- [0.7.3] - 2026-07-12
- [0.7.4] - 2026-07-23
- AI Skill
- Configuration
- Version Manager
- [0.1.2] - 2026-06-07
- [0.1.3] - 2026-06-07
- [0.1.4] - 2026-06-07
- [0.1.5] - 2026-06-08
- [0.1.6] - 2026-06-08
- [0.2.0] - 2026-06-08
- [0.3.1] - 2026-06-08
- [0.4.0] - 2026-06-09
- [0.5.0] - 2026-06-09
- [0.6.0] - 2026-06-14

## God Nodes (most connected - your core abstractions)
1. `ActionRef` - 25 edges
2. `model` - 25 edges
3. `Changelog` - 24 edges
4. `doUse()` - 18 edges
5. `doSync()` - 17 edges
6. `runNoTUI()` - 15 edges
7. `LoadDefault()` - 15 edges
8. `ApplyAllUpgrades()` - 15 edges
9. `run()` - 14 edges
10. `resolveLatestTag()` - 13 edges

## Surprising Connections (you probably didn't know these)
- `Testing` --references--> `resolveLatestTag()`  [INFERRED]
  AGENTS.md → internal/github/client.go
- `Changed` --references--> `ResolveVersion()`  [INFERRED]
  CHANGELOG.md → internal/github/client.go
- `Architecture` --references--> `actionSpec`  [INFERRED]
  AGENTS.md → cmd/common.go
- `Task 7: Per-action pins in `runNoTUI`` --references--> `runNoTUI()`  [INFERRED]
  docs/plans/2026-06-09-major-pin-and-config-file.md → cmd/root.go
- `Runtime behavior` --references--> `runCheck()`  [INFERRED]
  AGENTS.md → cmd/root.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **Continuous Integration Quality and Security Gates** — github_workflows_ci_job_test, github_workflows_ci_job_lint, github_workflows_ci_job_static_code_analysis, github_workflows_ci_job_vulnerability_check [EXTRACTED 1.00]
- **Actup Core Upgrade Architecture** — agents_parser_component, agents_breakingchanges_component, agents_tui_component, agents_upgrader_component, agents_atomic_file_replacement [EXTRACTED 1.00]
- **Graphify Knowledge Pipeline Subsystem** — agents_skills_graphify_skill, agents_skills_graphify_skill_pipeline, agents_rules_graphify, agents_skills_graphify_references_extraction_spec, agents_skills_graphify_references_query, agents_skills_graphify_references_update [EXTRACTED 1.00]
- **Graphify Traversal and Memory Synthesis** — agents_skills_graphify_references_query, agents_skills_graphify_references_query_memory_recording, graphify_out_memory_query_20260912_115939_f107bfaa_why_does_actionref_connect_workflow_action_parser, graphify_out_memory_query_20260912_120020_ac0879f9_trace_path_from_actionref_to_breakingchange_and_th, graphify_out_memory_query_20260912_120101_973593b2_explore_the_atomic_file_upgrade_mechanism_with_app [INFERRED 0.85]
- **Go Gopher and GitHub Octocat Branding** — logo_actup_logo, logo_go_gopher, logo_github_octocat [INFERRED 0.85]
- **Multi-Platform Artifact Publishing Flow** — github_workflows_release_workflow, github_workflows_update_homebrew_tap_workflow, goreleaser_distribution_targets [INFERRED 0.85]
- **Major Version Pinning and Configuration Subsystem** — actup_config, docs_plans_2026_06_09_major_pin_and_config_file_feature_major_pin, docs_plans_2026_06_09_major_pin_and_config_file_config_architecture, readme_config_file [INFERRED 0.95]

## Communities (49 total, 11 thin omitted)

### Community 0 - "Graphify Knowledge & Tooling"
Cohesion: 0.04
Nodes (38): Rule: graphify, Reference: add & watch, graphify add command, graphify watch daemon, Reference: exports & benchmark, graphify token reduction benchmark, graphify export falkordb, graphify MCP server (+30 more)

### Community 1 - "go_pkg_os"
Cohesion: 0.14
Nodes (16): [0.7.7] - 2026-09-12, Changed, Removed, discoverInDir(), DiscoverWorkflows(), isWorkflowFile(), TestDiscoverWorkflows(), TestDiscoverWorkflowsEmpty() (+8 more)

### Community 2 - "Architecture & Component Specs"
Cohesion: 0.05
Nodes (35): AGENTS.md — actup, internal/breakingchanges Component, CI / Release, actup CLI Architecture, Dependency notes, Developer commands, Docker, internal/github Component (+27 more)

### Community 3 - "root.go"
Cohesion: 0.06
Nodes (59): runAudit(), loadWorkflowActions(), mostCommonVersion(), printVulnWarnings(), resolveLatestForAll(), runLs(), runLsDetail(), runLsTable() (+51 more)

### Community 4 - "Actup Project Documentation"
Cohesion: 0.22
Nodes (10): actions/checkout v2 to v3 Breaking Change Definition, Embedded Breaking Changes Registry, Breaking Change Detection & Warning System, Actup Project Documentation, Bubble Tea Interactive TUI, Non-Interactive Upgrade Mode (--no-tui), Security Policy & Reporting, GitHub Actions Updater Agent Skill (+2 more)

### Community 5 - "testing.T"
Cohesion: 0.09
Nodes (47): Architecture, Added, parseActionSpec(), savePin(), doInit(), runInit(), initOptions, runUnpin() (+39 more)

### Community 6 - "Terminal User Interface"
Cohesion: 0.11
Nodes (17): stripANSI(), TestInitReturnsCommand(), TestUpdateChecklistTogglesSelection(), TestUpdateLoadingHandlesSpinnerTick(), TestUpdateLoadingIgnoresOtherMessages(), TestUpdateLoadingTransitionsToChecklist(), TestViewChecklistCursorHighlight(), TestViewChecklistRendering() (+9 more)

### Community 7 - "Git Hook Management"
Cohesion: 0.12
Nodes (20): hookFilePath(), installHook(), installHookDryRun(), runInstallHooks(), TestInstallHooks_CreatesHook(), TestInstallHooks_DryRun(), TestInstallHooks_ExistingHookForce(), TestInstallHooks_ExistingHookNoForce() (+12 more)

### Community 8 - "Workflow Configuration Schema"
Cohesion: 0.12
Nodes (16): additionalProperties, description, examples, type, additionalProperties, anyOf, description, $id (+8 more)

### Community 9 - "common.go"
Cohesion: 0.10
Nodes (29): Runtime behavior, Entry, Registry, actionSpec, breakingChangesFor(), confirmBreaking(), isTerminal(), resolveSpecRef() (+21 more)

### Community 10 - "Atomic File Upgrader"
Cohesion: 0.24
Nodes (13): ApplyAllUpgrades(), Upgrade, replaceInFile(), showDryRunDiff(), TestApplyAllUpgrades(), TestApplyUpgrades(), TestApplyUpgradesConcurrent(), TestApplyUpgradesDryRun() (+5 more)

### Community 11 - "Release & CI Pipelines"
Cohesion: 0.33
Nodes (6): Automated Release & Docker Publish Job, Release Pipeline Workflow, Homebrew Formula Generator, Update Homebrew Tap Workflow, GoReleaser Packaging Configuration, Multi-Platform Packaging Targets

### Community 12 - "Brand Assets & Visuals"
Cohesion: 0.67
Nodes (4): actup Logo, actup Brand Identity, GitHub Octocat Mascot, Go Gopher Mascot

### Community 14 - "vulnscanner.go"
Cohesion: 0.11
Nodes (20): outputAuditJSON(), TestAuditJSONOutputShape(), TestAuditSkipsSHARefs(), auditJSONFinding, auditJSONVuln, buildIntervals(), compareVersions(), eventsAffect() (+12 more)

### Community 15 - "vulnscanner_test.go"
Cohesion: 0.28
Nodes (13): newMockOSV(), simpleVuln(), TestScanFindsKnownVuln(), TestScanGroupsUsageSites(), TestScanLastAffected(), TestScanMixedCaseOwner(), TestScanMultipleIntervals(), TestScanNoActions() (+5 more)

### Community 16 - "Contributor Covenant Code of Conduct"
Cohesion: 0.15
Nodes (12): 1. Correction, 2. Warning, 3. Temporary Ban, 4. Permanent Ban, Attribution, Contributor Covenant Code of Conduct, Enforcement, Enforcement Guidelines (+4 more)

### Community 17 - "Changelog"
Cohesion: 0.18
Nodes (11): [0.1.0] - 2026-06-07, [0.1.1] - 2026-06-07, [0.3.0] - 2026-06-08, [0.5.1] - 2026-06-10, Added, Changed, Changelog, Documentation (+3 more)

### Community 18 - "README.md"
Cohesion: 0.18
Nodes (10): Acknowledgments, Breaking Changes, Demo, Features, License, Quick Start, Roadmap, Table of Contents (+2 more)

### Community 19 - "bug_report.md"
Cohesion: 0.25
Nodes (7): Actual Behavior / Logs, Additional Context, Bug Description, Environment, Expected Behavior, Steps to Reproduce, Workflow File

### Community 20 - "Actup Local Configuration"
Cohesion: 0.29
Nodes (6): Actup Local Configuration, Pinned Action: securego/gosec, Major Pin & Config File Implementation Plan, Major-Version Pinning Feature (--major), TagMode Struct Architecture, Project Configuration File (.actup.yaml)

### Community 21 - "GitHub Actions Updater"
Cohesion: 0.29
Nodes (6): Breaking-change registry, GitHub Actions Updater, Output template, Quick start, Rules, Workflow

### Community 22 - "CI Multi-OS Build Job"
Cohesion: 0.60
Nodes (6): CI Multi-OS Build Job, CI Lint Job, CI Gosec Static Analysis Job, CI Test Job, CI Govulncheck Job, Continuous Integration Workflow

### Community 23 - "[0.7.2] - 2026-07-02"
Cohesion: 0.40
Nodes (5): [0.7.2] - 2026-07-02, Added, Changed, Fixed, Removed

### Community 24 - "feature_request.md"
Cohesion: 0.40
Nodes (4): Additional Context, Describe alternatives you've considered, Describe the solution you'd like, Is your feature request related to a problem? Please describe.

### Community 25 - "PULL_REQUEST_TEMPLATE.md"
Cohesion: 0.40
Nodes (4): Checklist, Description, How Has This Been Tested?, Type of Change

### Community 26 - "Installation"
Cohesion: 0.40
Nodes (5): Build from Source, Docker, Installation, Package Managers, Pre-built Binaries

### Community 27 - "Security Policy"
Cohesion: 0.40
Nodes (4): Reporting a Vulnerability, Response Process, Security Policy, Supported Versions

### Community 28 - "[0.7.5] - 2026-07-24"
Cohesion: 0.50
Nodes (4): [0.7.5] - 2026-07-24, Added, Changed, Fixed

### Community 29 - "[0.7.6] - 2026-09-10"
Cohesion: 0.50
Nodes (4): [0.7.6] - 2026-09-10, Changed, Documentation, Removed

### Community 30 - "Contributing"
Cohesion: 0.50
Nodes (4): Bug Report Template, Feature Request Template, Pull Request Template, Contributing

### Community 31 - "Pre-commit Hooks"
Cohesion: 0.50
Nodes (4): actup-check Pre-commit Hook Specification, `actup install-hooks` (built-in), pre-commit framework, Pre-commit Hooks

### Community 32 - "[0.7.0] - 2026-06-16"
Cohesion: 0.67
Nodes (3): [0.7.0] - 2026-06-16, Added, Fixed

### Community 33 - "[0.7.1] - 2026-06-16"
Cohesion: 0.67
Nodes (3): [0.7.1] - 2026-06-16, Added, Fixed

### Community 34 - "[0.7.3] - 2026-07-12"
Cohesion: 0.67
Nodes (3): [0.7.3] - 2026-07-12, Changed, Documentation

### Community 35 - "[0.7.4] - 2026-07-23"
Cohesion: 0.67
Nodes (3): [0.7.4] - 2026-07-23, Added, Changed

### Community 36 - "AI Skill"
Cohesion: 0.67
Nodes (3): AI Skill, Install, Usage

### Community 37 - "Configuration"
Cohesion: 0.67
Nodes (3): Config File (`.actup.yaml`), Configuration, GitHub Token Resolution

### Community 38 - "Version Manager"
Cohesion: 0.67
Nodes (3): Ref resolution in `use`, Typical workflow, Version Manager

## Knowledge Gaps
- **41 isolated node(s):** `github.com/lynicis/actup`, `graphify add command`, `graphify token reduction benchmark`, `graphify export falkordb`, `graphify MCP server` (+36 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 201 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **11 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `AGENTS.md — actup` connect `Architecture & Component Specs` to `common.go`, `testing.T`?**
  _High betweenness centrality (0.198) - this node is a cross-community bridge._
- **Are the 2 inferred relationships involving `model` (e.g. with `Task 3: Update `internal/tui` for `--major`` and `Threading`) actually correct?**
  _`model` has 2 INFERRED edges - model-reasoned connections that need verification._
- **What connects `github.com/lynicis/actup`, `graphify add command`, `graphify token reduction benchmark` to the rest of the system?**
  _41 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Graphify Knowledge & Tooling` be split into smaller, more focused modules?**
  _Cohesion score 0.04081632653061224 - nodes in this community are weakly interconnected._
- **Why does `actup CLI Architecture` connect `Architecture & Component Specs` to `Graphify Knowledge & Tooling`?**
  _High betweenness centrality (0.186) - this node is a cross-community bridge._
- **Should `go_pkg_os` be split into smaller, more focused modules?**
  _Cohesion score 0.13768115942028986 - nodes in this community are weakly interconnected._
- **Why does `main()` connect `Git Hook Management` to `Pre-commit Hooks`?**
  _High betweenness centrality (0.175) - this node is a cross-community bridge._