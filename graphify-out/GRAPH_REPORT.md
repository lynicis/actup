# Graph Report - actup  (2026-11-14)

## Corpus Check
- 80 files · ~97,270 words
- Verdict: corpus is large enough that graph structure adds value.
- Unclassified: 7 file(s) not represented in the graph (top: (none) 6, .lock 1)

## Summary
- 1003 nodes · 2667 edges · 56 communities (50 shown, 6 thin omitted)
- Extraction: 85% EXTRACTED · 15% INFERRED · 0% AMBIGUOUS · INFERRED: 407 edges (avg confidence: 0.87)
- Token cost: 106,596 input · 4,724 output

## Graph Freshness
- Built from commit: `b0429d01`
- Run `git rev-parse HEAD` and compare to check if the graph is stale.
- Run `graphify update .` after code changes (no API cost).

## Community Hubs (Navigation)
- mcp_tools.go
- Audit CLI & Findings Formatting
- vulnscanner_test.go
- Terminal UI & Spinner Components
- Vulnerability Version Matching
- Release Workflow & Token Config
- Git Pre-Commit Hook Installer
- Breaking Changes Warning System
- Workflow File Reference Upgrader
- testing.T
- Homebrew Tap Automation
- Actup Manifest Action Declarations
- YAML Manifest Configuration Parser
- Workflow Uses Parser
- Action Pinning & Version Sync
- GitHub Tag & SHA Resolver
- Workflow Step Action Inputs
- ActionRef
- Pre-Commit Configuration Schema
- CI Workflow Execution Steps
- Workflow Job Runners & Conditions
- context.Context
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
- Cosign Docker Signing Config
- Documentation & Branding Assets
- Project Changelog History
- Release & Launch Strategy
- Product Marketing Context
- Config
- scanner_test.go
- Go Module Package Declaration
- Repository Security Policy
- Hyperframes Composition Brief: actup
- Design: GitHub Actions Lockfile (`actions.lock`) Integration & `gh actup` CLI Extension
- AGENTS.md
- isGHExtensionInvocation

## God Nodes (most connected - your core abstractions)
1. `ActionRef` - 54 edges
2. `writeWorkflow()` - 39 edges
3. `doUse()` - 33 edges
4. `doSync()` - 31 edges
5. `LoadDefault()` - 30 edges
6. `with` - 30 edges
7. `handleUpgradeActions()` - 28 edges
8. `runAuditFix()` - 27 edges
9. `model` - 24 edges
10. `ApplyAllUpgrades()` - 21 edges

## Surprising Connections (you probably didn't know these)
- `Packages` --references--> `NewClientWithBaseURL()`  [INFERRED]
  CLAUDE.md → internal/github/client.go
- `Non-obvious rules` --references--> `confirmBreaking()`  [INFERRED]
  CLAUDE.md → cmd/common.go
- `Non-obvious rules` --references--> `Save()`  [INFERRED]
  CLAUDE.md → internal/config/config.go
- `4.1 `internal/parser`: Skip `$/…` Local Action References` --references--> `parseActionRef()`  [INFERRED]
  docs/plans/2026-11-12-actions-lock-and-gh-extension-design.md → internal/parser/parser.go
- `4.3 Auto-Refresh `actions.lock` on Workflow Writes` --references--> `doUse()`  [INFERRED]
  docs/plans/2026-11-12-actions-lock-and-gh-extension-design.md → cmd/use.go

## Import Cycles
- None detected.

## Hyperedges (group relationships)
- **actup v0.9.1 Launch Readiness** — docs_plans_20261114launchplan_md, docs_plans_brainstormingideas_md, docs_plans_deepdiveanalysis_md [EXTRACTED 1.00]

## Communities (56 total, 6 thin omitted)

### Community 0 - "mcp_tools.go"
Cohesion: 0.10
Nodes (49): auditUnknownSite, auditVulnFinding, auditVulnFix, auditVulnsIn, auditVulnsOut, auditVulnVuln, checkOutdatedIn, checkOutdatedOut (+41 more)

### Community 1 - "Audit CLI & Findings Formatting"
Cohesion: 0.06
Nodes (37): auditJSONFinding, auditJSONVuln, fixTargetTooOldError, runInit(), newMCPServer(), runMCPTransport(), TestMCPRunToleratesClientDisconnect(), nopCloseWriter (+29 more)

### Community 2 - "vulnscanner_test.go"
Cohesion: 0.14
Nodes (23): multiIntervalVuln(), newMockOSV(), osvFromMap(), simpleVuln(), TestFixedFor_EarlierInterval(), TestFixedFor_MultiInterval(), TestFixedFor_NoFixedEvent(), TestFixedFor_VersionsOnly() (+15 more)

### Community 3 - "Terminal UI & Spinner Components"
Cohesion: 0.07
Nodes (31): allAtLatest(), allAtLatestMixed(), allAtLatestSHA(), anyRefPinned(), stripANSI(), TestAllAtLatest(), TestAllAtLatestMixed(), TestAllAtLatestSHA() (+23 more)

### Community 4 - "Vulnerability Version Matching"
Cohesion: 0.32
Nodes (19): auditTestRepo(), auditTransitiveFixture(), captureAuditStdout(), checkoutFinding(), resetAuditFixState(), runAuditCommand(), stubAuditScan(), stubTransitiveScan() (+11 more)

### Community 5 - "Release Workflow & Token Config"
Cohesion: 0.06
Nodes (34): GH_TOKEN, GITHUB_TOKEN, HOMEBREW_TAP_GITHUB_TOKEN, TAG, jobs, release, name, on (+26 more)

### Community 6 - "Git Pre-Commit Hook Installer"
Cohesion: 0.15
Nodes (16): hookFilePath(), installHook(), runInstallHooks(), TestInstallHooks_CreatesHook(), TestInstallHooks_DryRun(), TestInstallHooks_ExistingHookForce(), TestInstallHooks_ExistingHookNoForce(), TestInstallHooks_Uninstall() (+8 more)

### Community 7 - "Breaking Changes Warning System"
Cohesion: 0.12
Nodes (28): Entry, breakingChangesFor(), confirmBreaking(), confirmBreakingAll(), dedupeInputs(), TestConfirmBreakingAll_UnionInputsFromMultipleSites(), TestConfirmBreakingAllMixedVersions(), checkUseBreaking() (+20 more)

### Community 8 - "Workflow File Reference Upgrader"
Cohesion: 0.09
Nodes (42): loadRefs(), readFile(), TestApplyUseToFileRefs_MixedForceSHA(), TestApplyUseToFileRefs_MixedPerSiteStyle(), TestApplyUseToFileRefs_SHASameTargetSkipped(), TestApplyUseToFileRefs_SHAStaysSHA(), TestApplyUseToFileRefs_TagConvertsToSHA(), TestMostCommonVersionSkipsBareSHA() (+34 more)

### Community 9 - "testing.T"
Cohesion: 0.13
Nodes (24): TestAuditJSONOutputShape(), TestMinFixedFor(), TestPlanAuditFixes_DedupesAcrossFiles(), TestPlanAuditFixes_DifferentCurrents(), TestPlanAuditFixes_MixedVersionsFixesVulnerableSite(), TestPlanAuditFixes_UnfixableSkipsAllSites(), TestSavePins_NoManifestNoop(), filterActionsByPatterns() (+16 more)

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
Cohesion: 0.20
Nodes (20): DistinctVersions(), ExtractActions(), assertInputs(), inputWorkflow(), TestDistinctVersions(), TestExtractActions(), TestExtractActionsConcurrent(), TestExtractActionsMultilineSteps() (+12 more)

### Community 14 - "Action Pinning & Version Sync"
Cohesion: 0.13
Nodes (37): actionSpec, parseActionSpec(), recordUpgrades(), savePins(), doInit(), initOptions, runUnpin(), TestDoInitSkipsBareSHA() (+29 more)

### Community 15 - "GitHub Tag & SHA Resolver"
Cohesion: 0.13
Nodes (21): TestMCPCheckOutdated(), NewClientWithBaseURL(), resolveLatestTag(), resolvePin(), tagMajor(), TestFetchAllSemverTagsRetainsSHA(), TestIsMajorTag(), TestLatestTagInfoCachesTags() (+13 more)

### Community 16 - "Workflow Step Action Inputs"
Cohesion: 0.10
Nodes (21): with, advanced-security, annotations, args, cache, category, comment-summary-in-pr, dockerfile (+13 more)

### Community 17 - "ActionRef"
Cohesion: 0.06
Nodes (54): auditSiteVersion(), auditUpgrades(), auditVersionCompare(), auditVersionSegment(), maxAuditVersion(), minFixedFor(), planAuditFixes(), auditFix (+46 more)

### Community 18 - "Pre-Commit Configuration Schema"
Cohesion: 0.12
Nodes (16): default_install_hook_types, default_stages, always_run, args, description, entry, id, language (+8 more)

### Community 19 - "CI Workflow Execution Steps"
Cohesion: 0.24
Nodes (15): steps, steps, steps, steps, workflow-security, steps, steps, if (+7 more)

### Community 20 - "Workflow Job Runners & Conditions"
Cohesion: 0.14
Nodes (14): name, runs-on, if, name, runs-on, jobs, action-audit, dependency-review (+6 more)

### Community 21 - "context.Context"
Cohesion: 0.14
Nodes (25): auditMajor(), normalizedFixed(), resolveFixTarget(), runAuditFix(), anyRefPinned(), confirmVulnerableTarget(), isCommitSHA(), isTerminal() (+17 more)

### Community 22 - "GoReleaser Packaging Configuration"
Cohesion: 0.17
Nodes (11): before, hooks, checksum, algorithm, name_template, metadata, mod_timestamp, project_name (+3 more)

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
Cohesion: 0.33
Nodes (6): name, owner, release, draft, github, prerelease

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

### Community 42 - "Cosign Docker Signing Config"
Cohesion: 0.50
Nodes (4): docker_signs, args, cmd, output

### Community 47 - "Product Marketing Context"
Cohesion: 0.12
Nodes (15): Brand Voice, Changelog, Competitive Landscape, Customer Language, Differentiation, Goals, Learnings, Objections (+7 more)

### Community 48 - "Config"
Cohesion: 0.24
Nodes (13): Non-obvious rules, auditFindingsError(), outputAuditJSON(), runAudit(), applyActionFilters(), printVulnWarnings(), runOutdated(), filterSkippedActions() (+5 more)

### Community 49 - "scanner_test.go"
Cohesion: 0.31
Nodes (12): DiscoverWorkflows(), isWorkflowFile(), TestDiscoverWorkflows(), TestDiscoverWorkflows_Dedup(), TestDiscoverWorkflows_DefaultsIgnoreNonActionYAMLInActionsDir(), TestDiscoverWorkflows_DefaultsIncludeCompositeAndRootAction(), TestDiscoverWorkflows_DefaultsMissingActionsDir(), TestDiscoverWorkflows_ExplicitPathDoesNotAddDefaults() (+4 more)

### Community 52 - "Hyperframes Composition Brief: actup"
Cohesion: 0.20
Nodes (9): Audio, Creative Direction, Hyperframes Composition Brief: actup, Hyperframes Instructions, Objective, Output, Source Material, Storyboard (+1 more)

### Community 53 - "Design: GitHub Actions Lockfile (`actions.lock`) Integration & `gh actup` CLI Extension"
Cohesion: 0.25
Nodes (7): 1. Intent & Why, 2. Goals, Non-Goals, and Anti-Goals, 3. Approaches Considered, Anti-Goals, Design: GitHub Actions Lockfile (`actions.lock`) Integration & `gh actup` CLI Extension, Goals, Non-Goals

### Community 54 - "AGENTS.md"
Cohesion: 0.40
Nodes (4): AGENTS.md, Commands, External references, Packages

### Community 55 - "isGHExtensionInvocation"
Cohesion: 0.50
Nodes (4): init(), isGHExtensionInvocation(), TestIsGHExtensionInvocation(), Task 6: `gh` CLI Extension Support (`cmd/root.go`, `.goreleaser.yaml`, `README.md`, `AGENTS.md`)

## Knowledge Gaps
- **10 isolated node(s):** `go`, `gopls`, `github.com/lynicis/actup`, `actup — Brainstorming: Improvements & New Features`, `github-actions-updater Skill` (+5 more)
  These have ≤1 connection - possible missing edges or undocumented components. (Counts symbols only; 305 node(s) total have ≤1 connection when file, concept and rationale nodes are included.)
- **6 thin communities (<3 nodes) omitted from report** — run `graphify query` to explore isolated nodes.

## Suggested Questions
_Questions this graph is uniquely positioned to answer:_

- **Why does `ActionRef` connect `ActionRef` to `mcp_tools.go`, `Audit CLI & Findings Formatting`, `Terminal UI & Spinner Components`, `Breaking Changes Warning System`, `Workflow File Reference Upgrader`, `testing.T`, `Workflow Uses Parser`, `Config`, `context.Context`?**
  _High betweenness centrality (0.033) - this node is a cross-community bridge._
- **Are the 6 inferred relationships involving `ActionRef` (e.g. with `4.2 New Package: `internal/lockfile`` and `File Map`) actually correct?**
  _`ActionRef` has 6 INFERRED edges - model-reasoned connections that need verification._
- **What connects `go`, `gopls`, `github.com/lynicis/actup` to the rest of the system?**
  _10 weakly-connected nodes found - possible documentation gaps or missing edges._
- **Should `mcp_tools.go` be split into smaller, more focused modules?**
  _Cohesion score 0.09502262443438914 - nodes in this community are weakly interconnected._
- **Why does `4. Technical Specification` connect `ActionRef` to `Config`, `Design: GitHub Actions Lockfile (`actions.lock`) Integration & `gh actup` CLI Extension`, `context.Context`?**
  _High betweenness centrality (0.019) - this node is a cross-community bridge._
- **Are the 21 inferred relationships involving `writeWorkflow()` (e.g. with `TestMCPAuditVulnerabilities()` and `TestMCPCheckOutdated()`) actually correct?**
  _`writeWorkflow()` has 21 INFERRED edges - model-reasoned connections that need verification._
- **Should `Audit CLI & Findings Formatting` be split into smaller, more focused modules?**
  _Cohesion score 0.0567139282735613 - nodes in this community are weakly interconnected._
