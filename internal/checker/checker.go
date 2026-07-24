package checker

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/lynicis/actup/internal/github"
	"github.com/lynicis/actup/internal/parser"
	"golang.org/x/sync/errgroup"
)

type githubClient interface {
	LatestTag(ctx context.Context, owner, repo string, mode github.TagMode) (string, error)
}

type OutdatedAction struct {
	Owner   string
	Repo    string
	Current string
	Latest  string
	File    string
	Line    int
}

type Checker struct {
	client     githubClient
	semverMode bool
	majorVer   int
	cfgActions map[string]string
}

func New(client githubClient, semverMode bool, majorVer int, cfgActions map[string]string) *Checker {
	return &Checker{
		client:     client,
		semverMode: semverMode,
		majorVer:   majorVer,
		cfgActions: cfgActions,
	}
}

func (c *Checker) Run(ctx context.Context, actions []parser.ActionRef) ([]OutdatedAction, error) {
	groups := parser.GroupActions(actions)

	eg, egCtx := errgroup.WithContext(ctx)
	eg.SetLimit(5)

	var mu sync.Mutex
	var outdated []OutdatedAction

	for key, refs := range groups {
		eg.Go(func() error {
			owner := refs[0].Owner
			repo := refs[0].Repo

			resolvedMajor := c.majorVer
			exactVersion := ""
			if c.cfgActions != nil {
				if pin, ok := c.cfgActions[key]; ok {
					if n, err := strconv.Atoi(pin); err == nil {
						resolvedMajor = n
					} else if pin == "skip" {
						return nil
					} else {
						exactVersion = pin
					}
				}
			}

			var latest string
			var err error
			if exactVersion != "" {
				latest = exactVersion
			} else {
				latest, err = c.client.LatestTag(egCtx, owner, repo, github.TagMode{Semver: c.semverMode, Major: resolvedMajor})
				if err != nil {
					return fmt.Errorf("fetch latest for %s: %w", key, err)
				}
			}

			var localOutdated []OutdatedAction
			for _, ref := range refs {
				if ref.Current != latest {
					localOutdated = append(localOutdated, OutdatedAction{
						Owner:   ref.Owner,
						Repo:    ref.Repo,
						Current: ref.Current,
						Latest:  latest,
						File:    ref.File,
						Line:    ref.Line,
					})
				}
			}

			if len(localOutdated) > 0 {
				mu.Lock()
				outdated = append(outdated, localOutdated...)
				mu.Unlock()
			}

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return outdated, nil
}
