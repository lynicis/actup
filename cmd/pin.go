package cmd

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/parser"
)

var pinOpts struct {
	paths  []string
	dryRun bool
}

var pinCmd = &cobra.Command{
	Use:   "pin <action>[@<ref>]",
	Short: "Record an action version in .actup.yaml without touching workflows",
	Long: `Records a version pin for an action in .actup.yaml. Workflows are not modified;
run "actup sync" later to apply the pin. Like "nvm alias".

When no @ref is given, the version currently used in your workflows is pinned.`,
	Example: `  actup pin actions/checkout@v5
  actup pin actions/checkout@4        # pin to latest v4.x.x at sync time
  actup pin actions/checkout          # pin the version currently in use
  actup pin some-org/some-action@skip # exclude from upgrades`,
	Args: cobra.ExactArgs(1),
	RunE: runPin,
}

var unpinCmd = &cobra.Command{
	Use:     "unpin <action>",
	Aliases: []string{"rm"},
	Short:   "Remove an action pin from .actup.yaml",
	Example: `  actup unpin actions/checkout`,
	Args:    cobra.ExactArgs(1),
	RunE:    runUnpin,
}

func init() {
	pinCmd.Flags().StringArrayVarP(&pinOpts.paths, "path", "p", []string{}, "Paths to workflow files or directories")
	pinCmd.Flags().BoolVar(&pinOpts.dryRun, "dry-run", false, "Preview the pin without writing .actup.yaml")
	unpinCmd.Flags().BoolVar(&pinOpts.dryRun, "dry-run", false, "Preview the unpin without writing .actup.yaml")
	rootCmd.AddCommand(pinCmd)
	rootCmd.AddCommand(unpinCmd)
}

func runPin(cmd *cobra.Command, args []string) error {
	spec, err := parseActionSpec(args[0])
	if err != nil {
		return err
	}

	ref := spec.Ref
	if ref == "" {
		actions, err := loadWorkflowActions(context.Background(), pinOpts.paths)
		if err != nil {
			return fmt.Errorf("cannot determine current version; specify one explicitly (%s@<ref>): %w", spec.Key, err)
		}
		refs := parser.GroupActions(actions)[spec.Key]
		if len(refs) == 0 {
			return fmt.Errorf("%s is not used in any workflow; specify a version explicitly: actup pin %s@<ref>", spec.Key, spec.Key)
		}
		ref = mostCommonVersion(refs)
	}

	if pinOpts.dryRun {
		fmt.Printf("would pin %s: %s in .actup.yaml\n", spec.Key, ref)
		return nil
	}

	if err := savePin(spec.Key, ref); err != nil {
		return err
	}
	fmt.Printf("✓ pinned %s: %s in .actup.yaml\n", spec.Key, ref)
	return nil
}

func runUnpin(cmd *cobra.Command, args []string) error {
	spec, err := parseActionSpec(args[0])
	if err != nil {
		return err
	}

	cfg, err := config.LoadDefault()
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}
	if cfg == nil || cfg.Actions[spec.Key] == "" {
		fmt.Printf("⏭ %s is not pinned\n", spec.Key)
		return nil
	}

	if pinOpts.dryRun {
		fmt.Printf("would unpin %s in .actup.yaml\n", spec.Key)
		return nil
	}

	delete(cfg.Actions, spec.Key)
	if err := config.SaveDefault(cfg); err != nil {
		return err
	}
	fmt.Printf("✓ unpinned %s\n", spec.Key)
	return nil
}
