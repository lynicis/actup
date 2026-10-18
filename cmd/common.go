package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"golang.org/x/term"

	"github.com/lynicis/actup/internal/breakingchanges"
	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/github"
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
		return nil, errors.New("no workflow files found")
	}

	actions, err := parser.ExtractActions(ctx, files)
	if err != nil {
		return nil, fmt.Errorf("failed to parse workflows: %w", err)
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
// breaking ties in favor of the first occurrence.
func mostCommonVersion(refs []parser.ActionRef) string {
	if len(refs) == 0 {
		return ""
	}
	counts := make(map[string]int)
	best := refs[0].Current
	bestCount := 0
	for _, r := range refs {
		counts[r.Current]++
		if counts[r.Current] > bestCount {
			best = r.Current
			bestCount = counts[r.Current]
		}
	}
	return best
}

// savePin records an action pin in the default .actup.yaml, creating the file
// if needed and preserving other entries.
func savePin(key, ref string) error {
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
	seen := map[string]bool{}
	for _, ref := range refs {
		if seen[ref.Current] {
			continue
		}
		seen[ref.Current] = true
		bcs := breakingChangesFor(registry, key, ref.Current, target)
		ok, skip := confirmBreaking(key, ref.Current, target, bcs, dryRun, force, reader, isTTY)
		if skip {
			skipped++
		}
		if !ok {
			return false, skipped
		}
	}
	return true, skipped
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
func breakingChangesFor(registry *breakingchanges.Registry, action, from, to string) []breakingchanges.BreakingChange {
	if registry == nil {
		return nil
	}
	return registry.Check(action, from, to)
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
