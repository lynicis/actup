package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"

	"github.com/google/go-github/v62/github"
	"golang.org/x/mod/semver"
	"golang.org/x/oauth2"
)

var (
	ErrNoSemverTags = errors.New("no semver tags found for repository")
	majorTagRegex   = regexp.MustCompile(`^v\d+$`)
)

type TagMode struct {
	Semver bool
	Major  int
}

func (m TagMode) cacheKey(owner, repo string) string {
	return fmt.Sprintf("%s/%s/%v/%d", owner, repo, m.Semver, m.Major)
}

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
	return c.LatestTag(ctx, owner, repo, TagMode{Semver: true})
}

func (c *Client) LatestTag(ctx context.Context, owner, repo string, mode TagMode) (string, error) {
	cacheKey := mode.cacheKey(owner, repo)

	if cached, ok := c.cache.Load(cacheKey); ok {
		return cached.(string), nil
	}

	tags, err := c.fetchAllSemverTags(ctx, owner, repo)
	if err != nil {
		return "", err
	}

	tagName := resolveLatestTag(tags, mode)
	if tagName == "" {
		return "", ErrNoSemverTags
	}

	c.cache.Store(cacheKey, tagName)
	return tagName, nil
}

func (c *Client) fetchAllSemverTags(ctx context.Context, owner, repo string) ([]string, error) {
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
					return nil, fmt.Errorf("rate limit exceeded: %w", err)
				}
			}
			return nil, fmt.Errorf("list tags: %w", err)
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

	return semverTags, nil
}

func resolveLatestTag(tags []string, mode TagMode) string {
	if len(tags) == 0 {
		return ""
	}

	sort.Slice(tags, func(i, j int) bool {
		iC := tags[i]
		if !strings.HasPrefix(iC, "v") {
			iC = "v" + iC
		}
		jC := tags[j]
		if !strings.HasPrefix(jC, "v") {
			jC = "v" + jC
		}
		return semver.Compare(iC, jC) > 0
	})

	if mode.Major > 0 {
		prefix := fmt.Sprintf("v%d.", mode.Major)
		for _, tag := range tags {
			if strings.HasPrefix(tag, prefix) {
				return tag
			}
		}
		return ""
	}

	if mode.Semver {
		return tags[0]
	}

	for _, tag := range tags {
		if majorTagRegex.MatchString(tag) {
			return tag
		}
	}

	return tags[0]
}
