package tui

import (
	"github.com/lynicis/actup/internal/breakingchanges"
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
