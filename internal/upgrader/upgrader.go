package upgrader

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/lynicis/actup/internal/parser"
)

// Upgrade rewrites one action reference to a new tag.
type Upgrade struct {
	Action parser.ActionRef
	NewTag string
}

// usesPath returns the full action path (with subpath) to match in the
// workflow file. It falls back to owner/repo for refs built by hand.
func usesPath(action parser.ActionRef) string {
	if action.Path != "" {
		return action.Path
	}
	return action.Owner + "/" + action.Repo
}

func replaceInFile(file string, action parser.ActionRef, newTag string) error {
	// ponytail: file is checked and clean, bypass gosec check
	// #nosec G304
	content, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	found := false

	for i, line := range lines {
		if i+1 == action.Line {
			// Whitespace between "uses:" and the value may be spaces or tabs;
			// the value may carry a trailing comment. Match on the ref segment
			// anchored to the action path so exotic spacing cannot defeat the
			// rewrite.
			pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(usesPath(action)) + `@` + regexp.QuoteMeta(action.Current))
			if pattern.MatchString(line) {
				lines[i] = pattern.ReplaceAllLiteralString(line, usesPath(action)+"@"+newTag)
				found = true
			}
			break
		}
	}

	if !found {
		return fmt.Errorf("could not find expected uses line at line %d", action.Line)
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(file), "actup-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()

	defer func() {
		_ = os.Remove(tmpPath)
		_ = tmpFile.Close()
	}()

	if _, err := tmpFile.WriteString(strings.Join(lines, "\n")); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, file); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}

func showDryRunDiff(file string, action parser.ActionRef, newTag string) {
	oldLine := fmt.Sprintf("uses: %s@%s", usesPath(action), action.Current)
	newLine := fmt.Sprintf("uses: %s@%s", usesPath(action), newTag)

	fmt.Printf("--- a/%s\n+++ b/%s\n@@ -%d,1 +%d,1 @@\n-%s\n+%s\n", file, file, action.Line, action.Line, oldLine, newLine)
}

// ApplyAllUpgrades applies every upgrade, continuing past individual
// failures so one bad line does not block the rest. In dry-run mode it
// prints a diff-style preview instead of writing files.
func ApplyAllUpgrades(upgrades []Upgrade, dryRun bool) error {
	byFile := make(map[string][]Upgrade)
	for _, u := range upgrades {
		byFile[u.Action.File] = append(byFile[u.Action.File], u)
	}

	var errs []error
	for file, fileUpgrades := range byFile {
		for _, u := range fileUpgrades {
			if dryRun {
				showDryRunDiff(file, u.Action, u.NewTag)
				continue
			}
			if err := replaceInFile(file, u.Action, u.NewTag); err != nil {
				errs = append(errs, fmt.Errorf("%s: %w", file, err))
			}
		}
	}

	return errors.Join(errs...)
}
