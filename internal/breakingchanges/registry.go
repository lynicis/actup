package breakingchanges

import (
	_ "embed"
	"fmt"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
)

//go:embed registry.yaml
var registryData []byte

type Change struct {
	Type        string `yaml:"type"`
	Input       string `yaml:"input"`
	Replacement string `yaml:"replacement,omitempty"`
	Context     string `yaml:"context,omitempty"`
	Message     string `yaml:"message"`
}

type Entry struct {
	Action    string   `yaml:"action"`
	FromMajor int      `yaml:"from_major"`
	ToMajor   int      `yaml:"to_major"`
	Changes   []Change `yaml:"changes"`
}

type BreakingChange struct {
	Type        string
	Input       string
	Replacement string
	Context     string
	Message     string
}

type Registry struct {
	entries []Entry
}

func LoadRegistry() (*Registry, error) {
	var entries []Entry
	if err := yaml.Unmarshal(registryData, &entries); err != nil {
		return nil, fmt.Errorf("parse breaking-change registry: %w", err)
	}
	return &Registry{entries: entries}, nil
}

func (r *Registry) Check(action string, fromVersion string, toVersion string) []BreakingChange {
	fromMajor := parseMajor(fromVersion)
	toMajor := parseMajor(toVersion)

	if fromMajor < 0 || toMajor < 0 {
		return nil
	}

	var results []BreakingChange
	for _, entry := range r.entries {
		if entry.Action != action {
			continue
		}
		if fromMajor < entry.FromMajor {
			continue
		}
		if entry.ToMajor != 0 && toMajor > entry.ToMajor {
			continue
		}
		if fromMajor >= entry.ToMajor && entry.ToMajor != 0 {
			continue
		}
		for _, c := range entry.Changes {
			results = append(results, BreakingChange(c))
		}
	}
	return results
}

func parseMajor(version string) int {
	v := strings.TrimPrefix(version, "v")
	parts := strings.SplitN(v, ".", 2)
	if len(parts) == 0 || parts[0] == "" {
		return -1
	}
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return -1
	}
	return n
}

func ShouldUpgrade(breakingChanges []BreakingChange, force bool, dryRun bool) (upgrade bool, skip bool) {
	if len(breakingChanges) == 0 {
		return true, false
	}
	if dryRun {
		return true, false
	}
	if force {
		return true, false
	}
	return false, true
}
