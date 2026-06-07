package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	"github.com/google/go-github/v62/github"
	"golang.org/x/mod/semver"
	"golang.org/x/oauth2"
)

var ErrNoSemverTags = errors.New("no semver tags found for repository")

type Client struct {
	client *github.Client
	cache  sync.Map
}

func NewClient(token string) *Client {
	var tc *http.Client

	if token != "" {
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: token},
		)
		tc = oauth2.NewClient(context.Background(), ts)
	}

	client := github.NewClient(tc)

	return &Client{
		client: client,
		cache:  sync.Map{},
	}
}

func (c *Client) LatestSemverTag(ctx context.Context, owner, repo string) (string, error) {
	cacheKey := owner + "/" + repo

	if cached, ok := c.cache.Load(cacheKey); ok {
		return cached.(string), nil
	}

	tagName, err := c.fetchLatestTag(ctx, owner, repo)
	if err != nil {
		return "", err
	}

	c.cache.Store(cacheKey, tagName)

	return tagName, nil
}

func (c *Client) fetchLatestTag(ctx context.Context, owner, repo string) (string, error) {
	var allTags []*github.RepositoryTag
	page := 1

	for page <= 3 {
		tags, resp, err := c.client.Repositories.ListTags(ctx, owner, repo, &github.ListOptions{
			Page:    page,
			PerPage: 100,
		})
		if err != nil {
			if resp != nil && resp.StatusCode == http.StatusForbidden {
				if resp.Rate.Remaining == 0 {
					return "", fmt.Errorf("rate limit exceeded: %w", err)
				}
			}
			return "", fmt.Errorf("list tags: %w", err)
		}

		allTags = append(allTags, tags...)

		if resp.NextPage == 0 {
			break
		}
		page++
	}

	var semverTags []string
	for _, tag := range allTags {
		name := tag.GetName()
		canonical := name
		if !strings.HasPrefix(canonical, "v") {
			canonical = "v" + canonical
		}
		if semver.IsValid(canonical) {
			semverTags = append(semverTags, name)
		}
	}

	if len(semverTags) == 0 {
		return "", ErrNoSemverTags
	}

	sort.Slice(semverTags, func(i, j int) bool {
		iCanonical := semverTags[i]
		if !strings.HasPrefix(iCanonical, "v") {
			iCanonical = "v" + iCanonical
		}
		jCanonical := semverTags[j]
		if !strings.HasPrefix(jCanonical, "v") {
			jCanonical = "v" + jCanonical
		}
		return semver.Compare(iCanonical, jCanonical) > 0
	})

	return semverTags[0], nil
}