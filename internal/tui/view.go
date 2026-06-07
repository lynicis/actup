package tui

import (
	"fmt"
	"strings"
)

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
	b.WriteString(footerStyle.Render("  [space] toggle  [a] all  [n] none  [enter] apply  [q] quit"))
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
