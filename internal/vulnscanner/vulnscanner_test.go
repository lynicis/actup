package vulnscanner

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lynicis/actup/internal/parser"
)

// mockOSV wires a local httptest server that emulates the two OSV endpoints
// actup uses: POST /querybatch and GET /vulns/{id}.
type mockOSV struct {
	server *httptest.Server
	// vulnsByPackage maps package name to advisory IDs returned by querybatch.
	vulnsByPackage map[string][]string
	// vulns maps advisory id to the full advisory record.
	vulns map[string]map[string]any
}

func newMockOSV(t *testing.T) *mockOSV {
	t.Helper()
	m := &mockOSV{
		vulnsByPackage: map[string][]string{},
		vulns:          map[string]map[string]any{},
	}
	m.server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/querybatch"):
			var req struct {
				Queries []struct {
					Package struct {
						Ecosystem string `json:"ecosystem"`
						Name      string `json:"name"`
					} `json:"package"`
				} `json:"queries"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			results := make([]map[string]any, 0, len(req.Queries))
			for _, q := range req.Queries {
				ids, ok := m.vulnsByPackage[q.Package.Name]
				if !ok || len(ids) == 0 {
					results = append(results, map[string]any{})
					continue
				}
				vulns := make([]map[string]any, 0, len(ids))
				for _, id := range ids {
					vulns = append(vulns, map[string]any{"id": id})
				}
				results = append(results, map[string]any{"vulns": vulns})
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"results": results})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/vulns/"):
			id := strings.TrimPrefix(r.URL.Path, "/vulns/")
			v, ok := m.vulns[id]
			if !ok {
				w.WriteHeader(http.StatusNotFound)
				return
			}
			_ = json.NewEncoder(w).Encode(v)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	t.Cleanup(m.server.Close)
	return m
}

func (m *mockOSV) client() *Client {
	return newClient(m.server.URL)
}

func (m *mockOSV) addVuln(pkg string, v map[string]any) {
	id := v["id"].(string)
	m.vulns[id] = v
	m.vulnsByPackage[pkg] = append(m.vulnsByPackage[pkg], id)
}

func TestScanNoVulns(t *testing.T) {
	m := newMockOSV(t)
	client := m.client()

	actions := []parser.ActionRef{{Owner: "actions", Repo: "checkout", Current: "v5", File: "wf.yml", Line: 1}}
	findings, err := client.Scan(context.Background(), actions)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestScanFindsKnownVuln(t *testing.T) {
	m := newMockOSV(t)
	m.addVuln("tj-actions/changed-files", map[string]any{
		"id":      "GHSA-TEST-1",
		"summary": "command injection",
		"aliases": []string{"CVE-2023-51664"},
		"database_specific": map[string]any{
			"severity": "HIGH",
		},
		"affected": []map[string]any{
			{
				"package": map[string]any{"name": "tj-actions/changed-files", "ecosystem": "GitHub Actions"},
				"ranges": []map[string]any{
					{
						"type": "ECOSYSTEM",
						"events": []map[string]any{
							{"introduced": "0"},
							{"fixed": "41"},
						},
					},
				},
			},
		},
		"references": []map[string]any{
			{"type": "ADVISORY", "url": "https://github.com/tj-actions/changed-files/security/advisories/GHSA-TEST-1"},
		},
	})

	client := m.client()
	actions := []parser.ActionRef{
		{Owner: "tj-actions", Repo: "changed-files", Current: "v39.2", File: "wf.yml", Line: 10},
	}
	findings, err := client.Scan(context.Background(), actions)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	f := findings[0]
	if f.Vulns[0].ID != "GHSA-TEST-1" {
		t.Errorf("vuln id = %q, want GHSA-TEST-1", f.Vulns[0].ID)
	}
	if f.Vulns[0].Fixed != "41" {
		t.Errorf("fixed = %q, want 41", f.Vulns[0].Fixed)
	}
	if f.Vulns[0].Severity != "HIGH" {
		t.Errorf("severity = %q, want HIGH", f.Vulns[0].Severity)
	}
	if f.Vulns[0].URL != "https://github.com/tj-actions/changed-files/security/advisories/GHSA-TEST-1" {
		t.Errorf("url = %q", f.Vulns[0].URL)
	}
	if f.File != "wf.yml" || f.Line != 10 {
		t.Errorf("finding location = %s:%d", f.File, f.Line)
	}
}

func TestScanMixedCaseOwner(t *testing.T) {
	// OSV package names are lowercase; the client must normalize the query
	// so TJ-Actions/changed-files (as written in some workflows) still matches.
	m := newMockOSV(t)
	m.addVuln("tj-actions/changed-files", simpleVuln("GHSA-CASE-1", "tj-actions/changed-files", "0", "41"))

	client := m.client()
	actions := []parser.ActionRef{
		{Owner: "TJ-Actions", Repo: "changed-files", Current: "v40", File: "wf.yml", Line: 3},
	}
	findings, err := client.Scan(context.Background(), actions)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding for mixed-case owner, got %d", len(findings))
	}
}

func TestScanVersionRangeMatching(t *testing.T) {
	m := newMockOSV(t)
	m.addVuln("vuln/pkg", simpleVuln("GHSA-RANGE-1", "vuln/pkg", "0", "41"))

	client := m.client()
	cases := []struct {
		ref      string
		affected bool
	}{
		{"v1", true},
		{"v40", true},
		{"v40.9", true},
		{"v41", false},
		{"v41.0.0", false},
		{"v42", false},
		{"40", true}, // plain form without v prefix
	}
	for _, tc := range cases {
		actions := []parser.ActionRef{{Owner: "vuln", Repo: "pkg", Current: tc.ref, File: "wf.yml", Line: 1}}
		findings, err := client.Scan(context.Background(), actions)
		if err != nil {
			t.Fatalf("Scan(%s): %v", tc.ref, err)
		}
		if got := len(findings) > 0; got != tc.affected {
			t.Errorf("ref %s: affected = %v, want %v", tc.ref, got, tc.affected)
		}
	}
}

func TestScanMultipleIntervals(t *testing.T) {
	m := newMockOSV(t)
	m.addVuln("multi/pkg", map[string]any{
		"id":      "GHSA-MULTI-1",
		"summary": "multi-interval vuln",
		"affected": []map[string]any{
			{
				"package": map[string]any{"name": "multi/pkg", "ecosystem": "GitHub Actions"},
				"ranges": []map[string]any{
					{
						"type": "ECOSYSTEM",
						"events": []map[string]any{
							{"introduced": "0"},
							{"fixed": "2"},
							{"introduced": "5"},
							{"fixed": "5.3"},
						},
					},
				},
			},
		},
	})

	client := m.client()
	cases := []struct {
		ref      string
		affected bool
	}{
		{"v1", true},
		{"v2", false},
		{"v4.9", false},
		{"v5", true},
		{"v5.2.9", true},
		{"v5.3", false},
		{"v6", false},
	}
	for _, tc := range cases {
		actions := []parser.ActionRef{{Owner: "multi", Repo: "pkg", Current: tc.ref, File: "wf.yml", Line: 1}}
		findings, err := client.Scan(context.Background(), actions)
		if err != nil {
			t.Fatalf("Scan(%s): %v", tc.ref, err)
		}
		if got := len(findings) > 0; got != tc.affected {
			t.Errorf("ref %s: affected = %v, want %v", tc.ref, got, tc.affected)
		}
	}
}

func TestScanLastAffected(t *testing.T) {
	m := newMockOSV(t)
	m.addVuln("last/pkg", map[string]any{
		"id":      "GHSA-LAST-1",
		"summary": "last_affected vuln",
		"affected": []map[string]any{
			{
				"package": map[string]any{"name": "last/pkg", "ecosystem": "GitHub Actions"},
				"ranges": []map[string]any{
					{
						"type": "ECOSYSTEM",
						"events": []map[string]any{
							{"introduced": "0"},
							{"last_affected": "3.2"},
						},
					},
				},
			},
		},
	})

	client := m.client()
	cases := []struct {
		ref      string
		affected bool
	}{
		{"v3.2", true}, // last_affected is inclusive
		{"v3.2.5", true},
		{"v3.3", false},
		{"v4", false},
	}
	for _, tc := range cases {
		actions := []parser.ActionRef{{Owner: "last", Repo: "pkg", Current: tc.ref, File: "wf.yml", Line: 1}}
		findings, err := client.Scan(context.Background(), actions)
		if err != nil {
			t.Fatalf("Scan(%s): %v", tc.ref, err)
		}
		if got := len(findings) > 0; got != tc.affected {
			t.Errorf("ref %s: affected = %v, want %v", tc.ref, got, tc.affected)
		}
	}
}

func TestScanWithdrawnAdvisory(t *testing.T) {
	m := newMockOSV(t)
	v := simpleVuln("GHSA-WITHDRAWN-1", "wd/pkg", "0", "99")
	v["withdrawn"] = "2024-01-01T00:00:00Z"
	m.addVuln("wd/pkg", v)

	client := m.client()
	actions := []parser.ActionRef{{Owner: "wd", Repo: "pkg", Current: "v1", File: "wf.yml", Line: 1}}
	findings, err := client.Scan(context.Background(), actions)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("withdrawn advisory must not match, got %d findings", len(findings))
	}
}

func TestScanGroupsUsageSites(t *testing.T) {
	m := newMockOSV(t)
	m.addVuln("grp/pkg", simpleVuln("GHSA-GRP-1", "grp/pkg", "0", "10"))

	client := m.client()
	actions := []parser.ActionRef{
		{Owner: "grp", Repo: "pkg", Current: "v1", File: "a.yml", Line: 1},
		{Owner: "grp", Repo: "pkg", Current: "v1", File: "b.yml", Line: 7},
		{Owner: "grp", Repo: "pkg", Current: "v11", File: "c.yml", Line: 2}, // fixed version, no finding
	}
	findings, err := client.Scan(context.Background(), actions)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(findings) != 2 {
		t.Fatalf("expected 2 findings, got %d", len(findings))
	}
}

func TestScanNoActions(t *testing.T) {
	m := newMockOSV(t)
	client := m.client()

	findings, err := client.Scan(context.Background(), nil)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("expected no findings, got %d", len(findings))
	}
}

func TestScanPropagatesNetworkError(t *testing.T) {
	// Point the client at a closed server to force a network error.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	url := server.URL
	server.Close()

	client := newClient(url)
	actions := []parser.ActionRef{{Owner: "a", Repo: "b", Current: "v1", File: "wf.yml", Line: 1}}
	if _, err := client.Scan(context.Background(), actions); err == nil {
		t.Fatal("expected an error when OSV is unreachable")
	}
}

// simpleVuln builds a minimal advisory affecting [introduced, fixed).
func simpleVuln(id, pkg, introduced, fixed string) map[string]any {
	return map[string]any{
		"id":      id,
		"summary": "test vuln " + id,
		"database_specific": map[string]any{
			"severity": "MODERATE",
		},
		"affected": []map[string]any{
			{
				"package": map[string]any{"name": pkg, "ecosystem": "GitHub Actions"},
				"ranges": []map[string]any{
					{
						"type": "ECOSYSTEM",
						"events": []map[string]any{
							{"introduced": introduced},
							{"fixed": fixed},
						},
					},
				},
			},
		},
	}
}
