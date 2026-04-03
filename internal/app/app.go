package app

import (
	"qiao/internal/config"
	"qiao/internal/core"
	claudeprovider "qiao/internal/providers/claude"
	codexprovider "qiao/internal/providers/codex"
	"qiao/internal/providers/registry"
	tencentprovider "qiao/internal/providers/tencent"
)

const (
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

	r.registry.Register("claude", claudeprovider.New, claudeprovider.ConfigFields)
	r.registry.Register("codex", codexprovider.New, codexprovider.ConfigFields)
	r.registry.Register("tencent", tencentprovider.New, tencentprovider.ConfigFields)

	return r
}

func (r *Runtime) DefaultProvider() string {
	return r.config.DefaultProvider
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

func (r *Runtime) ProviderConfigFields(name string) []core.ConfigField {
	return r.registry.ConfigFields(name)
}
