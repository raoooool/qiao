package app

import (
	"qiao/internal/config"
	"qiao/internal/core"
	googleprovider "qiao/internal/providers/google"
	"qiao/internal/providers/registry"
)

const (
	defaultProvider = "google"
	defaultSource   = "auto"
	defaultTarget   = "zh"
)

type Runtime struct {
	config   config.Config
	registry *registry.Registry
}

func Load(path string) (*Runtime, error) {
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}

	return New(cfg), nil
}

func New(cfg config.Config) *Runtime {
	r := &Runtime{
		config:   cfg,
		registry: registry.New(),
	}

	r.registry.Register("google", googleprovider.New)

	return r
}

func (r *Runtime) DefaultProvider() string {
	if r.config.DefaultProvider != "" {
		return r.config.DefaultProvider
	}

	return defaultProvider
}

func (r *Runtime) DefaultSource() string {
	if r.config.DefaultSource != "" {
		return r.config.DefaultSource
	}

	return defaultSource
}

func (r *Runtime) DefaultTarget() string {
	if r.config.DefaultTarget != "" {
		return r.config.DefaultTarget
	}

	return defaultTarget
}

func (r *Runtime) ResolveProvider(name string) (core.Translator, error) {
	return r.registry.Resolve(name, r.config)
}

func (r *Runtime) ListProviders() []string {
	return r.registry.Providers()
}
