package github

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/google/go-github/v89/github"
	"golang.org/x/mod/semver"
)

var (
	ErrNoSemverTags = errors.New("no semver tags found for repository")
	majorTagRegex   = regexp.MustCompile(`^v\d+$`)
)

// maxConcurrency caps parallel GitHub API requests.
const maxConcurrency = 5

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
	opts := []github.ClientOptionsFunc{github.WithAuthToken(token)}
	if token == "" {
		opts = nil
	}
	client, err := github.NewClient(opts...)
	if err != nil {
		// Unreachable: WithAuthToken only rejects an empty token, which is
		// guarded above, and no other options are set.
		panic(fmt.Sprintf("go-github client options: %v", err))
	}
	return &Client{
		client: client,
		cache:  sync.Map{},
	}
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

	slices.SortFunc(tags, func(a, b string) int {
		aC := a
		if !strings.HasPrefix(aC, "v") {
			aC = "v" + aC
		}
		bC := b
		if !strings.HasPrefix(bC, "v") {
			bC = "v" + bC
		}
		// descending order: highest semver first
		return semver.Compare(bC, aC)
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

func ResolveVersion(
	ctx context.Context,
	client *Client,
	key string,
	semverMode bool,
	defaultMajor int,
	cfgActions map[string]string,
) (string, error) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid action key: %s", key)
	}
	owner := parts[0]
	repo := parts[1]

	resolvedMajor := defaultMajor
	exactVersion := ""
	if cfgActions != nil {
		if pin, ok := cfgActions[key]; ok {
			if n, err := strconv.Atoi(pin); err == nil {
				resolvedMajor = n
			} else if pin != "skip" {
				exactVersion = pin
			}
		}
	}

	if exactVersion != "" {
		return exactVersion, nil
	}

	return client.LatestTag(ctx, owner, repo, TagMode{Semver: semverMode, Major: resolvedMajor})
}

// ResolveAll resolves the latest version for every key in parallel (capped
// at maxConcurrency) and returns key -> latest version. Callers decide how
// to report per-key errors.
func ResolveAll(
	ctx context.Context,
	client *Client,
	keys []string,
	semverMode bool,
	defaultMajor int,
	cfgActions map[string]string,
) map[string]ResolveResult {
	results := make(map[string]ResolveResult, len(keys))
	var mu sync.Mutex
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrency)

	for _, key := range keys {
		wg.Add(1)
		go func(k string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			latest, err := ResolveVersion(ctx, client, k, semverMode, defaultMajor, cfgActions)

			mu.Lock()
			defer mu.Unlock()
			results[k] = ResolveResult{Latest: latest, Err: err}
		}(key)
	}
	wg.Wait()
	return results
}

// ResolveResult is the outcome of resolving one action key.
type ResolveResult struct {
	Latest string
	Err    error
}
