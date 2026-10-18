package config

import (
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"github.com/goccy/go-yaml"
)

const schemaHeader = "# yaml-language-server: $schema=https://raw.githubusercontent.com/lynicis/actup/main/schema.json"

type Config struct {
	Major   *int              `yaml:"major,omitempty"`
	Actions map[string]string `yaml:"actions,omitempty"`
}

func LoadDefault() (*Config, error) {
	paths := []string{".actup.yaml", ".actup.yml"}
	for _, path := range paths {
		cfg, err := Load(path)
		if err != nil {
			return nil, err
		}
		if cfg != nil {
			return cfg, nil
		}
	}
	return nil, nil
}

func Load(path string) (*Config, error) {
	// ponytail: file is checked and clean, bypass gosec check
	// #nosec G304
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveDefault writes the config to the default location: the existing
// .actup.yaml/.actup.yml if one is present, otherwise a new .actup.yaml.
func SaveDefault(cfg *Config) error {
	if _, err := os.Stat(".actup.yaml"); err == nil {
		return Save(".actup.yaml", cfg)
	}
	if _, err := os.Stat(".actup.yml"); err == nil {
		return Save(".actup.yml", cfg)
	}
	return Save(".actup.yaml", cfg)
}

// Save writes the config as deterministic YAML. Existing comments are not
// preserved; a schema header is emitted for editor autocomplete.
func Save(path string, cfg *Config) error {
	// ponytail: config contains no secrets, but 0600 avoids gosec G306
	// #nosec G306
	if err := os.WriteFile(filepath.Clean(path), []byte(Render(cfg)), 0600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

// Render renders the config as deterministic YAML with a schema header.
func Render(cfg *Config) string {
	var b strings.Builder
	b.WriteString(schemaHeader + "\n")

	if cfg == nil {
		return b.String()
	}

	if cfg.Major != nil {
		fmt.Fprintf(&b, "\nmajor: %d\n", *cfg.Major)
	}

	if len(cfg.Actions) > 0 {
		b.WriteString("\nactions:\n")
		keys := slices.Sorted(maps.Keys(cfg.Actions))
		for _, key := range keys {
			fmt.Fprintf(&b, "  %s: %s\n", formatScalar(key), formatScalar(cfg.Actions[key]))
		}
	}

	return b.String()
}

// formatScalar quotes YAML scalars that would otherwise parse as another type
// (numbers, booleans, null) or contain special characters.
func formatScalar(s string) string {
	if s == "" {
		return `""`
	}
	if _, err := strconv.Atoi(s); err == nil {
		return strconv.Quote(s)
	}
	lowered := strings.ToLower(s)
	switch lowered {
	case "true", "false", "yes", "no", "on", "off", "null", "~":
		return strconv.Quote(s)
	}
	if strings.ContainsAny(s, ":#{}[]&*!|>'\"%@` \t") {
		return strconv.Quote(s)
	}
	return s
}
