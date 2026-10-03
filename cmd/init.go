package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/parser"
)

var initOpts struct {
	paths  []string
	force  bool
	dryRun bool
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Bootstrap .actup.yaml with the action versions currently in use",
	Long: `Creates .actup.yaml pinning every action to the version currently used in
your workflows. The manifest then becomes the source of truth for
"actup sync".

If .actup.yaml already exists it is left untouched unless --force is given,
in which case missing entries are added and existing pins are preserved.`,
	Example: `  actup init
  actup init --dry-run
  actup init --force   # merge into an existing .actup.yaml`,
	RunE: runInit,
}

func init() {
	initCmd.Flags().StringArrayVarP(&initOpts.paths, "path", "p", []string{}, "Paths to workflow files or directories")
	initCmd.Flags().BoolVarP(&initOpts.force, "force", "f", false, "Merge into an existing .actup.yaml instead of refusing")
	initCmd.Flags().BoolVar(&initOpts.dryRun, "dry-run", false, "Preview the generated config without writing it")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	return doInit(context.Background(), initOpts)
}

func doInit(ctx context.Context, opts initOptions) error {
	actions, err := loadWorkflowActions(ctx, opts.paths)
	if err != nil {
		return err
	}

	existing, err := config.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to load existing config: %v\n", err)
	}
	if existing != nil && !opts.force && !opts.dryRun {
		return fmt.Errorf(".actup.yaml already exists; use --force to merge new pins into it")
	}

	cfg := existing
	if cfg == nil {
		cfg = &config.Config{}
	}
	if cfg.Actions == nil {
		cfg.Actions = map[string]string{}
	}

	grouped := parser.GroupActions(actions)
	added := 0
	for key, refs := range grouped {
		if _, exists := cfg.Actions[key]; exists {
			continue
		}
		cfg.Actions[key] = mostCommonVersion(refs)
		added++
	}

	if opts.dryRun {
		fmt.Println("Would write .actup.yaml:")
		fmt.Println()
		fmt.Print(config.Render(cfg))
		return nil
	}

	if err := config.SaveDefault(cfg); err != nil {
		return err
	}
	fmt.Printf("✓ wrote .actup.yaml with %d pinned action(s)\n", added)
	return nil
}

type initOptions struct {
	paths  []string
	force  bool
	dryRun bool
}
