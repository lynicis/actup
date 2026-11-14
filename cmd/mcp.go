package cmd

import (
	"context"
	"strings"
	"sync"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/spf13/cobra"

	"github.com/lynicis/actup/internal/github"
	"github.com/lynicis/actup/internal/lockfile"
	"github.com/lynicis/actup/internal/parser"
	"github.com/lynicis/actup/internal/vulnscanner"
)

var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run the actup MCP server over stdio for AI coding agents",
	Long: `Run the actup MCP (Model Context Protocol) server over stdio for AI coding agents.

Run it from the repository root; the tools list, upgrade, and pin the GitHub
Actions references found under .github/workflows. The GitHub token is resolved
from GITHUB_TOKEN or "gh auth token".`,
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMCPTransport(cmd.Context(), &mcp.StdioTransport{})
	},
}

func init() {
	rootCmd.AddCommand(mcpCmd)
}

// mcpNewGHClient is a test seam: tests swap it for github.NewClientWithBaseURL
// pointed at an httptest server.
var mcpNewGHClient = github.NewClient

// mcpVulnScan is a test seam for the OSV.dev scan.
var mcpVulnScan = func(ctx context.Context, actions []parser.ActionRef) ([]vulnscanner.Finding, error) {
	return vulnscanner.NewClient().Scan(ctx, actions)
}

// mcpWriteMu serializes the mutating tools; read tools stay concurrent.
var mcpWriteMu sync.Mutex

func newMCPServer() *mcp.Server {
	srv := mcp.NewServer(&mcp.Implementation{Name: "actup", Version: version}, nil)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "list_actions",
		Description: "List every GitHub Action used in the repository's workflows, with versions, usage sites, pins, and actions.lock status. No network.",
	}, handleListActions)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "check_outdated",
		Description: "Resolve the latest version of every workflow action (or the given owner/repo globs) and report which are outdated, with known breaking changes. Uses the GitHub API.",
	}, handleCheckOutdated)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "audit_vulnerabilities",
		Description: "Audit direct and actions.lock-transitive action versions against OSV.dev. Reports osv_unreachable instead of failing when OSV cannot be reached.",
	}, handleAuditVulnerabilities)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "upgrade_actions",
		Description: "Upgrade workflow references for the given owner/repo[@ref] specs (empty = every discovered action, honoring .actup.yaml pins) and record the result in a managed manifest. Skips breaking-change and vulnerable targets unless force is set; dry_run previews without writing.",
	}, handleUpgradeActions)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "pin_action",
		Description: "Record a version constraint for an action in .actup.yaml. An empty ref pins the most common version currently in use.",
	}, handlePinAction)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "unpin_action",
		Description: "Remove an action's pin from .actup.yaml.",
	}, handleUnpinAction)
	mcp.AddTool(srv, &mcp.Tool{
		Name:        "init_config",
		Description: "Create .actup.yaml recording every action's current version (skipping SHA pins without a version comment). Merges into an existing manifest only with force. Returns the rendered YAML either way.",
	}, handleInitConfig)
	return srv
}

// runMCPTransport runs the server until the transport closes. A client that
// disconnects while requests are in flight surfaces as a "server is closing"
// error from the SDK; for a stdio server that is a normal shutdown, not a
// failure.
func runMCPTransport(ctx context.Context, t mcp.Transport) error {
	err := newMCPServer().Run(ctx, t)
	if err != nil && strings.Contains(err.Error(), "server is closing") {
		return nil
	}
	return err
}

// lockStatusString renders a lockfile status for structured output.
func lockStatusString(s lockfile.Status) string {
	switch s {
	case lockfile.StatusLocked:
		return "locked"
	case lockfile.StatusStale:
		return "stale"
	default:
		return "none"
	}
}
