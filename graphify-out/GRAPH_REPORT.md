# Graph Report - actup  (2026-11-15)

## Corpus Check
- 73 files · ~75,568 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 8 file(s) not represented in the graph (top: (none) 6, .toml 1, .lock 1)

## Summary
- 975 nodes · 2583 edges · 51 communities (46 shown, 5 thin omitted)
- Extraction: 86% EXTRACTED · 14% INFERRED · 0% AMBIGUOUS · INFERRED: 351 edges (avg confidence: 0.85)
- Token cost: 106,596 input · 4,724 output

## Graph Freshness
- Built from commit: `ea11d04a`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- mcp_tools.go
- common.go
- vulnscanner_test.go
- Terminal UI & Spinner Components
- audit_test.go
- Release Workflow & Token Config
- install_hooks.go
- Breaking Changes Warning System
- Workflow File Reference Upgrader
- parser_test.go
- Homebrew Tap Automation
- Actup Manifest Action Declarations
- ActionRef
- context.Context
- testing.T
- client_test.go
- Workflow Step Action Inputs
- .StatusForAction
- Pre-Commit Configuration Schema
- CI Workflow Execution Steps
- Workflow Job Runners & Conditions
- doSync
- GoReleaser Packaging Configuration
- Brag Plan: actup
- Cross-Platform Build Targets
- NFPM Linux Package Config
- Homebrew & Scoop Distribution
- Brag Video Hyperframes Config
- Node.js Tooling Package Manifest
- Continuous Integration Workflow Triggers
- Docker Scan & Security Permissions
- OCI Container Image Metadata
- GoReleaser Changelog Formatting
- Multi-OS Build Matrix
- Secret Scanning & Gitleaks
- GoReleaser Archive Packaging
- Docker Multi-Arch Image Builder
- GitHub Release Draft Config
- MCP Server Editor Configuration
- eventsAffect
- Static Analysis & Linter Workflow
- Vulnerability Check Job
- go_pkg_testing
- Documentation & Branding Assets
- Project Changelog History
- scanner_test.go
- Product Marketing Context
- Go Module Package Declaration
- Repository Security Policy
- Hyperframes Composition Brief: actup

## God Nodes (most connected - your core abstractions)
1. `ActionRef` - 48 edges
2. `writeWorkflow()` - 39 edges
3. `doUse()` - 30 edges
4. `LoadDefault()` - 30 edges
5. `with` - 30 edges
6. `handleUpgradeActions()` - 28 edges
7. `doSync()` - 28 edges
8. `model` - 25 edges
9. `runAuditFix()` - 23 edges
10. `actions` - 20 edges

## Surprising Connections (you probably didn't know these)
- `Non-obvious rules` --references--> `runCheck()`  [INFERRED]
  CLAUDE.md → cmd/root.go
- `Non-obvious rules` --references--> `Save()`  [INFERRED]
  CLAUDE.md → internal/config/config.go
- `Packages` --references--> `NewClientWithBaseURL()`  [INFERRED]
  CLAUDE.md → internal/github/client.go
- `Non-obvious rules` --references--> `confirmBreaking()`  [INFERRED]
  CLAUDE.md → cmd/common.go
- `outdatedAction` --embeds--> `ActionRef`  [EXTRACTED]
  cmd/root.go → internal/parser/parser.go

## Import Cycles
- None detected.

## Communities (51 total, 5 thin omitted)

### Community 0 - "mcp_tools.go"
Cohesion: 0.12
Nodes (40): auditUnknownSite, auditVulnFinding, auditVulnFix, auditVulnsIn, auditVulnsOut, auditVulnVuln, checkOutdatedIn, checkOutdatedOut (+32 more)

### Community 1 - "common.go"
Cohesion: 0.06
Nodes (32): auditJSONFinding, auditJSONVuln, fixTargetTooOldError, runInit(), newMCPServer(), runMCPTransport(), TestMCPRunToleratesClientDisconnect(), nopCloseWriter (+24 more)

### Community 2 - "vulnscanner_test.go"
Cohesion: 0.15
Nodes (22): multiIntervalVuln(), newMockOSV(), osvFromMap(), simpleVuln(), TestFixedFor_EarlierInterval(), TestFixedFor_MultiInterval(), TestFixedFor_NoFixedEvent(), TestFixedFor_VersionsOnly() (+14 more)

### Community 3 - "Terminal UI & Spinner Components"
Cohesion: 0.07
Nodes (29): Sync(), TestSync_UsesExecSeam(), allAtLatest(), currentLabel(), stripANSI(), TestAllAtLatest(), TestAllAtLatestMixed(), TestAllAtLatestSHA() (+21 more)

### Community 4 - "audit_test.go"
Cohesion: 0.14
Nodes (34): auditUpgrades(), auditVersionCompare(), auditVersionSegment(), maxAuditVersion(), minFixedFor(), planAuditFixes(), auditTestRepo(), auditTransitiveFixture() (+26 more)

### Community 5 - "Release Workflow & Token Config"
Cohesion: 0.06
Nodes (34): GH_TOKEN, GITHUB_TOKEN, HOMEBREW_TAP_GITHUB_TOKEN, TAG, jobs, release, name, on (+26 more)

### Community 6 - "install_hooks.go"
Cohesion: 0.26
Nodes (10): hookFilePath(), installHook(), runInstallHooks(), TestInstallHooks_CreatesHook(), TestInstallHooks_DryRun(), TestInstallHooks_ExistingHookForce(), TestInstallHooks_ExistingHookNoForce(), TestInstallHooks_Uninstall() (+2 more)

### Community 7 - "Breaking Changes Warning System"
Cohesion: 0.11
Nodes (29): Entry, Non-obvious rules, breakingChangesFor(), confirmBreaking(), confirmBreakingAll(), dedupeInputs(), TestConfirmBreakingAll_UnionInputsFromMultipleSites(), TestConfirmBreakingAllMixedVersions() (+21 more)

### Community 8 - "Workflow File Reference Upgrader"
Cohesion: 0.11
Nodes (30): TestPrintUseDiffRendersSHAComment(), printUseDiff(), ApplyAllUpgrades(), Upgrade, replaceAllInFile(), replaceInFile(), showDryRunDiff(), captureStdout() (+22 more)

### Community 9 - "parser_test.go"
Cohesion: 0.13
Nodes (29): DistinctVersions(), ExtractActions(), extractFromFile(), inputsForStep(), leadingIndent(), parseActionRef(), parseBlockMap(), parseWithValue() (+21 more)

### Community 10 - "Homebrew Tap Automation"
Cohesion: 0.08
Nodes (25): SHA, TAG, TAP_TOKEN, jobs, update-tap, name, on, push (+17 more)

### Community 11 - "Actup Manifest Action Declarations"
Cohesion: 0.08
Nodes (24): actions, actions/attest, actions/checkout, actions/dependency-review-action, actions/setup-go, actions/upload-artifact, anchore/sbom-action, aquasecurity/setup-trivy (+16 more)

### Community 12 - "ActionRef"
Cohesion: 0.19
Nodes (19): applyActionFilters(), filterActionsByPatterns(), printVulnWarnings(), summarizeActionVersions(), runOutdated(), outdatedAction, filterSkippedActions(), run() (+11 more)

### Community 13 - "context.Context"
Cohesion: 0.19
Nodes (13): actionSpec, resolveSpecRef(), resolveToken(), resolveLatestForAll(), runLsTable(), clientForSpecs(), ResolveResult, TagInfo (+5 more)

### Community 14 - "testing.T"
Cohesion: 0.06
Nodes (101): isCommitSHA(), mostCommonVersion(), parseActionSpec(), recordUpgrades(), savePin(), doInit(), initOptions, assertWorkflowContains() (+93 more)

### Community 15 - "client_test.go"
Cohesion: 0.11
Nodes (24): AGENTS.md, Commands, External references, Packages, NewClientWithBaseURL(), resolveLatestTag(), resolvePin(), tagMajor() (+16 more)

### Community 16 - "Workflow Step Action Inputs"
Cohesion: 0.10
Nodes (21): with, advanced-security, annotations, args, cache, category, comment-summary-in-pr, dockerfile (+13 more)

### Community 17 - ".StatusForAction"
Cohesion: 0.33
Nodes (5): runLsDetail(), lockStatusString(), Status, lockSpecMatches(), TestStatusForAction_NarrowedLockRef()

### Community 18 - "Pre-Commit Configuration Schema"
Cohesion: 0.12
Nodes (16): default_install_hook_types, default_stages, always_run, args, description, entry, id, language (+8 more)

### Community 19 - "CI Workflow Execution Steps"
Cohesion: 0.24
Nodes (15): steps, steps, steps, steps, workflow-security, steps, steps, if (+7 more)

### Community 20 - "Workflow Job Runners & Conditions"
Cohesion: 0.14
Nodes (14): name, runs-on, if, name, runs-on, jobs, action-audit, dependency-review (+6 more)

### Community 21 - "doSync"
Cohesion: 0.15
Nodes (19): auditFindingsError(), auditMajor(), auditSiteVersion(), normalizedFixed(), outputAuditJSON(), resolveFixTarget(), runAudit(), runAuditFix() (+11 more)

### Community 22 - "GoReleaser Packaging Configuration"
Cohesion: 0.12
Nodes (15): before, hooks, checksum, algorithm, name_template, docker_signs, args, cmd (+7 more)

### Community 23 - "Brag Plan: actup"
Cohesion: 0.10
Nodes (19): Audio direction, Brag Plan: actup, Duration: 21 seconds, Format: landscape — 1920x1080, Hook (first 2-3 seconds), Key moments (the middle), Outro / punchline, Scene 1 — Hook: The Workflow Drift Problem — 3.8s (0.0s – 3.8s) (+11 more)

### Community 24 - "Cross-Platform Build Targets"
Cohesion: 0.18
Nodes (11): builds, env, goarch, goarm, goos, id, ignore, ldflags (+3 more)

### Community 25 - "NFPM Linux Package Config"
Cohesion: 0.20
Nodes (10): nfpms, bindir, description, formats, homepage, id, license, maintainer (+2 more)

### Community 26 - "Homebrew & Scoop Distribution"
Cohesion: 0.20
Nodes (10): name, owner, token, scoops, commit_msg_template, description, homepage, license (+2 more)

### Community 27 - "Brag Video Hyperframes Config"
Cohesion: 0.22
Nodes (8): media, autoProxy, paths, assets, blocks, components, registry, $schema

### Community 28 - "Node.js Tooling Package Manifest"
Cohesion: 0.22
Nodes (8): name, private, scripts, check, dev, publish, render, type

### Community 29 - "Continuous Integration Workflow Triggers"
Cohesion: 0.22
Nodes (8): name, on, pull_request, push, branches, paths-ignore, branches, paths-ignore

### Community 30 - "Docker Scan & Security Permissions"
Cohesion: 0.25
Nodes (9): name, permissions, runs-on, docker-scan, permissions, actions, contents, security-events (+1 more)

### Community 31 - "OCI Container Image Metadata"
Cohesion: 0.22
Nodes (9): org.opencontainers.image.created, org.opencontainers.image.description, org.opencontainers.image.licenses, org.opencontainers.image.revision, org.opencontainers.image.source, org.opencontainers.image.title, org.opencontainers.image.url, org.opencontainers.image.version (+1 more)

### Community 32 - "GoReleaser Changelog Formatting"
Cohesion: 0.22
Nodes (9): changelog, filters, groups, sort, use, exclude, order, regexp (+1 more)

### Community 33 - "Multi-OS Build Matrix"
Cohesion: 0.29
Nodes (7): name, needs, runs-on, strategy, build, os, matrix

### Community 34 - "Secret Scanning & Gitleaks"
Cohesion: 0.29
Nodes (7): GITHUB_TOKEN, GITLEAKS_ENABLE_COMMENTS, secret-scan, name, runs-on, steps, env

### Community 35 - "GoReleaser Archive Packaging"
Cohesion: 0.29
Nodes (7): archives, format_overrides, formats, id, name_template, formats, goos

### Community 36 - "Docker Multi-Arch Image Builder"
Cohesion: 0.33
Nodes (6): dockers_v2, dockerfile, flags, images, platforms, tags

### Community 37 - "GitHub Release Draft Config"
Cohesion: 0.29
Nodes (7): name, owner, release, draft, github, prerelease, replace_existing_artifacts

### Community 38 - "MCP Server Editor Configuration"
Cohesion: 0.33
Nodes (5): go, gopls, actup, gopls, producthunt

### Community 39 - "eventsAffect"
Cohesion: 0.22
Nodes (13): boundedEvents(), buildIntervals(), compareVersions(), eventsAffect(), inInterval(), lastAffectedEnd(), lastKnownBoundEvent(), normalizeVersion() (+5 more)

### Community 40 - "Static Analysis & Linter Workflow"
Cohesion: 0.40
Nodes (5): GO111MODULE, static-code-analysis, env, name, runs-on

### Community 41 - "Vulnerability Check Job"
Cohesion: 0.50
Nodes (4): vulnerability-check, name, runs-on, steps

### Community 42 - "go_pkg_testing"
Cohesion: 0.17
Nodes (13): FormatVersion(), init(), SetVersionInfo(), TestRootVersionFlag(), TestVersionOutput(), TestVersionShortOutput(), TestVersionSubcommandRegistered(), Resolve() (+5 more)

### Community 46 - "scanner_test.go"
Cohesion: 0.31
Nodes (12): DiscoverWorkflows(), isWorkflowFile(), TestDiscoverWorkflows(), TestDiscoverWorkflows_Dedup(), TestDiscoverWorkflows_DefaultsIgnoreNonActionYAMLInActionsDir(), TestDiscoverWorkflows_DefaultsIncludeCompositeAndRootAction(), TestDiscoverWorkflows_DefaultsMissingActionsDir(), TestDiscoverWorkflows_ExplicitPathDoesNotAddDefaults() (+4 more)

### Community 47 - "Product Marketing Context"
Cohesion: 0.12
Nodes (15): Brand Voice, Changelog, Competitive Landscape, Customer Language, Differentiation, Goals, Learnings, Objections (+7 more)

### Community 52 - "Hyperframes Composition Brief: actup"
Cohesion: 0.20
Nodes (9): Audio, Creative Direction, Hyperframes Composition Brief: actup, Hyperframes Instructions, Objective, Output, Source Material, Storyboard (+1 more)

## Knowledge Gaps
- **8 isolated node(s):** `go`, `gopls`, `github.com/lynicis/actup`, `github-actions-updater Skill`, `actup TUI Demo` (+3 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 296 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **5 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `ActionRef` connect `ActionRef` to `mcp_tools.go`, `common.go`, `Terminal UI & Spinner Components`, `audit_test.go`, `Breaking Changes Warning System`, `Workflow File Reference Upgrader`, `parser_test.go`, `context.Context`, `testing.T`, `.StatusForAction`, `doSync`?**
  _High betweenness centrality (0.018) - this node is a cross-community bridge._
- **Are the 21 inferred relationships involving `writeWorkflow()` (e.g. with `TestMCPAuditVulnerabilities()` and `TestMCPCheckOutdated()`) actually correct?**
  _`writeWorkflow()` has 21 INFERRED edges - model-reasoned connections that need verification._
- **What connects `go`, `gopls`, `github.com/lynicis/actup` to the rest of the system?**
  _8 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `mcp_tools.go` be split into smaller, more focused modules?**
  _Cohesion score 0.12181616832779624 - nodes in this community are weakly interconnected._
- **Why does `model` connect `Terminal UI & Spinner Components` to `common.go`, `ActionRef`?**
  _High betweenness centrality (0.015) - this node is a cross-community bridge._
- **Are the 18 inferred relationships involving `doUse()` (e.g. with `TestUseRejectsSHATarget()` and `anyRefPinned()`) actually correct?**
  _`doUse()` has 18 INFERRED edges - model-reasoned connections that need verification._
- **Should `common.go` be split into smaller, more focused modules?**
  _Cohesion score 0.06134731120170841 - nodes in this community are weakly interconnected._
