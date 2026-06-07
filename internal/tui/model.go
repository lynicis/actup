package tui

import (
	"context"
	"os"
	"os/signal"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lynicis/actup/internal/parser"
)

type model struct {
	state       state
	actions     []parser.ActionRef
	token       string
	dryRun      bool
	spinner     spinner.Model
	statusMsg   string
	items       []ActionItem
	selectedSet map[int]bool
	cursor      int
	progress    []progressItem
	summary     summaryResult
	quitting    bool
}

// Run launches the interactive TUI.
func Run(ctx context.Context, actions []parser.ActionRef, token string, dryRun bool) error {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt)

	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = normalStyle

	m := model{
		actions:     actions,
		token:       token,
		dryRun:      dryRun,
		spinner:     sp,
		statusMsg:   "Scanning workflow files...",
		selectedSet: make(map[int]bool),
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

	go func() {
		<-sigChan
		p.Quit()
	}()

	_, err := p.Run()
	return err
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,
		m.loadActions,
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch m.state {
	case stateLoading:
		return m.updateLoading(msg)
	case stateChecklist:
		return m.updateChecklist(msg)
	case stateProgress:
		return m.updateProgress(msg)
	case stateSummary:
		return m.updateSummary(msg)
	}
	return m, nil
}

func (m model) View() string {
	switch m.state {
	case stateLoading:
		return m.viewLoading()
	case stateChecklist:
		return m.viewChecklist()
	case stateProgress:
		return m.viewProgress()
	case stateSummary:
		return m.viewSummary()
	}
	return ""
}
