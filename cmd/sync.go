package cmd

import (
	"bufio"
	"context"
	"fmt"
	"maps"
	"os"
	"slices"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/lynicis/actup/internal/breakingchanges"
	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/github"
	"github.com/lynicis/actup/internal/parser"
	"github.com/lynicis/actup/internal/upgrader"
)

var syncOpts syncOptions

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
  actup sync --force
  actup sync --sha`,
	RunE: runSync,
}

func init() {
	syncCmd.Flags().StringArrayVarP(&syncOpts.paths, "path", "p", []string{}, "Paths to workflow files or directories")
	syncCmd.Flags().StringVarP(&syncOpts.token, "token", "t", "", "GitHub PAT (fallback: GITHUB_TOKEN env var, then gh auth token)")
	syncCmd.Flags().BoolVarP(&syncOpts.force, "force", "f", false, "Apply pins with breaking changes without prompting")
	syncCmd.Flags().BoolVar(&syncOpts.dryRun, "dry-run", false, "Preview changes without writing files")
	syncCmd.Flags().BoolVar(&syncOpts.sha, "sha", false, "Write refs as full commit SHAs with the tag in a trailing comment")
	rootCmd.AddCommand(syncCmd)
}

type syncOptions struct {
	paths  []string
	token  string
	force  bool
	dryRun bool
	sha    bool
	shaSet bool
}

func runSync(cmd *cobra.Command, args []string) error {
	syncOpts.shaSet = cmd.Flags().Changed("sha")
	return doSync(context.Background(), syncOpts)
}

func doSync(ctx context.Context, opts syncOptions) error {
	cfg, err := config.LoadDefault()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg == nil {
		fmt.Println("No action pins found in .actup.yaml.")
		fmt.Println("Run `actup init` to bootstrap one, or `actup pin <action>@<ref>` to add a pin.")
		return nil
	}
	pins := cfg.EffectivePins()
	if len(cfg.Actions) == 0 && len(pins) == 0 {
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

	keySet := map[string]struct{}{}
	for key := range cfg.Actions {
		keySet[key] = struct{}{}
	}
	for key := range pins {
		keySet[key] = struct{}{}
	}
	keys := slices.Sorted(maps.Keys(keySet))

	sha := cfg.SHA
	if opts.shaSet {
		sha = opts.sha
	}

	// Numeric pins need the GitHub API to resolve the latest tag in a major.
	// SHA mode also needs it to resolve each exact tag to a commit SHA, as does
	// any workflow ref already pinned to a SHA (it must stay pinned).
	needsClient := sha
	if !needsClient {
		for _, a := range actions {
			if a.Pinned() {
				needsClient = true
				break
			}
		}
	}
	if !needsClient {
		for _, key := range keys {
			pin := pins[key]
			if pin != "skip" {
				if _, err := strconv.Atoi(pin); err == nil {
					needsClient = true
					break
				}
			}
		}
	}
	var client *github.Client
	if needsClient {
		client = github.NewClient(resolveToken(opts.token))
	}

	reader := bufio.NewReader(os.Stdin)
	isTTY := isTerminal()

	// Warn about vulnerable versions currently in use before syncing pins.
	printVulnWarnings(ctx, actions)

	upgraded, skipped, breakingSkipped, missing := 0, 0, 0, 0
	dirty := false

	for _, key := range keys {
		pin, pinned := pins[key]
		snapshot := cfg.Actions[key]

		if pin == "skip" {
			fmt.Printf("⏭ %s: skipped (config)\n", key)
			skipped++
			continue
		}

		var target string
		if n, err := strconv.Atoi(pin); err == nil {
			owner, repo := splitActionKey(key)
			target, err = client.LatestTag(ctx, owner, repo, github.TagMode{Major: n})
			if err != nil {
				fmt.Printf("⚠ %s: API error (%v)\n", key, err)
				continue
			}
		} else if pinned {
			target = pin
		} else if snapshot != "" {
			target = snapshot // restore lock
		} else {
			skipped++
			continue
		}

		if isCommitSHA(target) {
			fmt.Printf("✗ %s: pin a tag and use sha: true\n", key)
			skipped++
			continue
		}

		refs, ok := grouped[key]
		if !ok {
			fmt.Printf("ⓘ %s: not used in any workflow\n", key)
			missing++
			continue
		}

		resolveSHA := sha || anyRefPinned(refs)
		newSHA := ""
		if resolveSHA {
			owner, repo := splitActionKey(key)
			newSHA, err = client.ResolveTagSHA(ctx, owner, repo, target)
			if err != nil {
				fmt.Printf("⚠ %s: API error (%v)\n", key, err)
				continue
			}
		}

		proceed, breakingSkips := confirmBreakingAll(key, refs, target, registry, opts.dryRun, opts.force, reader, isTTY)
		breakingSkipped += breakingSkips
		if !proceed {
			continue
		}

		// Guard against syncing to a version that is itself vulnerable.
		if !confirmVulnerableTarget(ctx, key, target, opts.dryRun, opts.force, reader, isTTY) {
			fmt.Printf("⏭ %s: skipped — target %s is vulnerable\n", key, target)
			continue
		}

		upgrades := make([]upgrader.Upgrade, 0, len(refs))
		for _, ref := range refs {
			writeSHA := sha || ref.Pinned()
			if writeSHA {
				if ref.SHA == newSHA && ref.Tag == target {
					continue
				}
			} else if ref.Version() == target {
				continue
			}
			refSHA := ""
			if writeSHA {
				refSHA = newSHA
			}
			upgrades = append(upgrades, upgrader.Upgrade{Action: ref, NewTag: target, NewSHA: refSHA})
		}
		if len(upgrades) == 0 {
			fmt.Printf("⏭ %s: already at %s\n", key, target)
			skipped++
			continue
		}

		if opts.dryRun {
			fmt.Printf("would update %s: %s → %s (%d reference(s))\n", key, refs[0].Current, target, len(upgrades))
			upgraded++
			continue
		}

		if err := upgrader.ApplyAllUpgrades(upgrades, false); err != nil {
			fmt.Printf("✗ %s: %v\n", key, err)
			continue
		}
		if cfg.Actions == nil {
			cfg.Actions = map[string]string{}
		}
		cfg.Actions[key] = target
		dirty = true
		fmt.Printf("✓ %s: %s → %s (%d reference(s))\n", key, refs[0].Current, target, len(upgrades))
		upgraded++
	}

	if dirty {
		if err := config.SaveDefault(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
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
