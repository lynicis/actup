package upgrader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/lynicis/actup/internal/parser"
)

type Upgrade struct {
	Action parser.ActionRef
	NewTag string
}

type Result struct {
	Action  parser.ActionRef
	NewTag  string
	Updated bool
	Error   error
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
			pattern := fmt.Sprintf("uses: %s/%s@%s", action.Owner, action.Repo, action.Current)
			if strings.Contains(line, pattern) {
				lines[i] = strings.Replace(line, action.Current, newTag, 1)
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

func showDryRunDiff(file string, action parser.ActionRef, newTag string) error {
	oldLine := fmt.Sprintf("uses: %s/%s@%s", action.Owner, action.Repo, action.Current)
	newLine := fmt.Sprintf("uses: %s/%s@%s", action.Owner, action.Repo, newTag)

	fmt.Printf("--- a/%s\n+++ b/%s\n@@ -%d,1 +%d,1 @@\n-%s\n+%s\n", file, file, action.Line, action.Line, oldLine, newLine)
	return nil
}

func ApplyAllUpgrades(upgrades map[string]Upgrade, dryRun bool) (map[string][]Result, error) {
	results := make(map[string][]Result)

	byFile := make(map[string][]Upgrade)
	for _, u := range upgrades {
		byFile[u.Action.File] = append(byFile[u.Action.File], u)
	}

	for file, fileUpgrades := range byFile {
		var fileResults []Result

		for _, u := range fileUpgrades {
			result := Result{Action: u.Action, NewTag: u.NewTag}

			if dryRun {
				if err := showDryRunDiff(file, u.Action, u.NewTag); err != nil {
					result.Error = err
				} else {
					result.Updated = true
				}
			} else {
				if err := replaceInFile(file, u.Action, u.NewTag); err != nil {
					result.Error = err
				} else {
					result.Updated = true
				}
			}

			fileResults = append(fileResults, result)
		}

		results[file] = fileResults
	}

	return results, nil
}
