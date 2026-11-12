package parser

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type ActionRef struct {
	Owner string
	Repo  string
	// Path is the full action path before "@", including any subpath
	// (e.g. "google/osv-scanner-action/osv-scanner-action").
	Path string
	// Current is the ref token exactly as written (tag, branch, or SHA).
	Current string
	// SHA is the lowercase 40-hex commit SHA, or "" when unpinned.
	SHA string
	// Tag is the version parsed from the trailing comment, or "".
	Tag string
	// Inputs are keys under the step's with: block, file order; nil if none.
	Inputs []string
	Line   int
	File   string
}

// Pinned reports whether the ref is a commit SHA.
func (a ActionRef) Pinned() bool { return a.SHA != "" }

// Version is the comparison value. A SHA pin carries its version in the
// trailing comment.
func (a ActionRef) Version() string {
	if a.SHA != "" {
		return a.Tag
	}
	return a.Current
}

// usesRegex captures the ref value and an optional trailing YAML comment.
var usesRegex = regexp.MustCompile(`^\s*(?:-\s+)?uses:\s*(.+?)(?:\s+#\s*(.*?))?\s*$`)

var shaRegex = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

// withRegex matches a with: key at the step's sibling indent.
var withRegex = regexp.MustCompile(`^with:\s*(.*)$`)

// withKeyRegex matches one key in a block map.
var withKeyRegex = regexp.MustCompile(`^\s*([^:#][^:#]*?)\s*:`)

// commentVersion accepts a strict leading version: "# v4", "# v4.2.2".
var commentVersion = regexp.MustCompile(`^(v?\d+(?:\.\d+){0,2})\b`)

func ExtractActions(_ context.Context, files []string) ([]ActionRef, error) {
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
	// ponytail: file is checked and clean, bypass gosec check
	// #nosec G304
	content, err := os.ReadFile(filepath.Clean(file))
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

		usesValue := trimMatchingQuotes(strings.TrimSpace(matches[1]))
		ref, err := parseActionRef(usesValue)
		if err != nil {
			continue
		}

		if ref == nil {
			continue
		}

		if len(matches) > 2 && ref.SHA != "" {
			if m := commentVersion.FindStringSubmatch(strings.TrimSpace(matches[2])); m != nil {
				ref.Tag = m[1]
			}
		}

		ref.Line = i + 1
		ref.File = file
		ref.Inputs = inputsForStep(lines, i)
		actions = append(actions, *ref)
	}

	return actions, nil
}

// leadingIndent counts the leading spaces on a line.
func leadingIndent(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

// inputsForStep returns the with: keys belonging to the step containing the
// uses: line at index i.
func inputsForStep(lines []string, i int) []string {
	keyIndent := strings.Index(lines[i], "uses:")

	start := i
	if leadingIndent(lines[i]) >= keyIndent {
		start = 0
		for j := i - 1; j >= 0; j-- {
			if strings.TrimSpace(lines[j]) == "" {
				continue
			}
			if leadingIndent(lines[j]) < keyIndent {
				start = j
				break
			}
		}
	}
	end := len(lines)
	for j := i + 1; j < len(lines); j++ {
		if strings.TrimSpace(lines[j]) == "" {
			continue
		}
		if leadingIndent(lines[j]) < keyIndent {
			end = j
			break
		}
	}

	for j := start; j < end; j++ {
		trimmed := strings.TrimSpace(lines[j])
		indent := leadingIndent(lines[j])
		if indent == keyIndent {
			// Sibling key.
		} else if indent+2 == keyIndent && strings.HasPrefix(trimmed, "- ") {
			// with: on the step's dash line, e.g. "- with: {...}".
			trimmed = strings.TrimSpace(strings.TrimPrefix(trimmed, "- "))
		} else {
			continue
		}
		m := withRegex.FindStringSubmatch(trimmed)
		if m == nil {
			continue
		}
		return parseWithValue(lines, j, end, keyIndent, m[1])
	}
	return nil
}

func parseWithValue(lines []string, idx, end, keyIndent int, value string) []string {
	value = strings.TrimSpace(value)
	if value == "" {
		return parseBlockMap(lines, idx, end, keyIndent)
	}
	if !strings.HasPrefix(value, "{") {
		// Anchors, null, and other scalars yield no keys.
		return nil
	}
	value = strings.TrimSuffix(strings.TrimPrefix(value, "{"), "}")
	var keys []string
	for _, entry := range strings.Split(value, ",") {
		key, _, _ := strings.Cut(entry, ":")
		key = trimMatchingQuotes(strings.TrimSpace(key))
		if key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

func parseBlockMap(lines []string, idx, end, keyIndent int) []string {
	var keys []string
	for j := idx + 1; j < end; j++ {
		if strings.TrimSpace(lines[j]) == "" {
			continue
		}
		if leadingIndent(lines[j]) <= keyIndent {
			break
		}
		if strings.HasPrefix(strings.TrimSpace(lines[j]), "#") {
			continue
		}
		m := withKeyRegex.FindStringSubmatch(lines[j])
		if m == nil {
			continue
		}
		key := trimMatchingQuotes(strings.TrimSpace(m[1]))
		if key != "" {
			keys = append(keys, key)
		}
	}
	return keys
}

func parseActionRef(uses string) (*ActionRef, error) {
	if uses == "" || strings.Contains(uses, "${{") {
		return nil, nil
	}

	if strings.HasPrefix(uses, "./") || strings.HasPrefix(uses, "$/") {
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

	if ref == "" {
		return nil, nil
	}

	pathParts := strings.Split(actionPath, "/")
	if len(pathParts) < 2 {
		return nil, nil
	}

	owner := pathParts[0]
	repo := pathParts[1]

	action := &ActionRef{
		Owner:   owner,
		Repo:    repo,
		Path:    actionPath,
		Current: ref,
	}
	if shaRegex.MatchString(ref) {
		action.SHA = strings.ToLower(ref)
	}
	return action, nil
}

// trimMatchingQuotes strips one matching pair of surrounding quotes.
func trimMatchingQuotes(s string) string {
	if len(s) >= 2 && (s[0] == '"' || s[0] == '\'') && s[len(s)-1] == s[0] {
		return s[1 : len(s)-1]
	}
	return s
}

// DistinctVersions returns distinct version strings in first-seen order.
func DistinctVersions(refs []ActionRef) []string {
	seen := make(map[string]bool, len(refs))
	var versions []string
	for _, ref := range refs {
		if seen[ref.Current] {
			continue
		}
		seen[ref.Current] = true
		versions = append(versions, ref.Current)
	}
	return versions
}

func GroupActions(actions []ActionRef) map[string][]ActionRef {
	grouped := make(map[string][]ActionRef)
	for _, a := range actions {
		key := a.Owner + "/" + a.Repo
		grouped[key] = append(grouped[key], a)
	}
	return grouped
}
