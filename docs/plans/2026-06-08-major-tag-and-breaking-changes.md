# actup — Major-Tag Mode & Breaking-Change Detection Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add (1) major-tag update mode as default with `--semver` opt-in, and (2) breaking-change detection and reporting for GitHub Actions upgrades.

**Architecture:** Feature 1 modifies `internal/github` tag resolution to prefer `^v\d+$` major tags and threads a `--semver` flag through `cmd/root.go`. Feature 2 introduces `internal/breakingchanges` with an embedded YAML registry, surfaces warnings in both TUI and `--no-tui` modes, and adds a `--force` flag for non-interactive breaking-change upgrades.

**Tech Stack:** Go 1.26, Cobra, Bubble Tea, `go-github/v62`, `golang.org/x/mod` semver, `gopkg.in/yaml.v3` (new dep for registry parsing).

---

## Feature 1 — Major-Tag Update Mode

### Task 1: Write failing tests for tag resolution modes

**Files:**
- Create: `internal/github/client_test.go`

**Step 1: Write the failing test**

```go
package github

import (
	"regexp"
	"testing"
)

func TestResolveLatestTag_MajorTagMode(t *testing.T) {
	tags := []string{"v4", "v4.1.0", "v4.2.1", "v5", "v5.0.1"}

	got := resolveLatestTag(tags, false)
	if got != "v5" {
		t.Errorf("resolveLatestTag major mode = %q, want %q", got, "v5")
	}
}

func TestResolveLatestTag_SemverMode(t *testing.T) {
	tags := []string{"v4", "v4.1.0", "v4.2.1", "v5", "v5.0.1"}

	got := resolveLatestTag(tags, true)
	if got != "v5.0.1" {
		t.Errorf("resolveLatestTag semver mode = %q, want %q", got, "v5.0.1")
	}
}

func TestResolveLatestTag_NoMajorTags_Fallback(t *testing.T) {
	tags := []string{"v3.1.0", "v3.2.0"}

	got := resolveLatestTag(tags, false)
	if got != "v3.2.0" {
		t.Errorf("resolveLatestTag fallback = %q, want %q", got, "v3.2.0")
	}
}

func TestResolveLatestTag_EmptyTags(t *testing.T) {
	got := resolveLatestTag(nil, false)
	if got != "" {
		t.Errorf("resolveLatestTag empty = %q, want empty", got)
	}
}

func TestIsMajorTag(t *testing.T) {
	majorTagRegex := regexp.MustCompile(`^v\d+$`)
	tests := []struct {
		tag  string
		want bool
	}{
		{"v3", true},
		{"v10", true},
		{"v3.1.0", false},
		{"v3.1", false},
		{"3", false},
		{"", false},
	}
	for _, tt := range tests {
		got := majorTagRegex.MatchString(tt.tag)
		if got != tt.want {
			t.Errorf("isMajorTag(%q) = %v, want %v", tt.tag, got, tt.want)
		}
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v -run TestResolveLatestTag ./internal/github`
Expected: FAIL — `resolveLatestTag` is undefined.

**Step 3: Commit test scaffold**

```bash
git add internal/github/client_test.go
git commit -m "test: add failing tests for major-tag and semver resolution modes"
```

---

### Task 2: Implement `resolveLatestTag` and refactor `LatestSemverTag`

**Files:**
- Modify: `internal/github/client.go`

**Step 1: Add `resolveLatestTag` function and `majorTagRegex`**

Add to `internal/github/client.go`:

```go
import "regexp"

var majorTagRegex = regexp.MustCompile(`^v\d+$`)

func resolveLatestTag(tags []string, semverMode bool) string {
	if len(tags) == 0 {
		return ""
	}

	sort.Slice(tags, func(i, j int) bool {
		iC := tags[i]
		if !strings.HasPrefix(iC, "v") {
			iC = "v" + iC
		}
		jC := tags[j]
		if !strings.HasPrefix(jC, "v") {
			jC = "v" + jC
		}
		return semver.Compare(iC, jC) > 0
	})

	if semverMode {
		return tags[0]
	}

	for _, tag := range tags {
		if majorTagRegex.MatchString(tag) {
			return tag
		}
	}

	return tags[0]
}
```

**Step 2: Refactor `fetchLatestTag` to use `resolveLatestTag`**

Replace the sorting and return logic at the bottom of `fetchLatestTag` with:

```go
return resolveLatestTag(semverTags, false), nil
```

**Step 3: Add `LatestTag` method with semver mode support**

```go
func (c *Client) LatestTag(ctx context.Context, owner, repo string, semverMode bool) (string, error) {
	cacheKey := fmt.Sprintf("%s/%s/%v", owner, repo, semverMode)

	if cached, ok := c.cache.Load(cacheKey); ok {
		return cached.(string), nil
	}

	tags, err := c.fetchAllSemverTags(ctx, owner, repo)
	if err != nil {
		return "", err
	}

	tagName := resolveLatestTag(tags, semverMode)
	if tagName == "" {
		return "", ErrNoSemverTags
	}

	c.cache.Store(cacheKey, tagName)
	return tagName, nil
}
```

**Step 4: Extract `fetchAllSemverTags` from `fetchLatestTag`**

Refactor `fetchLatestTag` into `fetchAllSemverTags` that returns `[]string` (the semver tag names) without picking one:

```go
func (c *Client) fetchAllSemverTags(ctx context.Context, owner, repo string) ([]string, error) {
	var allTags []*github.RepositoryTag
	page := 1

	for page <= 3 {
		tags, resp, err := c.client.Repositories.ListTags(ctx, owner, repo, &github.ListOptions{
			Page:    page,
			PerPage: 100,
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusForbidden {
				if resp.Rate.Remaining == 0 {
					return nil, fmt.Errorf("rate limit exceeded: %w", err)
				}
			}
			return nil, fmt.Errorf("list tags: %w", err)
		}

		allTags = append(allTags, tags...)

		if resp.NextPage == 0 {
			break
		}
		page++
	}

	var semverTags []string
	for _, tag := range allTags {
		name := tag.GetName()
		canonical := name
		if !strings.HasPrefix(canonical, "v") {
			canonical = "v" + canonical
		}
		if semver.IsValid(canonical) {
			semverTags = append(semverTags, name)
		}
	}

	return semverTags, nil
}
```

Keep `LatestSemverTag` as a backward-compatible wrapper:

```go
func (c *Client) LatestSemverTag(ctx context.Context, owner, repo string) (string, error) {
	return c.LatestTag(ctx, owner, repo, true)
}
```

**Step 5: Run tests to verify they pass**

Run: `go test -v -race ./internal/github`
Expected: All tests PASS.

**Step 6: Commit**

```bash
git add internal/github/client.go internal/github/client_test.go
git commit -m "feat(github): add resolveLatestTag with major-tag and semver modes"
```

---

### Task 3: Add `--semver` flag to `cmd/root.go` and wire it through

**Files:**
- Modify: `cmd/root.go`

**Step 1: Add the flag variable and registration**

Add to the `var` block:

```go
semverMode   bool
```

Add to `init()`:

```go
rootCmd.Flags().BoolVarP(&semverMode, "semver", "s", false, "Upgrade to the latest full semver tag instead of the latest major tag (e.g. v5.3.1 instead of v5)")
```

**Step 2: Thread `semverMode` through `run()` and `runNoTUI()`**

In `run()`, pass `semverMode` to `runNoTUI` and `tui.Run`:

```go
if noTUI {
    return runNoTUI(ctx, actions, githubToken, dryRun, semverMode)
}
return tui.Run(ctx, actions, githubToken, dryRun, semverMode)
```

Update `runNoTUI` signature:

```go
func runNoTUI(ctx context.Context, actions []parser.ActionRef, githubToken string, dryRun bool, semverMode bool) error {
```

Replace `ghClient.LatestSemverTag(ctx, owner, repo)` with:

```go
latest, err := ghClient.LatestTag(ctx, owner, repo, semverMode)
```

**Step 3: Update `tui.Run` signature**

Modify `internal/tui/model.go` `Run` function to accept `semverMode bool`, store it in the model, and use it in `loadActions`.

Add to `model` struct:

```go
semverMode  bool
```

Update `Run`:

```go
func Run(ctx context.Context, actions []parser.ActionRef, token string, dryRun bool, semverMode bool) error {
    // ...
    m := model{
        // ...
        semverMode:  semverMode,
    }
    // ...
}
```

**Step 4: Update `loadActions` in `internal/tui/actions.go`**

Replace `client.LatestSemverTag(...)` with:

```go
latest, err := client.LatestTag(context.Background(), owner, repo, m.semverMode)
```

**Step 5: Run all tests**

Run: `make test`
Expected: All tests PASS.

**Step 6: Run lint**

Run: `make fmt && make lint`
Expected: No errors.

**Step 7: Commit Feature 1**

```bash
git add -A
git commit -m "feat: add major-tag update mode as default with --semver opt-in"
```

---

## Feature 2 — Breaking-Change Detection & Reporting

### Task 4: Add `gopkg.in/yaml.v3` dependency

**Step 1: Add the dependency**

```bash
go get gopkg.in/yaml.v3
```

**Step 2: Commit**

```bash
git add go.mod go.sum
git commit -m "chore: add gopkg.in/yaml.v3 dependency for breaking-change registry"
```

---

### Task 5: Create `internal/breakingchanges` package with registry and tests

**Files:**
- Create: `internal/breakingchanges/registry.yaml`
- Create: `internal/breakingchanges/registry.go`
- Create: `internal/breakingchanges/registry_test.go`

**Step 1: Write `registry.yaml`**

```yaml
- action: "actions/checkout"
  from_major: 2
  to_major: 3
  changes:
    - type: renamed
      input: "github_token"
      replacement: "token"
      message: "The `github_token` input has been renamed to `token`."
    - type: deprecated
      input: "app_url"
      replacement: "url"
      context: "repository"
      message: "The `app_url` input is deprecated. Use `url` under the `repository` block instead."
```

**Step 2: Write `registry.go`**

```go
package breakingchanges

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed registry.yaml
var registryData []byte

type Change struct {
	Type        string `yaml:"type"`
	Input       string `yaml:"input"`
	Replacement string `yaml:"replacement,omitempty"`
	Context     string `yaml:"context,omitempty"`
	Message     string `yaml:"message"`
}

type Entry struct {
	Action    string   `yaml:"action"`
	FromMajor int      `yaml:"from_major"`
	ToMajor   int      `yaml:"to_major"`
	Changes   []Change `yaml:"changes"`
}

type BreakingChange struct {
	Type        string
	Input       string
	Replacement string
	Context     string
	Message     string
}

type Registry struct {
	entries []Entry
}

func LoadRegistry() (*Registry, error) {
	var entries []Entry
	if err := yaml.Unmarshal(registryData, &entries); err != nil {
		return nil, fmt.Errorf("parse breaking-change registry: %w", err)
	}
	return &Registry{entries: entries}, nil
}

func (r *Registry) Check(action string, fromVersion string, toVersion string) []BreakingChange {
	fromMajor := parseMajor(fromVersion)
	toMajor := parseMajor(toVersion)

	if fromMajor < 0 || toMajor < 0 {
		return nil
	}

	var results []BreakingChange
	for _, entry := range r.entries {
		if entry.Action != action {
			continue
		}
		if fromMajor < entry.FromMajor {
			continue
		}
		if entry.ToMajor != 0 && toMajor > entry.ToMajor {
			continue
		}
		if fromMajor >= entry.ToMajor && entry.ToMajor != 0 {
			continue
		}
		for _, c := range entry.Changes {
			results = append(results, BreakingChange{
				Type:        c.Type,
				Input:       c.Input,
				Replacement: c.Replacement,
				Context:     c.Context,
				Message:     c.Message,
			})
		}
	}
	return results
}

func parseMajor(version string) int {
	v := strings.TrimPrefix(version, "v")
	parts := strings.SplitN(v, ".", 2)
	if len(parts) == 0 || parts[0] == "" {
		return -1
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return -1
	}
	return n
}
```

**Step 3: Write `registry_test.go`**

```go
package breakingchanges

import "testing"

func TestCheck_V2ToV3(t *testing.T) {
	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	changes := reg.Check("actions/checkout", "v2", "v3")
	if len(changes) == 0 {
		t.Fatal("expected breaking changes for actions/checkout v2→v3")
	}

	found := false
	for _, c := range changes {
		if c.Input == "github_token" && c.Replacement == "token" {
			found = true
		}
	}
	if !found {
		t.Error("expected github_token→token rename in breaking changes")
	}
}

func TestCheck_V3ToV4_NoEntry(t *testing.T) {
	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	changes := reg.Check("actions/checkout", "v3", "v4")
	if len(changes) != 0 {
		t.Errorf("expected no breaking changes for v3→v4, got %d", len(changes))
	}
}

func TestCheck_UnknownAction(t *testing.T) {
	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	changes := reg.Check("unknown/action", "v1", "v2")
	if len(changes) != 0 {
		t.Errorf("expected no breaking changes for unknown action, got %d", len(changes))
	}
}

func TestCheck_SmokeTestSeededData(t *testing.T) {
	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	changes := reg.Check("actions/checkout", "v2", "v3")
	if len(changes) < 1 {
		t.Error("smoke test: expected at least 1 breaking change for actions/checkout v2→v3")
	}
}

func TestParseMajor(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"v3", 3},
		{"v4.1.0", 4},
		{"v10.0.0", 10},
		{"3", 3},
		{"", -1},
		{"v", -1},
	}
	for _, tt := range tests {
		got := parseMajor(tt.input)
		if got != tt.want {
			t.Errorf("parseMajor(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}

func TestLoadRegistry_ParsesYAML(t *testing.T) {
	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}
	if len(reg.entries) == 0 {
		t.Error("expected at least one entry in seeded registry")
	}
}
```

**Step 4: Run tests**

Run: `go test -v -race ./internal/breakingchanges`
Expected: All tests PASS.

**Step 5: Commit**

```bash
git add internal/breakingchanges/
git commit -m "feat(breakingchanges): add embedded registry with Check function"
```

---

### Task 6: Wire breaking-change detection into `cmd/root.go` and `runNoTUI`

**Files:**
- Modify: `cmd/root.go`

**Step 1: Add `--force` flag**

Add to `var` block:

```go
force        bool
```

Add to `init()`:

```go
rootCmd.Flags().BoolVarP(&force, "force", "f", false, "Upgrade actions with breaking changes without prompting in non-interactive mode")
```

**Step 2: Thread `force` through `runNoTUI`**

Update signature:

```go
func runNoTUI(ctx context.Context, actions []parser.ActionRef, githubToken string, dryRun bool, semverMode bool, force bool) error {
```

**Step 3: Add breaking-change check after version resolution**

In `runNoTUI`, after resolving `latest`, load the registry and call `Check`:

```go
registry, err := breakingchanges.LoadRegistry()
if err != nil {
    fmt.Fprintf(os.Stderr, "⚠ Failed to load breaking-change registry: %v\n", err)
}
```

After getting `latest` for each action, check:

```go
var bcs []breakingchanges.BreakingChange
if registry != nil {
    bcs = registry.Check(r.key, current, r.latest)
}
```

**Step 4: Print warnings and handle `--force` / TTY detection**

If `bcs` is non-empty and `!dryRun` and `!force`:
- Print warning block
- Check if stdin is a TTY via `term.IsTerminal(int(os.Stdin.Fd()))` (use `golang.org/x/term`)
- If not TTY: skip the action, print "Skipping breaking-change actions. Use --force to upgrade them anyway."
- If TTY: prompt `[y/N]`, skip unless confirmed

If `dryRun`: always print warnings without prompting.
If `force`: print warnings, proceed with upgrade.

**Step 5: Run tests and lint**

Run: `make test && make lint`
Expected: PASS.

**Step 6: Commit**

```bash
git add -A
git commit -m "feat: wire breaking-change detection into no-tui mode with --force flag"
```

---

### Task 7: Wire breaking-change detection into TUI

**Files:**
- Modify: `internal/tui/types.go`
- Modify: `internal/tui/model.go`
- Modify: `internal/tui/actions.go`
- Modify: `internal/tui/view.go`
- Modify: `internal/tui/update.go`
- Modify: `internal/tui/styles.go`

**Step 1: Add breaking-change fields to `ActionItem` and model**

In `types.go`, add to `ActionItem`:

```go
BreakingChanges []breakingchanges.BreakingChange
HasBreaking     bool
```

Add new state:

```go
stateDetail
```

In `model.go`, add to `model`:

```go
detailItem int
```

**Step 2: Update `loadActions` to check breaking changes**

In `actions.go`, after resolving `latest`:

```go
if registry != nil {
    bcs := registry.Check(key, acts[0].Current, latest)
    item.BreakingChanges = bcs
    item.HasBreaking = len(bcs) > 0
}
```

Load registry once at the top of `loadActions`.

**Step 3: Update checklist view to show breaking-change badge**

In `view.go` `viewChecklist`, after the existing suffix logic:

```go
if item.HasBreaking {
    suffix += " " + amberStyle.Render("⚠ breaking changes")
}
```

**Step 4: Add detail view**

Add `viewDetail()` method that renders the breaking-change messages for `m.items[m.detailItem]`.

**Step 5: Wire `i` key and `Esc` key in `updateChecklist`**

```go
case "i":
    if m.items[m.cursor].HasBreaking {
        m.detailItem = m.cursor
        m.state = stateDetail
    }
    return m, nil
```

Add `updateDetail` handler for `Esc` to return to checklist.

**Step 6: Update footer help text**

```
[space] toggle  [a] all  [n] none  [i] info  [enter] apply  [q] quit
```

**Step 7: Update `Run` signature to accept `semverMode`**

Already done in Task 3.

**Step 8: Run tests and lint**

Run: `make test && make lint`
Expected: PASS.

**Step 9: Commit**

```bash
git add -A
git commit -m "feat: add breaking-change detection and reporting"
```

---

### Task 8: Add upgrader tests for breaking-change skip behavior

**Files:**
- Modify: `internal/upgrader/upgrader_test.go`

**Step 1: Write tests for force flag behavior**

Test that `ApplyAllUpgrades` with a `force` parameter skips breaking-change actions when `force=false` and upgrades them when `force=true`.

Note: This depends on how the upgrader API evolves. If breaking-change filtering happens in `cmd/root.go` rather than the upgrader, test that the caller correctly filters.

**Step 2: Run tests**

Run: `make test`
Expected: PASS.

**Step 3: Final commit (if changes)**

```bash
git add -A
git commit -m "test: add upgrader breaking-change skip/force tests"
```

---

### Task 9: Final verification and cleanup

**Step 1: Run full test suite**

```bash
make test
```

**Step 2: Run lint**

```bash
make fmt && make mod-tidy && make lint
```

**Step 3: Verify `--help` output**

```bash
go run . --help
```

Confirm `--semver` and `--force` flags appear with correct descriptions.

**Step 4: Final squash/commit if needed**

Ensure the two main feature commits are clean:
1. `feat: add major-tag update mode as default with --semver opt-in`
2. `feat: add breaking-change detection and reporting`
