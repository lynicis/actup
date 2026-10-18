package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/token"
)

var outdatedOpts struct {
	paths  []string
	token  string
	semver bool
	major  int
}

var outdatedCmd = &cobra.Command{
	Use:   "outdated",
	Short: "List actions that can be upgraded (exits non-zero if any)",
	Long: `Prints every workflow action reference whose version differs from the target
version actup would upgrade it to, then exits with status 1 if any were
found — like "nvm outdated". Config pins are honored: an action pinned to
an exact version is never outdated, and a major pin only compares within
that major.

This is the subcommand form of the --check flag.`,
	Example: `  actup outdated
  actup outdated --semver`,
	RunE: runOutdated,
}

func init() {
	outdatedCmd.Flags().StringArrayVarP(&outdatedOpts.paths, "path", "p", []string{}, "Paths to workflow files or directories")
	outdatedCmd.Flags().StringVarP(&outdatedOpts.token, "token", "t", "", "GitHub PAT (fallback: GITHUB_TOKEN env var, then gh auth token)")
	outdatedCmd.Flags().BoolVarP(&outdatedOpts.semver, "semver", "s", false, "Resolve latest versions as full semver tags instead of major tags")
	outdatedCmd.Flags().IntVarP(&outdatedOpts.major, "major", "m", 0, "Compare within a specific major version only (e.g. 4 for latest v4.x.x)")
	rootCmd.AddCommand(outdatedCmd)
}

func runOutdated(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	ghToken := outdatedOpts.token
	if ghToken == "" {
		ghToken = token.Resolve("")
	}

	cfg, err := config.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to load config: %v\n", err)
		cfg = nil
	}

	actions, err := loadWorkflowActions(ctx, outdatedOpts.paths)
	if err != nil {
		return err
	}
	actions = filterSkippedActions(actions, cfg)
	if len(actions) == 0 {
		fmt.Println("No actions to check found.")
		return nil
	}

	// Surface vulnerable versions alongside the outdated report.
	printVulnWarnings(ctx, actions)

	return runCheck(ctx, actions, ghToken, outdatedOpts.semver, outdatedOpts.major, cfg)
}
