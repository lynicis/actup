package cmd

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/parser"
	"github.com/lynicis/actup/internal/upgrader"
)

const (
	testSHAOld = "0123456789abcdef0123456789abcdef01234567"
	testSHANew = "89abcdef0123456789abcdef0123456789abcdef"
)

func loadRefs(t *testing.T, dir string) []parser.ActionRef {
	t.Helper()
	refs, err := loadWorkflowActions(context.Background(), []string{dir})
	if err != nil {
		t.Fatalf("loadWorkflowActions: %v", err)
	}
	return refs
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(data)
}

func TestApplyUseToFileRefs_SHAStaysSHA(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := writeWorkflow(t, dir, "ci.yml", "steps:\n  - uses: actions/checkout@"+testSHAOld+" # v4\n")

	refs := loadRefs(t, dir)
	if !anyRefPinned(refs) {
		t.Fatal("expected a pinned ref")
	}
	if !applyUseToFileRefs("actions/checkout", refs, "v5", testSHANew, false, false) {
		t.Fatal("expected the SHA ref to be rewritten in SHA mode")
	}

	got := readFile(t, path)
	if !strings.Contains(got, "actions/checkout@"+testSHANew+" # v5") {
		t.Errorf("expected new SHA + tag, got:\n%s", got)
	}
	if strings.Contains(got, "@"+testSHAOld) {
		t.Errorf("old SHA should be gone:\n%s", got)
	}
}

func TestApplyUseToFileRefs_SHASameTargetSkipped(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := writeWorkflow(t, dir, "ci.yml", "steps:\n  - uses: actions/checkout@"+testSHAOld+" # v5\n")

	refs := loadRefs(t, dir)
	if applyUseToFileRefs("actions/checkout", refs, "v5", testSHAOld, false, false) {
		t.Fatal("same SHA + tag must not be rewritten")
	}
	if got := readFile(t, path); !strings.Contains(got, "@"+testSHAOld+" # v5") {
		t.Errorf("file changed unexpectedly:\n%s", got)
	}
}

func TestApplyUseToFileRefs_TagConvertsToSHA(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := writeWorkflow(t, dir, "ci.yml", "steps:\n  - uses: actions/checkout@v4\n")

	refs := loadRefs(t, dir)
	if anyRefPinned(refs) {
		t.Fatal("tag ref should not report pinned")
	}
	if !applyUseToFileRefs("actions/checkout", refs, "v5", testSHANew, true, false) {
		t.Fatal("expected conversion to a SHA ref")
	}
	if got := readFile(t, path); !strings.Contains(got, "actions/checkout@"+testSHANew+" # v5") {
		t.Errorf("expected SHA + tag, got:\n%s", got)
	}
}

func TestMostCommonVersionSkipsBareSHA(t *testing.T) {
	refs := []parser.ActionRef{
		{Current: testSHAOld, SHA: testSHAOld},
		{Current: "v4"},
	}
	if got := mostCommonVersion(refs); got != "v4" {
		t.Errorf("mostCommonVersion = %q, want v4", got)
	}
	allBare := []parser.ActionRef{{Current: testSHAOld, SHA: testSHAOld}}
	if got := mostCommonVersion(allBare); got != "" {
		t.Errorf("mostCommonVersion(bare SHA) = %q, want empty", got)
	}
}

func TestDoInitSkipsBareSHA(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	writeWorkflow(t, dir, "ci.yml", "steps:\n  - uses: actions/checkout@"+testSHAOld+"\n")

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	err = doInit(context.Background(), initOptions{})
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if err != nil {
		t.Fatalf("doInit: %v", err)
	}
	if !strings.Contains(buf.String(), "SHA-pinned without a version comment") {
		t.Errorf("expected the unknown-version notice, got %q", buf.String())
	}

	cfg, err := config.LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault: %v", err)
	}
	if _, exists := cfg.Actions["actions/checkout"]; exists {
		t.Errorf("bare SHA must not be recorded as an empty version: %v", cfg.Actions)
	}
}

func TestPrintUseDiffRendersSHAComment(t *testing.T) {
	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	printUseDiff(upgrader.Upgrade{
		Action: parser.ActionRef{Owner: "actions", Repo: "checkout", Path: "actions/checkout", Current: "v4", File: "ci.yml", Line: 3},
		NewTag: "v5",
		NewSHA: testSHANew,
	})
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)

	if !strings.Contains(buf.String(), "+uses: actions/checkout@"+testSHANew+" # v5") {
		t.Errorf("dry-run must render @sha # tag, got:\n%s", buf.String())
	}
}

func TestApplyUseToFileRefs_MixedPerSiteStyle(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := writeWorkflow(t, dir, "ci.yml",
		"steps:\n  - uses: actions/checkout@v4\n  - uses: actions/checkout@"+testSHAOld+" # v4\n")

	refs := loadRefs(t, dir)
	if !applyUseToFileRefs("actions/checkout", refs, "v5", testSHANew, false, false) {
		t.Fatal("expected both sites to be rewritten")
	}
	got := readFile(t, path)
	if !strings.Contains(got, "actions/checkout@v5") {
		t.Errorf("tag site must stay a tag without --sha:\n%s", got)
	}
	if !strings.Contains(got, "actions/checkout@"+testSHANew+" # v5") {
		t.Errorf("SHA site must stay a SHA:\n%s", got)
	}
}

func TestApplyUseToFileRefs_MixedForceSHA(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := writeWorkflow(t, dir, "ci.yml",
		"steps:\n  - uses: actions/checkout@v4\n  - uses: actions/checkout@"+testSHAOld+" # v4\n")

	refs := loadRefs(t, dir)
	if !applyUseToFileRefs("actions/checkout", refs, "v5", testSHANew, true, false) {
		t.Fatal("expected both sites to be rewritten")
	}
	got := readFile(t, path)
	if strings.Count(got, "actions/checkout@"+testSHANew+" # v5") != 2 {
		t.Errorf("--sha must convert both sites:\n%s", got)
	}
}

func TestSavePinRejectsSHAPin(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)

	err := savePin("actions/checkout", testSHAOld)
	if err == nil || !strings.Contains(err.Error(), "pin a tag and use sha: true") {
		t.Fatalf("savePin(SHA) = %v, want pin-a-tag error", err)
	}
	if _, statErr := os.Stat(".actup.yaml"); !os.IsNotExist(statErr) {
		t.Error("a rejected pin must not create the manifest")
	}
}

func TestSyncRejectsSHAPin(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := writeWorkflow(t, dir, "ci.yml", "steps:\n  - uses: actions/checkout@v4\n")
	if err := config.Save(".actup.yaml", &config.Config{
		Pins:    map[string]string{"actions/checkout": testSHAOld},
		Actions: map[string]string{"actions/checkout": testSHAOld},
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	syncErr := doSync(context.Background(), syncOptions{})
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if syncErr != nil {
		t.Fatalf("doSync: %v", syncErr)
	}
	if !strings.Contains(buf.String(), "pin a tag and use sha: true") {
		t.Errorf("expected SHA-pin rejection, got %q", buf.String())
	}
	if got := readFile(t, path); !strings.Contains(got, "actions/checkout@v4") {
		t.Errorf("workflow must be untouched:\n%s", got)
	}
}

func TestUseRejectsSHATarget(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	path := writeWorkflow(t, dir, "ci.yml", "steps:\n  - uses: actions/checkout@v4\n")

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	useErr := doUse(context.Background(), []actionSpec{{Key: "actions/checkout", Ref: testSHAOld}}, useOptions{})
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	if useErr != nil {
		t.Fatalf("doUse: %v", useErr)
	}
	if !strings.Contains(buf.String(), "pin a tag and use sha: true") {
		t.Errorf("expected SHA-target rejection, got %q", buf.String())
	}
	if got := readFile(t, path); !strings.Contains(got, "actions/checkout@v4") {
		t.Errorf("workflow must be untouched:\n%s", got)
	}
	if _, statErr := os.Stat(".actup.yaml"); !os.IsNotExist(statErr) {
		t.Error("rejected use must not save a pin")
	}
}
