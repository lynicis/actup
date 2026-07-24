package checker

import (
	"context"
	"fmt"
	"testing"

	"github.com/lynicis/actup/internal/github"
	"github.com/lynicis/actup/internal/parser"
)

type mockClient struct {
	tags     map[string]string
	err      error
	lastMode github.TagMode
}

func (m *mockClient) LatestTag(ctx context.Context, owner, repo string, mode github.TagMode) (string, error) {
	m.lastMode = mode
	if m.err != nil {
		return "", m.err
	}
	return m.tags[owner+"/"+repo], nil
}

func TestChecker_AllUpToDate(t *testing.T) {
	mc := &mockClient{tags: map[string]string{"actions/checkout": "v4"}}
	c := New(mc, false, 0, nil)
	actions := []parser.ActionRef{
		{Owner: "actions", Repo: "checkout", Current: "v4", File: "test.yml", Line: 5},
	}
	outdated, err := c.Run(context.Background(), actions)
	if err != nil {
		t.Fatal(err)
	}
	if len(outdated) != 0 {
		t.Errorf("expected 0 outdated, got %d", len(outdated))
	}
}

func TestChecker_Outdated(t *testing.T) {
	mc := &mockClient{tags: map[string]string{"actions/checkout": "v5"}}
	c := New(mc, false, 0, nil)
	actions := []parser.ActionRef{
		{Owner: "actions", Repo: "checkout", Current: "v3", File: "test.yml", Line: 5},
	}
	outdated, err := c.Run(context.Background(), actions)
	if err != nil {
		t.Fatal(err)
	}
	if len(outdated) != 1 {
		t.Fatalf("expected 1 outdated, got %d", len(outdated))
	}
	if outdated[0].Current != "v3" || outdated[0].Latest != "v5" {
		t.Errorf("got %s→%s, want v3→v5", outdated[0].Current, outdated[0].Latest)
	}
}

func TestChecker_APIError(t *testing.T) {
	mc := &mockClient{err: fmt.Errorf("API error")}
	c := New(mc, false, 0, nil)
	actions := []parser.ActionRef{
		{Owner: "actions", Repo: "checkout", Current: "v3", File: "test.yml", Line: 5},
	}
	_, err := c.Run(context.Background(), actions)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestChecker_CfgActions_Skip(t *testing.T) {
	mc := &mockClient{tags: map[string]string{"actions/checkout": "v5"}}
	cfgActions := map[string]string{"actions/checkout": "skip"}
	c := New(mc, false, 0, cfgActions)
	actions := []parser.ActionRef{
		{Owner: "actions", Repo: "checkout", Current: "v3", File: "test.yml", Line: 5},
	}
	outdated, err := c.Run(context.Background(), actions)
	if err != nil {
		t.Fatal(err)
	}
	if len(outdated) != 0 {
		t.Errorf("expected 0 outdated due to skip, got %d", len(outdated))
	}
}

func TestChecker_CfgActions_MajorOverride(t *testing.T) {
	mc := &mockClient{tags: map[string]string{"actions/checkout": "v4"}}
	cfgActions := map[string]string{"actions/checkout": "4"}
	c := New(mc, false, 5, cfgActions) // default major is 5, overridden to 4
	actions := []parser.ActionRef{
		{Owner: "actions", Repo: "checkout", Current: "v4", File: "test.yml", Line: 5},
	}
	outdated, err := c.Run(context.Background(), actions)
	if err != nil {
		t.Fatal(err)
	}
	if len(outdated) != 0 {
		t.Errorf("expected 0 outdated, got %d", len(outdated))
	}
	if mc.lastMode.Major != 4 {
		t.Errorf("expected major mode 4, got %d", mc.lastMode.Major)
	}
}

func TestChecker_CfgActions_ExactOverride(t *testing.T) {
	mc := &mockClient{tags: map[string]string{"actions/checkout": "v4.1.2"}}
	cfgActions := map[string]string{"actions/checkout": "v4.1.2"}
	c := New(mc, false, 0, cfgActions)
	actions := []parser.ActionRef{
		{Owner: "actions", Repo: "checkout", Current: "v4.1.2", File: "test.yml", Line: 5},
	}
	outdated, err := c.Run(context.Background(), actions)
	if err != nil {
		t.Fatal(err)
	}
	if len(outdated) != 0 {
		t.Errorf("expected 0 outdated, got %d", len(outdated))
	}
}
