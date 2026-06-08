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
