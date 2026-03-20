package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	DefaultProvider string                       `yaml:"default_provider"`
	DefaultSource   string                       `yaml:"default_source"`
	DefaultTarget   string                       `yaml:"default_target"`
	Providers       map[string]map[string]string `yaml:"providers,omitempty"`
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

type keyKind int

const (
	keyKindTopLevel keyKind = iota
	keyKindProvider
)

var validTopLevelKeys = map[string]bool{
	"default_provider": true,
	"default_source":   true,
	"default_target":   true,
}

func parseKeyPath(key string) (keyKind, []string, error) {
	parts := strings.Split(key, ".")

	switch len(parts) {
	case 1:
		if parts[0] == "providers" {
			return 0, nil, fmt.Errorf("%q is not a scalar key; use \"providers.<name>.<field>\"", key)
		}
		if !validTopLevelKeys[parts[0]] {
			return 0, nil, fmt.Errorf("unknown key %q: valid keys are default_provider, default_source, default_target", key)
		}
		return keyKindTopLevel, parts, nil
	case 3:
		if parts[0] != "providers" {
			return 0, nil, fmt.Errorf("invalid key %q: use \"field\" for top-level or \"providers.<name>.<field>\" for provider config", key)
		}
		return keyKindProvider, parts[1:], nil
	default:
		return 0, nil, fmt.Errorf("invalid key %q: use \"field\" for top-level or \"providers.<name>.<field>\" for provider config", key)
	}
}

func (c Config) Get(key string) (string, error) {
	kind, parts, err := parseKeyPath(key)
	if err != nil {
		return "", err
	}

	switch kind {
	case keyKindTopLevel:
		val := c.getTopLevel(parts[0])
		if val == "" {
			return "", fmt.Errorf("key %q not found", key)
		}
		return val, nil
	case keyKindProvider:
		if c.Providers == nil {
			return "", fmt.Errorf("key %q not found", key)
		}
		providerMap, ok := c.Providers[parts[0]]
		if !ok {
			return "", fmt.Errorf("key %q not found", key)
		}
		val, ok := providerMap[parts[1]]
		if !ok || val == "" {
			return "", fmt.Errorf("key %q not found", key)
		}
		return val, nil
	}
	return "", fmt.Errorf("key %q not found", key)
}

func (c Config) getTopLevel(field string) string {
	switch field {
	case "default_provider":
		return c.DefaultProvider
	case "default_source":
		return c.DefaultSource
	case "default_target":
		return c.DefaultTarget
	}
	return ""
}

func (c *Config) Set(key, value string) error {
	kind, parts, err := parseKeyPath(key)
	if err != nil {
		return err
	}

	switch kind {
	case keyKindTopLevel:
		c.setTopLevel(parts[0], value)
	case keyKindProvider:
		if c.Providers == nil {
			c.Providers = map[string]map[string]string{}
		}
		if c.Providers[parts[0]] == nil {
			c.Providers[parts[0]] = map[string]string{}
		}
		c.Providers[parts[0]][parts[1]] = value
	}
	return nil
}

func (c *Config) setTopLevel(field, value string) {
	switch field {
	case "default_provider":
		c.DefaultProvider = value
	case "default_source":
		c.DefaultSource = value
	case "default_target":
		c.DefaultTarget = value
	}
}

func (c *Config) Delete(key string) error {
	kind, parts, err := parseKeyPath(key)
	if err != nil {
		return err
	}

	switch kind {
	case keyKindTopLevel:
		val := c.getTopLevel(parts[0])
		if val == "" {
			return fmt.Errorf("key %q not found", key)
		}
		c.setTopLevel(parts[0], "")
	case keyKindProvider:
		if c.Providers == nil {
			return fmt.Errorf("key %q not found", key)
		}
		providerMap, ok := c.Providers[parts[0]]
		if !ok {
			return fmt.Errorf("key %q not found", key)
		}
		if _, ok := providerMap[parts[1]]; !ok {
			return fmt.Errorf("key %q not found", key)
		}
		delete(providerMap, parts[1])
		if len(providerMap) == 0 {
			delete(c.Providers, parts[0])
		}
	}
	return nil
}

func (c Config) List() map[string]string {
	result := map[string]string{}

	if c.DefaultProvider != "" {
		result["default_provider"] = c.DefaultProvider
	}
	if c.DefaultSource != "" {
		result["default_source"] = c.DefaultSource
	}
	if c.DefaultTarget != "" {
		result["default_target"] = c.DefaultTarget
	}

	for provider, fields := range c.Providers {
		for field, value := range fields {
			if value != "" {
				result[fmt.Sprintf("providers.%s.%s", provider, field)] = value
			}
		}
	}

	return result
}

func (c Config) Save(path string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
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
