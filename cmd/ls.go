package cmd

import (
	"context"
	"fmt"
	"maps"
	"os"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/github"
	"github.com/lynicis/actup/internal/parser"
)

var lsOpts struct {
	paths    []string
	token    string
	semver   bool
	outdated bool
}

var lsCmd = &cobra.Command{
	Use:   "ls [action]",
	Short: "List actions used in workflows and their versions",
	Long: `Lists every action used in your workflows with the versions in use and any
pin recorded in .actup.yaml — like "nvm ls".

By default no GitHub API calls are made. Pass --outdated to annotate each
action with its latest available version.`,
	Example: `  actup ls
  actup ls --outdated
  actup ls actions/checkout   # show every file:line where it is used`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLs,
}

func init() {
	lsCmd.Flags().StringArrayVarP(&lsOpts.paths, "path", "p", []string{}, "Paths to workflow files or directories")
	lsCmd.Flags().StringVarP(&lsOpts.token, "token", "t", "", "GitHub PAT (fallback: GITHUB_TOKEN env var, then gh auth token)")
	lsCmd.Flags().BoolVarP(&lsOpts.semver, "semver", "s", false, "Resolve latest versions as full semver tags instead of major tags")
	lsCmd.Flags().BoolVar(&lsOpts.outdated, "outdated", false, "Also resolve and show the latest available version (uses the GitHub API)")
	rootCmd.AddCommand(lsCmd)
}

func runLs(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	actions, err := loadWorkflowActions(ctx, lsOpts.paths)
	if err != nil {
		return err
	}
	grouped := parser.GroupActions(actions)

	cfg, err := config.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to load config: %v\n", err)
	}
	var cfgActions map[string]string
	if cfg != nil {
		cfgActions = cfg.Actions
	}

	if len(args) == 1 {
		return runLsDetail(grouped, cfgActions, args[0])
	}
	return runLsTable(ctx, grouped, cfgActions)
}

func runLsDetail(grouped map[string][]parser.ActionRef, cfgActions map[string]string, key string) error {
	refs, ok := grouped[key]
	if !ok {
		return fmt.Errorf("%s is not used in any workflow", key)
	}

	pin := cfgActions[key]
	if pin == "" {
		pin = "—"
	}
	fmt.Printf("%s (pin: %s)\n\n", key, pin)

	sorted := make([]parser.ActionRef, len(refs))
	copy(sorted, refs)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].File != sorted[j].File {
			return sorted[i].File < sorted[j].File
		}
		return sorted[i].Line < sorted[j].Line
	})

	for _, ref := range sorted {
		fmt.Printf("  %s:%d  %s\n", ref.File, ref.Line, ref.Current)
	}
	return nil
}

func runLsTable(ctx context.Context, grouped map[string][]parser.ActionRef, cfgActions map[string]string) error {
	keys := slices.Sorted(maps.Keys(grouped))

	var latest map[string]string
	if lsOpts.outdated {
		var err error
		latest, err = resolveLatestForAll(ctx, keys, cfgActions)
		if err != nil {
			return err
		}
	}

	keyWidth := len("ACTION")
	for _, key := range keys {
		if len(key) > keyWidth {
			keyWidth = len(key)
		}
	}

	for _, key := range keys {
		summary := summarizeActionVersions(grouped[key])
		line := fmt.Sprintf("%-*s  %-24s", keyWidth, key, summary)

		switch pin := cfgActions[key]; {
		case pin == "skip":
			line += "  skip"
		case pin != "":
			line += fmt.Sprintf("  pin: %s", pin)
		default:
			line += "  —"
		}

		if latest != nil {
			if l, ok := latest[key]; ok && l != grouped[key][0].Current {
				line += fmt.Sprintf("  → %s", l)
			} else if ok {
				line += "  ✓"
			}
		}
		fmt.Println(line)
	}

	if cfgActions != nil {
		var unused []string
		for key := range cfgActions {
			if _, used := grouped[key]; !used {
				unused = append(unused, key)
			}
		}
		sort.Strings(unused)
		if len(unused) > 0 {
			fmt.Printf("\nPinned but not used in any workflow:\n")
			for _, key := range unused {
				fmt.Printf("  %s: %s\n", key, cfgActions[key])
			}
		}
	}

	return nil
}

// summarizeActionVersions renders the distinct versions in use with their
// reference counts, e.g. "v4 ×3" or "v3 ×1, v4 ×2".
func summarizeActionVersions(refs []parser.ActionRef) string {
	counts := make(map[string]int)
	order := make([]string, 0, len(refs))
	for _, ref := range refs {
		if _, seen := counts[ref.Current]; !seen {
			order = append(order, ref.Current)
		}
		counts[ref.Current]++
	}
	if len(order) == 1 {
		return fmt.Sprintf("%s ×%d", order[0], counts[order[0]])
	}
	sort.Strings(order)
	parts := make([]string, 0, len(order))
	for _, v := range order {
		parts = append(parts, fmt.Sprintf("%s ×%d", v, counts[v]))
	}
	return strings.Join(parts, ", ")
}

// resolveLatestForAll resolves the latest version for every action key,
// honoring config pins the same way --check does.
func resolveLatestForAll(ctx context.Context, keys []string, cfgActions map[string]string) (map[string]string, error) {
	ghToken := resolveToken(lsOpts.token)
	client := github.NewClient(ghToken)

	latest := make(map[string]string, len(keys))
	for _, key := range keys {
		v, err := github.ResolveVersion(ctx, client, key, lsOpts.semver, 0, cfgActions)
		if err != nil {
			return nil, err
		}
		latest[key] = v
	}
	return latest, nil
}
