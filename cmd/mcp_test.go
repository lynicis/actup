package cmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/github"
	"github.com/lynicis/actup/internal/parser"
	"github.com/lynicis/actup/internal/vulnscanner"
)

const testSHA = "1111111111111111111111111111111111111111"

func TestMCPListActions(t *testing.T) {
	dir := t.TempDir()
	writeWorkflow(t, dir, "one.yml", "name: one\non: push\njobs:\n  b:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v3\n      - uses: actions/cache@v4\n")
	writeWorkflow(t, dir, "two.yml", "name: two\non: push\njobs:\n  b:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v4\n      - uses: actions/setup-go@"+testSHA+" # v2\n")
	t.Chdir(dir)

	res, out, err := handleListActions(t.Context(), &mcp.CallToolRequest{}, listActionsIn{})
	if err != nil {
		t.Fatalf("handleListActions: %v", err)
	}
	if res == nil || len(res.Content) == 0 {
		t.Fatal("expected a result with summary content")
	}

	var keys []string
	for _, a := range out.Actions {
		keys = append(keys, a.Key)
	}
	want := []string{"actions/cache", "actions/checkout", "actions/setup-go"}
	if !slices.Equal(keys, want) {
		t.Fatalf("keys = %v, want %v", keys, want)
	}
	if out.TotalSites != 4 {
		t.Errorf("total_sites = %d, want 4", out.TotalSites)
	}

	var checkout, setupgo listActionsAction
	for _, a := range out.Actions {
		switch a.Key {
		case "actions/checkout":
			checkout = a
		case "actions/setup-go":
			setupgo = a
		}
	}

	if len(checkout.Versions) != 2 {
		t.Fatalf("checkout versions = %+v, want 2 entries", checkout.Versions)
	}
	if checkout.Versions[0] != (listActionsVersion{Version: "v3", Count: 1, TagComment: ""}) ||
		checkout.Versions[1] != (listActionsVersion{Version: "v4", Count: 1}) {
		t.Errorf("checkout versions = %+v, want v3 ×1 then v4 ×1", checkout.Versions)
	}
	if len(checkout.Sites) != 2 {
		t.Fatalf("checkout sites = %+v, want 2", checkout.Sites)
	}
	if !strings.HasSuffix(checkout.Sites[0].File, "one.yml") || checkout.Sites[0].Line != 7 || checkout.Sites[0].Current != "v3" {
		t.Errorf("checkout site[0] = %+v, want one.yml:7 v3", checkout.Sites[0])
	}
	if !strings.HasSuffix(checkout.Sites[1].File, "two.yml") || checkout.Sites[1].Line != 7 || checkout.Sites[1].Current != "v4" {
		t.Errorf("checkout site[1] = %+v, want two.yml:7 v4", checkout.Sites[1])
	}

	if len(setupgo.Versions) != 1 || !setupgo.Versions[0].Pinned || setupgo.Versions[0].Version != "v2" || setupgo.Versions[0].TagComment != "v2" {
		t.Errorf("setup-go versions = %+v, want one pinned v2", setupgo.Versions)
	}
	if len(setupgo.Sites) != 1 || setupgo.Sites[0].SHA != testSHA || setupgo.Sites[0].TagComment != "v2" {
		t.Errorf("setup-go sites = %+v, want SHA pin with v2 comment", setupgo.Sites)
	}

	for _, a := range out.Actions {
		if a.Pin != "" {
			t.Errorf("%s pin = %q, want empty (no .actup.yaml)", a.Key, a.Pin)
		}
		if a.LockStatus != "none" {
			t.Errorf("%s lock_status = %q, want none", a.Key, a.LockStatus)
		}
	}
}

func TestMCPCheckOutdated(t *testing.T) {
	dir := t.TempDir()
	writeWorkflow(t, dir, "ci.yml", "name: ci\non: push\njobs:\n  b:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v3\n      - uses: actions/setup-go@v4\n")
	t.Chdir(dir)
	t.Setenv("GITHUB_TOKEN", "test")

	const sha = "dae09b1f80164e2d312981ce81045e7e1f4640d2"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch {
		case strings.HasSuffix(r.URL.Path, "/actions/checkout/tags"):
			_, _ = fmt.Fprintf(w, `[{"name":"v3","commit":{"sha":"%s"}},{"name":"v4","commit":{"sha":"%s"}},{"name":"v5","commit":{"sha":"%s"}}]`, sha, sha, sha)
		case strings.HasSuffix(r.URL.Path, "/actions/setup-go/tags"):
			_, _ = fmt.Fprintf(w, `[{"name":"v4","commit":{"sha":"%s"}}]`, sha)
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	orig := mcpNewGHClient
	mcpNewGHClient = func(string) *github.Client {
		c, err := github.NewClientWithBaseURL("", srv.URL)
		if err != nil {
			t.Fatalf("NewClientWithBaseURL: %v", err)
		}
		return c
	}
	t.Cleanup(func() { mcpNewGHClient = orig })

	res, out, err := handleCheckOutdated(t.Context(), &mcp.CallToolRequest{}, checkOutdatedIn{})
	if err != nil {
		t.Fatalf("handleCheckOutdated: %v", err)
	}
	if res == nil || len(res.Content) == 0 {
		t.Fatal("expected summary content")
	}
	if len(out.Results) != 2 {
		t.Fatalf("results = %+v, want 2", out.Results)
	}
	byKey := map[string]checkOutdatedResult{}
	for _, r := range out.Results {
		byKey[r.Key] = r
	}
	co := byKey["actions/checkout"]
	if co.Status != "outdated" || co.Latest != "v5" || co.Current != "v3" {
		t.Errorf("checkout = %+v, want outdated v3 → v5", co)
	}
	if len(co.BreakingChanges) == 0 {
		t.Errorf("checkout breaking_changes empty, want the v3→v4 registry entry")
	}
	sg := byKey["actions/setup-go"]
	if sg.Status != "up_to_date" || sg.Latest != "v4" {
		t.Errorf("setup-go = %+v, want up_to_date at v4", sg)
	}
}

func TestMCPAuditVulnerabilities(t *testing.T) {
	dir := t.TempDir()
	writeWorkflow(t, dir, "ci.yml", "name: ci\non: push\njobs:\n  b:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v3\n      - uses: foo/bar@"+testSHA+"\n")
	t.Chdir(dir)

	orig := mcpVulnScan
	mcpVulnScan = func(ctx context.Context, actions []parser.ActionRef) ([]vulnscanner.Finding, error) {
		if len(actions) != 1 || actions[0].Owner != "actions" || actions[0].Repo != "checkout" {
			t.Errorf("scan received %+v, want only actions/checkout", actions)
		}
		return []vulnscanner.Finding{{
			Owner: "actions", Repo: "checkout", Ref: "v3",
			File: actions[0].File, Line: actions[0].Line,
			Vulns: []vulnscanner.Vulnerability{{ID: "GHSA-test", Severity: "HIGH", Summary: "bad thing", Fixed: "v4", URL: "https://osv.dev/vulnerability/GHSA-test"}},
		}}, nil
	}
	t.Cleanup(func() { mcpVulnScan = orig })

	res, out, err := handleAuditVulnerabilities(t.Context(), &mcp.CallToolRequest{}, auditVulnsIn{})
	if err != nil {
		t.Fatalf("handleAuditVulnerabilities: %v", err)
	}
	if res == nil || len(res.Content) == 0 {
		t.Fatal("expected summary content")
	}
	if out.OsvUnreachable {
		t.Error("osv_unreachable = true, want false")
	}
	if len(out.Findings) != 1 {
		t.Fatalf("findings = %+v, want 1", out.Findings)
	}
	f := out.Findings[0]
	if f.Action != "actions/checkout" || f.Ref != "v3" || f.Transitive || f.Line != 7 {
		t.Errorf("finding = %+v, want actions/checkout v3 at line 7", f)
	}
	if len(f.Vulns) != 1 || f.Vulns[0].ID != "GHSA-test" || f.Vulns[0].Fixed != "v4" {
		t.Errorf("vulns = %+v, want GHSA-test fixed in v4", f.Vulns)
	}
	if len(out.UnknownVersion) != 1 {
		t.Fatalf("unknown_version = %+v, want 1", out.UnknownVersion)
	}
	u := out.UnknownVersion[0]
	if u.Action != "foo/bar" || u.Ref != testSHA || u.Line != 8 {
		t.Errorf("unknown_version = %+v, want foo/bar SHA at line 8", u)
	}
	if len(out.SuggestedFixes) != 1 {
		t.Fatalf("suggested_fixes = %+v, want 1", out.SuggestedFixes)
	}
	if fix := out.SuggestedFixes[0]; fix.Action != "actions/checkout" || fix.Current != "v3" || fix.MinFixed != "v4" {
		t.Errorf("suggested_fix = %+v, want checkout v3 → v4", fix)
	}
	if len(out.Unfixable) != 0 {
		t.Errorf("unfixable = %+v, want none", out.Unfixable)
	}
}

const checkoutV3Workflow = "name: ci\non: push\njobs:\n  b:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v3\n"

func assertWorkflowContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read workflow: %v", err)
	}
	if !strings.Contains(string(data), want) {
		t.Errorf("workflow %s does not contain %q:\n%s", path, want, data)
	}
}

func assertWorkflowNotContains(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read workflow: %v", err)
	}
	if strings.Contains(string(data), want) {
		t.Errorf("workflow %s unexpectedly contains %q:\n%s", path, want, data)
	}
}

func TestMCPUpgradeActions(t *testing.T) {
	const sha = "dae09b1f80164e2d312981ce81045e7e1f4640d2"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.HasSuffix(r.URL.Path, "/actions/checkout/tags") {
			_, _ = fmt.Fprintf(w, `[{"name":"v3","commit":{"sha":"%s"}},{"name":"v4","commit":{"sha":"%s"}},{"name":"v5","commit":{"sha":"%s"}}]`, sha, sha, sha)
			return
		}
		t.Errorf("unexpected path %s", r.URL.Path)
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	origClient, origScan := mcpNewGHClient, mcpVulnScan
	mcpNewGHClient = func(string) *github.Client {
		c, err := github.NewClientWithBaseURL("", srv.URL)
		if err != nil {
			t.Fatalf("NewClientWithBaseURL: %v", err)
		}
		return c
	}
	mcpVulnScan = func(context.Context, []parser.ActionRef) ([]vulnscanner.Finding, error) {
		return nil, nil
	}
	t.Cleanup(func() { mcpNewGHClient, mcpVulnScan = origClient, origScan })

	writeCI := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		path := writeWorkflow(t, dir, "ci.yml", checkoutV3Workflow)
		t.Chdir(dir)
		t.Setenv("GITHUB_TOKEN", "test")
		return path
	}

	t.Run("dry run plans without writing", func(t *testing.T) {
		path := writeCI(t)
		res, out, err := handleUpgradeActions(t.Context(), &mcp.CallToolRequest{}, upgradeActionsIn{
			Actions: []string{"actions/checkout@v5"},
			DryRun:  true,
		})
		if err != nil {
			t.Fatalf("handleUpgradeActions: %v", err)
		}
		if res == nil || len(res.Content) == 0 {
			t.Fatal("expected summary content")
		}
		if len(out.Results) != 1 {
			t.Fatalf("results = %+v, want 1", out.Results)
		}
		r := out.Results[0]
		if r.Status != "upgraded" || r.Target != "v5" || r.RefsUpdated != 1 {
			t.Errorf("result = %+v, want planned upgrade to v5", r)
		}
		if len(r.BreakingChanges) == 0 {
			t.Errorf("dry run should still list the breaking changes")
		}
		if len(out.FilesModified) != 0 || out.ConfigUpdated {
			t.Errorf("dry run must not report writes: %+v", out)
		}
		assertWorkflowContains(t, path, "actions/checkout@v3")
		assertWorkflowNotContains(t, path, "@v5")
	})

	t.Run("skips breaking changes without force", func(t *testing.T) {
		path := writeCI(t)
		_, out, err := handleUpgradeActions(t.Context(), &mcp.CallToolRequest{}, upgradeActionsIn{
			Actions: []string{"actions/checkout@v5"},
		})
		if err != nil {
			t.Fatalf("handleUpgradeActions: %v", err)
		}
		r := out.Results[0]
		if r.Status != "skipped_breaking" || len(r.BreakingChanges) == 0 || r.RefsUpdated != 0 {
			t.Errorf("result = %+v, want skipped_breaking", r)
		}
		assertWorkflowContains(t, path, "actions/checkout@v3")
	})

	t.Run("skips vulnerable targets without force", func(t *testing.T) {
		dir := t.TempDir()
		path := writeWorkflow(t, dir, "ci.yml", "name: ci\non: push\njobs:\n  b:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: foo/bar@v1\n")
		t.Chdir(dir)

		orig := mcpVulnScan
		mcpVulnScan = func(context.Context, []parser.ActionRef) ([]vulnscanner.Finding, error) {
			return []vulnscanner.Finding{{Owner: "foo", Repo: "bar", Ref: "v2", Vulns: []vulnscanner.Vulnerability{{ID: "GHSA-x", Fixed: "v3"}}}}, nil
		}
		t.Cleanup(func() { mcpVulnScan = orig })

		_, out, err := handleUpgradeActions(t.Context(), &mcp.CallToolRequest{}, upgradeActionsIn{
			Actions: []string{"foo/bar@v2"},
		})
		if err != nil {
			t.Fatalf("handleUpgradeActions: %v", err)
		}
		if r := out.Results[0]; r.Status != "skipped_vulnerable" || r.RefsUpdated != 0 {
			t.Errorf("result = %+v, want skipped_vulnerable", r)
		}
		assertWorkflowContains(t, path, "foo/bar@v1")
	})

	t.Run("force upgrades the workflow", func(t *testing.T) {
		path := writeCI(t)
		_, out, err := handleUpgradeActions(t.Context(), &mcp.CallToolRequest{}, upgradeActionsIn{
			Actions: []string{"actions/checkout@v5"},
			Force:   true,
		})
		if err != nil {
			t.Fatalf("handleUpgradeActions: %v", err)
		}
		r := out.Results[0]
		if r.Status != "upgraded" || r.RefsUpdated != 1 {
			t.Errorf("result = %+v, want upgraded", r)
		}
		if len(out.FilesModified) != 1 || !strings.HasSuffix(out.FilesModified[0], "ci.yml") {
			t.Errorf("files_modified = %v, want the workflow file", out.FilesModified)
		}
		if out.ConfigUpdated {
			t.Error("config_updated = true, want false without a manifest")
		}
		assertWorkflowContains(t, path, "actions/checkout@v5")
		assertWorkflowNotContains(t, path, "@v3")
	})

	t.Run("records managed manifest in all mode", func(t *testing.T) {
		dir := t.TempDir()
		path := writeWorkflow(t, dir, "ci.yml", checkoutV3Workflow)
		if err := os.WriteFile(filepath.Join(dir, ".actup.yaml"), []byte("pins:\n  actions/checkout: v4\n"), 0600); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
		t.Chdir(dir)
		t.Setenv("GITHUB_TOKEN", "test")

		_, out, err := handleUpgradeActions(t.Context(), &mcp.CallToolRequest{}, upgradeActionsIn{Force: true})
		if err != nil {
			t.Fatalf("handleUpgradeActions: %v", err)
		}
		r := out.Results[0]
		if r.Status != "upgraded" || r.Target != "v4" {
			t.Fatalf("result = %+v, want pin-honoring upgrade to v4", r)
		}
		if !out.ConfigUpdated {
			t.Error("config_updated = false, want true for a managed manifest")
		}
		cfg, err := config.LoadDefault()
		if err != nil {
			t.Fatalf("reload config: %v", err)
		}
		if cfg.Actions["actions/checkout"] != "v4" {
			t.Errorf("Actions[checkout] = %q, want v4", cfg.Actions["actions/checkout"])
		}
		if cfg.Pins["actions/checkout"] != "v4" {
			t.Errorf("Pins[checkout] = %q, want the pin to survive", cfg.Pins["actions/checkout"])
		}
		assertWorkflowContains(t, path, "actions/checkout@v4")
	})
}

func TestMCPPinUnpinInit(t *testing.T) {
	t.Run("pin_action writes .actup.yaml", func(t *testing.T) {
		dir := t.TempDir()
		writeWorkflow(t, dir, "ci.yml", checkoutV3Workflow)
		t.Chdir(dir)

		res, out, err := handlePinAction(t.Context(), &mcp.CallToolRequest{}, pinActionIn{Action: "actions/checkout"})
		if err != nil {
			t.Fatalf("handlePinAction: %v", err)
		}
		if res == nil || len(res.Content) == 0 {
			t.Fatal("expected summary content")
		}
		if !out.Pinned || out.Ref != "v3" {
			t.Errorf("out = %+v, want pinned v3", out)
		}
		cfg, err := config.LoadDefault()
		if err != nil {
			t.Fatalf("reload config: %v", err)
		}
		if cfg.Actions["actions/checkout"] != "v3" {
			t.Errorf("Actions[checkout] = %q, want v3", cfg.Actions["actions/checkout"])
		}
	})

	t.Run("pin_action reports unknown SHA pins", func(t *testing.T) {
		dir := t.TempDir()
		writeWorkflow(t, dir, "ci.yml", "name: ci\non: push\njobs:\n  b:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: foo/bar@"+testSHA+"\n")
		t.Chdir(dir)

		_, out, err := handlePinAction(t.Context(), &mcp.CallToolRequest{}, pinActionIn{Action: "foo/bar"})
		if err != nil {
			t.Fatalf("handlePinAction: %v", err)
		}
		if out.Pinned || out.Note == "" {
			t.Errorf("out = %+v, want pinned=false with a note", out)
		}
	})

	t.Run("pin_action dry run writes nothing", func(t *testing.T) {
		dir := t.TempDir()
		writeWorkflow(t, dir, "ci.yml", checkoutV3Workflow)
		t.Chdir(dir)

		_, out, err := handlePinAction(t.Context(), &mcp.CallToolRequest{}, pinActionIn{Action: "actions/checkout", DryRun: true})
		if err != nil {
			t.Fatalf("handlePinAction: %v", err)
		}
		if out.Pinned || out.Ref != "v3" {
			t.Errorf("out = %+v, want planned pin v3", out)
		}
		if _, err := os.Stat(".actup.yaml"); !os.IsNotExist(err) {
			t.Error("dry run must not write .actup.yaml")
		}
	})

	t.Run("unpin_action removes the entry", func(t *testing.T) {
		dir := t.TempDir()
		writeWorkflow(t, dir, "ci.yml", checkoutV3Workflow)
		t.Chdir(dir)

		if _, _, err := handlePinAction(t.Context(), &mcp.CallToolRequest{}, pinActionIn{Action: "actions/checkout"}); err != nil {
			t.Fatalf("handlePinAction: %v", err)
		}
		_, out, err := handleUnpinAction(t.Context(), &mcp.CallToolRequest{}, unpinActionIn{Action: "actions/checkout"})
		if err != nil {
			t.Fatalf("handleUnpinAction: %v", err)
		}
		if !out.Unpinned {
			t.Errorf("out = %+v, want unpinned", out)
		}
		cfg, err := config.LoadDefault()
		if err != nil {
			t.Fatalf("reload config: %v", err)
		}
		if cfg.EffectivePins()["actions/checkout"] != "" {
			t.Errorf("pin survived: %+v", cfg)
		}
	})

	t.Run("unpin_action on unpinned key", func(t *testing.T) {
		dir := t.TempDir()
		writeWorkflow(t, dir, "ci.yml", checkoutV3Workflow)
		t.Chdir(dir)

		_, out, err := handleUnpinAction(t.Context(), &mcp.CallToolRequest{}, unpinActionIn{Action: "actions/checkout"})
		if err != nil {
			t.Fatalf("handleUnpinAction: %v", err)
		}
		if out.Unpinned || out.Note == "" {
			t.Errorf("out = %+v, want unpinned=false with a note", out)
		}
	})

	t.Run("init_config records versions and skips unknown SHA pins", func(t *testing.T) {
		dir := t.TempDir()
		writeWorkflow(t, dir, "ci.yml", "name: ci\non: push\njobs:\n  b:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@v3\n      - uses: foo/bar@"+testSHA+"\n")
		t.Chdir(dir)

		res, out, err := handleInitConfig(t.Context(), &mcp.CallToolRequest{}, initConfigIn{})
		if err != nil {
			t.Fatalf("handleInitConfig: %v", err)
		}
		if res == nil || len(res.Content) == 0 {
			t.Fatal("expected summary content")
		}
		if !out.Written || out.Path != ".actup.yaml" {
			t.Errorf("out = %+v, want written .actup.yaml", out)
		}
		if out.Added != 1 {
			t.Errorf("added = %d, want 1", out.Added)
		}
		if len(out.Skipped) != 1 || out.Skipped[0] != "foo/bar" {
			t.Errorf("skipped = %v, want [foo/bar]", out.Skipped)
		}
		if !strings.Contains(out.ConfigYAML, "actions/checkout: v3") {
			t.Errorf("config_yaml missing checkout v3:\n%s", out.ConfigYAML)
		}
		cfg, err := config.LoadDefault()
		if err != nil {
			t.Fatalf("reload config: %v", err)
		}
		if cfg.Actions["actions/checkout"] != "v3" {
			t.Errorf("Actions[checkout] = %q, want v3", cfg.Actions["actions/checkout"])
		}
	})

	t.Run("init_config refuses to overwrite without force", func(t *testing.T) {
		dir := t.TempDir()
		writeWorkflow(t, dir, "ci.yml", checkoutV3Workflow)
		if err := os.WriteFile(filepath.Join(dir, ".actup.yaml"), []byte("actions:\n  actions/cache: v4\n"), 0600); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
		t.Chdir(dir)

		if _, _, err := handleInitConfig(t.Context(), &mcp.CallToolRequest{}, initConfigIn{}); err == nil {
			t.Fatal("expected an error without force")
		}
	})

	t.Run("init_config force merges", func(t *testing.T) {
		dir := t.TempDir()
		writeWorkflow(t, dir, "ci.yml", checkoutV3Workflow)
		if err := os.WriteFile(filepath.Join(dir, ".actup.yaml"), []byte("actions:\n  actions/cache: v4\n"), 0600); err != nil {
			t.Fatalf("write manifest: %v", err)
		}
		t.Chdir(dir)

		_, out, err := handleInitConfig(t.Context(), &mcp.CallToolRequest{}, initConfigIn{Force: true})
		if err != nil {
			t.Fatalf("handleInitConfig: %v", err)
		}
		if !out.Written || out.Added != 1 {
			t.Errorf("out = %+v, want merged write with 1 added", out)
		}
		cfg, err := config.LoadDefault()
		if err != nil {
			t.Fatalf("reload config: %v", err)
		}
		if cfg.Actions["actions/cache"] != "v4" || cfg.Actions["actions/checkout"] != "v3" {
			t.Errorf("merged Actions = %v, want cache v4 and checkout v3", cfg.Actions)
		}
	})
}

func TestMCPRunToleratesClientDisconnect(t *testing.T) {
	pr, pw := io.Pipe()
	var out bytes.Buffer

	done := make(chan error, 1)
	go func() {
		done <- runMCPTransport(context.Background(), &mcp.IOTransport{Reader: pr, Writer: nopCloseWriter{&out}})
	}()

	// A request immediately followed by stdin close: the client disconnects
	// while the initialize handshake is still in flight.
	_, _ = pw.Write([]byte(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"t","version":"0"}}}` + "\n"))
	_ = pw.Close()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not exit after client disconnect")
	}
}

type nopCloseWriter struct{ io.Writer }

func (nopCloseWriter) Close() error { return nil }
