package python

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/generator"
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
		{name: "admin", mutate: func(project *spec.Project) { project.Spec.Stack.AdminUI = "element-plus" }, want: "administration UI"},
		{name: "storefront", mutate: func(project *spec.Project) { project.Spec.Stack.Storefront = "nuxt" }, want: "storefront"},
		{name: "auth", mutate: func(project *spec.Project) { project.Spec.Auth.Modes = []string{"session"} }, want: "both session and token"},
		{name: "capability", mutate: func(project *spec.Project) {
			project.Spec.Capabilities = []spec.CapabilitySelection{{Name: "organization-tenancy", Version: "0.1.0"}}
		}, want: "does not support capability"},
		{name: "module", mutate: func(project *spec.Project) {
			project.Spec.Modules = []spec.Module{{Name: "tasks"}}
		}, want: "does not generate business"},
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

func outputContent(result generator.Result, path string) []byte {
	for _, output := range result.Outputs {
		if output.Path == path {
			return output.Content
		}
	}
	return nil
}
