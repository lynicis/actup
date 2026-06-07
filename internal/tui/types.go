package tui

import "fmt"

type state int

const (
	stateLoading state = iota
	stateChecklist
	stateProgress
	stateSummary
)

// ActionItem represents a grouped action for display in the TUI.
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
	items       []ActionItem
	selectedSet map[int]bool
}
