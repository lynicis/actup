package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
	short   bool
)

func SetVersionInfo(v, c, d string) {
	if v != "" {
		version = v
	}
	if c != "" {
		commit = c
	}
	if d != "" {
		date = d
	}
	rootCmd.Version = FormatVersion()
}

func FormatVersion() string {
	return fmt.Sprintf("actup version %s (commit: %s, built at: %s, %s/%s)", version, commit, date, runtime.GOOS, runtime.GOARCH)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print actup version information",
	RunE: func(cmd *cobra.Command, args []string) error {
		if short {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), version)
			return err
		}
		_, err := fmt.Fprintln(cmd.OutOrStdout(), FormatVersion())
		return err
	},
}

func init() {
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.Version = FormatVersion()
	versionCmd.Flags().BoolVarP(&short, "short", "s", false, "Print only the version number")
	rootCmd.AddCommand(versionCmd)
}
