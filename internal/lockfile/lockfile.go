// Package lockfile reads .github/workflows/actions.lock files produced by
// `gh actions-lock` and answers questions about pinned action references.
package lockfile

import (
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/lynicis/actup/internal/parser"
)

const DefaultPath = ".github/workflows/actions.lock"

type Status int

const (
	StatusNone Status = iota
	StatusLocked
	StatusStale
)

type Dependency struct {
	Ref     string   `yaml:"ref"`
	Commit  string   `yaml:"commit"`
	OwnerID int64    `yaml:"owner_id"`
	RepoID  int64    `yaml:"repo_id"`
	Uses    []string `yaml:"uses,omitempty"`
}

type Lockfile struct {
	Path         string                `yaml:"-"`
	Version      string                `yaml:"version"`
	Workflows    map[string][]string   `yaml:"workflows"`
	Dependencies map[string]Dependency `yaml:"dependencies"`
}

var shaRegex = regexp.MustCompile(`^[0-9a-fA-F]{40}$`)

func Load(path string) (*Lockfile, error) {
	// ponytail: path is caller-supplied; same read pattern as internal/parser.
	// #nosec G304
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, fmt.Errorf("read lockfile: %w", err)
	}

	var lf Lockfile
	if err := yaml.Unmarshal(data, &lf); err != nil {
		return nil, fmt.Errorf("parse lockfile %s: %w", path, err)
	}
	lf.Path = path
	return &lf, nil
}

func LoadForPaths(paths []string) (*Lockfile, error) {
	if len(paths) == 0 {
		if _, err := os.Stat(DefaultPath); err != nil {
			return nil, nil
		}
		return Load(DefaultPath)
	}

	for _, p := range paths {
		for _, candidate := range []string{
			filepath.Join(p, "actions.lock"),
			filepath.Join(p, ".github", "workflows", "actions.lock"),
			filepath.Join(filepath.Dir(p), "actions.lock"),
		} {
			if _, err := os.Stat(candidate); err == nil {
				return Load(candidate)
			}
		}
	}
	return nil, nil
}

func canonicalPinKey(owner, repo, ref string) string {
	return strings.ToLower(owner) + "/" + strings.ToLower(repo) + "@" + ref
}

func (l *Lockfile) TransitiveRefs(direct []parser.ActionRef) []parser.ActionRef {
	if l == nil {
		return nil
	}

	directSet := make(map[string]struct{}, len(direct)*2+len(l.Workflows))
	for _, a := range direct {
		directSet[canonicalPinKey(a.Owner, a.Repo, a.Current)] = struct{}{}
		directSet[canonicalPinKey(a.Owner, a.Repo, a.Version())] = struct{}{}
	}
	for _, refs := range l.Workflows {
		for _, entry := range refs {
			if key, ok := workflowKey(entry); ok {
				directSet[key] = struct{}{}
			}
		}
	}

	var out []parser.ActionRef
	seen := make(map[string]struct{})
	for _, key := range slices.Sorted(maps.Keys(l.Dependencies)) {
		repoPart, spec, ok := strings.Cut(key, "@")
		if !ok {
			continue
		}
		owner, repo, ok := strings.Cut(repoPart, "/")
		if !ok {
			continue
		}
		if _, ok := directSet[canonicalPinKey(owner, repo, spec)]; ok {
			continue
		}

		dep := l.Dependencies[key]
		ref := parser.ActionRef{
			Owner: owner,
			Repo:  repo,
			Path:  repoPart,
			Line:  0,
			File:  l.Path,
		}
		switch {
		case shaRegex.MatchString(dep.Ref):
			ref.Current = dep.Ref
			ref.SHA = strings.ToLower(dep.Ref)
		case dep.Ref != "":
			ref.Current = dep.Ref
		default:
			ref.Current = spec
		}

		dedupKey := canonicalPinKey(owner, repo, ref.Version()+"|"+ref.Current)
		if _, ok := seen[dedupKey]; ok {
			continue
		}
		seen[dedupKey] = struct{}{}
		out = append(out, ref)
	}
	return out
}

// workflowKey normalizes a workflows entry (e.g. "Actions/cache/save@v4") to a
// canonical repo-scoped key.
func workflowKey(entry string) (string, bool) {
	repoPart, ref, ok := strings.Cut(entry, "@")
	if !ok {
		return "", false
	}
	parts := strings.Split(repoPart, "/")
	if len(parts) < 2 {
		return "", false
	}
	return canonicalPinKey(parts[0], parts[1], ref), true
}

func (l *Lockfile) StatusForAction(key string, refs []parser.ActionRef) Status {
	if l == nil || len(refs) == 0 {
		return StatusNone
	}

	locked := make(map[string]struct{}, len(l.Dependencies))
	for k := range l.Dependencies {
		repoPart, ref, ok := strings.Cut(k, "@")
		if !ok {
			continue
		}
		locked[strings.ToLower(repoPart)+"@"+ref] = struct{}{}
	}

	for _, r := range refs {
		_, byCurrent := locked[canonicalPinKey(r.Owner, r.Repo, r.Current)]
		_, byVersion := locked[canonicalPinKey(r.Owner, r.Repo, r.Version())]
		if !byCurrent && !byVersion {
			return StatusStale
		}
	}
	return StatusLocked
}

var execCommand = exec.Command

func Sync(lockfilePath string) error {
	cmd := execCommand("gh", "actions-lock")
	d := filepath.ToSlash(lockfilePath)
	if d == DefaultPath || strings.HasSuffix(d, "/"+DefaultPath) {
		cmd.Dir = filepath.Dir(filepath.Dir(filepath.Dir(lockfilePath)))
	}

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("gh actions-lock: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
