# Graph Report - actup  (2026-11-14)

## Corpus Check
- 115 files · ~97,268 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 7 file(s) not represented in the graph (top: (none) 6, .lock 1)

## Summary
- 926 nodes · 2529 edges · 52 communities (43 shown, 9 thin omitted)
- Extraction: 86% EXTRACTED · 14% INFERRED · 0% AMBIGUOUS · INFERRED: 343 edges (avg confidence: 0.85)
- Token cost: 106,596 input · 4,724 output

## Community Hubs (Navigation)
- Vulnerability Audit & Auto-Fix
- Audit CLI & Findings Formatting
- HTTP & GitHub API Client
- Terminal UI & Spinner Components
- Vulnerability Version Matching
- Release Workflow & Token Config
- Git Pre-Commit Hook Installer
- Breaking Changes Warning System
- Workflow File Reference Upgrader
- CLI Root & Filter Execution
- Homebrew Tap Automation
- Actup Manifest Action Declarations
- YAML Manifest Configuration Parser
- Workflow Uses Parser
- Action Pinning & Version Sync
- GitHub Tag & SHA Resolver
- Workflow Step Action Inputs
- MCP Server Integration Tests
- Pre-Commit Configuration Schema
- CI Workflow Execution Steps
- Workflow Job Runners & Conditions
- Commit SHA Pinning Tests
- GoReleaser Packaging Configuration
- Version Flag & Build Info
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
- Model Context Protocol Server
- Static Analysis & Linter Workflow
- Vulnerability Check Job
- Cosign Docker Signing Config
- Documentation & Branding Assets
- Project Changelog History
- Release & Launch Strategy
- Breaking Changes Registry Spec
- GitHub API Client Architecture
- Go Module Package Declaration
- Repository Security Policy

## God Nodes (most connected - your core abstractions)
1. `ActionRef` - 48 edges
2. `writeWorkflow()` - 39 edges
3. `with` - 30 edges
4. `doUse()` - 30 edges
5. `LoadDefault()` - 30 edges
6. `handleUpgradeActions()` - 28 edges
7. `doSync()` - 28 edges
8. `model` - 24 edges
9. `runAuditFix()` - 23 edges
10. `actions` - 20 edges

## Surprising Connections (you probably didn't know these)
- `outdatedAction` --embeds--> `ActionRef`  [EXTRACTED]
  cmd/root.go → internal/parser/parser.go
- `auditFix` --references--> `ActionRef`  [EXTRACTED]
  cmd/audit.go → internal/parser/parser.go
- `runAuditFix()` --calls--> `LoadRegistry()`  [EXTRACTED]
  cmd/audit.go → internal/breakingchanges/registry.go
- `runAuditFix()` --calls--> `ApplyAllUpgrades()`  [EXTRACTED]
  cmd/audit.go → internal/upgrader/upgrader.go
- `runAuditFix()` --references--> `Finding`  [EXTRACTED]
  cmd/audit.go → internal/vulnscanner/vulnscanner.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **actup v0.9.1 Launch Readiness** — docs_plans_20261114launchplan_md, docs_plans_brainstormingideas_md, docs_plans_deepdiveanalysis_md [EXTRACTED 1.00]

## Communities (52 total, 9 thin omitted)

### Community 0 - "Vulnerability Audit & Auto-Fix"
Cohesion: 0.05
Nodes (100): actionSpec, normalizedFixed(), resolveFixTarget(), runAudit(), runAuditFix(), auditUnknownSite, auditVulnFinding, auditVulnFix (+92 more)

### Community 1 - "Audit CLI & Findings Formatting"
Cohesion: 0.07
Nodes (34): auditFindingsError(), auditMajor(), auditJSONFinding, auditJSONVuln, pluralY(), fixTargetTooOldError, runInit(), initOptions (+26 more)

### Community 2 - "HTTP & GitHub API Client"
Cohesion: 0.07
Nodes (42): boundedEvents(), buildIntervals(), compareVersions(), eventsAffect(), inInterval(), lastAffectedEnd(), lastKnownBoundEvent(), NewClient() (+34 more)

### Community 3 - "Terminal UI & Spinner Components"
Cohesion: 0.07
Nodes (28): Sync(), TestSync_UsesExecSeam(), allAtLatest(), stripANSI(), TestAllAtLatest(), TestAllAtLatestMixed(), TestAllAtLatestSHA(), TestAllAtLatestSHA_Predicate() (+20 more)

### Community 4 - "Vulnerability Version Matching"
Cohesion: 0.13
Nodes (37): auditSiteVersion(), auditUpgrades(), auditVersionCompare(), auditVersionSegment(), maxAuditVersion(), minFixedFor(), outputAuditJSON(), planAuditFixes() (+29 more)

### Community 5 - "Release Workflow & Token Config"
Cohesion: 0.06
Nodes (34): GH_TOKEN, GITHUB_TOKEN, HOMEBREW_TAP_GITHUB_TOKEN, TAG, jobs, release, name, on (+26 more)

### Community 6 - "Git Pre-Commit Hook Installer"
Cohesion: 0.10
Nodes (28): hookFilePath(), installHook(), runInstallHooks(), TestInstallHooks_CreatesHook(), TestInstallHooks_DryRun(), TestInstallHooks_ExistingHookForce(), TestInstallHooks_ExistingHookNoForce(), TestInstallHooks_Uninstall() (+20 more)

### Community 7 - "Breaking Changes Warning System"
Cohesion: 0.12
Nodes (29): Entry, breakingChangesFor(), confirmBreaking(), confirmBreakingAll(), dedupeInputs(), mcpBreakingChangesFor(), TestConfirmBreakingAll_UnionInputsFromMultipleSites(), TestConfirmBreakingAllMixedVersions() (+21 more)

### Community 8 - "Workflow File Reference Upgrader"
Cohesion: 0.12
Nodes (30): TestPrintUseDiffRendersSHAComment(), printUseDiff(), ApplyAllUpgrades(), Upgrade, replaceAllInFile(), replaceInFile(), showDryRunDiff(), captureStdout() (+22 more)

### Community 9 - "CLI Root & Filter Execution"
Cohesion: 0.11
Nodes (26): init(), isGHExtensionInvocation(), resetFilterTestState(), TestFilterActionsByPatterns(), TestFilterSkippedActions(), TestGroupFlagUnknown(), TestGroupFlagWithoutManifest(), TestGroupPlusPositionalUnion() (+18 more)

### Community 10 - "Homebrew Tap Automation"
Cohesion: 0.08
Nodes (25): SHA, TAG, TAP_TOKEN, jobs, update-tap, name, on, push (+17 more)

### Community 11 - "Actup Manifest Action Declarations"
Cohesion: 0.08
Nodes (24): actions, actions/attest, actions/checkout, actions/dependency-review-action, actions/setup-go, actions/upload-artifact, anchore/sbom-action, aquasecurity/setup-trivy (+16 more)

### Community 12 - "YAML Manifest Configuration Parser"
Cohesion: 0.15
Nodes (22): formatScalar(), Load(), Render(), Save(), TestEffectivePins_LegacyFallsBackToActions(), TestEffectivePins_ManagedPrefersPins(), TestLoad_InvalidYAML(), TestLoad_MissingFile() (+14 more)

### Community 13 - "Workflow Uses Parser"
Cohesion: 0.17
Nodes (23): DistinctVersions(), ExtractActions(), parseActionRef(), assertInputs(), inputWorkflow(), TestDistinctVersions(), TestExtractActions(), TestExtractActionsConcurrent() (+15 more)

### Community 14 - "Action Pinning & Version Sync"
Cohesion: 0.19
Nodes (21): parseActionSpec(), runUnpin(), captureStdout(), TestDoSync_AppliesPins(), TestDoSync_ExactRefsNeedNoToken(), TestDoUse_AlreadyAtTarget(), TestDoUse_DryRunLeavesEverythingUntouched(), TestDoUse_NoSaveSkipsConfig() (+13 more)

### Community 15 - "GitHub Tag & SHA Resolver"
Cohesion: 0.14
Nodes (21): TestMCPCheckOutdated(), NewClientWithBaseURL(), resolveLatestTag(), resolvePin(), tagMajor(), TestFetchAllSemverTagsRetainsSHA(), TestIsMajorTag(), TestLatestTagInfoCachesTags() (+13 more)

### Community 16 - "Workflow Step Action Inputs"
Cohesion: 0.10
Nodes (21): with, advanced-security, annotations, args, cache, category, comment-summary-in-pr, dockerfile (+13 more)

### Community 17 - "MCP Server Integration Tests"
Cohesion: 0.14
Nodes (8): assertWorkflowContains(), assertWorkflowNotContains(), TestMCPAuditVulnerabilities(), TestMCPInitConfigUnparseable(), TestMCPListActions(), TestMCPUpgradeActions(), TestMCPUpgradeActionsApplyError(), nopCloseWriter

### Community 18 - "Pre-Commit Configuration Schema"
Cohesion: 0.12
Nodes (15): default_install_hook_types, default_stages, always_run, description, entry, id, language, name (+7 more)

### Community 19 - "CI Workflow Execution Steps"
Cohesion: 0.24
Nodes (15): steps, steps, steps, steps, workflow-security, steps, steps, if (+7 more)

### Community 20 - "Workflow Job Runners & Conditions"
Cohesion: 0.14
Nodes (14): name, runs-on, if, name, runs-on, jobs, action-audit, dependency-review (+6 more)

### Community 21 - "Commit SHA Pinning Tests"
Cohesion: 0.36
Nodes (12): loadRefs(), readFile(), TestApplyUseToFileRefs_MixedForceSHA(), TestApplyUseToFileRefs_MixedPerSiteStyle(), TestApplyUseToFileRefs_SHASameTargetSkipped(), TestApplyUseToFileRefs_SHAStaysSHA(), TestApplyUseToFileRefs_TagConvertsToSHA(), TestMostCommonVersionSkipsBareSHA() (+4 more)

### Community 22 - "GoReleaser Packaging Configuration"
Cohesion: 0.17
Nodes (11): before, hooks, checksum, algorithm, name_template, metadata, mod_timestamp, project_name (+3 more)

### Community 23 - "Version Flag & Build Info"
Cohesion: 0.27
Nodes (7): FormatVersion(), init(), SetVersionInfo(), TestRootVersionFlag(), TestVersionOutput(), TestVersionShortOutput(), TestVersionSubcommandRegistered()

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
Cohesion: 0.33
Nodes (6): name, owner, release, draft, github, prerelease

### Community 38 - "MCP Server Editor Configuration"
Cohesion: 0.33
Nodes (5): go, gopls, actup, gopls, producthunt

### Community 39 - "Model Context Protocol Server"
Cohesion: 0.40
Nodes (3): newMCPServer(), runMCPTransport(), TestMCPRunToleratesClientDisconnect()

### Community 40 - "Static Analysis & Linter Workflow"
Cohesion: 0.40
Nodes (5): GO111MODULE, static-code-analysis, env, name, runs-on

### Community 41 - "Vulnerability Check Job"
Cohesion: 0.50
Nodes (4): vulnerability-check, name, runs-on, steps

### Community 42 - "Cosign Docker Signing Config"
Cohesion: 0.50
Nodes (4): docker_signs, args, cmd, output

## Knowledge Gaps
- **12 isolated node(s):** `go`, `gopls`, `github.com/lynicis/actup`, `Changelog`, `Security Policy` (+7 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 254 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **9 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `ActionRef` connect `Vulnerability Audit & Auto-Fix` to `Audit CLI & Findings Formatting`, `Terminal UI & Spinner Components`, `Vulnerability Version Matching`, `Breaking Changes Warning System`, `Workflow File Reference Upgrader`, `Workflow Uses Parser`, `Commit SHA Pinning Tests`?**
  _High betweenness centrality (0.019) - this node is a cross-community bridge._
- **Are the 21 inferred relationships involving `writeWorkflow()` (e.g. with `TestMCPAuditVulnerabilities()` and `TestMCPCheckOutdated()`) actually correct?**
  _`writeWorkflow()` has 21 INFERRED edges - model-reasoned connections that need verification._
- **What connects `go`, `gopls`, `github.com/lynicis/actup` to the rest of the system?**
  _12 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `Vulnerability Audit & Auto-Fix` be split into smaller, more focused modules?**
  _Cohesion score 0.0547673531655225 - nodes in this community are weakly interconnected._
- **Why does `model` connect `Terminal UI & Spinner Components` to `Vulnerability Audit & Auto-Fix`, `Audit CLI & Findings Formatting`?**
  _High betweenness centrality (0.015) - this node is a cross-community bridge._
- **Should `Audit CLI & Findings Formatting` be split into smaller, more focused modules?**
  _Cohesion score 0.0686641697877653 - nodes in this community are weakly interconnected._
- **Why does `doSync()` connect `Vulnerability Audit & Auto-Fix` to `Audit CLI & Findings Formatting`, `Breaking Changes Warning System`, `Workflow File Reference Upgrader`, `Action Pinning & Version Sync`, `Commit SHA Pinning Tests`?**
  _High betweenness centrality (0.005) - this node is a cross-community bridge._
