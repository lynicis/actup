package config

import (
	"os"

	"github.com/goccy/go-yaml"
)

type Config struct {
	Major   *int              `yaml:"major,omitempty"`
	Actions map[string]string `yaml:"actions,omitempty"`
}

func DefaultPath() string {
	return ".actup.yaml"
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
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
