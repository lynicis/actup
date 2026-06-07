package upgrader

import (
	"bytes"
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

func ApplyUpgrades(actions []parser.ActionRef, newTag string, dryRun bool) ([]Result, error) {
	grouped := groupByFile(actions)
	var results []Result

	for file, fileActions := range grouped {
		for _, action := range fileActions {
			result := Result{Action: action, NewTag: newTag}

			if dryRun {
				if err := showDryRunDiff(file, action, newTag); err != nil {
					result.Error = err
				} else {
					result.Updated = true
				}
			} else {
				if err := replaceInFile(file, action, newTag); err != nil {
					result.Error = err
				} else {
					result.Updated = true
				}
			}

			results = append(results, result)
		}
	}

	return results, nil
}

func groupByFile(actions []parser.ActionRef) map[string][]parser.ActionRef {
	grouped := make(map[string][]parser.ActionRef)
	for _, a := range actions {
		grouped[a.File] = append(grouped[a.File], a)
	}
	return grouped
}

func replaceInFile(file string, action parser.ActionRef, newTag string) error {
	content, err := os.ReadFile(file)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	found := false

	for i, line := range lines {
		if i+1 == action.Line {
			pattern := fmt.Sprintf("- uses: %s/%s@%s", action.Owner, action.Repo, action.Current)
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

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "--- a/%s\n", file)
	fmt.Fprintf(&buf, "+++ b/%s\n", file)
	fmt.Fprintf(&buf, "@@ -%d,%d +%d,%d @@\n", action.Line, 1, action.Line, 1)
	fmt.Fprintf(&buf, "-%s\n", oldLine)
	fmt.Fprintf(&buf, "+%s\n", newLine)

	fmt.Print(buf.String())
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
