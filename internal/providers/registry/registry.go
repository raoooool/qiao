package registry

import (
	"fmt"
	"slices"

	"qiao/internal/config"
	"qiao/internal/core"
)

type Factory func(config.Config) (core.Translator, error)

type providerInfo struct {
	factory      Factory
	configFields []core.ConfigField
}

type Registry struct {
	providers map[string]providerInfo
}

func New() *Registry {
	return &Registry{
		providers: map[string]providerInfo{},
	}
}

func (r *Registry) Register(name string, factory Factory, fields []core.ConfigField) {
	r.providers[name] = providerInfo{
		factory:      factory,
		configFields: fields,
	}
}

func (r *Registry) Resolve(name string, cfg config.Config) (core.Translator, error) {
	info, ok := r.providers[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", name)
	}

	return info.factory(cfg)
}

func (r *Registry) Providers() []string {
	providers := make([]string, 0, len(r.providers))
	for name := range r.providers {
		providers = append(providers, name)
	}

	slices.Sort(providers)

	return providers
}

func (r *Registry) ConfigFields(name string) []core.ConfigField {
	info, ok := r.providers[name]
	if !ok {
		return nil
	}

	return info.configFields
}
