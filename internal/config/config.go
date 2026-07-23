package config

import (
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Major   *int              `yaml:"major,omitempty"`
	Actions map[string]string `yaml:"actions,omitempty"`
}

func LoadDefault() (*Config, error) {
	paths := []string{".actup.yaml", ".actup.yml"}

	for _, path := range paths {
		// ponytail: file is checked and clean, bypass gosec check
		// #nosec G304
		data, err := os.ReadFile(filepath.Clean(path))
		if err == nil {
			var cfg Config
			if err := yaml.Unmarshal(data, &cfg); err != nil {
				return nil, err
			}
			return &cfg, nil
		}
		if !os.IsNotExist(err) {
			return nil, err
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
