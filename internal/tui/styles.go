package tui

import "github.com/charmbracelet/lipgloss"

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
