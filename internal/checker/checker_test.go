package checker

import (
	"context"
	"fmt"
	"testing"

	"github.com/lynicis/actup/internal/parser"
)

type mockClient struct {
	tags map[string]string
	err  error
}

func (m *mockClient) LatestTag(ctx context.Context, owner, repo string, semverMode bool) (string, error) {
	if m.err != nil {
		return "", m.err
	}
	return m.tags[owner+"/"+repo], nil
}

func TestChecker_AllUpToDate(t *testing.T) {
	mc := &mockClient{tags: map[string]string{"actions/checkout": "v4"}}
	c := New(mc, false)
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
	c := New(mc, false)
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
	c := New(mc, false)
	actions := []parser.ActionRef{
		{Owner: "actions", Repo: "checkout", Current: "v3", File: "test.yml", Line: 5},
	}
	_, err := c.Run(context.Background(), actions)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
