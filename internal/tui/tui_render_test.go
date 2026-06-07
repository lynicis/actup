package tui

import (
	"strings"
	"testing"

	"github.com/lynicis/actup/internal/parser"
)

func TestViewChecklistRendering(t *testing.T) {
	m := model{
		state: stateChecklist,
		items: []ActionItem{
			{Owner: "actions", Repo: "upload-artifact", Current: "v7", Latest: "v7.0.1", Selected: true},
			{Owner: "golangci", Repo: "golangci-lint-action", Current: "v9", Latest: "v9.2.1", Selected: true},
			{Owner: "goreleaser", Repo: "goreleaser-action", Current: "v7", Latest: "v7.2.2", Selected: true},
			{Owner: "actions", Repo: "checkout", Current: "v6", Latest: "v6.0.3", Selected: true, FileCount: 5},
			{Owner: "actions", Repo: "setup-go", Current: "v6", Latest: "v6.4.0", Selected: true, FileCount: 5},
		},
		actions: []parser.ActionRef{
			{File: "ci.yml"}, {File: "release.yml"},
		},
		selectedSet: map[int]bool{0: true, 1: true, 2: true, 3: true, 4: true},
	}

	output := m.viewChecklist()

	// Verify no tabs remain in output
	if strings.Contains(output, "\t") {
		t.Error("output still contains tab characters")
	}

	// Verify each line has consistent spacing by checking column positions
	lines := strings.Split(output, "\n")
	var dataLines []string
	for _, line := range lines {
		if strings.Contains(line, "[") && strings.Contains(line, "/") {
			dataLines = append(dataLines, line)
		}
	}

	if len(dataLines) != 5 {
		t.Fatalf("expected 5 data lines, got %d", len(dataLines))
	}

	// Check that version arrows appear at exactly the same column position
	arrowPositions := make(map[int]int)
	for i, line := range dataLines {
		pos := strings.Index(line, "→")
		if pos == -1 {
			t.Fatalf("line %d missing arrow: %q", i, line)
		}
		arrowPositions[pos]++
	}

	if len(arrowPositions) != 1 {
		var positions []int
		for pos := range arrowPositions {
			positions = append(positions, pos)
		}
		t.Errorf("arrow positions are inconsistent: %v (counts: %v)", positions, arrowPositions)
	}

	// Check that each line starts with the exact same prefix (2 spaces + prefix + space)
	expectedPrefix := "  [✓] "
	for i, line := range dataLines {
		cleanLine := stripANSI(line)
		if !strings.HasPrefix(cleanLine, expectedPrefix) {
			t.Errorf("line %d doesn't start with expected prefix %q: got %q", i, expectedPrefix, cleanLine)
		}
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == 27 { // ESC character
			inEscape = true
			continue
		}
		if inEscape {
			if c == 'm' {
				inEscape = false
			}
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}
