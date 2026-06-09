package github

import (
	"regexp"
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
