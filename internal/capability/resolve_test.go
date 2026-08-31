package capability

import (
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/spec"
)

func TestResolveReturnsDependenciesFirst(t *testing.T) {
	t.Parallel()

	catalog := map[string]spec.CapabilityPack{
		"auth": pack("auth", "1.0.0"),
		"rbac": withDependencies(pack("rbac", "1.2.0"), spec.PackDependency{Name: "auth", Constraint: "^1.0.0"}),
	}
	resolved, diagnostics := Resolve(catalog, []spec.CapabilitySelection{{Name: "rbac", Version: "1.2.0"}})

	if len(diagnostics) != 0 {
		t.Fatalf("Resolve() diagnostics = %#v, want none", diagnostics)
	}
	if len(resolved) != 2 || resolved[0].Metadata.Name != "auth" || resolved[1].Metadata.Name != "rbac" {
		t.Fatalf("Resolve() order = %#v, want auth then rbac", resolved)
	}
}

func TestResolveDetectsCycle(t *testing.T) {
	t.Parallel()

	catalog := map[string]spec.CapabilityPack{
		"first":  withDependencies(pack("first", "1.0.0"), spec.PackDependency{Name: "second", Constraint: "*"}),
		"second": withDependencies(pack("second", "1.0.0"), spec.PackDependency{Name: "first", Constraint: "*"}),
	}
	_, diagnostics := Resolve(catalog, []spec.CapabilitySelection{{Name: "first", Version: "1.0.0"}})
	assertCode(t, diagnostics, "capability.dependency.cycle")
}

func TestResolveDetectsConflict(t *testing.T) {
	t.Parallel()

	first := pack("first", "1.0.0")
	first.Spec.Conflicts = []string{"second"}
	catalog := map[string]spec.CapabilityPack{"first": first, "second": pack("second", "1.0.0")}
	_, diagnostics := Resolve(catalog, []spec.CapabilitySelection{
		{Name: "first", Version: "1.0.0"},
		{Name: "second", Version: "1.0.0"},
	})
	assertCode(t, diagnostics, "capability.conflict")
}

func TestSatisfiesSemVerConstraints(t *testing.T) {
	t.Parallel()

	tests := []struct {
		version    string
		constraint string
		want       bool
	}{
		{version: "1.4.0", constraint: "^1.2.3", want: true},
		{version: "2.0.0", constraint: "^1.2.3", want: false},
		{version: "0.2.9", constraint: "^0.2.3", want: true},
		{version: "0.3.0", constraint: "^0.2.3", want: false},
		{version: "1.0.0-beta.2", constraint: ">=1.0.0", want: false},
		{version: "1.0.0", constraint: ">=1.0.0-beta.2", want: true},
	}
	for _, test := range tests {
		if got := satisfies(test.version, test.constraint); got != test.want {
			t.Errorf("satisfies(%q, %q) = %v, want %v", test.version, test.constraint, got, test.want)
		}
	}
}

func pack(name, version string) spec.CapabilityPack {
	return spec.CapabilityPack{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindCapabilityPack,
		Metadata:   spec.Metadata{Name: name, Version: version},
		Spec: spec.CapabilityPackSpec{
			Description: "test",
			Backends:    []string{"go"},
			Databases:   []string{"postgresql"},
		},
	}
}

func withDependencies(pack spec.CapabilityPack, dependencies ...spec.PackDependency) spec.CapabilityPack {
	pack.Spec.Requires = dependencies
	return pack
}

func assertCode(t *testing.T, diagnostics []spec.Diagnostic, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("diagnostics = %#v, want code %q", diagnostics, code)
}
