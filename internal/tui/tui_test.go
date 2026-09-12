package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"

	"github.com/lynicis/actup/internal/parser"
)

func TestInitReturnsCommand(t *testing.T) {
	m := model{
		state: stateLoading,
	}

	cmd := m.Init()
	if cmd == nil {
		t.Fatal("expected Init() to return a non-nil command")
	}

	msg := cmd()
	batchMsg, ok := msg.(tea.BatchMsg)
	if !ok {
		t.Fatalf("expected Init() to return a tea.BatchMsg, got %T", msg)
	}
	if len(batchMsg) != 2 {
		t.Fatalf("expected BatchMsg to contain 2 commands, got %d", len(batchMsg))
	}
}

func TestUpdateLoadingTransitionsToChecklist(t *testing.T) {
	m := model{
		state: stateLoading,
	}

	msg := actionsLoadedMsg{
		items: []ActionItem{
			{Owner: "actions", Repo: "checkout", Current: "v3", Latest: "v4", Selected: true},
			{Owner: "actions", Repo: "setup-go", Current: "v4", Latest: "v5", Selected: true},
		},
	}

	newModel, cmd := m.Update(msg)
	updatedModel, ok := newModel.(model)
	if !ok {
		t.Fatalf("expected Update to return a model, got %T", newModel)
	}

	if updatedModel.state != stateChecklist {
		t.Errorf("expected state to be stateChecklist, got %d", updatedModel.state)
	}

	if len(updatedModel.items) != 2 {
		t.Errorf("expected 2 items, got %d", len(updatedModel.items))
	}

	if !updatedModel.items[0].Selected {
		t.Error("expected items[0].Selected to be true")
	}
	if !updatedModel.items[1].Selected {
		t.Error("expected items[1].Selected to be true")
	}

	if cmd != nil {
		t.Error("expected no command after transition to checklist")
	}
}

func TestUpdateLoadingIgnoresOtherMessages(t *testing.T) {
	m := model{
		state: stateLoading,
	}

	newModel, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	updatedModel, ok := newModel.(model)
	if !ok {
		t.Fatalf("expected Update to return a model, got %T", newModel)
	}

	if updatedModel.state != stateLoading {
		t.Errorf("expected state to remain stateLoading, got %d", updatedModel.state)
	}

	if cmd != nil {
		t.Error("expected no command for unhandled key message in loading state")
	}
}

func TestUpdateLoadingHandlesSpinnerTick(t *testing.T) {
	m := model{
		state: stateLoading,
	}

	cmd := m.Init()
	batchMsg := cmd().(tea.BatchMsg)

	// One of the batch commands should be the spinner tick
	var spinnerMsg tea.Msg
	for _, c := range batchMsg {
		msg := c()
		if _, ok := msg.(spinner.TickMsg); ok {
			spinnerMsg = msg
			break
		}
	}

	if spinnerMsg == nil {
		t.Fatal("expected one of the batch commands to produce a TickMsg")
	}

	newModel, cmd := m.Update(spinnerMsg)
	updatedModel, ok := newModel.(model)
	if !ok {
		t.Fatalf("expected Update to return a model, got %T", newModel)
	}

	if updatedModel.state != stateLoading {
		t.Errorf("expected state to remain stateLoading after spinner tick, got %d", updatedModel.state)
	}

	// After a spinner tick, a new tick command is typically returned
	_ = cmd
}

func TestUpdateChecklistTogglesSelection(t *testing.T) {
	m := model{
		state: stateChecklist,
		items: []ActionItem{
			{Owner: "actions", Repo: "checkout", Selected: true},
			{Owner: "actions", Repo: "setup-go", Selected: false},
		},
		cursor: 1,
	}

	// Toggle cursor item (index 1) with space
	newModel, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(" ")})
	updated := newModel.(model)
	if !updated.items[1].Selected {
		t.Errorf("expected item 1 to be selected after space toggle")
	}

	// Select none with 'n'
	newModel, _ = updated.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("n")})
	updated = newModel.(model)
	if updated.items[0].Selected || updated.items[1].Selected {
		t.Errorf("expected all items to be unselected after 'n'")
	}
}

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

func TestViewChecklistCursorHighlight(t *testing.T) {
	oldProfile := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(oldProfile)

	m := model{
		state: stateChecklist,
		items: []ActionItem{
			{Owner: "actions", Repo: "upload-artifact", Current: "v7", Latest: "v7.0.1", Selected: true},
			{Owner: "golangci", Repo: "golangci-lint-action", Current: "v9", Latest: "v9.2.1", Selected: true},
			{Owner: "goreleaser", Repo: "goreleaser-action", Current: "v7", Latest: "v7.2.2", Selected: true},
		},
		actions: []parser.ActionRef{{File: "ci.yml"}},
		cursor:  1,
	}

	output := m.viewChecklist()

	lines := strings.Split(output, "\n")
	var dataLines []string
	for _, line := range lines {
		if strings.Contains(line, "[") && strings.Contains(line, "/") {
			dataLines = append(dataLines, line)
		}
	}

	if len(dataLines) != 3 {
		t.Fatalf("expected 3 data lines, got %d", len(dataLines))
	}

	// The focused item (index 1) should contain the cursor text color escape sequence
	focusedLine := dataLines[1]
	if !strings.Contains(focusedLine, "38;5;87") {
		t.Errorf("focused line should contain cursor text color (87), got: %q", focusedLine)
	}

	// Other items should not have the cursor text color
	for i, line := range dataLines {
		if i == 1 {
			continue
		}
		if strings.Contains(line, "38;5;87") {
			t.Errorf("line %d should not contain cursor text color, got: %q", i, line)
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
