package parser

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type ActionRef struct {
	Owner   string
	Repo    string
	Current string
	Line    int
	File    string
}

var usesRegex = regexp.MustCompile(`-\s+uses:\s*(.+?)\s*$`)

func ExtractActions(ctx context.Context, files []string) ([]ActionRef, error) {
	var actions []ActionRef

	for _, file := range files {
		fileActions, err := extractFromFile(file)
		if err != nil {
			return nil, fmt.Errorf("parse %s: %w", file, err)
		}
		actions = append(actions, fileActions...)
	}

	return actions, nil
}

func extractFromFile(file string) ([]ActionRef, error) {
	content, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var actions []ActionRef
	lines := strings.Split(string(content), "\n")

	for i, line := range lines {
		matches := usesRegex.FindStringSubmatch(line)
		if len(matches) < 2 {
			continue
		}

		usesValue := strings.TrimSpace(matches[1])
		ref, err := parseActionRef(usesValue)
		if err != nil {
			continue
		}

		if ref == nil {
			continue
		}

		ref.Line = i + 1
		ref.File = file
		actions = append(actions, *ref)
	}

	return actions, nil
}

func parseActionRef(uses string) (*ActionRef, error) {
	if strings.HasPrefix(uses, "./") {
		return nil, nil
	}

	if strings.HasPrefix(uses, "docker://") {
		return nil, nil
	}

	parts := strings.Split(uses, "@")
	if len(parts) != 2 {
		return nil, nil
	}

	actionPath := parts[0]
	ref := parts[1]

	if len(ref) == 40 && isHex(ref) {
		return nil, nil
	}

	pathParts := strings.Split(actionPath, "/")
	if len(pathParts) < 2 {
		return nil, nil
	}

	owner := pathParts[0]
	repo := pathParts[1]

	return &ActionRef{
		Owner:   owner,
		Repo:    repo,
		Current: ref,
	}, nil
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

func GroupActions(actions []ActionRef) map[string][]ActionRef {
	grouped := make(map[string][]ActionRef)
	for _, a := range actions {
		key := a.Owner + "/" + a.Repo
		grouped[key] = append(grouped[key], a)
	}
	return grouped
}