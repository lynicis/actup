package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lynicis/actup/internal/breakingchanges"
	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/github"
	"github.com/lynicis/actup/internal/parser"
	"github.com/lynicis/actup/internal/upgrader"
)

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("87"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	greenStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("76"))
	amberStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	headerStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("252"))
	footerStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

type state int

const (
	stateLoading state = iota
	stateChecklist
	stateDetail
	stateProgress
	stateSummary
)

// ActionItem represents a grouped action for display in the TUI.
type ActionItem struct {
	Owner           string
	Repo            string
	Current         string
	Latest          string
	FileCount       int
	Selected        bool
	UpToDate        bool
	APIError        bool
	APIErrorMsg     string
	BreakingChanges []breakingchanges.BreakingChange
	HasBreaking     bool
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

type actionsLoadedMsg struct {
	items []ActionItem
}

type applyResult struct {
	progress []progressItem
	summary  summaryResult
}

type model struct {
	state      state
	actions    []parser.ActionRef
	token      string
	dryRun     bool
	semverMode bool
	majorVer   int
	cfg        *config.Config
	spinner    spinner.Model
	statusMsg  string
	items      []ActionItem
	cursor     int
	detailItem int
	progress   []progressItem
	summary    summaryResult
	quitting   bool
}

// Run launches the interactive TUI.
func Run(ctx context.Context, actions []parser.ActionRef, token string, dryRun bool, semverMode bool, majorVer int, cfg *config.Config) error {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = normalStyle

	m := model{
		actions:    actions,
		token:      token,
		dryRun:     dryRun,
		semverMode: semverMode,
		majorVer:   majorVer,
		cfg:        cfg,
		spinner:    sp,
		statusMsg:  "Scanning workflow files...",
	}

	p := tea.NewProgram(m, tea.WithAltScreen())

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
	case stateDetail:
		return m.updateDetail(msg)
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
	case stateDetail:
		return m.viewDetail()
	case stateProgress:
		return m.viewProgress()
	case stateSummary:
		return m.viewSummary()
	}
	return ""
}

func (m model) updateLoading(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case spinner.TickMsg:
		newSpinner, cmd := m.spinner.Update(msg)
		m.spinner = newSpinner
		return m, cmd
	case actionsLoadedMsg:
		m.items = msg.items
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
					item.Selected = true
				}
			}
			return m, nil
		case "n":
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
					item.Selected = !item.Selected
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

func (m model) loadActions() tea.Msg {
	client := github.NewClient(m.token)
	grouped := parser.GroupActions(m.actions)

	registry, _ := breakingchanges.LoadRegistry()

	type fetchResult struct {
		key    string
		latest string
		err    error
	}

	skipKeys := make(map[string]bool)
	if m.cfg != nil {
		for key, pin := range m.cfg.Actions {
			if pin == "skip" {
				skipKeys[key] = true
			}
		}
	}

	resultCh := make(chan fetchResult, len(grouped))

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for key := range grouped {
		if skipKeys[key] {
			continue
		}

		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			var cfgActions map[string]string
			if m.cfg != nil {
				cfgActions = m.cfg.Actions
			}
			latest, err := github.ResolveVersion(context.Background(), client, k, m.semverMode, m.majorVer, cfgActions)
			resultCh <- fetchResult{k, latest, err}
		}(key)
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
	for key, acts := range grouped {
		if skipKeys[key] {
			continue
		}
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
		}

		if registry != nil && !item.UpToDate && !item.APIError {
			bcs := registry.Check(key, item.Current, item.Latest)
			item.BreakingChanges = bcs
			item.HasBreaking = len(bcs) > 0
		}

		items = append(items, item)
	}

	return actionsLoadedMsg{
		items: items,
	}
}

func (m model) applyUpgrades() tea.Msg {
	upgrades := make(map[string]upgrader.Upgrade)

	for _, item := range m.items {
		if !item.Selected {
			continue
		}

		key := item.Owner + "/" + item.Repo
		for _, action := range m.actions {
			if action.Owner == item.Owner && action.Repo == item.Repo {
				upgradeKey := fmt.Sprintf("%s:%s:%d", key, action.File, action.Line)
				upgrades[upgradeKey] = upgrader.Upgrade{
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
		} else if item.Selected {
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

	summary := summaryResult{
		Upgraded:      upgradedCount,
		UpToDate:      upToDateCount,
		APIErrors:     apiErrCount,
		UpgradedFiles: upgradedFiles,
	}

	return applyResult{progress: progress, summary: summary}
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

	b.WriteString(headerStyle.Render(fmt.Sprintf("  actup — %d actions found across %d files", len(m.items), len(fileCount))))
	b.WriteString("\n\n")

	maxNameLen := 0
	for _, item := range m.items {
		nameLen := len(item.Owner) + 1 + len(item.Repo)
		if nameLen > maxNameLen {
			maxNameLen = nameLen
		}
	}
	nameWidth := maxNameLen + 2

	for i, item := range m.items {
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

		if item.HasBreaking {
			suffix += " " + amberStyle.Render("⚠ breaking changes")
		}

		if i == m.cursor {
			style = cursorStyle
		}

		change := ""
		if !item.UpToDate && !item.APIError {
			change = fmt.Sprintf(" → %s", item.Latest)
		}

		desc := ""
		if item.FileCount > 1 {
			desc = fmt.Sprintf(" (%d files)", item.FileCount)
		}

		name := item.Owner + "/" + item.Repo
		padding := strings.Repeat(" ", nameWidth-len(name))
		line := fmt.Sprintf("  %s %s%s %s%s%s%s", prefix, name, padding, item.Current, change, suffix, desc)
		b.WriteString(style.Render(line))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(footerStyle.Render("  [space] toggle  [a] all  [n] none  [i] info  [enter] apply  [q] quit"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewDetail() string {
	var b strings.Builder

	item := m.items[m.detailItem]

	b.WriteString(headerStyle.Render(fmt.Sprintf("  %s/%s — Breaking Changes", item.Owner, item.Repo)))
	b.WriteString("\n\n")

	fmt.Fprintf(&b, "  %s → %s\n\n", item.Current, item.Latest)

	for _, bc := range item.BreakingChanges {
		b.WriteString(amberStyle.Render(fmt.Sprintf("  ⚠ %s", bc.Message)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(footerStyle.Render("  [esc] back"))
	b.WriteString("\n")

	return b.String()
}

func (m model) viewProgress() string {
	var b strings.Builder
	b.WriteString(headerStyle.Render("  Upgrading actions..."))
	b.WriteString("\n\n")

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

		fmt.Fprintf(&b, "  %s %s/%s → %s\n", status, p.Owner, p.Repo, p.NewTag)
	}

	return b.String()
}

func (m model) viewSummary() string {
	var b strings.Builder

	b.WriteString("\n")

	if m.dryRun {
		b.WriteString(greenStyle.Render(fmt.Sprintf("  ✅ %d actions would be upgraded across %d files", m.summary.Upgraded, m.summary.UpgradedFiles)))
	} else {
		b.WriteString(greenStyle.Render(fmt.Sprintf("  ✅ %d actions upgraded across %d files", m.summary.Upgraded, m.summary.UpgradedFiles)))
	}
	b.WriteString("\n")

	if m.summary.UpToDate > 0 {
		b.WriteString(dimStyle.Render(fmt.Sprintf("  ⏭ %d action(s) skipped (up to date)", m.summary.UpToDate)))
		b.WriteString("\n")
	}

	if m.summary.APIErrors > 0 {
		b.WriteString(amberStyle.Render(fmt.Sprintf("  ⚠ %d action(s) skipped (API error)", m.summary.APIErrors)))
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(footerStyle.Render("  Press 'q' or Ctrl+C to exit"))
	b.WriteString("\n")

	return b.String()
}
