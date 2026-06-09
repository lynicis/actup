package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/lynicis/actup/internal/breakingchanges"
	"github.com/lynicis/actup/internal/checker"
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
	checkFlag   bool
)

var rootCmd = &cobra.Command{
	Use:   "actup",
	Short: "GitHub Actions Upgrader — upgrade action versions interactively",
	Long:  "A CLI tool that scans GitHub Actions workflow files and upgrades action versions to their latest semver tags via a terminal TUI.",
	RunE:  run,
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
	rootCmd.Flags().BoolVar(&checkFlag, "check", false, "Check for outdated actions and exit non-zero if found (implies --no-tui)")
}

func run(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	if githubToken == "" {
		resolver := token.NewResolver()
		githubToken = resolver.Resolve("")
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

	if len(actions) == 0 {
		fmt.Println("No actions to upgrade found.")
		return nil
	}

	if checkFlag {
		return runCheck(ctx, actions, githubToken, semverMode, majorVer)
	}

	if noTUI {
		return runNoTUI(ctx, actions, githubToken, dryRun, semverMode, majorVer, force)
	}

	return tui.Run(ctx, actions, githubToken, dryRun, semverMode, majorVer)
}

func runNoTUI(ctx context.Context, actions []parser.ActionRef, githubToken string, dryRun bool, semverMode bool, majorVer int, force bool) error {
	grouped := parser.GroupActions(actions)
	ghClient := github.NewClient(githubToken)

	registry, err := breakingchanges.LoadRegistry()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to load breaking-change registry: %v\n", err)
	}

	type result struct {
		key       string
		latest    string
		fileCount int
		err       error
	}

	resultCh := make(chan result, len(grouped))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for key, acts := range grouped {
		wg.Add(1)
		go func(key string, acts []parser.ActionRef) {
			sem <- struct{}{}
			defer func() { <-sem }()
			defer wg.Done()

			parts := strings.SplitN(key, "/", 2)
			owner := parts[0]
			repo := parts[1]

			latest, err := ghClient.LatestTag(ctx, owner, repo, github.TagMode{Semver: semverMode, Major: majorVer})

			resultCh <- result{
				key:       key,
				latest:    latest,
				fileCount: len(acts),
				err:       err,
			}
		}(key, acts)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	upgraded := 0
	skipped := 0
	apiErr := 0
	breakingSkipped := 0

	isTTY := term.IsTerminal(int(os.Stdin.Fd()))
	reader := bufio.NewReader(os.Stdin)

	for r := range resultCh {
		acts := grouped[r.key]
		current := acts[0].Current

		if r.err != nil {
			apiErr++
			fmt.Printf("⚠ %s: API error (%s)\n", r.key, r.err)
			continue
		}

		var needsUpgrade []parser.ActionRef
		for _, act := range acts {
			if act.Current != r.latest {
				needsUpgrade = append(needsUpgrade, act)
			}
		}

		if len(needsUpgrade) == 0 {
			skipped++
			fmt.Printf("⏭ %s: already at %s\n", r.key, r.latest)
			continue
		}

		var bcs []breakingchanges.BreakingChange
		if registry != nil {
			bcs = registry.Check(r.key, current, r.latest)
		}

		if len(bcs) > 0 && !dryRun && !force {
			fmt.Printf("\n⚠ %s: %s → %s has breaking changes:\n", r.key, current, r.latest)
			for _, bc := range bcs {
				fmt.Printf("  • %s\n", bc.Message)
			}

			if !isTTY {
				fmt.Printf("  Skipping breaking-change actions. Use --force to upgrade them anyway.\n")
				breakingSkipped++
				continue
			}

			fmt.Print("  Upgrade anyway? [y/N] ")
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer != "y" && answer != "yes" {
				breakingSkipped++
				continue
			}
		}

		if len(bcs) > 0 && (dryRun || force) {
			fmt.Printf("\n⚠ %s: %s → %s has breaking changes:\n", r.key, current, r.latest)
			for _, bc := range bcs {
				fmt.Printf("  • %s\n", bc.Message)
			}
		}

		upgraded++

		if dryRun {
			fmt.Printf("would update %s: %s → %s\n", r.key, current, r.latest)
		} else {
			upgrades := make(map[string]upgrader.Upgrade)
			for _, act := range needsUpgrade {
				upgradeKey := fmt.Sprintf("%s:%s:%d", r.key, act.File, act.Line)
				upgrades[upgradeKey] = upgrader.Upgrade{
					Action: act,
					NewTag: r.latest,
				}
			}

			_, err := upgrader.ApplyAllUpgrades(upgrades, false)
			if err != nil {
				fmt.Printf("✗ %s: %v\n", r.key, err)
				continue
			}
			fmt.Printf("✓ %s: %s → %s\n", r.key, current, r.latest)
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

func runCheck(ctx context.Context, actions []parser.ActionRef, ghToken string, semverMode bool, majorVer int) error {
	ghClient := github.NewClient(ghToken)

	c := checker.New(ghClient, semverMode, majorVer)
	outdated, err := c.Run(ctx, actions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(2)
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
