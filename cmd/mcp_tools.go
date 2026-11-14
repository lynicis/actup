package cmd

import (
	"context"
	"fmt"
	"maps"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/lynicis/actup/internal/breakingchanges"
	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/github"
	"github.com/lynicis/actup/internal/lockfile"
	"github.com/lynicis/actup/internal/parser"
	"github.com/lynicis/actup/internal/upgrader"
	"github.com/lynicis/actup/internal/vulnscanner"
)

// mcpResult wraps a short human summary; the typed Out value travels separately
// as StructuredContent.
func mcpResult(summary string) *mcp.CallToolResult {
	return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: summary}}}
}

// mcpLoadConfig loads .actup.yaml, degrading to nil with a stderr warning.
func mcpLoadConfig() *config.Config {
	cfg, err := config.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to load config: %v\n", err)
	}
	return cfg
}

// mcpLoadLockfile loads actions.lock, degrading to nil with a stderr warning.
func mcpLoadLockfile(paths []string) *lockfile.Lockfile {
	lf, err := lockfile.LoadForPaths(paths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to read actions.lock: %v\n", err)
	}
	return lf
}

type listActionsIn struct {
	// Paths mirror the -p flag; empty discovers from the repository root.
	Paths []string `json:"paths,omitempty"`
}

type listActionsOut struct {
	Actions    []listActionsAction `json:"actions"`
	TotalSites int                 `json:"total_sites"`
}

type listActionsAction struct {
	Key        string               `json:"key"`
	Pin        string               `json:"pin,omitempty"`
	LockStatus string               `json:"lock_status"`
	Versions   []listActionsVersion `json:"versions"`
	Sites      []listActionsSite    `json:"sites"`
}

type listActionsVersion struct {
	Version    string `json:"version"`
	Count      int    `json:"count"`
	Pinned     bool   `json:"pinned,omitempty"`
	TagComment string `json:"tag_comment,omitempty"`
}

type listActionsSite struct {
	File       string `json:"file"`
	Line       int    `json:"line"`
	Current    string `json:"current"`
	SHA        string `json:"sha,omitempty"`
	TagComment string `json:"tag_comment,omitempty"`
}

func handleListActions(ctx context.Context, req *mcp.CallToolRequest, in listActionsIn) (*mcp.CallToolResult, listActionsOut, error) {
	actions, err := loadWorkflowActions(ctx, in.Paths)
	if err != nil {
		return nil, listActionsOut{}, err
	}
	grouped := parser.GroupActions(actions)
	pins := mcpLoadConfig().EffectivePins()
	lf := mcpLoadLockfile(in.Paths)

	out := listActionsOut{Actions: make([]listActionsAction, 0, len(grouped)), TotalSites: len(actions)}
	for _, key := range slices.Sorted(maps.Keys(grouped)) {
		refs := grouped[key]
		entry := listActionsAction{
			Key:        key,
			Pin:        pins[key],
			LockStatus: lockStatusString(lf.StatusForAction(key, refs)),
		}

		versionIdx := map[string]int{}
		for _, ref := range refs {
			v := ref.Version()
			idx, ok := versionIdx[v]
			if !ok {
				idx = len(entry.Versions)
				versionIdx[v] = idx
				entry.Versions = append(entry.Versions, listActionsVersion{Version: v, TagComment: ref.Tag})
			}
			entry.Versions[idx].Count++
			if ref.Pinned() {
				entry.Versions[idx].Pinned = true
			}
		}
		slices.SortStableFunc(entry.Versions, func(a, b listActionsVersion) int {
			return strings.Compare(a.Version, b.Version)
		})

		sites := append([]parser.ActionRef(nil), refs...)
		sort.Slice(sites, func(i, j int) bool {
			if sites[i].File != sites[j].File {
				return sites[i].File < sites[j].File
			}
			return sites[i].Line < sites[j].Line
		})
		for _, ref := range sites {
			entry.Sites = append(entry.Sites, listActionsSite{
				File:       ref.File,
				Line:       ref.Line,
				Current:    ref.Current,
				SHA:        ref.SHA,
				TagComment: ref.Tag,
			})
		}

		out.Actions = append(out.Actions, entry)
	}

	return mcpResult(fmt.Sprintf("%d action(s) across %d usage site(s)", len(out.Actions), out.TotalSites)), out, nil
}

type checkOutdatedIn struct {
	// Paths mirror the -p flag; empty discovers from the repository root.
	Paths []string `json:"paths,omitempty"`
	// Actions are optional path.Match globs over owner/repo keys.
	Actions []string `json:"actions,omitempty"`
	Semver  bool     `json:"semver,omitempty"`
	Major   int      `json:"major,omitempty"`
}

type checkOutdatedOut struct {
	Results []checkOutdatedResult `json:"results"`
}

type checkOutdatedResult struct {
	Key             string              `json:"key"`
	Current         string              `json:"current,omitempty"`
	Latest          string              `json:"latest,omitempty"`
	Pinned          string              `json:"pinned,omitempty"`
	Status          string              `json:"status"`
	ResolveError    string              `json:"resolve_error,omitempty"`
	BreakingChanges []mcpBreakingChange `json:"breaking_changes,omitempty"`
}

type mcpBreakingChange struct {
	Type        string `json:"type"`
	Input       string `json:"input,omitempty"`
	Replacement string `json:"replacement,omitempty"`
	Context     string `json:"context,omitempty"`
	Message     string `json:"message"`
}

func handleCheckOutdated(ctx context.Context, req *mcp.CallToolRequest, in checkOutdatedIn) (*mcp.CallToolResult, checkOutdatedOut, error) {
	actions, err := loadWorkflowActions(ctx, in.Paths)
	if err != nil {
		return nil, checkOutdatedOut{}, err
	}
	cfg := mcpLoadConfig()
	actions, err = applyActionFilters(actions, in.Actions, nil, cfg)
	if err != nil {
		return nil, checkOutdatedOut{}, err
	}
	actions = filterSkippedActions(actions, cfg)
	grouped := parser.GroupActions(actions)

	keys := slices.Sorted(maps.Keys(grouped))
	client := mcpNewGHClient(resolveToken(""))
	results := github.ResolveAll(ctx, client, keys, in.Semver, in.Major, cfg.EffectivePins())

	registry, rerr := breakingchanges.LoadRegistry()
	if rerr != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to load breaking-change registry: %v\n", rerr)
	}

	pins := cfg.EffectivePins()
	out := checkOutdatedOut{Results: make([]checkOutdatedResult, 0, len(grouped))}
	outdated := 0
	for _, key := range keys {
		refs := grouped[key]
		res := checkOutdatedResult{Key: key, Pinned: pins[key]}

		r := results[key]
		if r.Err != nil {
			res.Status = "error"
			res.ResolveError = r.Err.Error()
			out.Results = append(out.Results, res)
			continue
		}
		res.Latest = r.Latest

		distinct := map[string]bool{}
		for _, ref := range refs {
			if v := ref.Version(); v != "" {
				distinct[v] = true
			}
		}
		if len(distinct) == 0 {
			// Every site is a SHA pin without a version comment.
			res.Status = "unknown_version"
			out.Results = append(out.Results, res)
			continue
		}
		versions := slices.Sorted(maps.Keys(distinct))
		res.Current = strings.Join(versions, ", ")
		for _, v := range versions {
			if v != r.Latest {
				res.Status = "outdated"
				break
			}
		}
		if res.Status == "" {
			res.Status = "up_to_date"
		} else {
			outdated++
			res.BreakingChanges = mcpBreakingChangesFor(registry, key, refs, r.Latest)
		}
		out.Results = append(out.Results, res)
	}

	return mcpResult(fmt.Sprintf("%d of %d action(s) outdated", outdated, len(out.Results))), out, nil
}

// mcpBreakingChangesFor unions the registry entries across the distinct
// current versions of an action, deduplicated. Mirrors the version walk in
// confirmBreakingAll (common.go) without the prompting.
func mcpBreakingChangesFor(registry *breakingchanges.Registry, key string, refs []parser.ActionRef, target string) []mcpBreakingChange {
	inputsByVersion := map[string][]string{}
	var versions []string
	for _, ref := range refs {
		v := ref.Version()
		if v == "" {
			continue // unknown version cannot be checked
		}
		if _, ok := inputsByVersion[v]; !ok {
			versions = append(versions, v)
		}
		inputsByVersion[v] = append(inputsByVersion[v], ref.Inputs...)
	}
	slices.Sort(versions)

	seen := map[mcpBreakingChange]struct{}{}
	var out []mcpBreakingChange
	for _, v := range versions {
		for _, bc := range breakingChangesFor(registry, key, v, target, dedupeInputs(inputsByVersion[v])) {
			entry := mcpBreakingChange{Type: bc.Type, Input: bc.Input, Replacement: bc.Replacement, Context: bc.Context, Message: bc.Message}
			if _, dup := seen[entry]; dup {
				continue
			}
			seen[entry] = struct{}{}
			out = append(out, entry)
		}
	}
	return out
}

type auditVulnsIn struct {
	// Paths mirror the -p flag; empty discovers from the repository root.
	Paths []string `json:"paths,omitempty"`
}

type auditVulnsOut struct {
	OsvUnreachable bool               `json:"osv_unreachable,omitempty"`
	Findings       []auditVulnFinding `json:"findings"`
	UnknownVersion []auditUnknownSite `json:"unknown_version,omitempty"`
	SuggestedFixes []auditVulnFix     `json:"suggested_fixes,omitempty"`
	Unfixable      []auditVulnFinding `json:"unfixable,omitempty"`
}

type auditVulnFinding struct {
	Action     string          `json:"action"`
	Ref        string          `json:"ref"`
	File       string          `json:"file"`
	Line       int             `json:"line"`
	Transitive bool            `json:"transitive,omitempty"`
	Vulns      []auditVulnVuln `json:"vulns"`
}

type auditVulnVuln struct {
	ID       string `json:"id"`
	Severity string `json:"severity,omitempty"`
	Summary  string `json:"summary,omitempty"`
	Fixed    string `json:"fixed,omitempty"`
	URL      string `json:"url,omitempty"`
}

type auditUnknownSite struct {
	Action string `json:"action"`
	File   string `json:"file"`
	Line   int    `json:"line"`
	Ref    string `json:"ref"`
}

type auditVulnFix struct {
	Action   string `json:"action"`
	Current  string `json:"current"`
	MinFixed string `json:"min_fixed"`
}

func handleAuditVulnerabilities(ctx context.Context, req *mcp.CallToolRequest, in auditVulnsIn) (*mcp.CallToolResult, auditVulnsOut, error) {
	cfg := mcpLoadConfig()
	actions, err := loadWorkflowActions(ctx, in.Paths)
	if err != nil {
		return nil, auditVulnsOut{}, err
	}
	actions = filterSkippedActions(actions, cfg)
	direct := actions
	if lf := mcpLoadLockfile(in.Paths); lf != nil {
		actions = append(actions, filterSkippedActions(lf.TransitiveRefs(direct), cfg)...)
	}

	out := auditVulnsOut{Findings: []auditVulnFinding{}}

	// A SHA pin without a version comment cannot be matched against OSV
	// advisory ranges; report it explicitly rather than implying it is safe.
	var checkable []parser.ActionRef
	for _, a := range actions {
		if a.Pinned() && a.Version() == "" {
			out.UnknownVersion = append(out.UnknownVersion, auditUnknownSite{
				Action: a.Owner + "/" + a.Repo,
				File:   a.File,
				Line:   a.Line,
				Ref:    a.Current,
			})
			continue
		}
		checkable = append(checkable, a)
	}

	if len(checkable) == 0 {
		return mcpResult("no checkable action versions"), out, nil
	}

	findings, err := mcpVulnScan(ctx, checkable)
	if err != nil {
		// Non-fatal: mirror the CLI's audit distinction between findings
		// and an unreachable OSV.
		out.OsvUnreachable = true
		return mcpResult("OSV.dev unreachable; vulnerability status unknown"), out, nil
	}
	for _, f := range findings {
		out.Findings = append(out.Findings, toAuditVulnFinding(f))
	}

	// Transitive lockfile refs (Line == 0) are reported but never handed to
	// the fix planner: their source is actions.lock, not a workflow file.
	var directFindings []vulnscanner.Finding
	for _, f := range findings {
		if f.Line != 0 {
			directFindings = append(directFindings, f)
		}
	}
	directActions := make([]parser.ActionRef, 0, len(direct))
	for _, a := range direct {
		if a.Line > 0 {
			directActions = append(directActions, a)
		}
	}
	fixes, unfixable, _ := planAuditFixes(directFindings, directActions)
	for _, f := range fixes {
		out.SuggestedFixes = append(out.SuggestedFixes, auditVulnFix{Action: f.Key, Current: f.Current, MinFixed: f.MinFixed})
	}
	for _, f := range unfixable {
		out.Unfixable = append(out.Unfixable, toAuditVulnFinding(f))
	}

	summary := "no known vulnerable action versions"
	if len(out.Findings) > 0 {
		summary = fmt.Sprintf("%d vulnerable usage(s), %d suggested fix(es)", len(out.Findings), len(out.SuggestedFixes))
	}
	return mcpResult(summary), out, nil
}

func toAuditVulnFinding(f vulnscanner.Finding) auditVulnFinding {
	jf := auditVulnFinding{
		Action:     f.Owner + "/" + f.Repo,
		Ref:        f.Ref,
		File:       f.File,
		Line:       f.Line,
		Transitive: f.Line == 0,
		Vulns:      make([]auditVulnVuln, 0, len(f.Vulns)),
	}
	for _, v := range f.Vulns {
		jf.Vulns = append(jf.Vulns, auditVulnVuln{ID: v.ID, Severity: v.Severity, Summary: v.Summary, Fixed: v.Fixed, URL: v.URL})
	}
	return jf
}

// mcpClientForSpecs mirrors clientForSpecs (use.go) but builds the client
// through the mcpNewGHClient seam so tests can point it at an httptest server.
func mcpClientForSpecs(specs []actionSpec, needsAPI bool) *github.Client {
	for _, spec := range specs {
		if spec.Ref == "" || spec.Ref == "latest" {
			needsAPI = true
			break
		}
		if _, err := strconv.Atoi(spec.Ref); err == nil {
			needsAPI = true
			break
		}
	}
	if !needsAPI {
		return nil
	}
	return mcpNewGHClient(resolveToken(""))
}

type upgradeActionsIn struct {
	// Paths mirror the -p flag; empty discovers from the repository root.
	Paths []string `json:"paths,omitempty"`
	// Actions are owner/repo[@ref] specs; empty upgrades every discovered
	// action, honoring .actup.yaml pins.
	Actions []string `json:"actions,omitempty"`
	DryRun  bool     `json:"dry_run,omitempty"`
	Force   bool     `json:"force,omitempty"`
	Semver  bool     `json:"semver,omitempty"`
	Major   int      `json:"major,omitempty"`
	SHA     bool     `json:"sha,omitempty"`
}

type upgradeActionsOut struct {
	Results       []upgradeActionResult `json:"results"`
	FilesModified []string              `json:"files_modified,omitempty"`
	ConfigUpdated bool                  `json:"config_updated"`
	LockfileNote  string                `json:"lockfile_note,omitempty"`
}

type upgradeActionResult struct {
	Key             string              `json:"key"`
	Target          string              `json:"target,omitempty"`
	Status          string              `json:"status"`
	RefsUpdated     int                 `json:"refs_updated"`
	BreakingChanges []mcpBreakingChange `json:"breaking_changes,omitempty"`
	Error           string              `json:"error,omitempty"`
}

func handleUpgradeActions(ctx context.Context, req *mcp.CallToolRequest, in upgradeActionsIn) (*mcp.CallToolResult, upgradeActionsOut, error) {
	mcpWriteMu.Lock()
	defer mcpWriteMu.Unlock()

	actions, err := loadWorkflowActions(ctx, in.Paths)
	if err != nil {
		return nil, upgradeActionsOut{}, err
	}
	grouped := parser.GroupActions(actions)

	registry, rerr := breakingchanges.LoadRegistry()
	if rerr != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to load breaking-change registry: %v\n", rerr)
	}
	cfg := mcpLoadConfig()
	shaEffective := cfg != nil && cfg.SHA || in.SHA

	var specs []actionSpec
	for _, a := range in.Actions {
		spec, err := parseActionSpec(a)
		if err != nil {
			return nil, upgradeActionsOut{}, err
		}
		specs = append(specs, spec)
	}

	type upgradeTarget struct {
		key string
		ref string
		err error
	}
	var targets []upgradeTarget
	var client *github.Client
	specKeys := map[string]bool{}
	if len(specs) == 0 {
		// All mode: honor "skip" pins (mirrors runOutdated, runAudit, and
		// the bare `actup` upgrade flow) and resolve pin-honoring latest.
		actions = filterSkippedActions(actions, cfg)
		grouped = parser.GroupActions(actions)
		client = mcpNewGHClient(resolveToken(""))
		keys := slices.Sorted(maps.Keys(grouped))
		results := github.ResolveAll(ctx, client, keys, in.Semver, in.Major, cfg.EffectivePins())
		for _, key := range keys {
			r := results[key]
			t := upgradeTarget{key: key, ref: r.Latest}
			if r.Err != nil {
				t.err = r.Err
			}
			targets = append(targets, t)
		}
	} else {
		client = mcpClientForSpecs(specs, shaEffective || anyRefPinned(actions))
		seen := map[string]bool{}
		for _, spec := range specs {
			if seen[spec.Key] {
				continue
			}
			seen[spec.Key] = true
			specKeys[spec.Key] = true
			t := upgradeTarget{key: spec.Key}
			t.ref, t.err = resolveSpecRef(ctx, client, spec, in.Semver)
			targets = append(targets, t)
		}
	}
	files := map[string]bool{}
	upgraded := map[string]string{}
	planned := 0
	applied := 0
	out := upgradeActionsOut{Results: make([]upgradeActionResult, 0, len(targets))}
	for _, t := range targets {
		res := upgradeActionResult{Key: t.key, Target: t.ref}
		if t.err != nil {
			res.Status = "error"
			res.Error = t.err.Error()
			out.Results = append(out.Results, res)
			continue
		}

		refs := grouped[t.key]
		if len(refs) == 0 {
			res.Status = "not_used"
			out.Results = append(out.Results, res)
			continue
		}

		if isCommitSHA(t.ref) {
			res.Status = "skipped_sha"
			out.Results = append(out.Results, res)
			continue
		}

		resolveSHA := shaEffective || anyRefPinned(refs)
		newSHA := ""
		if resolveSHA {
			owner, repo := splitActionKey(t.key)
			newSHA, err = client.ResolveTagSHA(ctx, owner, repo, t.ref)
			if err != nil {
				res.Status = "error"
				res.Error = err.Error()
				out.Results = append(out.Results, res)
				continue
			}
		}

		// Gates: breaking changes and vulnerable targets skip the action
		// unless dry_run or force. Deliberately stricter than the
		// non-interactive CLI, which proceeds on vulnerable targets.
		res.BreakingChanges = mcpBreakingChangesFor(registry, t.key, refs, t.ref)
		if !in.DryRun && !in.Force {
			if len(res.BreakingChanges) > 0 {
				res.Status = "skipped_breaking"
				out.Results = append(out.Results, res)
				continue
			}
			owner, repo := splitActionKey(t.key)
			if findings, serr := mcpVulnScan(ctx, []parser.ActionRef{{Owner: owner, Repo: repo, Current: t.ref}}); serr == nil && len(findings) > 0 {
				res.Status = "skipped_vulnerable"
				out.Results = append(out.Results, res)
				continue
			}
		}

		// Skip already-current refs; mirrors applyUseToFileRefs (use.go).
		var upgrades []upgrader.Upgrade
		for _, ref := range refs {
			writeSHA := shaEffective || ref.Pinned()
			if writeSHA {
				if ref.SHA == newSHA && ref.Tag == t.ref {
					continue
				}
			} else if ref.Version() == t.ref {
				continue
			}
			refSHA := ""
			if writeSHA {
				refSHA = newSHA
			}
			upgrades = append(upgrades, upgrader.Upgrade{Action: ref, NewTag: t.ref, NewSHA: refSHA})
		}
		if len(upgrades) == 0 {
			res.Status = "already_current"
			out.Results = append(out.Results, res)
			continue
		}
		if in.DryRun {
			// Planned only; never call ApplyAllUpgrades here — its dry-run
			// branch prints to stdout and would corrupt the protocol.
			res.Status = "upgraded"
			res.RefsUpdated = len(upgrades)
			planned++
			out.Results = append(out.Results, res)
			continue
		}

		if err := upgrader.ApplyAllUpgrades(upgrades, false); err != nil {
			res.Status = "error"
			res.Error = err.Error()
			out.Results = append(out.Results, res)
			continue
		}
		res.Status = "upgraded"
		res.RefsUpdated = len(upgrades)
		applied++
		for _, u := range upgrades {
			files[u.Action.File] = true
		}
		upgraded[t.key] = t.ref
		out.Results = append(out.Results, res)
	}

	if !in.DryRun && len(upgraded) > 0 {
		if cfg != nil && cfg.IsManaged() {
			// Mirror `actup use`: an explicit target moves the pin with it,
			// keeping the manifest consistent for later CLI runs.
			for key, ref := range upgraded {
				if specKeys[key] {
					cfg.Pins[key] = ref
				}
			}
		}
		if err := recordUpgrades(cfg, upgraded); err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Failed to update .actup.yaml: %v\n", err)
		} else {
			out.ConfigUpdated = cfg != nil && cfg.IsManaged()
		}
	}
	if lf := mcpLoadLockfile(in.Paths); lf != nil {
		out.LockfileNote = "actions.lock exists; run gh actions-lock or actup sync to refresh it"
	}
	out.FilesModified = slices.Sorted(maps.Keys(files))

	summary := fmt.Sprintf("%d action(s) upgraded, %d file(s) modified", applied, len(out.FilesModified))
	if in.DryRun {
		summary = fmt.Sprintf("dry run: %d action(s) would change; no files were modified", planned)
	}
	return mcpResult(summary), out, nil
}

type pinActionIn struct {
	Action string   `json:"action"`
	Paths  []string `json:"paths,omitempty"`
	DryRun bool     `json:"dry_run,omitempty"`
}

type pinActionOut struct {
	Action string `json:"action"`
	Ref    string `json:"ref,omitempty"`
	Pinned bool   `json:"pinned"`
	Note   string `json:"note,omitempty"`
}

func handlePinAction(ctx context.Context, req *mcp.CallToolRequest, in pinActionIn) (*mcp.CallToolResult, pinActionOut, error) {
	mcpWriteMu.Lock()
	defer mcpWriteMu.Unlock()

	spec, err := parseActionSpec(in.Action)
	if err != nil {
		return nil, pinActionOut{}, err
	}

	ref := spec.Ref
	if ref == "" {
		actions, err := loadWorkflowActions(ctx, in.Paths)
		if err != nil {
			return nil, pinActionOut{}, fmt.Errorf("cannot determine current version; specify one explicitly (%s@<ref>): %w", spec.Key, err)
		}
		refs := parser.GroupActions(actions)[spec.Key]
		if len(refs) == 0 {
			return nil, pinActionOut{}, fmt.Errorf("%s is not used in any workflow; specify a version explicitly: %s@<ref>", spec.Key, spec.Key)
		}
		ref = mostCommonVersion(refs)
		if ref == "" {
			return mcpResult(spec.Key + " is SHA-pinned without a version comment"),
				pinActionOut{Action: spec.Key, Note: "SHA-pinned without a version comment; specify a ref explicitly"}, nil
		}
	}

	if in.DryRun {
		return mcpResult(fmt.Sprintf("dry run: would pin %s: %s in .actup.yaml", spec.Key, ref)),
			pinActionOut{Action: spec.Key, Ref: ref, Note: "dry run: no files were modified"}, nil
	}

	if err := savePin(spec.Key, ref); err != nil {
		return nil, pinActionOut{}, err
	}
	return mcpResult(fmt.Sprintf("pinned %s: %s in .actup.yaml", spec.Key, ref)),
		pinActionOut{Action: spec.Key, Ref: ref, Pinned: true}, nil
}

type unpinActionIn struct {
	Action string `json:"action"`
	DryRun bool   `json:"dry_run,omitempty"`
}

type unpinActionOut struct {
	Action   string `json:"action"`
	Unpinned bool   `json:"unpinned"`
	Note     string `json:"note,omitempty"`
}

func handleUnpinAction(ctx context.Context, req *mcp.CallToolRequest, in unpinActionIn) (*mcp.CallToolResult, unpinActionOut, error) {
	mcpWriteMu.Lock()
	defer mcpWriteMu.Unlock()

	spec, err := parseActionSpec(in.Action)
	if err != nil {
		return nil, unpinActionOut{}, err
	}

	cfg, err := config.LoadDefault()
	if err != nil {
		return nil, unpinActionOut{}, fmt.Errorf("load config: %w", err)
	}
	if cfg == nil || cfg.EffectivePins()[spec.Key] == "" {
		return mcpResult(spec.Key + " is not pinned"),
			unpinActionOut{Action: spec.Key, Note: "not pinned"}, nil
	}

	if in.DryRun {
		return mcpResult("dry run: would unpin " + spec.Key),
			unpinActionOut{Action: spec.Key, Note: "dry run: no files were modified"}, nil
	}

	delete(cfg.Actions, spec.Key)
	delete(cfg.Pins, spec.Key)
	if err := config.SaveDefault(cfg); err != nil {
		return nil, unpinActionOut{}, err
	}
	return mcpResult("unpinned " + spec.Key), unpinActionOut{Action: spec.Key, Unpinned: true}, nil
}

type initConfigIn struct {
	Paths  []string `json:"paths,omitempty"`
	Force  bool     `json:"force,omitempty"`
	DryRun bool     `json:"dry_run,omitempty"`
}

type initConfigOut struct {
	Written    bool     `json:"written"`
	Path       string   `json:"path,omitempty"`
	Added      int      `json:"added"`
	Skipped    []string `json:"skipped,omitempty"`
	ConfigYAML string   `json:"config_yaml"`
}

func handleInitConfig(ctx context.Context, req *mcp.CallToolRequest, in initConfigIn) (*mcp.CallToolResult, initConfigOut, error) {
	mcpWriteMu.Lock()
	defer mcpWriteMu.Unlock()

	actions, err := loadWorkflowActions(ctx, in.Paths)
	if err != nil {
		return nil, initConfigOut{}, err
	}

	existing, err := config.LoadDefault()
	if err != nil {
		if !in.Force && !in.DryRun {
			return nil, initConfigOut{}, fmt.Errorf(".actup.yaml exists but could not be parsed; fix or remove it, or use force: true: %w", err)
		}
		fmt.Fprintf(os.Stderr, "⚠ Failed to load existing config: %v\n", err)
	}
	if existing != nil && !in.Force && !in.DryRun {
		return nil, initConfigOut{}, fmt.Errorf(".actup.yaml already exists; use force: true to merge new pins into it")
	}
	cfg := existing

	if cfg == nil {
		cfg = &config.Config{Pins: map[string]string{}}
	} else if in.Force && cfg.Pins == nil {
		// One-way migration: the legacy actions: entries become recorded state.
		cfg.Pins = map[string]string{}
	}
	if cfg.Actions == nil {
		cfg.Actions = map[string]string{}
	}

	grouped := parser.GroupActions(actions)
	var out initConfigOut
	for _, key := range slices.Sorted(maps.Keys(grouped)) {
		if _, exists := cfg.Actions[key]; exists {
			continue
		}
		version := mostCommonVersion(grouped[key])
		if version == "" {
			out.Skipped = append(out.Skipped, key)
			continue
		}
		cfg.Actions[key] = version
		out.Added++
	}
	out.ConfigYAML = config.Render(cfg)

	if in.DryRun {
		return mcpResult("dry run: would write .actup.yaml"), out, nil
	}

	path := ".actup.yaml"
	if _, err := os.Stat(".actup.yaml"); err != nil {
		if _, err := os.Stat(".actup.yml"); err == nil {
			path = ".actup.yml"
		}
	}
	if err := config.SaveDefault(cfg); err != nil {
		return nil, initConfigOut{}, err
	}
	out.Written = true
	out.Path = path
	return mcpResult(fmt.Sprintf("wrote %s with %d recorded action(s)", path, out.Added)), out, nil
}
