package checker

import (
	"context"
	"fmt"

	"github.com/lynicis/actup/internal/github"
	"github.com/lynicis/actup/internal/parser"
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
}

func New(client githubClient, semverMode bool, majorVer int) *Checker {
	return &Checker{client: client, semverMode: semverMode, majorVer: majorVer}
}

func (c *Checker) Run(ctx context.Context, actions []parser.ActionRef) ([]OutdatedAction, error) {
	groups := parser.GroupActions(actions)

	var outdated []OutdatedAction
	for key, refs := range groups {
		owner := refs[0].Owner
		repo := refs[0].Repo
		latest, err := c.client.LatestTag(ctx, owner, repo, github.TagMode{Semver: c.semverMode, Major: c.majorVer})
		if err != nil {
			return nil, fmt.Errorf("fetch latest for %s: %w", key, err)
		}
		for _, ref := range refs {
			if ref.Current != latest {
				outdated = append(outdated, OutdatedAction{
					Owner:   ref.Owner,
					Repo:    ref.Repo,
					Current: ref.Current,
					Latest:  latest,
					File:    ref.File,
					Line:    ref.Line,
				})
			}
		}
	}
	return outdated, nil
}
