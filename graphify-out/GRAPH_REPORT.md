# Graph Report - actup  (2026-11-15)

## Corpus Check
- 73 files · ~78,276 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 8 file(s) not represented in the graph (top: (none) 6, .toml 1, .lock 1)

## Summary
- 978 nodes · 2594 edges · 50 communities (43 shown, 7 thin omitted)
- Extraction: 86% EXTRACTED · 14% INFERRED · 0% AMBIGUOUS · INFERRED: 354 edges (avg confidence: 0.85)
- Token cost: 106,596 input · 4,724 output

## Graph Freshness
- Built from commit: `bdb7b2f3`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- mcp_tools.go
- ls.go
- parser_test.go
- Terminal UI & Spinner Components
- audit.go
- Release Workflow & Token Config
- go_pkg_testing
- Breaking Changes Warning System
- Workflow File Reference Upgrader
- main.go
- Homebrew Tap Automation
- Actup Manifest Action Declarations
- scanner_test.go
- root.go
- testing.T
- mcp_test.go
- Workflow Step Action Inputs
- lockfile.go
- Pre-Commit Configuration Schema
- CI Workflow Execution Steps
- Workflow Job Runners & Conditions
- go_pkg_os
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
- go_pkg_strings
- Documentation & Branding Assets
- Project Changelog History
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
- `Non-obvious rules` --references--> `Save()`  [INFERRED]
  CLAUDE.md → internal/config/config.go
- `Packages` --references--> `NewClientWithBaseURL()`  [INFERRED]
  CLAUDE.md → internal/github/client.go
- `Non-obvious rules` --references--> `confirmBreaking()`  [INFERRED]
  CLAUDE.md → cmd/common.go
- `Non-obvious rules` --references--> `runCheck()`  [INFERRED]
  CLAUDE.md → cmd/root.go
- `outdatedAction` --embeds--> `ActionRef`  [EXTRACTED]
  cmd/root.go → internal/parser/parser.go

## Import Cycles
- None detected.

## Communities (50 total, 7 thin omitted)

### Community 0 - "mcp_tools.go"
Cohesion: 0.06
Nodes (92): Non-obvious rules, actionSpec, auditFindingsError(), normalizedFixed(), outputAuditJSON(), resolveFixTarget(), runAudit(), runAuditFix() (+84 more)

### Community 2 - "parser_test.go"
Cohesion: 0.14
Nodes (29): DistinctVersions(), ExtractActions(), extractFromFile(), inputsForStep(), leadingIndent(), parseActionRef(), parseBlockMap(), parseWithValue() (+21 more)

### Community 3 - "Terminal UI & Spinner Components"
Cohesion: 0.08
Nodes (30): allAtLatest(), currentLabel(), stripANSI(), TestAllAtLatest(), TestAllAtLatestMixed(), TestAllAtLatestSHA(), TestAllAtLatestSHA_Predicate(), TestInitReturnsCommand() (+22 more)

### Community 4 - "audit.go"
Cohesion: 0.11
Nodes (39): auditMajor(), auditSiteVersion(), auditUpgrades(), auditVersionCompare(), auditVersionSegment(), maxAuditVersion(), minFixedFor(), planAuditFixes() (+31 more)

### Community 5 - "Release Workflow & Token Config"
Cohesion: 0.06
Nodes (34): GH_TOKEN, GITHUB_TOKEN, HOMEBREW_TAP_GITHUB_TOKEN, TAG, jobs, release, name, on (+26 more)

### Community 6 - "go_pkg_testing"
Cohesion: 0.23
Nodes (10): hookFilePath(), installHook(), runInstallHooks(), TestInstallHooks_CreatesHook(), TestInstallHooks_DryRun(), TestInstallHooks_ExistingHookForce(), TestInstallHooks_ExistingHookNoForce(), TestInstallHooks_Uninstall() (+2 more)

### Community 7 - "Breaking Changes Warning System"
Cohesion: 0.09
Nodes (27): Entry, breakingChangesFor(), TestConfirmBreakingAll_UnionInputsFromMultipleSites(), TestConfirmBreakingAllMixedVersions(), syncOptions, checkUseBreaking(), useOptions, BreakingChange (+19 more)

### Community 8 - "Workflow File Reference Upgrader"
Cohesion: 0.12
Nodes (28): ApplyAllUpgrades(), Upgrade, replaceAllInFile(), replaceInFile(), showDryRunDiff(), captureStdout(), TestApplyAllUpgrades(), TestApplyAllUpgradesContinuesPastErrors() (+20 more)

### Community 9 - "main.go"
Cohesion: 0.22
Nodes (9): Execute(), FormatVersion(), init(), SetVersionInfo(), TestRootVersionFlag(), TestVersionOutput(), TestVersionShortOutput(), TestVersionSubcommandRegistered() (+1 more)

### Community 10 - "Homebrew Tap Automation"
Cohesion: 0.08
Nodes (25): SHA, TAG, TAP_TOKEN, jobs, update-tap, name, on, push (+17 more)

### Community 11 - "Actup Manifest Action Declarations"
Cohesion: 0.08
Nodes (24): actions, actions/attest, actions/checkout, actions/dependency-review-action, actions/setup-go, actions/upload-artifact, anchore/sbom-action, aquasecurity/setup-trivy (+16 more)

### Community 12 - "scanner_test.go"
Cohesion: 0.31
Nodes (12): DiscoverWorkflows(), isWorkflowFile(), TestDiscoverWorkflows(), TestDiscoverWorkflows_Dedup(), TestDiscoverWorkflows_DefaultsIgnoreNonActionYAMLInActionsDir(), TestDiscoverWorkflows_DefaultsIncludeCompositeAndRootAction(), TestDiscoverWorkflows_DefaultsMissingActionsDir(), TestDiscoverWorkflows_ExplicitPathDoesNotAddDefaults() (+4 more)

### Community 13 - "root.go"
Cohesion: 0.27
Nodes (6): outdatedAction, init(), isGHExtensionInvocation(), TestIsGHExtensionInvocation(), appendPath(), discoverInDir()

### Community 14 - "testing.T"
Cohesion: 0.06
Nodes (97): parseActionSpec(), recordUpgrades(), savePin(), doInit(), assertWorkflowContains(), assertWorkflowNotContains(), TestMCPAuditVulnerabilities(), TestMCPCheckOutdated() (+89 more)

### Community 15 - "mcp_test.go"
Cohesion: 0.08
Nodes (28): AGENTS.md, Commands, External references, Packages, ResolveResult, TagInfo, TagMode, NewClientWithBaseURL() (+20 more)

### Community 16 - "Workflow Step Action Inputs"
Cohesion: 0.10
Nodes (21): with, advanced-security, annotations, args, cache, category, comment-summary-in-pr, dockerfile (+13 more)

### Community 17 - "lockfile.go"
Cohesion: 0.14
Nodes (18): lockStatusString(), canonicalPinKey(), Lockfile, Status, Load(), LoadForPaths(), lockSpecMatches(), Sync() (+10 more)

### Community 18 - "Pre-Commit Configuration Schema"
Cohesion: 0.12
Nodes (16): default_install_hook_types, default_stages, always_run, args, description, entry, id, language (+8 more)

### Community 19 - "CI Workflow Execution Steps"
Cohesion: 0.24
Nodes (15): steps, steps, steps, steps, workflow-security, steps, steps, if (+7 more)

### Community 20 - "Workflow Job Runners & Conditions"
Cohesion: 0.14
Nodes (14): name, runs-on, if, name, runs-on, jobs, action-audit, dependency-review (+6 more)

### Community 21 - "go_pkg_os"
Cohesion: 0.33
Nodes (6): Resolve(), TestResolve_EnvVarUsedWhenFlagEmpty(), TestResolve_FlagValueTakesPriority(), TestResolve_GhAuthErrorReturnsEmpty(), TestResolve_GhNotFoundReturnsEmpty(), TestResolve_GhTokenUsedWhenFlagAndEnvEmpty()

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
Cohesion: 0.07
Nodes (42): boundedEvents(), buildIntervals(), compareVersions(), eventsAffect(), inInterval(), lastAffectedEnd(), lastKnownBoundEvent(), NewClient() (+34 more)

### Community 40 - "Static Analysis & Linter Workflow"
Cohesion: 0.40
Nodes (5): GO111MODULE, static-code-analysis, env, name, runs-on

### Community 41 - "Vulnerability Check Job"
Cohesion: 0.50
Nodes (4): vulnerability-check, name, runs-on, steps

### Community 47 - "Product Marketing Context"
Cohesion: 0.12
Nodes (15): Brand Voice, Changelog, Competitive Landscape, Customer Language, Differentiation, Goals, Learnings, Objections (+7 more)

### Community 52 - "Hyperframes Composition Brief: actup"
Cohesion: 0.20
Nodes (9): Audio, Creative Direction, Hyperframes Composition Brief: actup, Hyperframes Instructions, Objective, Output, Source Material, Storyboard (+1 more)

## Knowledge Gaps
- **8 isolated node(s):** `go`, `gopls`, `github.com/lynicis/actup`, `github-actions-updater Skill`, `actup TUI Demo` (+3 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 296 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **7 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `ActionRef` connect `mcp_tools.go` to `parser_test.go`, `Terminal UI & Spinner Components`, `audit.go`, `Breaking Changes Warning System`, `Workflow File Reference Upgrader`, `root.go`, `testing.T`, `lockfile.go`?**
  _High betweenness centrality (0.017) - this node is a cross-community bridge._
- **Are the 21 inferred relationships involving `writeWorkflow()` (e.g. with `TestMCPAuditVulnerabilities()` and `TestMCPCheckOutdated()`) actually correct?**
  _`writeWorkflow()` has 21 INFERRED edges - model-reasoned connections that need verification._
- **What connects `go`, `gopls`, `github.com/lynicis/actup` to the rest of the system?**
  _8 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `mcp_tools.go` be split into smaller, more focused modules?**
  _Cohesion score 0.05554628857381151 - nodes in this community are weakly interconnected._
- **Why does `model` connect `Terminal UI & Spinner Components` to `mcp_tools.go`, `lockfile.go`?**
  _High betweenness centrality (0.014) - this node is a cross-community bridge._
- **Are the 18 inferred relationships involving `doUse()` (e.g. with `TestUseRejectsSHATarget()` and `anyRefPinned()`) actually correct?**
  _`doUse()` has 18 INFERRED edges - model-reasoned connections that need verification._
- **Should `parser_test.go` be split into smaller, more focused modules?**
  _Cohesion score 0.13978494623655913 - nodes in this community are weakly interconnected._
