// Package vulnscanner checks GitHub Actions workflow references against
// known vulnerabilities from the OSV.dev database (GitHub Actions ecosystem).
package vulnscanner

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/semver"

	"github.com/lynicis/actup/internal/parser"
)

// Default OSV.dev API endpoint, overridable in tests via newTestClient.
const defaultOSVBaseURL = "https://api.osv.dev/v1"

// maxConcurrency limits parallel vuln-detail fetches.
const maxConcurrency = 5

// ecosystem is the OSV ecosystem identifier for GitHub Actions.
const ecosystem = "GitHub Actions"

// Client queries the OSV.dev API for GitHub Actions advisories.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates an OSV client pointed at the public OSV.dev API.
func NewClient() *Client {
	return newClient(defaultOSVBaseURL)
}

// newClient creates an OSV client against an explicit base URL (tests).
func newClient(baseURL string) *Client {
	return &Client{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Vulnerability is a single OSV advisory relevant to an action.
type Vulnerability struct {
	ID       string // OSV/GHSA id, e.g. "GHSA-mcph-m25j-8j63"
	Summary  string // one-line description
	Severity string // e.g. "HIGH", "CRITICAL" (from database_specific)
	Fixed    string // first fixed version ("" when unknown)
	URL      string // advisory link
}

// Finding is a vulnerable action usage site in the scanned workflows.
type Finding struct {
	Owner string
	Repo  string
	Ref   string // workflow ref as written (e.g. "v45.0.7")
	Vulns []Vulnerability
	File  string
	Line  int
}

type osvEvent struct {
	Introduced   string `json:"introduced,omitempty"`
	Fixed        string `json:"fixed,omitempty"`
	LastAffected string `json:"last_affected,omitempty"`
}

type osvQueryBatchRequest struct {
	Queries []osvQuery `json:"queries"`
}

type osvQuery struct {
	Package struct {
		Ecosystem string `json:"ecosystem"`
		Name      string `json:"name"`
	} `json:"package"`
}

type osvQueryBatchResponse struct {
	Results []struct {
		Vulns []struct {
			ID string `json:"id"`
		} `json:"vulns"`
	} `json:"results"`
}

type osvVuln struct {
	ID               string  `json:"id"`
	Summary          string  `json:"summary"`
	Withdrawn        *string `json:"withdrawn,omitempty"`
	DatabaseSpecific struct {
		Severity string `json:"severity"`
	} `json:"database_specific"`
	Affected []struct {
		Package struct {
			Name      string `json:"name"`
			Ecosystem string `json:"ecosystem"`
		} `json:"package"`
		Ranges []struct {
			Type   string     `json:"type"`
			Events []osvEvent `json:"events"`
		} `json:"ranges"`
		Versions []string `json:"versions"`
	} `json:"affected"`
	References []struct {
		Type string `json:"type"`
		URL  string `json:"url"`
	} `json:"references"`
}

// Scan queries OSV for advisories against every unique action key and matches
// each workflow ref against the advisory ranges client-side (OSV does not
// perform server-side matching for the GitHub Actions ecosystem).
func (c *Client) Scan(ctx context.Context, actions []parser.ActionRef) ([]Finding, error) {
	// Group by normalized key (lowercased owner/repo) with all usage sites.
	type keyInfo struct {
		owner, repo string
		sites       []parser.ActionRef
	}
	keys := make(map[string]*keyInfo)
	for _, a := range actions {
		key := strings.ToLower(a.Owner + "/" + a.Repo)
		info, ok := keys[key]
		if !ok {
			info = &keyInfo{owner: strings.ToLower(a.Owner), repo: strings.ToLower(a.Repo)}
			keys[key] = info
		}
		info.sites = append(info.sites, a)
	}

	// Batch-query OSV for all keys at once.
	queries := make([]osvQuery, 0, len(keys))
	for _, info := range keys {
		var q osvQuery
		q.Package.Ecosystem = ecosystem
		q.Package.Name = info.owner + "/" + info.repo
		queries = append(queries, q)
	}

	batchResp, err := c.queryBatch(ctx, queries)
	if err != nil {
		return nil, fmt.Errorf("query OSV: %w", err)
	}

	// querybatch returns only vuln ids; fetch full details in parallel.
	var allIDs []string
	for _, r := range batchResp.Results {
		for _, v := range r.Vulns {
			allIDs = append(allIDs, v.ID)
		}
	}

	vulnDetails := make(map[string]*osvVuln)
	var mu sync.Mutex
	var fetchErrs []error
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrency)

	for _, id := range allIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			v, err := c.fetchVuln(ctx, id)
			mu.Lock()
			defer mu.Unlock()
			if err != nil {
				fetchErrs = append(fetchErrs, err)
				return
			}
			vulnDetails[id] = v
		}(id)
	}
	wg.Wait()

	if len(fetchErrs) > 0 && len(vulnDetails) == 0 {
		return nil, fetchErrs[0]
	}

	// Match each ref against each advisory's ranges, client-side.
	var findings []Finding
	for key, info := range keys {
		for _, site := range info.sites {
			var vulns []Vulnerability
			for _, v := range vulnDetails {
				if !v.affects(key, site.Current) {
					continue
				}
				vulns = append(vulns, Vulnerability{
					ID:       v.ID,
					Summary:  v.Summary,
					Severity: v.DatabaseSpecific.Severity,
					Fixed:    v.firstFixed(),
					URL:      v.advisoryURL(),
				})
			}
			if len(vulns) == 0 {
				continue
			}
			findings = append(findings, Finding{
				Owner: site.Owner,
				Repo:  site.Repo,
				Ref:   site.Current,
				Vulns: vulns,
				File:  site.File,
				Line:  site.Line,
			})
		}
	}

	return findings, nil
}

// affects reports whether the advisory covers actionKey at the given ref.
// It only considers entries whose package matches (case-insensitive) and
// applies ECOSYSTEM-type range events.
func (v *osvVuln) affects(actionKey, ref string) bool {
	if v.Withdrawn != nil && *v.Withdrawn != "" {
		return false
	}
	for _, aff := range v.Affected {
		if !strings.EqualFold(aff.Package.Ecosystem, ecosystem) {
			continue
		}
		if !strings.EqualFold(aff.Package.Name, actionKey) {
			continue
		}
		if len(aff.Versions) > 0 {
			for _, ver := range aff.Versions {
				if versionsEqual(ver, ref) {
					return true
				}
			}
			continue
		}
		for _, r := range aff.Ranges {
			if r.Type != "ECOSYSTEM" {
				continue
			}
			if eventsAffect(r.Events, ref) {
				return true
			}
		}
	}
	return false
}

// eventsAffect implements OSV range semantics: a version is affected when
// it lies inside a vulnerable interval. Both plain ("41") and v-prefixed
// ("v41") forms are normalized.
func eventsAffect(events []osvEvent, ref string) bool {
	intervals := buildIntervals(events)
	if len(intervals) == 0 {
		// An event list without any "introduced" marks nothing affected.
		return false
	}
	norm := normalizeVersion(ref)
	for _, iv := range intervals {
		if compareVersions(norm, iv.start) >= 0 {
			if iv.end == "" || compareVersions(norm, iv.end) < 0 {
				return true
			}
		}
	}
	return false
}

// interval is a vulnerable version range [start, end). An empty end means
// no upper bound.
type interval struct {
	start, end string
}

// buildIntervals converts OSV events into vulnerable intervals. Introduced
// opens an interval; fixed (or a following introduced) closes it.
func buildIntervals(events []osvEvent) []interval {
	var out []interval
	cur := interval{start: ""}
	started := false
	for _, e := range events {
		switch {
		case e.Introduced != "":
			if started {
				// A new introduction closes the previous interval at this version.
				cur.end = e.Introduced
				out = append(out, cur)
			}
			started = true
			cur = interval{start: e.Introduced}
		case e.Fixed != "":
			if started {
				cur.end = e.Fixed
				out = append(out, cur)
				started = false
				cur = interval{start: ""}
			}
			// A fixed event without an open interval marks nothing (OSV
			// semantics: everything before introduced=0 is not affected).
		case e.LastAffected != "":
			// last_affected: vulnerable up to and including this version.
			if started {
				// Treat as fixed at last_affected+1 by comparing with <=.
				// We approximate by setting end to the next patch bump is not
				// possible generically, so we store it and handle comparison
				// separately via lastAffected intervals.
				cur.end = lastAffectedEnd(e.LastAffected)
				out = append(out, cur)
				started = false
				cur = interval{start: ""}
			}
		}
	}
	if started {
		out = append(out, cur)
	}
	return out
}

// lastAffectedEnd converts a last_affected version into an exclusive upper
// bound by bumping its last specified segment: "3.2" → "3.3" (so 3.2.5 is
// still affected), "3.2.4" → "3.2.5". Non-numeric inputs are returned
// unchanged.
func lastAffectedEnd(ver string) string {
	v := strings.TrimPrefix(strings.TrimSpace(ver), "v")
	parts := strings.Split(v, ".")
	last, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return v
	}
	parts[len(parts)-1] = strconv.Itoa(last + 1)
	return strings.Join(parts, ".")
}

func (c *Client) queryBatch(ctx context.Context, queries []osvQuery) (*osvQueryBatchResponse, error) {
	if len(queries) == 0 {
		return &osvQueryBatchResponse{}, nil
	}

	body, err := json.Marshal(osvQueryBatchRequest{Queries: queries})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/querybatch", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			_ = cerr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OSV querybatch: status %d", resp.StatusCode)
	}

	var out osvQueryBatchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode querybatch response: %w", err)
	}
	return &out, nil
}

func (c *Client) fetchVuln(ctx context.Context, id string) (*osvVuln, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/vulns/"+url.PathEscape(id), nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", id, err)
	}
	defer func() {
		if cerr := resp.Body.Close(); cerr != nil {
			_ = cerr
		}
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: status %d", id, resp.StatusCode)
	}

	var v osvVuln
	if err := json.NewDecoder(resp.Body).Decode(&v); err != nil {
		return nil, fmt.Errorf("decode %s: %w", id, err)
	}
	return &v, nil
}

// firstFixed returns the earliest fixed version across all affected entries.
func (v *osvVuln) firstFixed() string {
	var best string
	for _, aff := range v.Affected {
		for _, r := range aff.Ranges {
			for _, e := range r.Events {
				if e.Fixed != "" {
					if best == "" || compareVersions(e.Fixed, best) < 0 {
						best = e.Fixed
					}
				}
			}
		}
	}
	return best
}

// advisoryURL returns a human-friendly link for the advisory. It prefers a
// reference whose URL embeds the advisory's own id (some OSV records link to
// related advisories with different ids); otherwise it falls back to the
// canonical osv.dev page for this id.
func (v *osvVuln) advisoryURL() string {
	for _, ref := range v.References {
		if ref.URL != "" && strings.Contains(ref.URL, v.ID) {
			return ref.URL
		}
	}
	return "https://osv.dev/vulnerability/" + v.ID
}

// normalizeVersion returns a canonical semver string ("v"-prefixed), or ""
// when the input is not valid semver.
func normalizeVersion(s string) string {
	v := strings.TrimSpace(s)
	if v == "" {
		return ""
	}
	if !strings.HasPrefix(v, "v") {
		v = "v" + v
	}
	if semver.IsValid(v) {
		return semver.Canonical(v)
	}
	return ""
}

// versionsEqual compares two version strings ignoring a leading "v" and
// trailing zero segments ("41" == "v41.0.0").
func versionsEqual(a, b string) bool {
	return normalizeVersion(a) == normalizeVersion(b) && normalizeVersion(a) != ""
}

// compareVersions compares two version strings numerically per segment.
// It returns -1, 0, or 1. Non-semver inputs compare equal to each other and
// unequal to any semver version.
func compareVersions(a, b string) int {
	an := normalizeVersion(a)
	bn := normalizeVersion(b)
	if an == "" || bn == "" {
		if an == bn {
			return 0
		}
		if an == "" {
			return -1
		}
		return 1
	}
	return semver.Compare(an, bn)
}
