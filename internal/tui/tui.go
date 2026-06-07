package tui

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lynicis/actup/internal/github"
	"github.com/lynicis/actup/internal/parser"
	"github.com/lynicis/actup/internal/upgrader"
)

type state int

const (
	stateLoading state = iota
	stateChecklist
	stateProgress
	stateSummary
)

type ActionItem struct {
	Owner       string
	Repo        string
	Current     string
	Latest      string
	FileCount   int
	Selected    bool
	UpToDate    bool
	APIError    bool
	APIErrorMsg string
}

func (a ActionItem) Title() string {
	return fmt.Sprintf("%s/%s", a.Owner, a.Repo)
}

func (a ActionItem) Description() string {
	if a.UpToDate {
		return fmt.Sprintf("%s (up to date)", a.Latest)
	}
	if a.APIError {
		return "API error"
	}
	return fmt.Sprintf("%s → %s (%d files)", a.Current, a.Latest, a.FileCount)
}

func (a ActionItem) FilterValue() string {
	return fmt.Sprintf("%s/%s", a.Owner, a.Repo)
}

type model struct {
	state       state
	actions     []parser.ActionRef
	token       string
	dryRun      bool
	spinner     spinner.Model
	statusMsg   string
	items       []ActionItem
	list        list.Model
	selectedSet map[int]bool
	ghClient    *github.Client
	progress    []progressItem
	summary     summaryResult
	quitting    bool
}

type progressItem struct {
	Owner  string
	Repo   string
	NewTag string
	Status string
	Error  error
}

type summaryResult struct {
	Upgraded      int
	UpToDate      int
	APIErrors     int
	UpgradedFiles int
}

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	greenStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("76"))
	amberStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	footerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

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
		func() tea.Msg {
			m.spinner.Tick()
			return spinner.TickMsg{}
		},
		m.loadActions,
	)
}

func (m model) loadActions() tea.Msg {
	m.ghClient = github.NewClient(m.token)

	grouped := parser.GroupActions(m.actions)

	type fetchResult struct {
		key    string
		latest string
		err    error
	}

	resultCh := make(chan fetchResult, len(grouped))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for key, acts := range grouped {
		wg.Add(1)
		go func(key string, acts []parser.ActionRef) {
			sem <- struct{}{}
			defer func() { <-sem }()
			defer wg.Done()

			parts := strings.SplitN(key, "/", 2)
			owner := parts[0]
			repo := parts[1]

			latest, err := m.ghClient.LatestSemverTag(context.Background(), owner, repo)

			select {
			case resultCh <- fetchResult{key, latest, err}:
			default:
			}
		}(key, acts)
	}

	go func() {
		wg.Wait()
		close(resultCh)
	}()

	results := make(map[string]fetchResult)
	for r := range resultCh {
		results[r.key] = r
	}

	var items []ActionItem
	itemIndex := 0
	for key, acts := range grouped {
		parts := strings.SplitN(key, "/", 2)
		owner := parts[0]
		repo := parts[1]

		item := ActionItem{
			Owner:     owner,
			Repo:      repo,
			Current:   acts[0].Current,
			FileCount: len(acts),
		}

		if r, ok := results[key]; ok {
			if r.err != nil {
				item.APIError = true
				item.APIErrorMsg = r.err.Error()
			} else {
				item.Latest = r.latest
				if r.latest == acts[0].Current {
					item.UpToDate = true
				}
			}
		}

		if !item.UpToDate && !item.APIError {
			item.Selected = true
			m.selectedSet[itemIndex] = true
		}

		items = append(items, item)
		itemIndex++
	}

	m.items = items
	m.state = stateChecklist

	l := list.New(m.makeListItems(), list.NewDefaultDelegate(), 0, 0)
	l.SetShowTitle(false)
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetWidth(80)
	l.SetHeight(20)
	m.list = l

	return nil
}

func (m model) makeListItems() []list.Item {
	items := make([]list.Item, len(m.items))
	for i, item := range m.items {
		items[i] = item
	}
	return items
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

func (m model) updateLoading(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		newSpinner, cmd := m.spinner.Update(msg)
		m.spinner = newSpinner
		return m, cmd

	case []ActionItem:
		m.items = msg
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

		case " ":
			selected := m.list.Index()
			if selected >= 0 && selected < len(m.items) {
				item := &m.items[selected]
				if !item.UpToDate && !item.APIError {
					m.selectedSet[selected] = !m.selectedSet[selected]
					item.Selected = m.selectedSet[selected]
				}
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m model) updateProgress(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case []progressItem:
		m.progress = msg
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

func (m model) applyUpgrades() tea.Msg {
	upgrades := make(map[string]upgrader.Upgrade)

	for i, item := range m.items {
		if !m.selectedSet[i] {
			continue
		}

		key := item.Owner + "/" + item.Repo
		for _, action := range m.actions {
			if action.Owner == item.Owner && action.Repo == item.Repo {
				upgrades[key] = upgrader.Upgrade{
					Action: action,
					NewTag: item.Latest,
				}
			}
		}
	}

	_, err := upgrader.ApplyAllUpgrades(upgrades, m.dryRun)

	var progress []progressItem
	upgradedCount := 0
	upToDateCount := 0
	apiErrCount := 0
	upgradedFiles := 0

	for _, item := range m.items {
		if item.UpToDate {
			upToDateCount++
			progress = append(progress, progressItem{
				Owner:  item.Owner,
				Repo:   item.Repo,
				NewTag: item.Latest,
				Status: "up to date",
			})
		} else if item.APIError {
			apiErrCount++
			progress = append(progress, progressItem{
				Owner:  item.Owner,
				Repo:   item.Repo,
				NewTag: item.Latest,
				Status: "API error",
				Error:  fmt.Errorf("%s", item.APIErrorMsg),
			})
		} else if m.selectedSet[getItemIndex(m.items, item)] {
			upgradedCount++
			upgradedFiles += item.FileCount
			progress = append(progress, progressItem{
				Owner:  item.Owner,
				Repo:   item.Repo,
				NewTag: item.Latest,
				Status: "done",
			})
		}
	}

	_ = err

	m.summary = summaryResult{
		Upgraded:      upgradedCount,
		UpToDate:      upToDateCount,
		APIErrors:     apiErrCount,
		UpgradedFiles: upgradedFiles,
	}

	return progress
}

func getItemIndex(items []ActionItem, target ActionItem) int {
	for i, item := range items {
		if item.Owner == target.Owner && item.Repo == target.Repo {
			return i
		}
	}
	return -1
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

func (m model) viewLoading() string {
	return fmt.Sprintf("\n\n   %s %s\n\n", m.spinner.View(), m.statusMsg)
}

func (m model) viewChecklist() string {
	var b strings.Builder

	fileCount := make(map[string]bool)
	for _, action := range m.actions {
		fileCount[action.File] = true
	}

	b.WriteString(headerStyle.Render(fmt.Sprintf("  actup — %d actions found across %d files\n\n", len(m.items), len(fileCount))))

	for _, item := range m.items {
		prefix := "[ ]"
		style := normalStyle
		suffix := ""

		if item.Selected {
			prefix = "[✓]"
			style = selectedStyle
		}

		if item.UpToDate {
			prefix = "[⏭]"
			style = dimStyle
			suffix = " (up to date)"
		}

		if item.APIError {
			prefix = "[⚠]"
			style = amberStyle
			suffix = " (API error)"
		}

		change := ""
		if !item.UpToDate && !item.APIError {
			change = fmt.Sprintf(" → %s", item.Latest)
		}

		desc := ""
		if item.FileCount > 1 {
			desc = fmt.Sprintf(" (%d files)", item.FileCount)
		}

		b.WriteString(style.Render(fmt.Sprintf("  %s %s/%s\t%s%s%s%s\n", prefix, item.Owner, item.Repo, item.Current, change, suffix, desc)))
	}

	b.WriteString("\n")
	b.WriteString(footerStyle.Render("  [space] toggle  [a] all  [n] none  [enter] apply  [q] quit\n"))

	return b.String()
}

func (m model) viewProgress() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("  Upgrading actions...\n\n"))

	for _, p := range m.progress {
		status := ""
		switch p.Status {
		case "done":
			status = greenStyle.Render("✓")
		case "up to date":
			status = dimStyle.Render("⏭")
		case "API error":
			status = amberStyle.Render("⚠")
		default:
			status = m.spinner.View()
		}

		b.WriteString(fmt.Sprintf("  %s %s/%s → %s\n", status, p.Owner, p.Repo, p.NewTag))
	}

	return b.String()
}

func (m model) viewSummary() string {
	var b strings.Builder

	b.WriteString("\n")

	if m.dryRun {
		b.WriteString(greenStyle.Render(fmt.Sprintf("  ✅ %d actions would be upgraded across %d files\n", m.summary.Upgraded, m.summary.UpgradedFiles)))
	} else {
		b.WriteString(greenStyle.Render(fmt.Sprintf("  ✅ %d actions upgraded across %d files\n", m.summary.Upgraded, m.summary.UpgradedFiles)))
	}

	if m.summary.UpToDate > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ⏭ %d action(s) skipped (up to date)\n", m.summary.UpToDate)))
	}

	if m.summary.APIErrors > 0 {
		b.WriteString(amberStyle.Render(fmt.Sprintf("  ⚠ %d action(s) skipped (API error)\n", m.summary.APIErrors)))
	}

	b.WriteString("\n")
	b.WriteString(footerStyle.Render("  Press 'q' or Ctrl+C to exit\n"))

	return b.String()
}
