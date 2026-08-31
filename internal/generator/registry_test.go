package generator

import (
	"context"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/spec"
)

type testAdapter struct {
	backend string
}

func (adapter testAdapter) Backend() string {
	return adapter.backend
}

func (adapter testAdapter) Generate(context.Context, spec.Project) (Result, error) {
	return Result{}, nil
}

func TestRegistryResolvesAdapter(t *testing.T) {
	t.Parallel()

	registry := NewRegistry(testAdapter{backend: "go"})
	if _, exists := registry.Get("go"); !exists {
		t.Fatal("Get(go) exists = false, want true")
	}
	if _, exists := registry.Get("java"); exists {
		t.Fatal("Get(java) exists = true, want false")
	}
}
