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
	shaRegex        = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)
)

// maxConcurrency caps parallel GitHub API requests.
const maxConcurrency = 5

// TagInfo pairs a tag name with the commit SHA the GitHub API reports for it
// (already peeled for annotated tags).
type TagInfo struct {
	Name string
	SHA  string
}

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

// NewClientWithBaseURL builds a client pointed at a custom API root (used by
// tests to point at an httptest server, which go-github treats as an
// enterprise API root).
func NewClientWithBaseURL(token, baseURL string) (*Client, error) {
	opts := []github.ClientOptionsFunc{github.WithEnterpriseURLs(baseURL, baseURL)}
	if token != "" {
		opts = append(opts, github.WithAuthToken(token))
	}
	client, err := github.NewClient(opts...)
	if err != nil {
		return nil, err
	}
	return &Client{client: client, cache: sync.Map{}}, nil
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

	tagName := resolveLatestTag(tagNames(tags), mode)
	if tagName == "" {
		return "", ErrNoSemverTags
	}

	c.cache.Store(cacheKey, tagName)
	return tagName, nil
}

// LatestTagInfo resolves the latest tag and returns its name plus commit SHA
// from the cached tag list, without a second API call.
func (c *Client) LatestTagInfo(ctx context.Context, owner, repo string, mode TagMode) (TagInfo, error) {
	tags, err := c.fetchAllSemverTags(ctx, owner, repo)
	if err != nil {
		return TagInfo{}, err
	}

	tagName := resolveLatestTag(tagNames(tags), mode)
	if tagName == "" {
		return TagInfo{}, ErrNoSemverTags
	}

	for _, tag := range tags {
		if tag.Name == tagName {
			return tag, nil
		}
	}
	return TagInfo{}, ErrNoSemverTags
}

// ResolveTagSHA resolves any tag or branch ref to its peeled commit SHA. One
// API call per (owner, repo, ref), cached.
func (c *Client) ResolveTagSHA(ctx context.Context, owner, repo, ref string) (string, error) {
	cacheKey := "sha/" + owner + "/" + repo + "/" + ref
	if cached, ok := c.cache.Load(cacheKey); ok {
		return cached.(string), nil
	}

	sha, resp, err := c.client.Repositories.GetCommitSHA1(ctx, owner, repo, ref, "")
	if err != nil {
		if resp != nil && resp.StatusCode == http.StatusForbidden && resp.Rate.Remaining == 0 {
			return "", fmt.Errorf("rate limit exceeded: %w", err)
		}
		return "", fmt.Errorf("resolve tag %s: %w", ref, err)
	}

	sha = strings.TrimSpace(sha)
	if !shaRegex.MatchString(sha) {
		return "", fmt.Errorf("invalid commit SHA %q for %s/%s@%s", sha, owner, repo, ref)
	}
	sha = strings.ToLower(sha)

	c.cache.Store(cacheKey, sha)
	return sha, nil
}

func tagNames(tags []TagInfo) []string {
	names := make([]string, len(tags))
	for i, tag := range tags {
		names[i] = tag.Name
	}
	return names
}

func (c *Client) fetchAllSemverTags(ctx context.Context, owner, repo string) ([]TagInfo, error) {
	cacheKey := "tags/" + owner + "/" + repo
	if cached, ok := c.cache.Load(cacheKey); ok {
		return cached.([]TagInfo), nil
	}

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

	var semverTags []TagInfo
	for _, tag := range allTags {
		name := tag.GetName()
		canonical := name
		if !strings.HasPrefix(canonical, "v") {
			canonical = "v" + canonical
		}
		if semver.IsValid(canonical) || majorTagRegex.MatchString(canonical) {
			semverTags = append(semverTags, TagInfo{Name: name, SHA: tag.GetCommit().GetSHA()})
		}
	}

	c.cache.Store(cacheKey, semverTags)
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
		// Match by parsed major so bare vN tags and missing v prefixes work;
		// return the original spelling.
		for _, tag := range tags {
			if tagMajor(tag) == mode.Major {
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

// tagMajor returns the numeric major of a tag ("v41" or "41.2.0" -> 41),
// or -1 when it cannot be parsed.
func tagMajor(tag string) int {
	v := strings.TrimPrefix(tag, "v")
	seg, _, _ := strings.Cut(v, ".")
	n, err := strconv.Atoi(seg)
	if err != nil {
		return -1
	}
	return n
}

// resolvePin applies the config pin precedence: numeric pin -> major, other
// non-skip pin -> exact, skip/absent -> default major with no exact pin.
func resolvePin(cfgActions map[string]string, key string, defaultMajor int) (major int, exact string) {
	if cfgActions == nil {
		return defaultMajor, ""
	}
	pin, ok := cfgActions[key]
	if !ok {
		return defaultMajor, ""
	}
	if n, err := strconv.Atoi(pin); err == nil {
		return n, ""
	}
	if pin != "skip" {
		return defaultMajor, pin
	}
	return defaultMajor, ""
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

	resolvedMajor, exactVersion := resolvePin(cfgActions, key, defaultMajor)

	if exactVersion != "" {
		if shaRegex.MatchString(exactVersion) {
			return "", fmt.Errorf("pin a tag and use sha: true")
		}
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
