// Package generator defines the real boundary between the model-neutral Engine and language adapters.
package generator

import (
	"context"
	"fmt"

	"github.com/hkx5414375/scaffold-agent/internal/change"
	"github.com/hkx5414375/scaffold-agent/internal/spec"
)

// Result contains the complete deterministic desired output from one language adapter.
type Result struct {
	CapabilityLock map[string]string
	Outputs        []change.Output
}

// Adapter generates one backend without accessing the target project filesystem.
type Adapter interface {
	Backend() string
	Generate(context.Context, spec.Project) (Result, error)
}

// Registry resolves one adapter per backend.
type Registry struct {
	adapters map[string]Adapter
}

// NewRegistry creates a registry and panics on duplicate built-in backends.
func NewRegistry(adapters ...Adapter) Registry {
	registry := Registry{adapters: make(map[string]Adapter, len(adapters))}
	for _, adapter := range adapters {
		backend := adapter.Backend()
		if backend == "" {
			panic("generator adapter backend is required")
		}
		if _, exists := registry.adapters[backend]; exists {
			panic(fmt.Sprintf("duplicate generator adapter for backend %q", backend))
		}
		registry.adapters[backend] = adapter
	}
	return registry
}

// Get returns the adapter registered for a backend.
func (registry Registry) Get(backend string) (Adapter, bool) {
	adapter, exists := registry.adapters[backend]
	return adapter, exists
}
