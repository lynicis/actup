package cmd

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lynicis/actup/internal/breakingchanges"
	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/github"
	"github.com/lynicis/actup/internal/lockfile"
	"github.com/lynicis/actup/internal/parser"
	"github.com/lynicis/actup/internal/upgrader"
	"github.com/lynicis/actup/internal/vulnscanner"
)

// errAuditFindings signals that the audit left at least one finding. Execute
// maps it to exit status 1.
var errAuditFindings = errors.New("audit findings remain")

// errAuditUnavailable signals that the OSV scan could not run. Execute maps
// it to exit status 2.
var errAuditUnavailable = errors.New("audit scan unavailable")

var auditOpts struct {
	paths  []string
	json   bool
	fix    bool
	dryRun bool
	force  bool
	semver bool
	token  string
	sha    bool
}

// auditScan is a seam so tests can supply findings without OSV network access.
var auditScan = func(ctx context.Context, refs []parser.ActionRef) ([]vulnscanner.Finding, error) {
	return vulnscanner.NewClient().Scan(ctx, refs)
}

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Scan workflows for known vulnerable action versions (exits non-zero if found)",
	Long: `Checks every action reference in the workflow files against the OSV.dev
vulnerability database (GitHub Actions ecosystem advisories) and reports
usages of versions known to be vulnerable.

Findings include the advisory ID, severity, and the first fixed version
when known. Exits with status 1 when at least one vulnerable usage is
found, and 2 when the OSV database cannot be reached.

With --fix, actup upgrades every vulnerable action to a version that clears
all of its advisories and records the new versions in .actup.yaml (when one
already exists). Actions with an advisory that has no fixed version are
reported and left untouched.

Vulnerability data comes from OSV.dev; results require network access.
The scan skips actions listed as "skip" in .actup.yaml.`,
	Example: `  actup audit
  actup audit --json
  actup audit --fix
  actup audit --fix --dry-run
  actup audit --fix --semver --force
  actup audit --path .github/workflows`,
	RunE: runAudit,
}

func init() {
	auditCmd.Flags().StringArrayVarP(&auditOpts.paths, "path", "p", []string{}, "Paths to workflow files or directories")
	auditCmd.Flags().BoolVar(&auditOpts.json, "json", false, "Output findings as JSON")
	auditCmd.Flags().BoolVar(&auditOpts.fix, "fix", false, "Upgrade vulnerable actions to a fixed version")
	auditCmd.Flags().BoolVar(&auditOpts.dryRun, "dry-run", false, "Preview fixes without writing files or saving pins")
	auditCmd.Flags().BoolVarP(&auditOpts.force, "force", "f", false, "Apply fixes with breaking changes without prompting")
	auditCmd.Flags().BoolVarP(&auditOpts.semver, "semver", "s", false, "Fix to the exact fixed version instead of the latest tag in its major")
	auditCmd.Flags().StringVarP(&auditOpts.token, "token", "t", "", "GitHub PAT (fallback: GITHUB_TOKEN env var, then gh auth token)")
	auditCmd.Flags().BoolVar(&auditOpts.sha, "sha", false, "Write refs as full commit SHAs with the tag in a trailing comment")
	rootCmd.AddCommand(auditCmd)
}

// auditFix is one planned per-action upgrade.
type auditFix struct {
	Key      string
	Refs     []parser.ActionRef
	Current  string // max version among refs, for reporting
	MinFixed string // max of all applicable Fixed versions
	Target   string // tag to write
}

// auditJSONFinding is the JSON representation of one vulnerable usage site.
type auditJSONFinding struct {
	Action     string          `json:"action"`
	Ref        string          `json:"ref"`
	File       string          `json:"file"`
	Line       int             `json:"line"`
	Transitive bool            `json:"transitive,omitempty"`
	Vulns      []auditJSONVuln `json:"vulns"`
}

type auditJSONVuln struct {
	ID       string `json:"id"`
	Severity string `json:"severity,omitempty"`
	Summary  string `json:"summary,omitempty"`
	Fixed    string `json:"fixed,omitempty"`
	URL      string `json:"url,omitempty"`
}

func runAudit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if auditOpts.json && auditOpts.fix {
		return errors.New("--json cannot be combined with --fix")
	}

	cfg, err := config.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to load config: %v\n", err)
		cfg = nil
	}

	actions, err := loadWorkflowActions(ctx, auditOpts.paths)
	if err != nil {
		return err
	}
	actions = filterSkippedActions(actions, cfg)
	directActions := actions
	if lf, lerr := lockfile.LoadForPaths(auditOpts.paths); lerr != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to read actions.lock: %v\n", lerr)
	} else if lf != nil {
		actions = append(actions, filterSkippedActions(lf.TransitiveRefs(directActions), cfg)...)
	}
	if len(actions) == 0 {
		if auditOpts.json {
			fmt.Println(`{"findings":[]}`)
		} else {
			fmt.Println("No actions found.")
		}
		return nil
	}

	// A SHA pin without a version comment cannot be matched against OSV
	// advisory ranges; report it explicitly rather than implying it is safe.
	checkable := make([]parser.ActionRef, 0, len(actions))
	unknownKeys := map[string]bool{}
	var unknownOrder []string
	for _, a := range actions {
		if a.Pinned() && a.Version() == "" {
			key := a.Owner + "/" + a.Repo
			if !unknownKeys[key] {
				unknownKeys[key] = true
				unknownOrder = append(unknownOrder, key)
			}
			continue
		}
		checkable = append(checkable, a)
	}
	noticeOut := os.Stdout
	if auditOpts.json {
		noticeOut = os.Stderr // keep stdout machine-readable
	}
	for _, key := range unknownOrder {
		_, _ = fmt.Fprintf(noticeOut, "ⓘ %s: version unknown (SHA-pinned without a version comment); skipping vulnerability check\n", key)
	}

	if len(checkable) == 0 {
		if auditOpts.json {
			fmt.Println(`{"findings":[]}`)
		}
		return nil
	}

	findings, err := auditScan(ctx, checkable)
	if err != nil {
		if auditOpts.json {
			return fmt.Errorf("%w: %v", errAuditUnavailable, err)
		}
		fmt.Fprintf(os.Stderr, "⚠ Vulnerability scan failed (network unavailable?): %v\n", err)
		os.Exit(2)
	}

	if auditOpts.json {
		if err := outputAuditJSON(findings); err != nil {
			if errors.Is(err, errAuditFindings) {
				return auditFindingsError(cmd)
			}
			return err
		}
		return nil
	}

	if len(findings) == 0 {
		fmt.Println("No known vulnerable action versions found.")
		return nil
	}

	if auditOpts.fix {
		return runAuditFix(ctx, cmd, cfg, directActions, findings)
	}

	for _, f := range findings {
		action := f.Owner + "/" + f.Repo
		if f.Line == 0 {
			fmt.Printf("%s (transitive): %s@%s\n", f.File, action, f.Ref)
		} else {
			fmt.Printf("%s:%d: %s@%s\n", f.File, f.Line, action, f.Ref)
		}
		for _, v := range f.Vulns {
			severity := v.Severity
			if severity == "" {
				severity = "UNKNOWN"
			}
			fixed := ""
			if v.Fixed != "" {
				fixed = fmt.Sprintf(", fixed in %s", v.Fixed)
			}
			fmt.Printf("  ⚠ %s [%s]%s %s\n", v.ID, severity, fixed, v.Summary)
			if v.URL != "" {
				fmt.Printf("    %s\n", v.URL)
			}
		}
	}

	fmt.Printf("\n%d vulnerable action usage(s) found.\n", len(findings))
	fmt.Fprintln(os.Stderr, "Run 'actup audit --help' for options, or upgrade with 'actup use <action>@latest'.")
	os.Exit(1)
	return nil
}

// runAuditFix plans and applies upgrades for every actionable finding.
func runAuditFix(ctx context.Context, cmd *cobra.Command, cfg *config.Config, actions []parser.ActionRef, findings []vulnscanner.Finding) error {
	// Transitive lockfile refs (Line == 0) are reported but never handed to
	// the upgrader: their source is actions.lock, not a workflow file.
	var directFindings, transitiveFindings []vulnscanner.Finding
	for _, f := range findings {
		if f.Line == 0 {
			transitiveFindings = append(transitiveFindings, f)
			continue
		}
		directFindings = append(directFindings, f)
	}
	directActions := make([]parser.ActionRef, 0, len(actions))
	for _, a := range actions {
		if a.Line > 0 {
			directActions = append(directActions, a)
		}
	}

	fixes, unfixable, alreadyPast := planAuditFixes(directFindings, directActions)
	for _, f := range unfixable {
		fmt.Printf("✗ %s/%s: no fixed version known (%s:%d)\n", f.Owner, f.Repo, f.File, f.Line)
	}
	for _, f := range alreadyPast {
		fmt.Printf("⏭ %s: already past the reported fix %s\n", f.Key, f.MinFixed)
	}
	residual := len(unfixable)
	actionable := false

	shaMode := cfg != nil && cfg.SHA
	if cmd.Flags().Changed("sha") {
		shaMode = auditOpts.sha
	}

	var client *github.Client
	if !auditOpts.semver || shaMode || anyRefPinned(directActions) {
		client = github.NewClient(resolveToken(auditOpts.token))
	}
	registry, err := breakingchanges.LoadRegistry()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to load breaking-change registry: %v\n", err)
	}
	reader := bufio.NewReader(os.Stdin)
	isTTY := isTerminal()

	var pending []upgrader.Upgrade
	type appliedFix struct {
		key, target string
		n           int
	}
	var applied []appliedFix
	pins := map[string]string{}

	for _, f := range fixes {
		target, err := resolveFixTarget(ctx, client, f.Key, f.MinFixed, auditOpts.semver)
		if err != nil {
			var tooOld *fixTargetTooOldError
			if errors.As(err, &tooOld) {
				fmt.Printf("✗ %s: %v\n", f.Key, err)
				residual++
				continue
			}
			target = normalizedFixed(f.MinFixed)
			fmt.Printf("⚠ %s: using %s (GitHub API unavailable)\n", f.Key, target)
		}
		f.Target = target

		newSHA := ""
		if shaMode || anyRefPinned(f.Refs) {
			owner, repo := splitActionKey(f.Key)
			sha, err := client.ResolveTagSHA(ctx, owner, repo, target)
			if err != nil {
				fmt.Printf("✗ %s: %v\n", f.Key, err)
				residual++
				continue
			}
			newSHA = sha
		}

		proceed, _ := confirmBreakingAll(f.Key, f.Refs, target, registry, auditOpts.dryRun, auditOpts.force, reader, isTTY)
		if !proceed {
			residual++
			continue
		}

		upgrades := auditUpgrades(f, newSHA, shaMode)
		if len(upgrades) == 0 {
			fmt.Printf("⏭ %s: already at %s\n", f.Key, target)
			pins[f.Key] = target
			continue
		}
		actionable = true
		if auditOpts.dryRun {
			fmt.Printf("would update %s: %s → %s (%d refs)\n", f.Key, f.Current, target, len(upgrades))
			continue
		}
		pending = append(pending, upgrades...)
		applied = append(applied, appliedFix{key: f.Key, target: target, n: len(upgrades)})
		pins[f.Key] = target
	}

	if len(pending) > 0 {
		if err := upgrader.ApplyAllUpgrades(pending, false); err != nil {
			fmt.Printf("✗ %v\n", err)
			residual += len(applied)
			pins = map[string]string{}
			applied = nil
		}
	}
	for _, a := range applied {
		fmt.Printf("✓ %s → %s (%d refs)\n", a.key, a.target, a.n)
	}

	if !auditOpts.dryRun && len(pins) > 0 {
		if err := savePins(pins); err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Failed to save pins: %v\n", err)
		}
	}
	if !auditOpts.dryRun && len(applied) > 0 {
		syncLockfile(auditOpts.paths)
	}

	for _, tf := range transitiveFindings {
		fmt.Printf("✗ %s/%s@%s: transitive dependency in %s (upgrade parent action or run 'gh actions-lock --relock')\n", tf.Owner, tf.Repo, tf.Ref, tf.File)
		residual++
	}

	if auditOpts.dryRun {
		if actionable || residual > 0 {
			return auditFindingsError(cmd)
		}
		return nil
	}
	if residual > 0 {
		fmt.Printf("\n%d vulnerable action usage(s) remain.\n", residual)
		return auditFindingsError(cmd)
	}
	fmt.Println("\nAll vulnerable action usages fixed.")
	return nil
}

// auditFindingsError silences Cobra's error printing so the summary stays
// clean, then returns the sentinel that Execute maps to exit status 1.
func auditFindingsError(cmd *cobra.Command) error {
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	return errAuditFindings
}

// planAuditFixes groups findings per action and works out the target version.
// Keys with any advisory lacking a fixed version are returned as unfixable;
// keys already past the reported fix are returned as alreadyPast.
func planAuditFixes(findings []vulnscanner.Finding, actions []parser.ActionRef) (fixes []auditFix, unfixable []vulnscanner.Finding, alreadyPast []auditFix) {
	type group struct {
		key   string
		sites []vulnscanner.Finding
		vulns []vulnscanner.Vulnerability
	}
	groups := map[string]*group{}
	var order []string
	for _, f := range findings {
		lower := strings.ToLower(f.Owner + "/" + f.Repo)
		g := groups[lower]
		if g == nil {
			g = &group{key: f.Owner + "/" + f.Repo}
			groups[lower] = g
			order = append(order, lower)
		}
		g.sites = append(g.sites, f)
		g.vulns = append(g.vulns, f.Vulns...)
	}

	refsByKey := map[string][]parser.ActionRef{}
	for _, a := range actions {
		lower := strings.ToLower(a.Owner + "/" + a.Repo)
		refsByKey[lower] = append(refsByKey[lower], a)
	}

	for _, lower := range order {
		g := groups[lower]
		minFixed, ok := minFixedFor(g.vulns)
		if !ok {
			unfixable = append(unfixable, g.sites...)
			continue
		}
		refs := refsByKey[lower]
		current := maxAuditVersion(refs)

		// A key is only "already past" when every vulnerable site is at or
		// above the fix; a sibling ref above it must not mask a vulnerable one.
		needsFix := false
		for _, site := range g.sites {
			v := auditSiteVersion(site, refs)
			if v == "" || auditVersionCompare(v, minFixed) < 0 {
				needsFix = true
				break
			}
		}
		if !needsFix {
			alreadyPast = append(alreadyPast, auditFix{Key: g.key, Refs: refs, Current: current, MinFixed: minFixed})
			continue // already past the reported fix; never downgrade
		}
		fixes = append(fixes, auditFix{Key: g.key, Refs: refs, Current: current, MinFixed: minFixed})
	}
	return fixes, unfixable, alreadyPast
}

// auditSiteVersion returns the version used at a vulnerable site, preferring
// the parsed ref at the same file and line.
func auditSiteVersion(f vulnscanner.Finding, refs []parser.ActionRef) string {
	for _, ref := range refs {
		if ref.File == f.File && ref.Line == f.Line {
			return ref.Version()
		}
	}
	return f.Ref
}

// minFixedFor returns the highest fixed version across deduplicated advisories.
// Multiple occurrences of the same advisory are merged by their maximum fixed
// version; any occurrence without a fix makes the whole key unfixable.
func minFixedFor(vulns []vulnscanner.Vulnerability) (string, bool) {
	seen := map[string]int{}
	var deduped []vulnscanner.Vulnerability
	for _, v := range vulns {
		if idx, ok := seen[v.ID]; ok {
			prior := deduped[idx].Fixed
			switch {
			case prior == "" || v.Fixed == "":
				deduped[idx].Fixed = ""
			case auditVersionCompare(v.Fixed, prior) > 0:
				deduped[idx].Fixed = v.Fixed
			}
			continue
		}
		seen[v.ID] = len(deduped)
		deduped = append(deduped, v)
	}

	best := ""
	for _, v := range deduped {
		if v.Fixed == "" {
			return "", false
		}
		if best == "" || auditVersionCompare(v.Fixed, best) > 0 {
			best = v.Fixed
		}
	}
	if best == "" {
		return "", false
	}
	return best, true
}

// fixTargetTooOldError means the newest tag in the fixed major is still below
// the required fixed version.
type fixTargetTooOldError struct {
	fixed string
	major int
}

func (e *fixTargetTooOldError) Error() string {
	return fmt.Sprintf("no tag ≥ %s in major v%d", e.fixed, e.major)
}

// resolveFixTarget resolves the tag to write. In semver mode it returns the
// fixed version verbatim; otherwise the latest tag in the fixed major.
func resolveFixTarget(ctx context.Context, client *github.Client, key, fixed string, semverMode bool) (string, error) {
	if semverMode {
		return normalizedFixed(fixed), nil
	}
	owner, repo := splitActionKey(key)
	major := auditMajor(fixed)
	if major <= 0 {
		return "", fmt.Errorf("cannot determine major version from %q", fixed)
	}
	tag, err := client.LatestTag(ctx, owner, repo, github.TagMode{Major: major})
	if err != nil {
		return "", err
	}
	if auditVersionCompare(tag, fixed) < 0 {
		return "", &fixTargetTooOldError{fixed: fixed, major: major}
	}
	return tag, nil
}

// normalizedFixed renders a fixed version as written by audit --fix --semver.
func normalizedFixed(fixed string) string {
	v := strings.TrimSpace(fixed)
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	return v
}

// auditMajor returns the numeric major of a version, or -1.
func auditMajor(version string) int {
	v := strings.TrimPrefix(strings.TrimSpace(version), "v")
	seg, _, _ := strings.Cut(v, ".")
	n, err := strconv.Atoi(seg)
	if err != nil {
		return -1
	}
	return n
}

// maxAuditVersion returns the highest version among refs, ignoring bare SHAs.
func maxAuditVersion(refs []parser.ActionRef) string {
	best := ""
	for _, ref := range refs {
		v := ref.Version()
		if v == "" {
			continue
		}
		if best == "" || auditVersionCompare(v, best) > 0 {
			best = v
		}
	}
	return best
}

// auditVersionCompare compares dotted versions per numeric segment, ignoring
// a leading "v". It handles OSV versions like "41" and "5.2".
func auditVersionCompare(a, b string) int {
	ap := strings.Split(strings.TrimPrefix(strings.TrimSpace(a), "v"), ".")
	bp := strings.Split(strings.TrimPrefix(strings.TrimSpace(b), "v"), ".")
	n := max(len(ap), len(bp))
	for i := 0; i < n; i++ {
		x, y := auditVersionSegment(ap, i), auditVersionSegment(bp, i)
		if x != y {
			if x < y {
				return -1
			}
			return 1
		}
	}
	return 0
}

func auditVersionSegment(parts []string, i int) int {
	if i >= len(parts) {
		return 0
	}
	n, err := strconv.Atoi(parts[i])
	if err != nil {
		return 0
	}
	return n
}

// auditUpgrades builds the upgrades for one action, skipping refs already at
// the target. SHA sites stay SHA; tag sites stay tag unless --sha is effective.
func auditUpgrades(f auditFix, newSHA string, forceSHA bool) []upgrader.Upgrade {
	upgrades := make([]upgrader.Upgrade, 0, len(f.Refs))
	for _, ref := range f.Refs {
		writeSHA := forceSHA || ref.Pinned()
		if writeSHA {
			if ref.SHA == newSHA && ref.Tag == f.Target {
				continue
			}
		} else if ref.Version() == f.Target {
			continue
		}
		refSHA := ""
		if writeSHA {
			refSHA = newSHA
		}
		upgrades = append(upgrades, upgrader.Upgrade{Action: ref, NewTag: f.Target, NewSHA: refSHA})
	}
	return upgrades
}

// outputAuditJSON writes findings as JSON to stdout and returns
// errAuditFindings when any were found. No extra hints are printed so the
// output stays machine-readable.
func outputAuditJSON(findings []vulnscanner.Finding) error {
	out := struct {
		Findings []auditJSONFinding `json:"findings"`
	}{
		Findings: make([]auditJSONFinding, 0, len(findings)),
	}

	for _, f := range findings {
		jf := auditJSONFinding{
			Action:     f.Owner + "/" + f.Repo,
			Ref:        f.Ref,
			File:       f.File,
			Line:       f.Line,
			Transitive: f.Line == 0,
			Vulns:      make([]auditJSONVuln, 0, len(f.Vulns)),
		}
		for _, v := range f.Vulns {
			jf.Vulns = append(jf.Vulns, auditJSONVuln{
				ID:       v.ID,
				Severity: v.Severity,
				Summary:  v.Summary,
				Fixed:    v.Fixed,
				URL:      v.URL,
			})
		}
		out.Findings = append(out.Findings, jf)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(&out); err != nil {
		return err
	}

	if len(findings) > 0 {
		return errAuditFindings
	}
	return nil
}
