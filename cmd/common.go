package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"path"
	"regexp"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/lynicis/actup/internal/breakingchanges"
	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/github"
	"github.com/lynicis/actup/internal/lockfile"
	"github.com/lynicis/actup/internal/parser"
	"github.com/lynicis/actup/internal/scanner"
	"github.com/lynicis/actup/internal/token"
	"github.com/lynicis/actup/internal/vulnscanner"
)

// loadWorkflowActions discovers workflow files under the given paths and
// extracts all action references from them.
func loadWorkflowActions(ctx context.Context, paths []string) ([]parser.ActionRef, error) {
	files, err := scanner.DiscoverWorkflows(ctx, paths)
	if err != nil {
		return nil, fmt.Errorf("failed to discover workflows: %w", err)
	}
	if len(files) == 0 {
		return nil, errors.New("no workflow or action files found")
	}

	actions, err := parser.ExtractActions(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflows: %w", err)
	}
	return actions, nil
}

// filterActionsByPatterns keeps refs whose owner/repo key matches at least one
// path.Match pattern. Patterns are a union; empty patterns keeps everything.
func filterActionsByPatterns(actions []parser.ActionRef, patterns []string) ([]parser.ActionRef, error) {
	if len(patterns) == 0 {
		return actions, nil
	}
	var out []parser.ActionRef
	for _, a := range actions {
		key := a.Owner + "/" + a.Repo
		for _, p := range patterns {
			ok, err := path.Match(p, key)
			if err != nil {
				return nil, fmt.Errorf("invalid filter %q: %w", p, err)
			}
			if ok {
				out = append(out, a)
				break
			}
		}
	}
	return out, nil
}

// applyActionFilters unions positional patterns with the named config groups,
// filters the actions, and errors when nothing matches.
func applyActionFilters(actions []parser.ActionRef, args, groups []string, cfg *config.Config) ([]parser.ActionRef, error) {
	patterns := append([]string{}, args...)
	for _, g := range groups {
		if cfg == nil {
			return nil, fmt.Errorf("--group requires .actup.yaml")
		}
		pats, ok := cfg.Groups[g]
		if !ok {
			return nil, fmt.Errorf("unknown group %q", g)
		}
		patterns = append(patterns, pats...)
	}
	actions, err := filterActionsByPatterns(actions, patterns)
	if err != nil {
		return nil, err
	}
	if len(patterns) > 0 && len(actions) == 0 {
		return nil, fmt.Errorf("no actions match %s", strings.Join(patterns, ", "))
	}
	return actions, nil
}

// actionSpec is a user-provided action reference of the form
// owner/repo[@<ref>]. An empty Ref means "not specified".
type actionSpec struct {
	Key string
	Ref string
}

// parseActionSpec parses an argument like "actions/checkout@v5". The key is
// normalized to owner/repo, matching how actup keys subpath actions
// (owner/repo/sub@v1) throughout the parser.
func parseActionSpec(arg string) (actionSpec, error) {
	parts := strings.SplitN(arg, "@", 2)

	pathParts := strings.Split(parts[0], "/")
	if len(pathParts) < 2 || pathParts[0] == "" || pathParts[1] == "" {
		return actionSpec{}, fmt.Errorf("invalid action %q (expected owner/repo[@ref])", arg)
	}

	spec := actionSpec{Key: pathParts[0] + "/" + pathParts[1]}
	if len(parts) == 2 {
		spec.Ref = parts[1]
	}
	return spec, nil
}

// resolveSpecRef maps a spec to the concrete ref that should be written:
//   - "" or "latest" → latest tag (major-tag mode, or full semver with semver)
//   - a bare number (e.g. "4") → latest tag within that major (e.g. v4.3.0)
//   - anything else (e.g. "v5", "v5.3.1", "master") → verbatim
func resolveSpecRef(ctx context.Context, client *github.Client, spec actionSpec, semver bool) (string, error) {
	ref := spec.Ref
	if ref == "" || ref == "latest" {
		owner, repo := splitActionKey(spec.Key)
		return client.LatestTag(ctx, owner, repo, github.TagMode{Semver: semver})
	}
	if n, err := strconv.Atoi(ref); err == nil {
		owner, repo := splitActionKey(spec.Key)
		return client.LatestTag(ctx, owner, repo, github.TagMode{Major: n})
	}
	return ref, nil
}

func splitActionKey(key string) (owner, repo string) {
	parts := strings.SplitN(key, "/", 2)
	return parts[0], parts[1]
}

// mostCommonVersion returns the most frequently used version among refs,
// breaking ties in favor of the first occurrence. Bare SHA refs have an empty
// Version and are ignored.
func mostCommonVersion(refs []parser.ActionRef) string {
	counts := make(map[string]int)
	best := ""
	bestCount := 0
	for _, r := range refs {
		v := r.Version()
		if v == "" {
			continue
		}
		counts[v]++
		if counts[v] > bestCount {
			best = v
			bestCount = counts[v]
		}
	}
	return best
}

// anyRefPinned reports whether any ref is a commit SHA. An existing SHA pin
// must stay a SHA pin even when --sha is not passed.
func anyRefPinned(refs []parser.ActionRef) bool {
	for _, ref := range refs {
		if ref.Pinned() {
			return true
		}
	}
	return false
}

var commitSHARegex = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

// isCommitSHA reports whether s is a full 40-hex commit SHA.
func isCommitSHA(s string) bool { return commitSHARegex.MatchString(s) }

// savePin records an action pin in the default .actup.yaml, creating the file
// if needed and preserving other entries.
func savePin(key, ref string) error {
	if isCommitSHA(ref) {
		return fmt.Errorf("pin a tag and use sha: true")
	}
	cfg, err := config.LoadDefault()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg == nil {
		cfg = &config.Config{}
	}
	if cfg.Actions == nil {
		cfg.Actions = map[string]string{}
	}
	cfg.Actions[key] = ref
	if cfg.IsManaged() {
		cfg.Pins[key] = ref
	}
	return config.SaveDefault(cfg)
}

// savePins records multiple pins with one load/save. It does nothing when no
// manifest exists (audit must not create one).
func savePins(pins map[string]string) error {
	if len(pins) == 0 {
		return nil
	}
	cfg, err := config.LoadDefault()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg == nil {
		return nil
	}
	if cfg.Actions == nil {
		cfg.Actions = map[string]string{}
	}
	for key, ref := range pins {
		cfg.Actions[key] = ref
		if cfg.IsManaged() {
			cfg.Pins[key] = ref
		}
	}
	return config.SaveDefault(cfg)
}

// recordUpgrades updates a managed manifest after successful rewrites.
// Legacy and absent manifests are left untouched.
func recordUpgrades(cfg *config.Config, upgraded map[string]string) error {
	if cfg == nil || !cfg.IsManaged() || len(upgraded) == 0 {
		return nil
	}
	if cfg.Actions == nil {
		cfg.Actions = map[string]string{}
	}
	for key, ref := range upgraded {
		cfg.Actions[key] = ref
	}
	return config.SaveDefault(cfg)
}

// confirmBreaking prints breaking-change warnings and returns whether to
// proceed. Proceed when dry-run or force, prompt on a TTY, otherwise skip.
func confirmBreaking(action, from, to string, bcs []breakingchanges.BreakingChange, dryRun, force bool, reader *bufio.Reader, isTTY bool) (proceed bool, skipped bool) {
	if len(bcs) == 0 {
		return true, false
	}

	fmt.Printf("\n⚠ %s: %s → %s has breaking changes:\n", action, from, to)
	for _, bc := range bcs {
		fmt.Printf("  • %s\n", bc.Message)
	}

	if dryRun || force {
		return true, false
	}

	if !isTTY {
		fmt.Printf("  Skipping breaking-change actions. Use --force to apply anyway.\n")
		return false, true
	}

	fmt.Print("  Apply anyway? [y/N] ")
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	if answer != "y" && answer != "yes" {
		return false, true
	}
	return true, false
}

// confirmBreakingAll warns about breaking changes for every distinct
// current version used with the action. It returns whether to proceed and
// how many distinct versions were declined.
func confirmBreakingAll(key string, refs []parser.ActionRef, target string, registry *breakingchanges.Registry, dryRun, force bool, reader *bufio.Reader, isTTY bool) (proceed bool, skipped int) {
	inputsByVersion := map[string][]string{}
	var versions []string
	for _, ref := range refs {
		version := ref.Version()
		if version == "" {
			continue // unknown version cannot be checked
		}
		if _, ok := inputsByVersion[version]; !ok {
			versions = append(versions, version)
		}
		inputsByVersion[version] = append(inputsByVersion[version], ref.Inputs...)
	}
	for _, version := range versions {
		bcs := breakingChangesFor(registry, key, version, target, dedupeInputs(inputsByVersion[version]))
		ok, skip := confirmBreaking(key, version, target, bcs, dryRun, force, reader, isTTY)
		if skip {
			skipped++
		}
		if !ok {
			return false, skipped
		}
	}
	return true, skipped
}

// dedupeInputs removes duplicate with: keys, preserving first-seen order.
func dedupeInputs(inputs []string) []string {
	seen := make(map[string]bool, len(inputs))
	var out []string
	for _, input := range inputs {
		if seen[input] {
			continue
		}
		seen[input] = true
		out = append(out, input)
	}
	return out
}

// resolveToken returns the GitHub token to use: an explicit flag value,
// the GITHUB_TOKEN env var, or the gh CLI — warning when nothing resolves.
func resolveToken(flagToken string) string {
	if flagToken == "" {
		flagToken = token.Resolve("")
	}
	if flagToken == "" {
		fmt.Fprintf(os.Stderr, "\033[33m⚠ No GitHub token set — unauthenticated requests are rate-limited to 60/hour\033[0m\n\n")
	}
	return flagToken
}

// isTerminal reports whether stdin is an interactive terminal.
func isTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// breakingChangesFor returns known breaking changes between two versions of
// an action, or nil when the registry is unavailable.
func breakingChangesFor(registry *breakingchanges.Registry, action, from, to string, inputs []string) []breakingchanges.BreakingChange {
	if registry == nil {
		return nil
	}
	return registry.Check(action, from, to, inputs)
}

// printVulnWarnings scans the given action refs against OSV.dev and prints a
// warning for every usage of a known-vulnerable version. Failures (e.g. no
// network) degrade gracefully: a short notice is printed and the caller's
// flow continues.
func printVulnWarnings(ctx context.Context, actions []parser.ActionRef) {
	findings, _ := scanVulns(ctx, actions)
	if len(findings) == 0 {
		return
	}

	fmt.Fprintf(os.Stderr, "\n⚠ Vulnerable action versions found (%d usage site(s)):\n", len(findings))
	for _, f := range findings {
		fmt.Fprintf(os.Stderr, "  %s:%d: %s/%s@%s\n", f.File, f.Line, f.Owner, f.Repo, f.Ref)
		for _, v := range f.Vulns {
			severity := v.Severity
			if severity == "" {
				severity = "UNKNOWN"
			}
			fixed := ""
			if v.Fixed != "" {
				fixed = fmt.Sprintf(", fixed in %s", v.Fixed)
			}
			fmt.Fprintf(os.Stderr, "    • %s [%s]%s — %s\n", v.ID, severity, fixed, v.Summary)
		}
	}
	fmt.Fprintf(os.Stderr, "  Run 'actup audit' for details, or upgrade with 'actup use <action>@latest'.\n\n")
}

// scanVulns runs the OSV audit over actions and returns the findings, plus a
// map of vulnerable action key -> advisories. A nil map means the scan
// failed (offline); callers should degrade gracefully in that case.
func scanVulns(ctx context.Context, actions []parser.ActionRef) ([]vulnscanner.Finding, map[string][]vulnscanner.Vulnerability) {
	if len(actions) == 0 {
		return nil, make(map[string][]vulnscanner.Vulnerability)
	}

	client := vulnscanner.NewClient()
	findings, err := client.Scan(ctx, actions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Vulnerability check unavailable: %v\n", err)
		return nil, nil
	}

	vulnsByKey := make(map[string][]vulnscanner.Vulnerability)
	for _, f := range findings {
		key := strings.ToLower(f.Owner + "/" + f.Repo)
		vulnsByKey[key] = f.Vulns
	}
	return findings, vulnsByKey
}

// confirmVulnerableTarget warns when the target version of an upgrade is
// itself affected by known advisories and asks whether to proceed. Mirrors
// confirmBreaking's TTY semantics: auto-proceed on dry-run/force, prompt on
// TTY, proceed with a notice when not a TTY.
func confirmVulnerableTarget(ctx context.Context, key, target string, dryRun, force bool, reader *bufio.Reader, isTTY bool) bool {
	owner, repo := splitActionKey(key)
	probe := []parser.ActionRef{{Owner: owner, Repo: repo, Current: target}}
	_, vulnsByKey := scanVulns(ctx, probe)
	vulns := vulnsByKey[strings.ToLower(key)]
	if len(vulns) == 0 {
		return true
	}

	fmt.Printf("\n⚠ %s: target version %s is affected by %d known vulnerabilit%s:\n", key, target, len(vulns), pluralY(len(vulns)))
	for _, v := range vulns {
		severity := v.Severity
		if severity == "" {
			severity = "UNKNOWN"
		}
		fixed := ""
		if v.Fixed != "" {
			fixed = fmt.Sprintf(", fixed in %s", v.Fixed)
		}
		fmt.Printf("  • %s [%s]%s — %s\n", v.ID, severity, fixed, v.Summary)
	}

	if dryRun || force {
		return true
	}

	if !isTTY {
		fmt.Printf("  Proceeding anyway (non-interactive). Use --force to suppress this notice.\n")
		return true
	}

	fmt.Print("  Pin this version anyway? [y/N] ")
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

// pluralY returns "y" for one, "ies" otherwise.
func pluralY(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

// relockFunc is a test seam for refreshing actions.lock via `gh actions-lock`.
var relockFunc = lockfile.Sync

// syncLockfile refreshes actions.lock after workflow writes, when one exists.
func syncLockfile(paths []string) {
	lf, err := lockfile.LoadForPaths(paths)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to read actions.lock: %v\n", err)
		return
	}
	if lf == nil {
		return
	}
	if err := relockFunc(lf.Path); err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to update actions.lock (run 'gh actions-lock' manually): %v\n", err)
		return
	}
	fmt.Println("✓ Updated .github/workflows/actions.lock")
}
