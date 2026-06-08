package tui

import (
	"context"
	"fmt"
	"strings"
	"sync"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lynicis/actup/internal/github"
	"github.com/lynicis/actup/internal/parser"
	"github.com/lynicis/actup/internal/upgrader"
)

type applyResult struct {
	progress []progressItem
	summary  summaryResult
}

func (m model) loadActions() tea.Msg {
	client := github.NewClient(m.token)
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

			latest, err := client.LatestTag(context.Background(), owner, repo, m.semverMode)
			resultCh <- fetchResult{key, latest, err}
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
	selectedSet := make(map[int]bool)
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
			selectedSet[itemIndex] = true
		}

		items = append(items, item)
		itemIndex++
	}

	return actionsLoadedMsg{
		items:       items,
		selectedSet: selectedSet,
	}
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

	for i, item := range m.items {
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
		} else if m.selectedSet[i] {
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
