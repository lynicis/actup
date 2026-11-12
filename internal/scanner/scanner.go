package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultWorkflowDir = ".github/workflows"
	defaultActionsDir  = ".github/actions"
)

func DiscoverWorkflows(ctx context.Context, paths []string) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	if len(paths) == 0 {
		// Defaults only: workflows, composite actions, then the repo-root action file.
		if err := appendPath(&files, seen, defaultWorkflowDir, isWorkflowFile); err != nil {
			return nil, err
		}
		if err := appendPath(&files, seen, defaultActionsDir, isActionFile); err != nil {
			return nil, err
		}
		for _, p := range []string{"action.yml", "action.yaml"} {
			if err := appendPath(&files, seen, p, isActionFile); err != nil {
				return nil, err
			}
		}
		return files, nil
	}

	// Explicit -p semantics unchanged: no defaults are appended.
	for _, p := range paths {
		if err := appendPath(&files, seen, p, isWorkflowFile); err != nil {
			return nil, err
		}
	}

	return files, nil
}

func appendPath(files *[]string, seen map[string]bool, p string, match func(string) bool) error {
	absPath, err := filepath.Abs(p)
	if err != nil {
		return fmt.Errorf("resolve path %q: %w", p, err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat path %q: %w", p, err)
	}

	if info.IsDir() {
		dirFiles, err := discoverInDir(absPath, match)
		if err != nil {
			return fmt.Errorf("discover in directory %q: %w", p, err)
		}
		for _, f := range dirFiles {
			if !seen[f] {
				seen[f] = true
				*files = append(*files, f)
			}
		}
		return nil
	}

	if match(absPath) && !seen[absPath] {
		seen[absPath] = true
		*files = append(*files, absPath)
	}

	return nil
}

func discoverInDir(dir string, match func(string) bool) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if match(path) {
			files = append(files, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

func isWorkflowFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return ext == ".yml" || ext == ".yaml"
}

func isActionFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return base == "action.yml" || base == "action.yaml"
}
