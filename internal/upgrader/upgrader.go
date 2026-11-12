package upgrader

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/lynicis/actup/internal/parser"
)

// Upgrade rewrites one action reference. NewSHA non-empty writes a SHA pin
// with "# <NewTag>" as its comment.
type Upgrade struct {
	Action parser.ActionRef
	NewTag string
	NewSHA string
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
	return replaceAllInFile(file, []Upgrade{{Action: action, NewTag: newTag}})
}

// replaceAllInFile applies every upgrade for one file in a single
// read-modify-write and preserves the original permission bits.
func replaceAllInFile(file string, upgrades []Upgrade) error {
	if len(upgrades) == 0 {
		return nil
	}

	info, err := os.Stat(file)
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}
	mode := info.Mode().Perm()

	// ponytail: file is checked and clean, bypass gosec check
	// #nosec G304
	content, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return fmt.Errorf("read file: %w", err)
	}

	lines := strings.Split(string(content), "\n")

	var errs []error
	seen := make(map[int]Upgrade, len(upgrades))
	for _, u := range upgrades {
		line := u.Action.Line
		if prev, ok := seen[line]; ok {
			// Stable line sort keeps the first caller-supplied upgrade authoritative.
			if prev.NewTag != u.NewTag || prev.NewSHA != u.NewSHA {
				if prev.NewSHA == "" && u.NewSHA == "" {
					errs = append(errs, fmt.Errorf("conflicting tags for line %d: %q and %q", line, prev.NewTag, u.NewTag))
				} else {
					errs = append(errs, fmt.Errorf("conflicting upgrades for line %d: %q/%q and %q/%q", line, prev.NewTag, prev.NewSHA, u.NewTag, u.NewSHA))
				}
			}
			continue
		}
		seen[line] = u

		if u.NewTag == "" {
			errs = append(errs, fmt.Errorf("invalid empty tag at line %d", line))
			continue
		}

		// Whitespace between "uses:" and the value may be spaces or tabs;
		// the value may carry a trailing comment. Match on the ref segment
		// anchored to the action path so exotic spacing cannot defeat the
		// rewrite.
		var pattern *regexp.Regexp
		replacement := usesPath(u.Action) + "@" + u.NewTag
		if u.NewSHA != "" {
			// SHA mode replaces the ref and normalizes (or strips) the comment.
			// The optional quote sits immediately after the ref so quoted uses
			// values keep their quotes with the comment outside them; the
			// trailing \r is preserved for CRLF files.
			pattern = regexp.MustCompile(`(` + regexp.QuoteMeta(usesPath(u.Action)+"@"+u.Action.Current) + `)(['"]?)(?:[ \t]+#.*)?[ \t]*\r?$`)
		} else {
			pattern = regexp.MustCompile(`\b` + regexp.QuoteMeta(usesPath(u.Action)) + `@` + regexp.QuoteMeta(u.Action.Current))
		}
		if line < 1 || line > len(lines) || !pattern.MatchString(lines[line-1]) {
			errs = append(errs, fmt.Errorf("could not find expected uses line at line %d", line))
			continue
		}
		if u.NewSHA != "" {
			oldLine := lines[line-1]
			idx := pattern.FindStringSubmatchIndex(oldLine)
			quote := oldLine[idx[4]:idx[5]]
			newLine := oldLine[:idx[0]] + usesPath(u.Action) + "@" + u.NewSHA + quote + " # " + u.NewTag
			if strings.HasSuffix(oldLine, "\r") {
				newLine += "\r"
			}
			lines[line-1] = newLine
		} else {
			lines[line-1] = pattern.ReplaceAllLiteralString(lines[line-1], replacement)
		}
	}

	tmpFile, err := os.CreateTemp(filepath.Dir(file), "actup-*.tmp")
	if err != nil {
		errs = append(errs, fmt.Errorf("create temp file: %w", err))
		return errors.Join(errs...)
	}
	tmpPath := tmpFile.Name()

	defer func() {
		_ = os.Remove(tmpPath)
		_ = tmpFile.Close()
	}()

	if _, err := tmpFile.WriteString(strings.Join(lines, "\n")); err != nil {
		errs = append(errs, fmt.Errorf("write temp file: %w", err))
		return errors.Join(errs...)
	}

	if err := os.Chmod(tmpPath, mode); err != nil {
		errs = append(errs, fmt.Errorf("chmod temp file: %w", err))
		return errors.Join(errs...)
	}

	if err := tmpFile.Close(); err != nil {
		errs = append(errs, fmt.Errorf("close temp file: %w", err))
		return errors.Join(errs...)
	}

	if err := os.Rename(tmpPath, file); err != nil {
		errs = append(errs, fmt.Errorf("rename temp file: %w", err))
	}

	return errors.Join(errs...)
}

func showDryRunDiff(file string, u Upgrade) {
	action := u.Action
	oldLine := fmt.Sprintf("uses: %s@%s", usesPath(action), action.Current)
	newRef := u.NewTag
	if u.NewSHA != "" {
		newRef = u.NewSHA + " # " + u.NewTag
	}
	newLine := fmt.Sprintf("uses: %s@%s", usesPath(action), newRef)

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
	for _, file := range slices.Sorted(maps.Keys(byFile)) {
		fileUpgrades := byFile[file]
		slices.SortStableFunc(fileUpgrades, func(a, b Upgrade) int {
			return cmp.Compare(a.Action.Line, b.Action.Line)
		})

		if dryRun {
			for _, u := range fileUpgrades {
				showDryRunDiff(file, u)
			}
			continue
		}

		if err := replaceAllInFile(file, fileUpgrades); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", file, err))
		}
	}

	return errors.Join(errs...)
}
