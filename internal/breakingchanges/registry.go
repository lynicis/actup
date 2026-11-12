package breakingchanges

import (
	_ "embed"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
)

//go:embed registry.yaml
var registryData []byte

type BreakingChange struct {
	Type        string `yaml:"type"`
	Input       string `yaml:"input"`
	Replacement string `yaml:"replacement,omitempty"`
	Context     string `yaml:"context,omitempty"`
	Message     string `yaml:"message"`
}

type Entry struct {
	Action    string           `yaml:"action"`
	FromMajor int              `yaml:"from_major"`
	ToMajor   int              `yaml:"to_major"`
	Changes   []BreakingChange `yaml:"changes"`
}

type Registry struct {
	entries []Entry
}

// New builds a registry from explicit entries for tests.
func New(entries []Entry) *Registry { return &Registry{entries: entries} }

func LoadRegistry() (*Registry, error) {
	var entries []Entry
	if err := yaml.Unmarshal(registryData, &entries); err != nil {
		return nil, fmt.Errorf("parse breaking-change registry: %w", err)
	}
	return &Registry{entries: entries}, nil
}

func (r *Registry) Check(action string, fromVersion string, toVersion string, inputs []string) []BreakingChange {
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
		if entry.ToMajor != 0 {
			// Include when the upgrade crosses this boundary.
			if fromMajor >= entry.ToMajor || toMajor < entry.ToMajor {
				continue
			}
		} else {
			// to_major: 0 means "introduced in FromMajor and still relevant".
			if fromMajor >= entry.FromMajor || toMajor < entry.FromMajor {
				continue
			}
		}
		// ponytail: simplified loop to slice append
		for _, bc := range entry.Changes {
			if bc.Input != "" && !slices.Contains(inputs, bc.Input) {
				continue
			}
			results = append(results, bc)
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
