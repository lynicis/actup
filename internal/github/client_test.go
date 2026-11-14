package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
)

func TestResolveLatestTag_MajorTagMode(t *testing.T) {
	tags := []string{"v4", "v4.1.0", "v4.2.1", "v5", "v5.0.1"}

	got := resolveLatestTag(tags, TagMode{})
	if got != "v5" {
		t.Errorf("resolveLatestTag major mode = %q, want %q", got, "v5")
	}
}

func TestResolveLatestTag_SemverMode(t *testing.T) {
	tags := []string{"v4", "v4.1.0", "v4.2.1", "v5", "v5.0.1"}

	got := resolveLatestTag(tags, TagMode{Semver: true})
	if got != "v5.0.1" {
		t.Errorf("resolveLatestTag semver mode = %q, want %q", got, "v5.0.1")
	}
}

func TestResolveLatestTag_NoMajorTags_Fallback(t *testing.T) {
	tags := []string{"v3.1.0", "v3.2.0"}

	got := resolveLatestTag(tags, TagMode{})
	if got != "v3.2.0" {
		t.Errorf("resolveLatestTag fallback = %q, want %q", got, "v3.2.0")
	}
}

func TestResolveLatestTag_EmptyTags(t *testing.T) {
	got := resolveLatestTag(nil, TagMode{})
	if got != "" {
		t.Errorf("resolveLatestTag empty = %q, want empty", got)
	}
}

func TestResolveLatestTag_MajorPin(t *testing.T) {
	tags := []string{"v4.0.0", "v4.2.1", "v5.0.0", "v5.1.0"}
	got := resolveLatestTag(tags, TagMode{Major: 4})
	if got != "v4.2.1" {
		t.Errorf("major pin 4 = %q, want %q", got, "v4.2.1")
	}
}

func TestResolveLatestTag_MajorPinNoMatch(t *testing.T) {
	tags := []string{"v5.0.0", "v5.1.0"}
	got := resolveLatestTag(tags, TagMode{Major: 4})
	if got != "" {
		t.Errorf("major pin 4 = %q, want empty", got)
	}
}

func TestResolveLatestTag_MajorPinBareTag(t *testing.T) {
	tags := []string{"v40.1.0", "v41", "v42.0.0"}
	got := resolveLatestTag(tags, TagMode{Major: 41})
	if got != "v41" {
		t.Errorf("major pin 41 = %q, want %q", got, "v41")
	}
}

func TestResolveLatestTag_MajorPinNoVPrefix(t *testing.T) {
	tags := []string{"41.0.0", "41.2.0"}
	got := resolveLatestTag(tags, TagMode{Major: 41})
	if got != "41.2.0" {
		t.Errorf("major pin 41 = %q, want %q", got, "41.2.0")
	}
}

func TestIsMajorTag(t *testing.T) {
	majorTagRegex := regexp.MustCompile(`^v\d+$`)
	tests := []struct {
		tag  string
		want bool
	}{
		{"v3", true},
		{"v10", true},
		{"v3.1.0", false},
		{"v3.1", false},
		{"3", false},
		{"", false},
	}
	for _, tt := range tests {
		got := majorTagRegex.MatchString(tt.tag)
		if got != tt.want {
			t.Errorf("isMajorTag(%q) = %v, want %v", tt.tag, got, tt.want)
		}
	}
}

func TestResolvePin(t *testing.T) {
	tests := []struct {
		name     string
		actions  map[string]string
		key      string
		defMajor int
		wantM    int
		wantX    string
	}{
		{name: "numeric", actions: map[string]string{"a/b": "4"}, key: "a/b", defMajor: 1, wantM: 4},
		{name: "exact", actions: map[string]string{"a/b": "v5.1.0"}, key: "a/b", defMajor: 1, wantM: 1, wantX: "v5.1.0"},
		{name: "skip", actions: map[string]string{"a/b": "skip"}, key: "a/b", defMajor: 1, wantM: 1},
		{name: "absent", actions: map[string]string{"a/b": "v5"}, key: "c/d", defMajor: 2, wantM: 2},
		{name: "nil map", actions: nil, key: "a/b", defMajor: 3, wantM: 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, exact := resolvePin(tt.actions, tt.key, tt.defMajor)
			if major != tt.wantM || exact != tt.wantX {
				t.Errorf("resolvePin() = (%d, %q), want (%d, %q)", major, exact, tt.wantM, tt.wantX)
			}
		})
	}
}

func TestResolveTagSHA(t *testing.T) {
	const sha = "dae09b1f80164e2d312981ce81045e7e1f4640d2"

	tests := []struct {
		name string
		ref  string
	}{
		{name: "lightweight tag", ref: "v1.0.0"},
		// The API peels annotated tags, so both shapes return the commit SHA.
		{name: "annotated tag", ref: "v2.0.0"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasSuffix(r.URL.Path, "/commits/"+tt.ref) {
					t.Errorf("unexpected path %s", r.URL.Path)
					http.NotFound(w, r)
					return
				}
				_, _ = w.Write([]byte(sha))
			}))
			defer srv.Close()

			c, err := NewClientWithBaseURL("token", srv.URL)
			if err != nil {
				t.Fatalf("NewClientWithBaseURL: %v", err)
			}

			got, err := c.ResolveTagSHA(t.Context(), "actions", "checkout", tt.ref)
			if err != nil {
				t.Fatalf("ResolveTagSHA: %v", err)
			}
			if got != sha {
				t.Errorf("ResolveTagSHA = %q, want %q", got, sha)
			}
		})
	}
}

func TestResolveTagSHA_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.NotFound(w, nil)
	}))
	defer srv.Close()

	c, err := NewClientWithBaseURL("", srv.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL: %v", err)
	}

	if _, err := c.ResolveTagSHA(t.Context(), "actions", "checkout", "v9"); err == nil {
		t.Fatal("expected an error for a missing tag")
	}
}

func TestResolveTagSHA_RateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(`{"message":"API rate limit exceeded"}`))
	}))
	defer srv.Close()

	c, err := NewClientWithBaseURL("", srv.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL: %v", err)
	}

	_, err = c.ResolveTagSHA(t.Context(), "actions", "checkout", "v4")
	if err == nil {
		t.Fatal("expected a rate-limit error")
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("error = %q, want it to mention the rate limit", err)
	}
}

func TestLatestTagInfoCachesTags(t *testing.T) {
	const sha = "dae09b1f80164e2d312981ce81045e7e1f4640d2"
	var requests int

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/tags") {
			t.Errorf("unexpected path %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}
		requests++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"v4","commit":{"sha":"` + sha + `"}},{"name":"v4.2.2","commit":{"sha":"` + sha + `"}}]`))
	}))
	defer srv.Close()

	c, err := NewClientWithBaseURL("", srv.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL: %v", err)
	}

	info, err := c.LatestTagInfo(t.Context(), "actions", "checkout", TagMode{})
	if err != nil {
		t.Fatalf("LatestTagInfo: %v", err)
	}
	if info.Name != "v4" || info.SHA != sha {
		t.Errorf("LatestTagInfo = %+v, want name v4 and SHA %s", info, sha)
	}

	if _, err := c.LatestTagInfo(t.Context(), "actions", "checkout", TagMode{}); err != nil {
		t.Fatalf("second LatestTagInfo: %v", err)
	}
	if requests != 1 {
		t.Errorf("expected tag list to be cached (1 request), got %d", requests)
	}
}

func TestFetchAllSemverTagsRetainsSHA(t *testing.T) {
	const sha = "dae09b1f80164e2d312981ce81045e7e1f4640d2"

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/tags") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"v4","commit":{"sha":"` + sha + `"}},{"name":"not-semver","commit":{"sha":"` + sha + `"}}]`))
	}))
	defer srv.Close()

	c, err := NewClientWithBaseURL("", srv.URL)
	if err != nil {
		t.Fatalf("NewClientWithBaseURL: %v", err)
	}

	tags, err := c.fetchAllSemverTags(t.Context(), "actions", "checkout")
	if err != nil {
		t.Fatalf("fetchAllSemverTags: %v", err)
	}
	if len(tags) != 1 || tags[0].Name != "v4" || tags[0].SHA != sha {
		t.Errorf("fetchAllSemverTags = %+v, want [{v4 %s}]", tags, sha)
	}
}

func TestResolveVersionRejectsSHAPin(t *testing.T) {
	sha := "0123456789abcdef0123456789abcdef01234567"
	_, err := ResolveVersion(context.Background(), nil, "actions/checkout", false, 0, map[string]string{"actions/checkout": sha})
	if err == nil || !strings.Contains(err.Error(), "pin a tag and use sha: true") {
		t.Fatalf("ResolveVersion(SHA pin) = %v, want pin-a-tag error", err)
	}
}
