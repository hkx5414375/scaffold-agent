package python

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/generator"
	adminui "github.com/hkx5414375/scaffold-agent/internal/generator/admin"
	"github.com/hkx5414375/scaffold-agent/internal/spec"
	"go.yaml.in/yaml/v3"
)

func TestGenerateFoundationForBothDatabases(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			project := validProject()
			project.Spec.Database.Engine = database
			first, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			second, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate(second) error = %v", err)
			}
			if !reflect.DeepEqual(first, second) {
				t.Fatal("Generate() is not deterministic")
			}
			if first.CapabilityLock[baseOwner] != baseVersion || len(first.Outputs) != 29 {
				t.Fatalf("Generate() result = %#v", first)
			}
			for _, path := range []string{
				"pyproject.toml",
				"uv.lock",
				"api/openapi.yaml",
				"src/demo_service/main.py",
				"src/demo_service/identity/service.py",
				"src/demo_service/migration/versions/000001_identity.py",
				"tests/test_architecture.py",
			} {
				if outputContent(first, path) == nil {
					t.Errorf("Generate() did not produce %s", path)
				}
			}
			var contract map[string]any
			if err := yaml.Unmarshal(outputContent(first, "api/openapi.yaml"), &contract); err != nil {
				t.Fatalf("generated OpenAPI is invalid YAML: %v", err)
			}
			configuration := string(outputContent(first, "pyproject.toml"))
			if !strings.Contains(configuration, databases[database].DriverDependency) {
				t.Fatalf("generated pyproject does not select %s driver:\n%s", database, configuration)
			}
		})
	}
}

func TestGenerateRejectsIncompleteSelections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*spec.Project)
		want   string
	}{
		{name: "admin", mutate: func(project *spec.Project) { project.Spec.Stack.AdminUI = "unsupported" }, want: "Element Plus"},
		{name: "storefront", mutate: func(project *spec.Project) { project.Spec.Stack.Storefront = "nuxt" }, want: "storefront"},
		{name: "auth", mutate: func(project *spec.Project) { project.Spec.Auth.Modes = []string{"session"} }, want: "both session and token"},
		{name: "capability", mutate: func(project *spec.Project) {
			project.Spec.Capabilities = []spec.CapabilitySelection{{Name: "unsupported", Version: "0.1.0"}}
		}, want: "not present in the catalog"},
		{name: "capability configuration", mutate: func(project *spec.Project) {
			project.Spec.Capabilities = []spec.CapabilitySelection{{
				Name: tenancyOwner, Version: tenancyVersion, Config: map[string]any{"mode": "custom"},
			}}
		}, want: "does not accept configuration"},
		{name: "multiple modules", mutate: func(project *spec.Project) {
			project.Spec.Modules = []spec.Module{{Name: "tasks"}, {Name: "notes"}}
		}, want: "at most one business module"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			project := validProject()
			test.mutate(&project)
			_, err := New().Generate(context.Background(), project)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Generate() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestGenerateOrganizationTenancyForBothDatabases(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			project := validBusinessProject()
			project.Spec.Database.Engine = database
			project.Spec.Stack.AdminUI = "element-plus"
			project.Spec.Capabilities = []spec.CapabilitySelection{{
				Name: tenancyOwner, Version: tenancyVersion,
			}}
			generated, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate(%s) error = %v", database, err)
			}
			if generated.CapabilityLock[tenancyOwner] != tenancyVersion || len(generated.Outputs) != 66 {
				t.Fatalf("Generate(%s) result = %#v", database, generated)
			}
			for _, path := range []string{
				"src/demo_service/tenancy/service.py",
				"src/demo_service/tenancy/repository.py",
				"src/demo_service/migration/versions/000050_tenancy.py",
				"src/demo_service/migration/versions/000051_tenant_business.py",
				"tests/test_tenancy.py",
				"tests/test_tenancy_database.py",
			} {
				if outputContent(generated, path) == nil || outputOwner(generated, path) != tenancyOwner {
					t.Errorf("Generate(%s) did not produce tenancy-owned %s", database, path)
				}
			}
			assertContains := map[string][]string{
				"api/openapi.yaml":                     {"X-Organization-ID", "/api/v1/organizations"},
				"src/demo_service/tasks/repository.py": {"organization_id"},
				"src/demo_service/tasks/service.py":    {"actor.organization_id"},
				"web/admin/src/api/client.ts":          {"X-Organization-ID"},
				"web/admin/src/stores/session.ts":      {"loadOrganizations"},
			}
			for path, fragments := range assertContains {
				content := string(outputContent(generated, path))
				for _, fragment := range fragments {
					if !strings.Contains(content, fragment) {
						t.Errorf("%s does not contain %q", path, fragment)
					}
				}
			}
		})
	}
}

func TestGenerateOrganizationTenancyWithoutBusiness(t *testing.T) {
	t.Parallel()

	project := validProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{{
		Name: tenancyOwner, Version: tenancyVersion,
	}}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if generated.CapabilityLock[tenancyOwner] != tenancyVersion || len(generated.Outputs) != 37 {
		t.Fatalf("Generate() result = %#v", generated)
	}
	if outputContent(generated, "src/demo_service/migration/versions/000051_tenant_business.py") != nil {
		t.Fatal("Generate() produced a tenant business migration without a business entity")
	}
}

func TestTenantCompositionImportsRemainSortedAroundBusinessPackage(t *testing.T) {
	t.Parallel()

	for _, moduleName := range []string{"accounts", "users"} {
		project := validBusinessProject()
		project.Spec.Modules[0].Name = moduleName
		for index := range project.Spec.Modules[0].Permissions {
			suffix := strings.TrimPrefix(project.Spec.Modules[0].Permissions[index].Code, "tasks")
			project.Spec.Modules[0].Permissions[index].Code = moduleName + suffix
		}
		project.Spec.Capabilities = []spec.CapabilitySelection{{
			Name: tenancyOwner, Version: tenancyVersion,
		}}
		generated, err := New().Generate(context.Background(), project)
		if err != nil {
			t.Fatalf("Generate(%s) error = %v", moduleName, err)
		}
		mainSource := string(outputContent(generated, "src/demo_service/main.py"))
		businessImport := "from demo_service." + moduleName + ".http import"
		tenancyImport := "from demo_service.tenancy.http import"
		businessIndex := strings.Index(mainSource, businessImport)
		tenancyIndex := strings.Index(mainSource, tenancyImport)
		if businessIndex < 0 || tenancyIndex < 0 {
			t.Fatalf("generated imports are incomplete:\n%s", mainSource)
		}
		if (moduleName < "tenancy") != (businessIndex < tenancyIndex) {
			t.Fatalf("generated imports are not sorted for %s:\n%s", moduleName, mainSource)
		}
		firstIndex, secondIndex := businessIndex, tenancyIndex
		if firstIndex > secondIndex {
			firstIndex, secondIndex = secondIndex, firstIndex
		}
		if strings.Contains(mainSource[firstIndex:secondIndex], "\n\n") {
			t.Fatalf("generated imports contain an empty group for %s:\n%s", moduleName, mainSource)
		}
	}
}

func TestGenerateSharedAdministration(t *testing.T) {
	t.Parallel()

	project := validBusinessProject()
	project.Spec.Stack.AdminUI = "element-plus"
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if generated.CapabilityLock[adminui.Owner] != adminui.Version || len(generated.Outputs) != 57 {
		t.Fatalf("Generate() result = %#v", generated)
	}
	for _, path := range []string{
		"web/admin/package-lock.json",
		"web/admin/src/App.vue",
		"web/admin/src/types.ts",
		"web/admin/src/views/BusinessView.vue",
	} {
		if outputContent(generated, path) == nil {
			t.Errorf("Generate() did not produce %s", path)
		}
	}
}

func TestGenerateBlueprintCRUDForBothDatabases(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		project := validBusinessProject()
		project.Spec.Database.Engine = database
		generated, err := New().Generate(context.Background(), project)
		if err != nil {
			t.Fatalf("Generate(%s) error = %v", database, err)
		}
		if generated.CapabilityLock[crudOwner] != crudVersion || len(generated.Outputs) != 37 {
			t.Fatalf("Generate(%s) result = %#v", database, generated)
		}
		for _, path := range []string{
			"src/demo_service/tasks/models.py",
			"src/demo_service/tasks/repository.py",
			"src/demo_service/tasks/service.py",
			"src/demo_service/tasks/http.py",
			"src/demo_service/migration/versions/000002_tasks.py",
			"tests/test_tasks.py",
			"tests/test_tasks_database.py",
		} {
			if outputContent(generated, path) == nil {
				t.Errorf("Generate(%s) did not produce %s", database, path)
			}
		}
		var contract map[string]any
		if err := yaml.Unmarshal(outputContent(generated, "api/openapi.yaml"), &contract); err != nil {
			t.Fatalf("generated CRUD OpenAPI is invalid YAML: %v", err)
		}
	}
}

func TestGenerateRejectsUnsafeBusinessShapes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*spec.Project)
		want   string
	}{
		{name: "missing entity", mutate: func(project *spec.Project) {
			project.Spec.Modules[0].Entities = nil
		}, want: "exactly one entity"},
		{name: "workflow", mutate: func(project *spec.Project) {
			project.Spec.Modules[0].Workflows = []spec.Workflow{{Name: "review"}}
		}, want: "does not support workflows"},
		{name: "keyword", mutate: func(project *spec.Project) {
			project.Spec.Modules[0].Entities[0].Fields[0].Name = "class"
		}, want: "language keyword"},
		{name: "reserved", mutate: func(project *spec.Project) {
			project.Spec.Modules[0].Entities[0].Fields[0].Name = "version"
		}, want: "reserved"},
		{name: "permission", mutate: func(project *spec.Project) {
			project.Spec.Modules[0].Permissions = project.Spec.Modules[0].Permissions[:3]
		}, want: "must declare"},
		{name: "mysql unique text", mutate: func(project *spec.Project) {
			project.Spec.Database.Engine = "mysql"
			project.Spec.Modules[0].Entities[0].Fields[1].Unique = true
		}, want: "cannot use"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			project := validBusinessProject()
			test.mutate(&project)
			_, err := New().Generate(context.Background(), project)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("Generate() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestPythonIdentifierIsImportSafe(t *testing.T) {
	t.Parallel()

	tests := map[string]string{
		"demo-service": "demo_service",
		"class":        "service_class",
		"123-service":  "service_123_service",
		"---":          "generated_service",
	}
	for input, expected := range tests {
		if actual := pythonIdentifier(input); actual != expected {
			t.Errorf("pythonIdentifier(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func validProject() spec.Project {
	return spec.Project{
		Metadata: spec.Metadata{Name: "demo-service"},
		Spec: spec.ProjectSpec{
			Stack:    spec.StackSpec{Backend: "python", AdminUI: "none", Storefront: "none"},
			Database: spec.DatabaseSpec{Engine: "postgresql"},
			Auth:     spec.AuthSpec{Modes: []string{"session", "token"}},
		},
	}
}

func validBusinessProject() spec.Project {
	project := validProject()
	project.Spec.Modules = []spec.Module{{
		Name: "tasks",
		Entities: []spec.Entity{{Name: "task", Fields: []spec.Field{
			{Name: "title", Type: "string", Required: true, Unique: true},
			{Name: "description", Type: "text"},
			{Name: "done", Type: "bool", Required: true},
			{Name: "priority", Type: "int64", Required: true},
			{Name: "due_at", Type: "datetime"},
		}}},
		Permissions: []spec.Permission{
			{Code: "tasks:task:create"},
			{Code: "tasks:task:read"},
			{Code: "tasks:task:update"},
			{Code: "tasks:task:delete"},
		},
	}}
	return project
}

func outputContent(result generator.Result, path string) []byte {
	for _, output := range result.Outputs {
		if output.Path == path {
			return output.Content
		}
	}
	return nil
}

func outputOwner(result generator.Result, path string) string {
	for _, output := range result.Outputs {
		if output.Path == path {
			return output.Owner
		}
	}
	return ""
}
