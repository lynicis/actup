package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"sort"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/lynicis/actup/internal/breakingchanges"
	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/github"
	"github.com/lynicis/actup/internal/parser"
	"github.com/lynicis/actup/internal/token"
	"github.com/lynicis/actup/internal/upgrader"
)

var syncOpts struct {
	paths  []string
	token  string
	force  bool
	dryRun bool
}

var syncCmd = &cobra.Command{
	Use:     "sync",
	Aliases: []string{"install"},
	Short:   "Apply the action pins from .actup.yaml to all workflows",
	Long: `Makes workflows follow .actup.yaml — the manifest is the source of truth.

Pin semantics:
  exact ref (v5, v5.3.1, master)  applied verbatim
  major number (4)                resolved to the latest tag within v4
  skip                            ignored

Known breaking changes are warned about; on a TTY you are prompted,
otherwise pass --force to apply them anyway.`,
	Example: `  actup sync
  actup sync --dry-run
  actup sync --force`,
	RunE: runSync,
}

func init() {
	syncCmd.Flags().StringArrayVarP(&syncOpts.paths, "path", "p", []string{}, "Paths to workflow files or directories")
	syncCmd.Flags().StringVarP(&syncOpts.token, "token", "t", "", "GitHub PAT (fallback: GITHUB_TOKEN env var, then gh auth token)")
	syncCmd.Flags().BoolVarP(&syncOpts.force, "force", "f", false, "Apply pins with breaking changes without prompting")
	syncCmd.Flags().BoolVar(&syncOpts.dryRun, "dry-run", false, "Preview changes without writing files")
	rootCmd.AddCommand(syncCmd)
}

type syncOptions struct {
	paths  []string
	token  string
	force  bool
	dryRun bool
}

func runSync(cmd *cobra.Command, args []string) error {
	return doSync(context.Background(), syncOpts)
}

func doSync(ctx context.Context, opts syncOptions) error {
	cfg, err := config.LoadDefault()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg == nil || len(cfg.Actions) == 0 {
		fmt.Println("No action pins found in .actup.yaml.")
		fmt.Println("Run `actup init` to bootstrap one, or `actup pin <action>@<ref>` to add a pin.")
		return nil
	}

	actions, err := loadWorkflowActions(ctx, opts.paths)
	if err != nil {
		return err
	}
	grouped := parser.GroupActions(actions)

	registry, err := breakingchanges.LoadRegistry()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to load breaking-change registry: %v\n", err)
	}

	keys := make([]string, 0, len(cfg.Actions))
	for key := range cfg.Actions {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Numeric pins need the GitHub API to resolve the latest tag in a major;
	// exact refs are applied verbatim without any API call.
	needsClient := false
	for _, key := range keys {
		pin := cfg.Actions[key]
		if pin != "skip" {
			if _, err := strconv.Atoi(pin); err == nil {
				needsClient = true
				break
			}
		}
	}
	var client *github.Client
	if needsClient {
		ghToken := opts.token
		if ghToken == "" {
			ghToken = token.Resolve("")
		}
		if ghToken == "" {
			fmt.Fprintf(os.Stderr, "\033[33m⚠ No GitHub token set — unauthenticated requests are rate-limited to 60/hour\033[0m\n\n")
		}
		client = github.NewClient(ghToken)
	}

	reader := bufio.NewReader(os.Stdin)
	isTTY := isTerminal()

	// Warn about vulnerable versions currently in use before syncing pins.
	printVulnWarnings(ctx, actions)

	upgraded, skipped, breakingSkipped, missing := 0, 0, 0, 0

	for _, key := range keys {
		pin := cfg.Actions[key]

		if pin == "skip" {
			fmt.Printf("⏭ %s: skipped (config)\n", key)
			skipped++
			continue
		}

		target := pin
		if n, err := strconv.Atoi(pin); err == nil {
			owner, repo := splitActionKey(key)
			target, err = client.LatestTag(ctx, owner, repo, github.TagMode{Major: n})
			if err != nil {
				fmt.Printf("⚠ %s: API error (%v)\n", key, err)
				continue
			}
		}

		refs, ok := grouped[key]
		if !ok {
			fmt.Printf("ⓘ %s: not used in any workflow\n", key)
			missing++
			continue
		}

		proceed := true
		seen := map[string]bool{}
		for _, ref := range refs {
			if seen[ref.Current] {
				continue
			}
			seen[ref.Current] = true
			bcs := breakingChangesFor(registry, key, ref.Current, target)
			ok, skip := confirmBreaking(key, ref.Current, target, bcs, opts.dryRun, opts.force, reader, isTTY)
			if skip {
				breakingSkipped++
			}
			if !ok {
				proceed = false
				break
			}
		}
		if !proceed {
			continue
		}

		// Guard against syncing to a version that is itself vulnerable.
		if !confirmVulnerableTarget(ctx, key, target, opts.dryRun, opts.force, reader, isTTY) {
			fmt.Printf("⏭ %s: skipped — target %s is vulnerable\n", key, target)
			continue
		}

		uniqueCurrents := map[string]bool{}
		for _, ref := range refs {
			uniqueCurrents[ref.Current] = true
		}
		if len(uniqueCurrents) == 1 && refs[0].Current == target {
			fmt.Printf("⏭ %s: already at %s\n", key, target)
			skipped++
			continue
		}

		upgrades := make([]upgrader.Upgrade, 0, len(refs))
		for _, ref := range refs {
			if ref.Current == target {
				continue
			}
			upgrades = append(upgrades, upgrader.Upgrade{Action: ref, NewTag: target})
		}
		if len(upgrades) == 0 {
			fmt.Printf("⏭ %s: already at %s\n", key, target)
			skipped++
			continue
		}

		if opts.dryRun {
			fmt.Printf("would update %s: %s → %s (%d reference(s))\n", key, pin, target, len(upgrades))
			upgraded++
			continue
		}

		if err := upgrader.ApplyAllUpgrades(upgrades, false); err != nil {
			fmt.Printf("✗ %s: %v\n", key, err)
			continue
		}
		fmt.Printf("✓ %s: %s → %s (%d reference(s))\n", key, refs[0].Current, target, len(upgrades))
		upgraded++
	}

	fmt.Printf("\n%d action(s) synced, %d already up to date", upgraded, skipped)
	if missing > 0 {
		fmt.Printf(", %d not used in any workflow", missing)
	}
	if breakingSkipped > 0 {
		fmt.Printf(", %d skipped (breaking changes)", breakingSkipped)
	}
	fmt.Println()
	if opts.dryRun {
		fmt.Println("Run without --dry-run to apply changes.")
	}
	return nil
}
