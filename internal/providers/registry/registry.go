package registry

import (
	"fmt"
	"slices"

	"qiao/internal/config"
	"qiao/internal/core"
)

type Factory func(config.Config) (core.Translator, error)

type Registry struct {
	factories map[string]Factory
}

func New() *Registry {
	return &Registry{
		factories: map[string]Factory{},
	}
}

func (r *Registry) Register(name string, factory Factory) {
	r.factories[name] = factory
}

func (r *Registry) Resolve(name string, cfg config.Config) (core.Translator, error) {
	factory, ok := r.factories[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider %q", name)
	}

	return factory(cfg)
}

func (r *Registry) Providers() []string {
	providers := make([]string, 0, len(r.factories))
	for name := range r.factories {
		providers = append(providers, name)
	}

	slices.Sort(providers)

	return providers
}
