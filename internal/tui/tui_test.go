package tui

import (
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
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
