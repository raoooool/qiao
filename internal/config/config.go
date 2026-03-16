package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultProvider string                       `yaml:"default_provider"`
	DefaultSource   string                       `yaml:"default_source"`
	DefaultTarget   string                       `yaml:"default_target"`
	Providers       map[string]map[string]string `yaml:"providers"`
}

func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "qiao", "config.yaml"), nil
}

func Load(path string) (Config, error) {
	if path == "" {
		defaultPath, err := DefaultPath()
		if err != nil {
			return Config{}, err
		}
		path = defaultPath
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Config{}, nil
		}
		return Config{}, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}

	if cfg.Providers == nil {
		cfg.Providers = map[string]map[string]string{}
	}

	return cfg, nil
}

func (c Config) ProviderConfig(name string) (map[string]string, bool) {
	if c.Providers == nil {
		return nil, false
	}

	provider, ok := c.Providers[name]
	if !ok {
		return nil, false
	}

	cloned := make(map[string]string, len(provider))
	for key, value := range provider {
		cloned[key] = value
	}

	return cloned, true
}
