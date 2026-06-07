package scanner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const defaultWorkflowDir = ".github/workflows"

func DiscoverWorkflows(ctx context.Context, paths []string) ([]string, error) {
	var files []string
	seen := make(map[string]bool)

	if len(paths) == 0 {
		cwd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get current working directory: %w", err)
		}
		paths = []string{filepath.Join(cwd, defaultWorkflowDir)}
	}

	for _, p := range paths {
		absPath, err := filepath.Abs(p)
		if err != nil {
			return nil, fmt.Errorf("resolve path %q: %w", p, err)
		}

		info, err := os.Stat(absPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("stat path %q: %w", p, err)
		}

		if info.IsDir() {
			dirFiles, err := discoverInDir(absPath)
			if err != nil {
				return nil, fmt.Errorf("discover in directory %q: %w", p, err)
			}
			for _, f := range dirFiles {
				if !seen[f] {
					seen[f] = true
					files = append(files, f)
				}
			}
		} else {
			if isWorkflowFile(absPath) && !seen[absPath] {
				seen[absPath] = true
				files = append(files, absPath)
			}
		}
	}

	return files, nil
}

func discoverInDir(dir string) ([]string, error) {
	var files []string

	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		if isWorkflowFile(path) {
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
