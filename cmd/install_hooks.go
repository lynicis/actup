package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

const hookScript = `#!/bin/sh
# actup pre-commit hook — checks for outdated GitHub Actions versions
# Installed by: actup install-hooks
exec actup --check --no-tui
`

type installHooksOptions struct {
	path      string
	uninstall bool
	dryRun    bool
	force     bool
}

var installOpts installHooksOptions

var installHooksCmd = &cobra.Command{
	Use:   "install-hooks",
	Short: "Install git pre-commit hooks for actup",
	Long:  "Installs a pre-commit git hook that checks for outdated GitHub Actions versions before each commit.",
	RunE:  runInstallHooks,
}

func init() {
	rootCmd.AddCommand(installHooksCmd)
	installHooksCmd.Flags().StringVarP(&installOpts.path, "path", "p", ".", "Path to the git repository")
	installHooksCmd.Flags().BoolVar(&installOpts.uninstall, "uninstall", false, "Remove the installed hook")
	installHooksCmd.Flags().BoolVar(&installOpts.dryRun, "dry-run", false, "Preview what would be done")
	installHooksCmd.Flags().BoolVarP(&installOpts.force, "force", "f", false, "Overwrite existing hook without prompting")
}

func runInstallHooks(cmd *cobra.Command, args []string) error {
	if installOpts.uninstall {
		return uninstallHook(installOpts.path)
	}
	if installOpts.dryRun {
		return installHookDryRun(installOpts.path)
	}
	return installHook(installOpts.path, installOpts.force)
}

func hookFilePath(repoPath string) string {
	return filepath.Join(repoPath, ".git", "hooks", "pre-commit")
}

func installHook(repoPath string, force bool) error {
	path := hookFilePath(repoPath)

	if _, err := os.Stat(path); err == nil && !force {
		fmt.Fprintf(os.Stderr, "Warning: %s already exists. Use --force to overwrite.\n", path)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create hooks directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(hookScript), 0755); err != nil {
		return fmt.Errorf("write hook: %w", err)
	}

	fmt.Printf("Installed actup pre-commit hook at %s\n", path)
	return nil
}

func installHookDryRun(repoPath string) error {
	path := hookFilePath(repoPath)
	if _, err := os.Stat(path); err == nil {
		fmt.Printf("Would overwrite: %s\n", path)
	} else {
		fmt.Printf("Would create: %s\n", path)
	}
	return nil
}

func uninstallHook(repoPath string) error {
	path := hookFilePath(repoPath)
	if err := os.Remove(path); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No hook found to remove.")
			return nil
		}
		return fmt.Errorf("remove hook: %w", err)
	}
	fmt.Printf("Removed pre-commit hook at %s\n", path)
	return nil
}
