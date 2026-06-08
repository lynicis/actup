package tui

import (
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

func (m model) updateLoading(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		newSpinner, cmd := m.spinner.Update(msg)
		m.spinner = newSpinner
		return m, cmd
	case actionsLoadedMsg:
		m.items = msg.items
		m.selectedSet = msg.selectedSet
		m.state = stateChecklist
		return m, nil
	}
	return m, nil
}

func (m model) updateChecklist(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			m.state = stateProgress
			return m, m.applyUpgrades
		case "a":
			for i := range m.items {
				item := &m.items[i]
				if !item.UpToDate && !item.APIError {
					m.selectedSet[i] = true
					item.Selected = true
				}
			}
			return m, nil
		case "n":
			m.selectedSet = make(map[int]bool)
			for i := range m.items {
				m.items[i].Selected = false
			}
			return m, nil
		case "up", "k":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		case "down", "j":
			if m.cursor < len(m.items)-1 {
				m.cursor++
			}
			return m, nil
		case " ":
			if m.cursor >= 0 && m.cursor < len(m.items) {
				item := &m.items[m.cursor]
				if !item.UpToDate && !item.APIError {
					m.selectedSet[m.cursor] = !m.selectedSet[m.cursor]
					item.Selected = m.selectedSet[m.cursor]
				}
			}
			return m, nil
		case "i":
			if m.cursor >= 0 && m.cursor < len(m.items) && m.items[m.cursor].HasBreaking {
				m.detailItem = m.cursor
				m.state = stateDetail
			}
			return m, nil
		}
	}
	return m, nil
}

func (m model) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			m.state = stateChecklist
			return m, nil
		}
	}
	return m, nil
}

func (m model) updateProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case applyResult:
		m.progress = msg.progress
		m.summary = msg.summary
		m.state = stateSummary
		return m, nil
	}
	return m, nil
}

func (m model) updateSummary(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" || msg.String() == "q" {
			return m, tea.Quit
		}
	}
	return m, nil
}
