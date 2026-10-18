package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/vulnscanner"
)

var auditOpts struct {
	paths []string
	json  bool
}

var auditCmd = &cobra.Command{
	Use:   "audit",
	Short: "Scan workflows for known vulnerable action versions (exits non-zero if found)",
	Long: `Checks every action reference in the workflow files against the OSV.dev
vulnerability database (GitHub Actions ecosystem advisories) and reports
usages of versions known to be vulnerable.

Findings include the advisory ID, severity, and the first fixed version
when known. Exits with status 1 when at least one vulnerable usage is
found, and 2 when the OSV database cannot be reached.

Vulnerability data comes from OSV.dev; results require network access.
The scan skips actions listed as "skip" in .actup.yaml.`,
	Example: `  actup audit
  actup audit --json
  actup audit --path .github/workflows`,
	RunE: runAudit,
}

func init() {
	auditCmd.Flags().StringArrayVarP(&auditOpts.paths, "path", "p", []string{}, "Paths to workflow files or directories")
	auditCmd.Flags().BoolVar(&auditOpts.json, "json", false, "Output findings as JSON")
	rootCmd.AddCommand(auditCmd)
}

// auditJSONFinding is the JSON representation of one vulnerable usage site.
type auditJSONFinding struct {
	Action string          `json:"action"`
	Ref    string          `json:"ref"`
	File   string          `json:"file"`
	Line   int             `json:"line"`
	Vulns  []auditJSONVuln `json:"vulns"`
}

type auditJSONVuln struct {
	ID       string `json:"id"`
	Severity string `json:"severity,omitempty"`
	Summary  string `json:"summary,omitempty"`
	Fixed    string `json:"fixed,omitempty"`
	URL      string `json:"url,omitempty"`
}

func runAudit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.LoadDefault()
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Failed to load config: %v\n", err)
		cfg = nil
	}

	actions, err := loadWorkflowActions(ctx, auditOpts.paths)
	if err != nil {
		return err
	}
	actions = filterSkippedActions(actions, cfg)
	if len(actions) == 0 {
		if auditOpts.json {
			fmt.Println(`{"findings":[]}`)
		} else {
			fmt.Println("No actions found.")
		}
		return nil
	}

	client := vulnscanner.NewClient("")
	findings, err := client.Scan(ctx, actions)
	if err != nil {
		if auditOpts.json {
			return fmt.Errorf("vulnerability scan failed: %w", err)
		}
		fmt.Fprintf(os.Stderr, "⚠ Vulnerability scan failed (network unavailable?): %v\n", err)
		os.Exit(2)
	}

	if auditOpts.json {
		return outputAuditJSON(findings)
	}

	if len(findings) == 0 {
		fmt.Println("No known vulnerable action versions found.")
		return nil
	}

	for _, f := range findings {
		action := f.Owner + "/" + f.Repo
		fmt.Printf("%s:%d: %s@%s\n", f.File, f.Line, action, f.Ref)
		for _, v := range f.Vulns {
			severity := v.Severity
			if severity == "" {
				severity = "UNKNOWN"
			}
			fixed := ""
			if v.Fixed != "" {
				fixed = fmt.Sprintf(", fixed in %s", v.Fixed)
			}
			fmt.Printf("  ⚠ %s [%s]%s %s\n", v.ID, severity, fixed, v.Summary)
			if v.URL != "" {
				fmt.Printf("    %s\n", v.URL)
			}
		}
	}

	fmt.Printf("\n%d vulnerable action usage(s) found.\n", len(findings))
	fmt.Fprintln(os.Stderr, "Run 'actup audit --help' for options, or upgrade with 'actup use <action>@latest'.")
	os.Exit(1)
	return nil
}

// outputAuditJSON writes findings as JSON to stdout and exits 1 when any
// were found. No extra hints are printed so the output stays machine-readable.
func outputAuditJSON(findings []vulnscanner.Finding) error {
	out := struct {
		Findings []auditJSONFinding `json:"findings"`
	}{
		Findings: make([]auditJSONFinding, 0, len(findings)),
	}

	for _, f := range findings {
		jf := auditJSONFinding{
			Action: f.Owner + "/" + f.Repo,
			Ref:    f.Ref,
			File:   f.File,
			Line:   f.Line,
			Vulns:  make([]auditJSONVuln, 0, len(f.Vulns)),
		}
		for _, v := range f.Vulns {
			jf.Vulns = append(jf.Vulns, auditJSONVuln{
				ID:       v.ID,
				Severity: v.Severity,
				Summary:  v.Summary,
				Fixed:    v.Fixed,
				URL:      v.URL,
			})
		}
		out.Findings = append(out.Findings, jf)
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(&out); err != nil {
		return err
	}

	if len(findings) > 0 {
		os.Exit(1)
	}
	return nil
}
