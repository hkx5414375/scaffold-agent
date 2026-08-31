package spec

import "testing"

func TestValidateProjectAcceptsPortableProject(t *testing.T) {
	t.Parallel()

	project := validProject()
	if diagnostics := ValidateProject(project); len(diagnostics) != 0 {
		t.Fatalf("ValidateProject() diagnostics = %#v, want none", diagnostics)
	}
}

func TestValidateProjectRejectsDuplicateAndNonPortableFields(t *testing.T) {
	t.Parallel()

	project := validProject()
	project.Spec.Auth.Modes = []string{"session", "session"}
	project.Spec.Modules[0].Entities[0].Fields = append(project.Spec.Modules[0].Entities[0].Fields,
		Field{Name: "DisplayName", Type: "varchar"},
	)

	diagnostics := ValidateProject(project)
	assertDiagnosticCode(t, diagnostics, "project.auth.mode.duplicate")
	assertDiagnosticCode(t, diagnostics, "project.field.name.invalid")
	assertDiagnosticCode(t, diagnostics, "project.field.type.invalid")
}

func TestValidateCapabilityPackRejectsSelfDependency(t *testing.T) {
	t.Parallel()

	pack := validPack("rbac", "1.0.0")
	pack.Spec.Requires = []PackDependency{{Name: "rbac", Constraint: "^1.0.0"}}

	diagnostics := ValidateCapabilityPack(pack)
	assertDiagnosticCode(t, diagnostics, "pack.dependency.self")
}

func TestValidateProjectRejectsInvalidWorkflowAndPermission(t *testing.T) {
	t.Parallel()

	project := validProject()
	project.Spec.Modules[0].Workflows = []Workflow{{Name: "approval", States: []string{"draft", "draft"}}}
	project.Spec.Modules[0].Permissions = []Permission{{Code: "Invalid Permission"}}

	diagnostics := ValidateProject(project)
	assertDiagnosticCode(t, diagnostics, "project.workflow.state.duplicate")
	assertDiagnosticCode(t, diagnostics, "project.permission.code.invalid")
}

func validProject() Project {
	return Project{
		APIVersion: APIVersionV1Alpha1,
		Kind:       KindProject,
		Metadata:   Metadata{Name: "inventory-demo", DisplayName: "Inventory Demo"},
		Spec: ProjectSpec{
			Stack:    StackSpec{Backend: "go", AdminUI: "element-plus", Storefront: "nuxt"},
			Database: DatabaseSpec{Engine: "postgresql"},
			Auth:     AuthSpec{Modes: []string{"session", "token"}},
			Capabilities: []CapabilitySelection{
				{Name: "rbac", Version: "1.0.0", Config: map[string]any{"data_scope": true}},
			},
			Modules: []Module{{
				Name: "inventory",
				Entities: []Entity{{
					Name:   "inventory_order",
					Fields: []Field{{Name: "order_no", Type: "string", Required: true, Unique: true}},
				}},
			}},
		},
	}
}

func validPack(name, version string) CapabilityPack {
	return CapabilityPack{
		APIVersion: APIVersionV1Alpha1,
		Kind:       KindCapabilityPack,
		Metadata:   Metadata{Name: name, Version: version},
		Spec: CapabilityPackSpec{
			Description: "Test capability",
			Backends:    []string{"go"},
			Databases:   []string{"postgresql"},
		},
	}
}

func assertDiagnosticCode(t *testing.T, diagnostics []Diagnostic, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("diagnostics = %#v, want code %q", diagnostics, code)
}
