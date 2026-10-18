package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/lynicis/actup/internal/breakingchanges"
	"github.com/lynicis/actup/internal/github"
	"github.com/lynicis/actup/internal/parser"
	"github.com/lynicis/actup/internal/upgrader"
)

type useOptions struct {
	paths  []string
	token  string
	semver bool
	force  bool
	dryRun bool
	noSave bool
}

var useOpts useOptions

var useCmd = &cobra.Command{
	Use:   "use <action>[@<ref>]...",
	Short: "Set the version of one or more actions across all workflows",
	Long: `Sets the version of one or more actions across every workflow that uses them,
and records the version as a pin in .actup.yaml (like "nvm use" or "fvm use").

Ref resolution:
  @v5        written verbatim (the standard moving major tag)
  @5         resolved to the latest tag within major v5 (e.g. v5.3.1)
  @latest    resolved to the latest tag (major tag, or full semver with --semver)
  @v5.3.1    written verbatim (exact version)
  (no @ref)  same as @latest

Use --no-save to update workflows without touching .actup.yaml.`,
	Example: `  actup use actions/checkout@v5
  actup use actions/checkout@5 actions/setup-go@latest
  actup use actions/checkout@v5 --no-save
  actup use actions/checkout@v5 --dry-run`,
	Args: cobra.MinimumNArgs(1),
	RunE: runUse,
}

func init() {
	useCmd.Flags().StringArrayVarP(&useOpts.paths, "path", "p", []string{}, "Paths to workflow files or directories")
	useCmd.Flags().StringVarP(&useOpts.token, "token", "t", "", "GitHub PAT (fallback: GITHUB_TOKEN env var, then gh auth token)")
	useCmd.Flags().BoolVarP(&useOpts.semver, "semver", "s", false, "Resolve @latest to the latest full semver tag instead of the latest major tag")
	useCmd.Flags().BoolVarP(&useOpts.force, "force", "f", false, "Apply upgrades with breaking changes without prompting")
	useCmd.Flags().BoolVar(&useOpts.dryRun, "dry-run", false, "Preview changes without writing files or saving pins")
	useCmd.Flags().BoolVar(&useOpts.noSave, "no-save", false, "Do not record the version in .actup.yaml")
	rootCmd.AddCommand(useCmd)
}

func runUse(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	specs := make([]actionSpec, 0, len(args))
	for _, arg := range args {
		spec, err := parseActionSpec(arg)
		if err != nil {
			return err
		}
		specs = append(specs, spec)
	}

	return doUse(ctx, specs, useOpts)
}

func doUse(ctx context.Context, specs []actionSpec, opts useOptions) error {
	actions, err := loadWorkflowActions(ctx, opts.paths)
	if err != nil {
		return err
	}
	grouped := parser.GroupActions(actions)

	registry, err := breakingchanges.LoadRegistry()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to load breaking-change registry: %v\n", err)
	}

	client := clientForSpecs(specs, opts.token)
	reader := bufio.NewReader(os.Stdin)
	isTTY := isTerminal()

	// Warn about vulnerable versions currently in use before changing anything.
	printVulnWarnings(ctx, actions)

	updatedActions := 0
	for _, spec := range specs {
		target, err := resolveSpecRef(ctx, client, spec, opts.semver)
		if err != nil {
			fmt.Printf("✗ %s: %v\n", spec.Key, err)
			continue
		}

		refs := grouped[spec.Key]

		proceed := true
		if len(refs) > 0 {
			proceed = checkUseBreaking(spec.Key, refs, target, registry, opts, reader, isTTY)
		}
		if !proceed {
			continue
		}

		// Guard against pinning a version that is itself vulnerable.
		if len(refs) > 0 && !confirmVulnerableTarget(ctx, spec.Key, target, opts.dryRun, opts.force, reader, isTTY) {
			fmt.Printf("⏭ %s: skipped — target %s is vulnerable\n", spec.Key, target)
			continue
		}

		if len(refs) > 0 {
			if changed := applyUseToFileRefs(spec.Key, refs, target, opts.dryRun); changed {
				updatedActions++
			}
		} else {
			fmt.Printf("ⓘ %s: not used in any workflow\n", spec.Key)
		}

		if opts.dryRun {
			fmt.Printf("would pin %s: %s in .actup.yaml\n", spec.Key, target)
			continue
		}
		if opts.noSave {
			continue
		}
		if err := savePin(spec.Key, target); err != nil {
			fmt.Fprintf(os.Stderr, "⚠ Failed to save pin for %s: %v\n", spec.Key, err)
			continue
		}
	}

	if opts.dryRun {
		fmt.Println("\nDry run: no files were modified. Run without --dry-run to apply.")
	} else if updatedActions > 0 || !opts.noSave {
		fmt.Printf("\n%d action(s) updated.", updatedActions)
		if !opts.noSave {
			fmt.Printf(" Pins saved to .actup.yaml.")
		}
		fmt.Println()
	}
	return nil
}

// clientForSpecs returns a GitHub client, or nil when no spec needs one
// (all specs carry a verbatim ref, so no API calls are required).
func clientForSpecs(specs []actionSpec, ghToken string) *github.Client {
	needsClient := false
	for _, spec := range specs {
		if spec.Ref == "" || spec.Ref == "latest" {
			needsClient = true
			break
		}
		if _, err := strconv.Atoi(spec.Ref); err == nil {
			needsClient = true
			break
		}
	}
	if !needsClient {
		return nil
	}
	return github.NewClient(resolveToken(ghToken))
}

// checkUseBreaking warns about breaking changes for every distinct current
// version used with the action and returns whether to proceed.
func checkUseBreaking(key string, refs []parser.ActionRef, target string, registry *breakingchanges.Registry, opts useOptions, reader *bufio.Reader, isTTY bool) bool {
	proceed, _ := confirmBreakingAll(key, refs, target, registry, opts.dryRun, opts.force, reader, isTTY)
	return proceed
}

// applyUseToFileRefs rewrites every workflow reference for the action to
// target and reports whether anything changed.
func applyUseToFileRefs(key string, refs []parser.ActionRef, target string, dryRun bool) bool {
	uniqueCurrents := map[string]bool{}
	for _, ref := range refs {
		uniqueCurrents[ref.Current] = true
	}
	if len(uniqueCurrents) == 1 {
		for current := range uniqueCurrents {
			if current == target {
				fmt.Printf("⏭ %s: already at %s\n", key, target)
				return false
			}
		}
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
		return false
	}

	if dryRun {
		for _, u := range upgrades {
			printUseDiff(u)
		}
		fmt.Printf("would update %s → %s (%d reference(s))\n", key, target, len(upgrades))
		return true
	}

	if err := upgrader.ApplyAllUpgrades(upgrades, false); err != nil {
		fmt.Printf("✗ %s: %v\n", key, err)
		return false
	}
	fmt.Printf("✓ %s → %s (%d reference(s))\n", key, target, len(upgrades))
	return true
}

func printUseDiff(u upgrader.Upgrade) {
	a := u.Action
	path := a.Path
	if path == "" {
		path = a.Owner + "/" + a.Repo
	}
	oldLine := fmt.Sprintf("uses: %s@%s", path, a.Current)
	newLine := fmt.Sprintf("uses: %s@%s", path, u.NewTag)
	fmt.Printf("--- a/%s\n+++ b/%s\n@@ -%d,1 +%d,1 @@\n-%s\n+%s\n", a.File, a.File, a.Line, a.Line, oldLine, newLine)
}
