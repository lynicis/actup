package tui

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/lynicis/actup/internal/breakingchanges"
	"github.com/lynicis/actup/internal/config"
	"github.com/lynicis/actup/internal/github"
	"github.com/lynicis/actup/internal/parser"
	"github.com/lynicis/actup/internal/upgrader"
	"github.com/lynicis/actup/internal/vulnscanner"
)

var (
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("205"))
	normalStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("87"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
	greenStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("76"))
	amberStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
	redStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
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
	Currents        []string // all distinct versions, first-seen order
	Latest          string
	LatestSHA       string
	FileCount       int
	Selected        bool
	UpToDate        bool
	APIError        bool
	APIErrorMsg     string
	BreakingChanges []breakingchanges.BreakingChange
	HasBreaking     bool
	Vulns           []vulnscanner.Vulnerability
	HasVulns        bool
	VulnErrorMsg    string
}

type progressItem struct {
	Owner  string
	Repo   string
	NewTag string
	Status string
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
	shaMode    bool
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
func Run(actions []parser.ActionRef, token string, dryRun bool, semverMode bool, majorVer int, shaMode bool, cfg *config.Config) error {
	sp := spinner.New()
	sp.Spinner = spinner.Dot
	sp.Style = normalStyle

	m := model{
		actions:    actions,
		token:      token,
		dryRun:     dryRun,
		semverMode: semverMode,
		shaMode:    shaMode,
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
			if m.cursor >= 0 && m.cursor < len(m.items) && (len(m.items[m.cursor].BreakingChanges) > 0 || len(m.items[m.cursor].Vulns) > 0) {
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

// allAtLatest reports whether every ref already uses the latest version.
func allAtLatest(acts []parser.ActionRef, latest string) bool {
	for _, act := range acts {
		if act.Version() != latest {
			return false
		}
	}
	return true
}

// allAtLatestSHA reports whether every ref is pinned to the latest SHA and
// carries the latest tag in its comment.
func allAtLatestSHA(acts []parser.ActionRef, latestSHA, latestTag string) bool {
	for _, act := range acts {
		if act.SHA != latestSHA || act.Tag != latestTag {
			return false
		}
	}
	return true
}

// allAtLatestMixed reports whether tag refs match the latest version while
// SHA refs match both the latest SHA and its tag.
func allAtLatestMixed(acts []parser.ActionRef, latestSHA, latestTag string) bool {
	for _, act := range acts {
		if act.Pinned() {
			if act.SHA != latestSHA || act.Tag != latestTag {
				return false
			}
			continue
		}
		if act.Version() != latestTag {
			return false
		}
	}
	return true
}

// versionsInUse returns distinct non-empty comparison versions in first-seen
// order. Bare SHA refs have no version.
func versionsInUse(refs []parser.ActionRef) []string {
	seen := make(map[string]bool, len(refs))
	var versions []string
	for _, ref := range refs {
		v := ref.Version()
		if v == "" || seen[v] {
			continue
		}
		seen[v] = true
		versions = append(versions, v)
	}
	return versions
}

// anyRefPinned reports whether any ref is a commit SHA.
func anyRefPinned(refs []parser.ActionRef) bool {
	for _, ref := range refs {
		if ref.Pinned() {
			return true
		}
	}
	return false
}

// breakingForAll unions breaking changes for every distinct current version,
// deduping on (Type, Input, Message) in first-seen order.
func breakingForAll(registry *breakingchanges.Registry, key string, currents []string, inputsByVersion map[string][]string, latest string) []breakingchanges.BreakingChange {
	if registry == nil {
		return nil
	}
	seen := map[string]bool{}
	var results []breakingchanges.BreakingChange
	for _, current := range currents {
		for _, bc := range registry.Check(key, current, latest, inputsByVersion[current]) {
			id := bc.Type + "\x00" + bc.Input + "\x00" + bc.Message
			if seen[id] {
				continue
			}
			seen[id] = true
			results = append(results, bc)
		}
	}
	return results
}

func (m model) loadActions() tea.Msg {
	client := github.NewClient(m.token)
	grouped := parser.GroupActions(m.actions)

	registry, _ := breakingchanges.LoadRegistry()

	cfgActions := m.cfg.EffectivePins()
	skipKeys := make(map[string]bool)
	for key, pin := range cfgActions {
		if pin == "skip" {
			skipKeys[key] = true
		}
	}

	keys := make([]string, 0, len(grouped))
	for key := range grouped {
		if !skipKeys[key] {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys) // skipped keys filtered above; sorted for stable display
	results := github.ResolveAll(context.Background(), client, keys, m.semverMode, m.majorVer, cfgActions)

	var items []ActionItem
	for _, key := range keys {
		acts := grouped[key]
		parts := strings.SplitN(key, "/", 2)
		owner := parts[0]
		repo := parts[1]
		currents := versionsInUse(acts)
		inputsByVersion := map[string][]string{}
		for _, ref := range acts {
			if v := ref.Version(); v != "" {
				inputsByVersion[v] = append(inputsByVersion[v], ref.Inputs...)
			}
		}

		item := ActionItem{
			Owner:     owner,
			Repo:      repo,
			Current:   acts[0].Current,
			Currents:  currents,
			FileCount: len(acts),
		}

		if r, ok := results[key]; ok {
			if r.Err != nil {
				item.APIError = true
				item.APIErrorMsg = r.Err.Error()
			} else {
				item.Latest = r.Latest
				anyPinned := anyRefPinned(acts)
				if m.shaMode || anyPinned {
					sha, err := client.ResolveTagSHA(context.Background(), owner, repo, r.Latest)
					if err != nil {
						item.APIError = true
						item.APIErrorMsg = err.Error()
					} else {
						item.LatestSHA = sha
						if m.shaMode {
							item.UpToDate = allAtLatestSHA(acts, item.LatestSHA, r.Latest)
						} else {
							item.UpToDate = allAtLatestMixed(acts, item.LatestSHA, r.Latest)
						}
					}
				} else {
					item.UpToDate = allAtLatest(acts, r.Latest)
				}
			}
		}

		if !item.UpToDate && !item.APIError {
			item.Selected = true
		}

		if registry != nil && !item.UpToDate && !item.APIError {
			item.BreakingChanges = breakingForAll(registry, key, item.Currents, inputsByVersion, item.Latest)
		}

		items = append(items, item)
	}

	// Vulnerability audit against OSV.dev. Failures degrade gracefully to
	// an amber note so the checklist still loads offline.
	vulnClient := vulnscanner.NewClient()
	findings, err := vulnClient.Scan(context.Background(), m.actions)
	if err != nil {
		for i := range items {
			items[i].VulnErrorMsg = err.Error()
		}
	} else {
		for _, f := range findings {
			key := strings.ToLower(f.Owner + "/" + f.Repo)
			for i := range items {
				if strings.ToLower(items[i].Owner+"/"+items[i].Repo) == key {
					items[i].Vulns = f.Vulns
				}
			}
		}
	}

	return actionsLoadedMsg{
		items: items,
	}
}

func (m model) applyUpgrades() tea.Msg {
	var upgrades []upgrader.Upgrade

	for _, item := range m.items {
		if !item.Selected {
			continue
		}

		var itemRefs []parser.ActionRef
		for _, action := range m.actions {
			if action.Owner == item.Owner && action.Repo == item.Repo {
				itemRefs = append(itemRefs, action)
			}
		}
		for _, action := range itemRefs {
			writeSHA := m.shaMode || action.Pinned()
			if writeSHA {
				if action.SHA == item.LatestSHA && action.Tag == item.Latest {
					continue
				}
			} else if action.Version() == item.Latest {
				continue
			}
			newSHA := ""
			if writeSHA {
				newSHA = item.LatestSHA
			}
			upgrades = append(upgrades, upgrader.Upgrade{
				Action: action,
				NewTag: item.Latest,
				NewSHA: newSHA,
			})
		}
	}

	err := upgrader.ApplyAllUpgrades(upgrades, m.dryRun)
	if !m.dryRun && err == nil && m.cfg != nil && m.cfg.IsManaged() {
		if m.cfg.Actions == nil {
			m.cfg.Actions = map[string]string{}
		}
		for _, item := range m.items {
			if item.Selected && !item.UpToDate && !item.APIError {
				m.cfg.Actions[item.Owner+"/"+item.Repo] = item.Latest
			}
		}
		// A failed save must not lose the run summary; the workflows were rewritten.
		if saveErr := config.SaveDefault(m.cfg); saveErr != nil {
			err = saveErr
		}
	}

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

// versionSummary renders "v2 ×1, v4 ×1" for mixed-version groups, sorted for
// deterministic output.
func (m model) versionSummary(item ActionItem) string {
	counts := map[string]int{}
	for _, a := range m.actions {
		if a.Owner == item.Owner && a.Repo == item.Repo {
			v := a.Version()
			if v == "" {
				v = a.Current
			}
			counts[v]++
		}
	}
	if len(counts) == 0 {
		for _, v := range item.Currents {
			counts[v]++
		}
	}
	versions := make([]string, 0, len(counts))
	for v := range counts {
		versions = append(versions, v)
	}
	sort.Strings(versions)
	parts := make([]string, 0, len(versions))
	for _, v := range versions {
		parts = append(parts, fmt.Sprintf("%s ×%d", v, counts[v]))
	}
	return strings.Join(parts, ", ")
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

		if len(item.BreakingChanges) > 0 {
			suffix += " " + amberStyle.Render("⚠ breaking changes")
		}

		if len(item.Vulns) > 0 {
			suffix += " " + redStyle.Render("⚠ vulnerable")
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
		current := item.Current
		if len(item.Currents) > 1 {
			current = m.versionSummary(item)
		}
		line := fmt.Sprintf("  %s %s%s %s%s%s%s", prefix, name, padding, current, change, suffix, desc)
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

	if len(item.Vulns) > 0 {
		b.WriteString(headerStyle.Render(fmt.Sprintf("  %s/%s — Vulnerabilities", item.Owner, item.Repo)))
		b.WriteString("\n\n")

		currentLabel := item.Current
		if len(item.Currents) > 1 {
			currentLabel = strings.Join(item.Currents, ", ")
		}
		fmt.Fprintf(&b, "  current version: %s\n\n", currentLabel)

		for _, v := range item.Vulns {
			severity := v.Severity
			if severity == "" {
				severity = "UNKNOWN"
			}
			fixed := "no fix known"
			if v.Fixed != "" {
				fixed = "fixed in " + v.Fixed
			}
			b.WriteString(redStyle.Render(fmt.Sprintf("  ⚠ %s [%s] (%s)", v.ID, severity, fixed)))
			b.WriteString("\n")
			if v.Summary != "" {
				fmt.Fprintf(&b, "    %s\n", v.Summary)
			}
			if v.URL != "" {
				b.WriteString(dimStyle.Render(fmt.Sprintf("    %s\n", v.URL)))
			}
			b.WriteString("\n")
		}

		if len(item.BreakingChanges) > 0 {
			b.WriteString(headerStyle.Render("  Breaking changes when upgrading to " + item.Latest))
			b.WriteString("\n\n")
			for _, bc := range item.BreakingChanges {
				b.WriteString(amberStyle.Render(fmt.Sprintf("  ⚠ %s", bc.Message)))
				b.WriteString("\n")
			}
		}

		b.WriteString("\n")
		b.WriteString(footerStyle.Render("  [esc] back"))
		b.WriteString("\n")

		return b.String()
	}

	b.WriteString(headerStyle.Render(fmt.Sprintf("  %s/%s — Breaking Changes", item.Owner, item.Repo)))
	b.WriteString("\n\n")

	if len(item.Currents) > 1 {
		for _, current := range item.Currents {
			fmt.Fprintf(&b, "  %s → %s\n", current, item.Latest)
		}
		b.WriteString("\n")
	} else {
		fmt.Fprintf(&b, "  %s → %s\n\n", item.Current, item.Latest)
	}

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
