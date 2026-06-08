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

	if noTUI {
		return runNoTUI(ctx, actions, githubToken, dryRun, semverMode)
	}

	return tui.Run(ctx, actions, githubToken, dryRun, semverMode)
}

func runNoTUI(ctx context.Context, actions []parser.ActionRef, githubToken string, dryRun bool, semverMode bool) error {
	grouped := parser.GroupActions(actions)
	ghClient := github.NewClient(githubToken)

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

			latest, err := ghClient.LatestTag(ctx, owner, repo, semverMode)

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

	for r := range resultCh {
		acts := grouped[r.key]
		current := acts[0].Current

		if r.err != nil {
			apiErr++
			fmt.Printf("⚠ %s: API error (%s)\n", r.key, r.err)
			continue
		}

		if r.latest == current {
			skipped++
			fmt.Printf("⏭ %s: already at %s\n", r.key, r.latest)
			continue
		}

		upgraded++

		if dryRun {
			fmt.Printf("would update %s: %s → %s\n", r.key, current, r.latest)
		} else {
			upgrades := make(map[string]upgrader.Upgrade)
			for _, act := range acts {
				upgrades[r.key] = upgrader.Upgrade{
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

	fmt.Printf("\n%d actions upgraded, %d skipped (up to date), %d skipped (API error)\n", upgraded, skipped, apiErr)
	if dryRun {
		fmt.Println("Run without --dry-run to apply changes.")
	}
	return nil
}
