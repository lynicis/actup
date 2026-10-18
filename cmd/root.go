package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/lynicis/actup/internal/breakingchanges"
	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/github"
	"github.com/lynicis/actup/internal/parser"
	"github.com/lynicis/actup/internal/scanner"
	"github.com/lynicis/actup/internal/token"
	"github.com/lynicis/actup/internal/tui"
	"github.com/lynicis/actup/internal/upgrader"
)

var (
	paths       []string
	githubToken string
	dryRun      bool
	noTUI       bool
	semverMode  bool
	majorVer    int
	force       bool
)

var rootCmd = &cobra.Command{
	Use:   "actup",
	Short: "Version manager for GitHub Actions — pin, switch, and upgrade action versions",
	Long: `actup is a version manager for GitHub Actions, like nvm is for Node or fvm for Flutter.

Run it bare to interactively upgrade actions to their latest versions via a TUI,
or use the version-manager subcommands:

  use       set an action version across all workflows (and pin it)
  pin/unpin manage version pins in .actup.yaml
  ls        list actions in use, their versions, and pins
  init      bootstrap .actup.yaml from your current workflows
  sync      apply .actup.yaml pins to all workflows
  outdated  list actions that can be upgraded
  audit     check action versions against known vulnerabilities

It detects known breaking changes between major versions and warns before
upgrading, and checks workflows against OSV.dev advisories.`,
	RunE: run,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringArrayVarP(&paths, "path", "p", []string{}, "Paths to workflow files or directories")
	rootCmd.Flags().StringVarP(&githubToken, "token", "t", "", "GitHub PAT (fallback: GITHUB_TOKEN env var, then gh auth token)")
	rootCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without writing files")
	rootCmd.Flags().BoolVar(&noTUI, "no-tui", false, "Upgrade all discovered actions non-interactively")
	rootCmd.Flags().BoolVarP(&semverMode, "semver", "s", false, "Upgrade to the latest full semver tag instead of the latest major tag (e.g. v5.3.1 instead of v5)")
	rootCmd.Flags().IntVarP(&majorVer, "major", "m", 0, "Pin upgrades to a specific major version (e.g. 4 for latest v4.x.x)")
	rootCmd.Flags().BoolVarP(&force, "force", "f", false, "Upgrade actions with breaking changes without prompting in non-interactive mode")
}

func run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if githubToken == "" {
		githubToken = token.Resolve("")
	}

	if githubToken == "" {
		fmt.Fprintf(os.Stderr, "\033[33m⚠ No GitHub token set — unauthenticated requests are rate-limited to 60/hour\033[0m\n\n")
	}

	files, err := scanner.DiscoverWorkflows(ctx, paths)
	if err != nil {
		return fmt.Errorf("failed to discover workflows: %w", err)
	}

	if len(files) == 0 {
		return fmt.Errorf("no workflow files found")
	}

	actions, err := parser.ExtractActions(ctx, files)
	if err != nil {
		return fmt.Errorf("failed to parse workflows: %w", err)
	}

	cfg, err := config.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to load config: %v\n", err)
	}
	if cfg != nil {
		if majorVer == 0 && cfg.Major != nil {
			majorVer = *cfg.Major
		}
	}

	actions = filterSkippedActions(actions, cfg)
	if len(actions) == 0 {
		fmt.Println("No actions to upgrade found.")
		return nil
	}

	// Vulnerability audit against OSV.dev. Warnings only: network failures
	// degrade gracefully so the upgrade flow still works offline. The TUI
	// performs its own scan during loading instead (stderr output would be
	// hidden by the alt screen).
	if noTUI {
		printVulnWarnings(ctx, actions)
		return runNoTUI(ctx, actions, githubToken, dryRun, semverMode, majorVer, force, cfg)
	}

	return tui.Run(actions, githubToken, dryRun, semverMode, majorVer, cfg)
}

func filterSkippedActions(actions []parser.ActionRef, cfg *config.Config) []parser.ActionRef {
	if cfg == nil || len(cfg.Actions) == 0 {
		return actions
	}
	var filtered []parser.ActionRef
	for _, a := range actions {
		key := a.Owner + "/" + a.Repo
		if pin, ok := cfg.Actions[key]; ok && pin == "skip" {
			continue
		}
		filtered = append(filtered, a)
	}
	return filtered
}

func runNoTUI(ctx context.Context, actions []parser.ActionRef, githubToken string, dryRun bool, semverMode bool, majorVer int, force bool, cfg *config.Config) error {
	grouped := parser.GroupActions(actions)
	ghClient := github.NewClient(githubToken)

	registry, err := breakingchanges.LoadRegistry()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to load breaking-change registry: %v\n", err)
	}

	var cfgActions map[string]string
	if cfg != nil {
		cfgActions = cfg.Actions
	}

	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	results := github.ResolveAll(ctx, ghClient, keys, semverMode, majorVer, cfgActions)

	upgraded := 0
	skipped := 0
	apiErr := 0
	breakingSkipped := 0

	isTTY := isTerminal()
	reader := bufio.NewReader(os.Stdin)

	for _, key := range keys {
		acts := grouped[key]
		current := acts[0].Current
		r := results[key]

		if r.Err != nil {
			if errors.Is(r.Err, github.ErrNoSemverTags) {
				skipped++
				fmt.Printf("⏭ %s: no tags found for major v%d\n", key, majorVer)
				continue
			}
			apiErr++
			fmt.Printf("⚠ %s: API error (%s)\n", key, r.Err)
			continue
		}

		var needsUpgrade []parser.ActionRef
		for _, act := range acts {
			if act.Current != r.Latest {
				needsUpgrade = append(needsUpgrade, act)
			}
		}

		if len(needsUpgrade) == 0 {
			skipped++
			fmt.Printf("⏭ %s: already at %s\n", key, r.Latest)
			continue
		}

		bcs := breakingChangesFor(registry, key, current, r.Latest)

		proceed, wasSkipped := confirmBreaking(key, current, r.Latest, bcs, dryRun, force, reader, isTTY)
		if wasSkipped {
			breakingSkipped++
			continue
		}
		if !proceed {
			continue
		}

		upgraded++

		if dryRun {
			fmt.Printf("would update %s: %s → %s\n", key, current, r.Latest)
		} else {
			upgrades := make([]upgrader.Upgrade, 0, len(needsUpgrade))
			for _, act := range needsUpgrade {
				upgrades = append(upgrades, upgrader.Upgrade{
					Action: act,
					NewTag: r.Latest,
				})
			}

			if err := upgrader.ApplyAllUpgrades(upgrades, false); err != nil {
				fmt.Printf("✗ %s: %v\n", key, err)
				upgraded--
				continue
			}
			fmt.Printf("✓ %s: %s → %s\n", key, current, r.Latest)
		}
	}

	fmt.Printf("\n%d actions upgraded, %d skipped (up to date), %d skipped (API error)", upgraded, skipped, apiErr)
	if breakingSkipped > 0 {
		fmt.Printf(", %d skipped (breaking changes)", breakingSkipped)
	}
	fmt.Println()
	if dryRun {
		fmt.Println("Run without --dry-run to apply changes.")
	}
	return nil
}

type outdatedAction struct {
	parser.ActionRef
	Latest string
}

func runCheck(ctx context.Context, actions []parser.ActionRef, ghToken string, semverMode bool, majorVer int, cfg *config.Config) error {
	ghClient := github.NewClient(ghToken)
	grouped := parser.GroupActions(actions)

	var cfgActions map[string]string
	if cfg != nil {
		cfgActions = cfg.Actions
	}

	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	results := github.ResolveAll(ctx, ghClient, keys, semverMode, majorVer, cfgActions)

	for _, r := range results {
		if r.Err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", r.Err)
			os.Exit(2)
		}
	}

	var outdated []outdatedAction
	for _, key := range keys {
		latest := results[key].Latest
		for _, act := range grouped[key] {
			if act.Current != latest {
				outdated = append(outdated, outdatedAction{
					ActionRef: act,
					Latest:    latest,
				})
			}
		}
	}

	if len(outdated) == 0 {
		fmt.Println("All actions are up to date.")
		return nil
	}

	for _, oa := range outdated {
		fmt.Printf("%s:%d: %s/%s %s → %s\n", oa.File, oa.Line, oa.Owner, oa.Repo, oa.Current, oa.Latest)
	}
	fmt.Printf("\n%d action(s) can be upgraded.\n", len(outdated))
	os.Exit(1)
	return nil
}
