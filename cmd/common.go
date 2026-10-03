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
// proceed. It mirrors the --no-tui behavior in root.go: proceed when dry-run
// or force, prompt on a TTY, otherwise skip.
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
