package breakingchanges

import (
	"strings"
	"testing"
)

func TestCheck_V2ToV3(t *testing.T) {
	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	changes := reg.Check("actions/checkout", "v2", "v3", nil)
	if len(changes) == 0 {
		t.Fatal("expected breaking changes for actions/checkout v2→v3")
	}

	found := false
	for _, c := range changes {
		if strings.Contains(c.Message, "Node 16") {
			found = true
		}
	}
	if !found {
		t.Error("expected checkout v3 Node 16 runtime note in breaking changes")
	}
}

// Formerly "no entry"; the registry now carries the checkout 3→4 runtime note.
func TestCheck_V3ToV4_NoEntry(t *testing.T) {
	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	changes := reg.Check("actions/checkout", "v3", "v4", nil)
	if len(changes) != 1 {
		t.Fatalf("expected 1 breaking change for v3→v4, got %d", len(changes))
	}
	if !strings.Contains(changes[0].Message, "Node 20") {
		t.Errorf("expected Node 20 runtime note, got %q", changes[0].Message)
	}
}

func TestCheck_UnknownAction(t *testing.T) {
	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	changes := reg.Check("unknown/action", "v1", "v2", nil)
	if len(changes) != 0 {
		t.Errorf("expected no breaking changes for unknown action, got %d", len(changes))
	}
}

func TestCheck_SmokeTestSeededData(t *testing.T) {
	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	changes := reg.Check("actions/checkout", "v2", "v3", nil)
	if len(changes) < 1 {
		t.Error("smoke test: expected at least 1 breaking change for actions/checkout v2→v3")
	}
}

func TestCheck_FiltersUnusedInput(t *testing.T) {
	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	without := reg.Check("codecov/codecov-action", "v3", "v4", nil)
	if len(without) != 0 {
		t.Errorf("expected no changes without the file input, got %d", len(without))
	}

	with := reg.Check("codecov/codecov-action", "v3", "v4", []string{"file"})
	if len(with) != 1 || with[0].Input != "file" {
		t.Errorf("expected the file deprecation, got %v", with)
	}
}

func TestCheck_GeneralNoteAlwaysShown(t *testing.T) {
	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	for _, inputs := range [][]string{nil, {}, {"file"}} {
		changes := reg.Check("actions/checkout", "v3", "v4", inputs)
		if len(changes) != 1 {
			t.Errorf("inputs %v: expected general note, got %v", inputs, changes)
		}
	}
}

func TestCheck_ToMajorZeroCoversLater(t *testing.T) {
	reg := New([]Entry{{
		Action:    "owner/repo",
		FromMajor: 2,
		ToMajor:   0,
		Changes:   []BreakingChange{{Message: "still relevant"}},
	}})

	for _, target := range []string{"v2", "v3", "v5"} {
		if len(reg.Check("owner/repo", "v1", target, nil)) == 0 {
			t.Errorf("expected to_major 0 entry for v1→%s", target)
		}
	}
	if len(reg.Check("owner/repo", "v2", "v5", nil)) != 0 {
		t.Error("expected no entry when already at FromMajor")
	}
}

func TestLoadRegistry_AllEntriesHaveMessage(t *testing.T) {
	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}
	if len(reg.entries) == 0 {
		t.Fatal("expected at least one entry")
	}
	for _, entry := range reg.entries {
		if len(entry.Changes) == 0 {
			t.Errorf("%s %d→%d has no changes", entry.Action, entry.FromMajor, entry.ToMajor)
		}
		for _, bc := range entry.Changes {
			if bc.Message == "" {
				t.Errorf("%s %d→%d has a change without a message", entry.Action, entry.FromMajor, entry.ToMajor)
			}
		}
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

func TestCheck_BoundaryCrossing(t *testing.T) {
	// Embedded actions/checkout entries: 2→3 and 3→4.
	reg, err := LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry failed: %v", err)
	}

	tests := []struct {
		from, to string
		want     bool
	}{
		{"v1", "v4", true},
		{"v2", "v4", true},
		{"v3", "v4", true},
		{"v2", "v3", true},
		{"v1", "v2", false},
	}
	for _, tt := range tests {
		changes := reg.Check("actions/checkout", tt.from, tt.to, nil)
		if got := len(changes) > 0; got != tt.want {
			t.Errorf("Check(%s→%s) has changes = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}

func TestNew_ToMajorZero(t *testing.T) {
	reg := New([]Entry{{
		Action:    "owner/repo",
		FromMajor: 2,
		ToMajor:   0,
		Changes:   []BreakingChange{{Message: "still relevant"}},
	}})

	tests := []struct {
		from, to string
		want     bool
	}{
		{"v1", "v4", true},
		{"v2", "v4", false},
		{"v1", "v2", true},
		{"v2", "v1", false},
	}
	for _, tt := range tests {
		changes := reg.Check("owner/repo", tt.from, tt.to, nil)
		if got := len(changes) > 0; got != tt.want {
			t.Errorf("Check(%s→%s) has changes = %v, want %v", tt.from, tt.to, got, tt.want)
		}
	}
}
