package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/parser"
	"github.com/lynicis/actup/internal/vulnscanner"
)

// TestAuditJSONOutputShape verifies the audit --json output is valid JSON with
// the expected top-level structure. The OSV network call is skipped by running
// in a directory with no workflows (command errors before any network use).
func TestAuditJSONOutputShape(t *testing.T) {
	dir := t.TempDir()

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"audit", "--json", "--path", dir})
	defer rootCmd.SetArgs(nil)

	err := rootCmd.Execute()
	if err == nil {
		t.Fatal("expected error: no workflow files found in empty dir")
	}
}

// TestAuditBareSHAUnknownVersion ensures a bare SHA ref is parsed but is not
// claimed to be safe: the audit reports an explicit unknown-version notice and
// performs no vulnerability check.
func TestAuditBareSHAUnknownVersion(t *testing.T) {
	resetAuditFixState()
	defer resetAuditFixState()
	dir := t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wf := filepath.Join(wfDir, "ci.yml")
	content := "name: CI\non: [push]\njobs:\n  b:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@8f4b7f84864484a7bf31766abe9204da3cbe65b3\n"
	if err := os.WriteFile(wf, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	rootCmd.SetOut(&stdout)
	rootCmd.SetErr(&stderr)
	rootCmd.SetArgs([]string{"audit", "--json", "--path", dir})
	defer rootCmd.SetArgs(nil)

	// Run in the temp dir so config loading finds no .actup.yaml.
	oldWd, _ := os.Getwd()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	oldStdout, oldStderr := os.Stdout, os.Stderr
	rOut, wOut, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	rErr, wErr, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout, os.Stderr = wOut, wErr

	// Zero network: the only action is a bare SHA, so the scan must be skipped.
	execErr := rootCmd.Execute()

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout, os.Stderr = oldStdout, oldStderr
	var bufOut, bufErr bytes.Buffer
	_, _ = bufOut.ReadFrom(rOut)
	_, _ = bufErr.ReadFrom(rErr)

	if execErr != nil {
		t.Fatalf("audit with only a bare SHA ref should succeed, got %v", execErr)
	}

	out := bufOut.String()
	var parsed struct {
		Findings []json.RawMessage `json:"findings"`
	}
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output not valid JSON: %q (%v)", out, err)
	}
	if len(parsed.Findings) != 0 {
		t.Errorf("expected 0 findings, got %d", len(parsed.Findings))
	}
	notice := bufErr.String()
	if !strings.Contains(notice, "version unknown (SHA-pinned without a version comment)") {
		t.Errorf("expected explicit unknown-version notice on stderr, got %q", notice)
	}
	if strings.Contains(out, "No actions found") {
		t.Errorf("must not claim there are no actions: %q", out)
	}
}

// resetAuditFixState zeroes audit flags between command executions. Cobra
// keeps flag values and "changed" state on the shared command, so tests must
// clear both before and after each audit run.
func resetAuditFixState() {
	auditOpts = struct {
		paths  []string
		json   bool
		fix    bool
		dryRun bool
		force  bool
		semver bool
		token  string
		sha    bool
	}{}
	for _, name := range []string{"sha", "fix", "json", "dry-run", "semver", "force", "token", "path"} {
		if f := auditCmd.Flags().Lookup(name); f != nil {
			f.Changed = false
		}
	}
}

// auditTestRepo creates a temp repo with one workflow using ref and chdirs
// into it for the duration of the test.
func auditTestRepo(t *testing.T, ref string) (dir, wf string) {
	t.Helper()
	dir = t.TempDir()
	wfDir := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wfDir, 0o755); err != nil {
		t.Fatal(err)
	}
	wf = filepath.Join(wfDir, "ci.yml")
	content := "name: CI\non: [push]\njobs:\n  b:\n    runs-on: ubuntu-latest\n    steps:\n      - uses: actions/checkout@" + ref + "\n"
	if err := os.WriteFile(wf, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
	return dir, wf
}

// stubAuditScan replaces the OSV scan with fixed findings for the test.
func stubAuditScan(t *testing.T, findings []vulnscanner.Finding) {
	t.Helper()
	old := auditScan
	auditScan = func(context.Context, []parser.ActionRef) ([]vulnscanner.Finding, error) {
		return findings, nil
	}
	t.Cleanup(func() { auditScan = old })
}

func checkoutFinding(fixed string) vulnscanner.Finding {
	return vulnscanner.Finding{
		Owner: "actions",
		Repo:  "checkout",
		Ref:   "v1",
		Vulns: []vulnscanner.Vulnerability{{ID: "GHSA-test", Fixed: fixed}},
		File:  ".github/workflows/ci.yml",
		Line:  7,
	}
}

func runAuditCommand(t *testing.T, args ...string) error {
	t.Helper()
	rootCmd.SetArgs(args)
	t.Cleanup(func() { rootCmd.SetArgs(nil) })
	return rootCmd.Execute()
}

func TestMinFixedFor(t *testing.T) {
	got, ok := minFixedFor([]vulnscanner.Vulnerability{
		{ID: "A", Fixed: "2"},
		{ID: "B", Fixed: "5.3"},
	})
	if !ok || got != "5.3" {
		t.Fatalf("minFixedFor = %q, %v; want 5.3, true", got, ok)
	}

	// Duplicates by ID merge to the max fixed across occurrences: fixedFor is
	// site-specific, so the same advisory can report different fixes per site.
	got, ok = minFixedFor([]vulnscanner.Vulnerability{
		{ID: "A", Fixed: "2"},
		{ID: "A", Fixed: "5.3"},
	})
	if !ok || got != "5.3" {
		t.Fatalf("merged minFixedFor = %q, %v; want 5.3, true", got, ok)
	}

	// An occurrence without a fix makes the advisory unfixable.
	if _, ok := minFixedFor([]vulnscanner.Vulnerability{
		{ID: "A", Fixed: "5.3"},
		{ID: "A", Fixed: ""},
	}); ok {
		t.Error("minFixedFor with an unfixed occurrence ok = true, want false")
	}

	if _, ok := minFixedFor(nil); ok {
		t.Error("minFixedFor(nil) ok = true, want false")
	}
	if _, ok := minFixedFor([]vulnscanner.Vulnerability{{ID: "A", Fixed: ""}}); ok {
		t.Error("minFixedFor with an unfixed advisory ok = true, want false")
	}
}

func TestPlanAuditFixes_DedupesAcrossFiles(t *testing.T) {
	refs := []parser.ActionRef{
		{Owner: "actions", Repo: "checkout", Path: "actions/checkout", Current: "v1", File: "a.yml", Line: 1},
		{Owner: "actions", Repo: "checkout", Path: "actions/checkout", Current: "v2", File: "b.yml", Line: 1},
		{Owner: "actions", Repo: "checkout", Path: "actions/checkout", Current: "v3", File: "c.yml", Line: 1},
	}
	v := vulnscanner.Vulnerability{ID: "GHSA-x", Fixed: "5.3"}
	findings := []vulnscanner.Finding{
		{Owner: "actions", Repo: "checkout", Ref: "v1", File: "a.yml", Line: 1, Vulns: []vulnscanner.Vulnerability{v}},
		{Owner: "actions", Repo: "checkout", Ref: "v2", File: "b.yml", Line: 1, Vulns: []vulnscanner.Vulnerability{v}},
	}

	fixes, unfixable, _ := planAuditFixes(findings, refs)
	if len(unfixable) != 0 {
		t.Fatalf("unfixable = %d, want 0", len(unfixable))
	}
	if len(fixes) != 1 {
		t.Fatalf("fixes = %d, want 1", len(fixes))
	}
	f := fixes[0]
	if f.MinFixed != "5.3" {
		t.Errorf("MinFixed = %q, want 5.3", f.MinFixed)
	}
	if len(f.Refs) != 3 {
		t.Errorf("Refs = %d, want all 3 refs for the key", len(f.Refs))
	}
	if f.Current != "v3" {
		t.Errorf("Current = %q, want max v3", f.Current)
	}
}

func TestPlanAuditFixes_UnfixableSkipsAllSites(t *testing.T) {
	refs := []parser.ActionRef{
		{Owner: "actions", Repo: "checkout", Current: "v1", File: "a.yml", Line: 1},
		{Owner: "actions", Repo: "checkout", Current: "v1", File: "b.yml", Line: 1},
	}
	findings := []vulnscanner.Finding{
		{Owner: "actions", Repo: "checkout", Ref: "v1", File: "a.yml", Line: 1,
			Vulns: []vulnscanner.Vulnerability{{ID: "A", Fixed: "5"}}},
		{Owner: "actions", Repo: "checkout", Ref: "v1", File: "b.yml", Line: 1,
			Vulns: []vulnscanner.Vulnerability{{ID: "B", Fixed: ""}}},
	}

	fixes, unfixable, _ := planAuditFixes(findings, refs)
	if len(fixes) != 0 {
		t.Fatalf("fixes = %d, want 0 (no partial fixes)", len(fixes))
	}
	if len(unfixable) != 2 {
		t.Fatalf("unfixable = %d, want both sites", len(unfixable))
	}
}

func TestPlanAuditFixes_DifferentCurrents(t *testing.T) {
	refs := []parser.ActionRef{
		{Owner: "actions", Repo: "checkout", Current: "v1", File: "a.yml", Line: 1},
		{Owner: "actions", Repo: "checkout", Current: "v2.4", File: "b.yml", Line: 1},
	}
	findings := []vulnscanner.Finding{
		{Owner: "actions", Repo: "checkout", Ref: "v1", File: "a.yml", Line: 1,
			Vulns: []vulnscanner.Vulnerability{{ID: "A", Fixed: "5"}}},
	}

	fixes, _, _ := planAuditFixes(findings, refs)
	if len(fixes) != 1 {
		t.Fatalf("fixes = %d, want 1", len(fixes))
	}
	if fixes[0].Current != "v2.4" {
		t.Errorf("Current = %q, want max v2.4", fixes[0].Current)
	}
	if fixes[0].MinFixed != "5" {
		t.Errorf("MinFixed = %q, want 5", fixes[0].MinFixed)
	}
}

func TestPlanAuditFixes_MixedVersionsFixesVulnerableSite(t *testing.T) {
	refs := []parser.ActionRef{
		{Owner: "actions", Repo: "checkout", Current: "v1", File: "a.yml", Line: 1},
		{Owner: "actions", Repo: "checkout", Current: "v5", File: "b.yml", Line: 1},
	}
	// Only the v1 site is vulnerable; the v5 sibling must not mask it.
	findings := []vulnscanner.Finding{
		{Owner: "actions", Repo: "checkout", Ref: "v1", File: "a.yml", Line: 1,
			Vulns: []vulnscanner.Vulnerability{{ID: "A", Fixed: "5"}}},
	}

	fixes, _, alreadyPast := planAuditFixes(findings, refs)
	if len(alreadyPast) != 0 {
		t.Fatalf("alreadyPast = %d, want 0", len(alreadyPast))
	}
	if len(fixes) != 1 {
		t.Fatalf("fixes = %d, want 1 (vulnerable site must be fixed)", len(fixes))
	}
	if fixes[0].Current != "v5" {
		t.Errorf("Current = %q, want max v5", fixes[0].Current)
	}

	f := fixes[0]
	f.Target = "v5"
	upgrades := auditUpgrades(f, "", false)
	if len(upgrades) != 1 || upgrades[0].Action.File != "a.yml" {
		t.Fatalf("upgrades = %+v, want only the v1 site", upgrades)
	}

	// When the vulnerable site itself is already past the fix, skip the key.
	safeFindings := []vulnscanner.Finding{
		{Owner: "actions", Repo: "checkout", Ref: "v5", File: "b.yml", Line: 1,
			Vulns: []vulnscanner.Vulnerability{{ID: "A", Fixed: "5"}}},
	}
	fixes, _, alreadyPast = planAuditFixes(safeFindings, refs)
	if len(fixes) != 0 || len(alreadyPast) != 1 {
		t.Fatalf("all-sites-past = fixes %d, alreadyPast %d; want 0, 1", len(fixes), len(alreadyPast))
	}
}

func TestAuditFixDryRunWritesNothing(t *testing.T) {
	resetAuditFixState()
	defer resetAuditFixState()
	_, wf := auditTestRepo(t, "v1")
	stubAuditScan(t, []vulnscanner.Finding{checkoutFinding("5")})

	err := runAuditCommand(t, "audit", "--fix", "--dry-run", "--semver", "--path", ".")
	if !errors.Is(err, errAuditFindings) {
		t.Fatalf("err = %v, want errAuditFindings", err)
	}

	got, err := os.ReadFile(wf)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "actions/checkout@v1") {
		t.Errorf("dry run rewrote workflow: %s", got)
	}
	if _, err := os.Stat(".actup.yaml"); !os.IsNotExist(err) {
		t.Errorf("dry run created .actup.yaml (stat err %v)", err)
	}
}

func TestAuditFixWritesTargetAndPin(t *testing.T) {
	cases := []struct {
		name       string
		manifest   string
		wantPinned bool
	}{
		{"managed", "pins: {}\nactions: {}\n", true},
		{"legacy", "actions: {}\n", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resetAuditFixState()
			defer resetAuditFixState()
			_, wf := auditTestRepo(t, "v1")
			if err := os.WriteFile(".actup.yaml", []byte(tc.manifest), 0o644); err != nil {
				t.Fatal(err)
			}
			stubAuditScan(t, []vulnscanner.Finding{checkoutFinding("5")})

			err := runAuditCommand(t, "audit", "--fix", "--semver", "--force", "--path", ".")
			if err != nil {
				t.Fatalf("audit --fix = %v, want nil", err)
			}

			got, err := os.ReadFile(wf)
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(got), "actions/checkout@v5") {
				t.Errorf("workflow not rewritten: %s", got)
			}

			cfg, err := config.Load(".actup.yaml")
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Actions["actions/checkout"] != "v5" {
				t.Errorf("Actions = %q, want v5", cfg.Actions["actions/checkout"])
			}
			if tc.wantPinned {
				if cfg.Pins["actions/checkout"] != "v5" {
					t.Errorf("Pins = %q, want v5", cfg.Pins["actions/checkout"])
				}
			} else if cfg.Pins != nil {
				t.Errorf("legacy manifest gained pins: %v", cfg.Pins)
			}
		})
	}
}

func TestAuditFixNoManifestCreated(t *testing.T) {
	resetAuditFixState()
	defer resetAuditFixState()
	_, wf := auditTestRepo(t, "v1")
	stubAuditScan(t, []vulnscanner.Finding{checkoutFinding("5")})

	if err := runAuditCommand(t, "audit", "--fix", "--semver", "--force", "--path", "."); err != nil {
		t.Fatalf("audit --fix = %v, want nil", err)
	}
	got, err := os.ReadFile(wf)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "actions/checkout@v5") {
		t.Errorf("workflow not rewritten: %s", got)
	}
	if _, err := os.Stat(".actup.yaml"); !os.IsNotExist(err) {
		t.Errorf("audit --fix created .actup.yaml (stat err %v)", err)
	}
}

func TestAuditFixNoFixKnown(t *testing.T) {
	resetAuditFixState()
	defer resetAuditFixState()
	_, wf := auditTestRepo(t, "v1")
	stubAuditScan(t, []vulnscanner.Finding{checkoutFinding("")})

	err := runAuditCommand(t, "audit", "--fix", "--semver", "--force", "--path", ".")
	if !errors.Is(err, errAuditFindings) {
		t.Fatalf("err = %v, want errAuditFindings", err)
	}
	got, readErr := os.ReadFile(wf)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(got), "actions/checkout@v1") {
		t.Errorf("unfixable action was rewritten: %s", got)
	}
}

func TestAuditFixJSONRejected(t *testing.T) {
	resetAuditFixState()
	defer resetAuditFixState()
	dir := t.TempDir()

	old := auditScan
	called := false
	auditScan = func(context.Context, []parser.ActionRef) ([]vulnscanner.Finding, error) {
		called = true
		return nil, errors.New("scan must not run")
	}
	defer func() { auditScan = old }()

	err := runAuditCommand(t, "audit", "--json", "--fix", "--path", dir)
	if err == nil || !strings.Contains(err.Error(), "--json cannot be combined with --fix") {
		t.Fatalf("err = %v, want json+fix rejection", err)
	}
	if called {
		t.Error("scan ran before rejecting --json with --fix")
	}
}

func TestSavePinsBatched(t *testing.T) {
	resetAuditFixState()
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	pins := map[string]string{"a/b": "v2", "c/d": "v3"}

	if err := os.WriteFile(".actup.yaml", []byte("pins: {}\nactions: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := savePins(pins); err != nil {
		t.Fatalf("savePins: %v", err)
	}
	cfg, err := config.Load(".actup.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range pins {
		if cfg.Actions[key] != want || cfg.Pins[key] != want {
			t.Errorf("managed %s: Actions=%q Pins=%q, want %q", key, cfg.Actions[key], cfg.Pins[key], want)
		}
	}

	if err := os.WriteFile(".actup.yaml", []byte("actions: {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := savePins(pins); err != nil {
		t.Fatalf("savePins legacy: %v", err)
	}
	cfg, err = config.Load(".actup.yaml")
	if err != nil {
		t.Fatal(err)
	}
	for key, want := range pins {
		if cfg.Actions[key] != want {
			t.Errorf("legacy %s: Actions=%q, want %q", key, cfg.Actions[key], want)
		}
	}
	if cfg.Pins != nil {
		t.Errorf("legacy manifest gained pins: %v", cfg.Pins)
	}
}

func TestSavePins_NoManifestNoop(t *testing.T) {
	dir := t.TempDir()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(oldWd) }()

	if err := savePins(map[string]string{"a/b": "v2"}); err != nil {
		t.Fatalf("savePins: %v", err)
	}
	if _, err := os.Stat(".actup.yaml"); !os.IsNotExist(err) {
		t.Errorf("savePins created a manifest (stat err %v)", err)
	}
}

func TestAuditFixAlreadyPast(t *testing.T) {
	resetAuditFixState()
	defer resetAuditFixState()
	_, wf := auditTestRepo(t, "v3")
	finding := checkoutFinding("1")
	finding.Ref = "v3"
	finding.File = wf
	finding.Line = 7
	stubAuditScan(t, []vulnscanner.Finding{finding})

	oldStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = oldStdout }()

	cmdErr := runAuditCommand(t, "audit", "--fix", "--dry-run", "--semver", "--path", ".")
	_ = w.Close()
	os.Stdout = oldStdout
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	if cmdErr != nil {
		t.Fatalf("err = %v, want nil (nothing actionable)", cmdErr)
	}
	if !strings.Contains(string(out), "already past the reported fix 1") {
		t.Errorf("missing already-past notice: %q", out)
	}
	got, err := os.ReadFile(wf)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(got), "actions/checkout@v3") {
		t.Errorf("already-past key was rewritten: %s", got)
	}
}
