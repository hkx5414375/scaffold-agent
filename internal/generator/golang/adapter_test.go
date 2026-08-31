package gogen

import (
	"bytes"
	"context"
	"go/format"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/hkx5414375/scaffold-agent/internal/spec"
)

func TestGenerateBaseServiceIsDeterministic(t *testing.T) {
	t.Parallel()

	project := validProject()
	first, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate(first) error = %v", err)
	}
	second, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate(second) error = %v", err)
	}
	if len(first.Outputs) != len(outputTemplates) || len(second.Outputs) != len(first.Outputs) {
		t.Fatalf("Generate() output counts = %d, %d, want %d", len(first.Outputs), len(second.Outputs), len(outputTemplates))
	}
	for index := range first.Outputs {
		if first.Outputs[index].Path != second.Outputs[index].Path || string(first.Outputs[index].Content) != string(second.Outputs[index].Content) {
			t.Fatalf("Generate() output %d is not deterministic", index)
		}
		if strings.HasSuffix(first.Outputs[index].Path, ".go") {
			formatted, err := format.Source(first.Outputs[index].Content)
			if err != nil {
				t.Fatalf("generated %q is invalid Go: %v", first.Outputs[index].Path, err)
			}
			if !bytes.Equal(formatted, first.Outputs[index].Content) {
				t.Fatalf("generated %q is not gofmt formatted", first.Outputs[index].Path)
			}
		}
	}
}

func TestGenerateRejectsUnsupportedBusinessModules(t *testing.T) {
	t.Parallel()

	project := spec.Project{
		Metadata: spec.Metadata{Name: "demo"},
		Spec: spec.ProjectSpec{
			Stack:    spec.StackSpec{Backend: "go"},
			Database: spec.DatabaseSpec{Engine: "postgresql"},
			Auth:     spec.AuthSpec{Modes: []string{"session", "token"}},
			Modules:  []spec.Module{{Name: "orders"}},
		},
	}
	if _, err := New().Generate(context.Background(), project); err == nil {
		t.Fatal("Generate() error = nil, want unsupported module error")
	}
}

func TestGenerateBusinessModuleIsFormatted(t *testing.T) {
	t.Parallel()

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
			{Code: "tasks:task:create"}, {Code: "tasks:task:read"},
			{Code: "tasks:task:update"}, {Code: "tasks:task:delete"},
		},
	}}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if generated.CapabilityLock[businessCapability] != businessVersion {
		t.Fatalf("Generate() capability lock = %#v", generated.CapabilityLock)
	}
	openAPIFound := false
	for _, output := range generated.Outputs {
		if output.Path == "api/openapi.yaml" {
			openAPIFound = true
			var document struct {
				OpenAPI string         `yaml:"openapi"`
				Paths   map[string]any `yaml:"paths"`
			}
			if err := yaml.Unmarshal(output.Content, &document); err != nil {
				t.Fatalf("generated OpenAPI is invalid YAML: %v\n%s", err, output.Content)
			}
			if document.OpenAPI != "3.1.0" || document.Paths["/api/v1/tasks"] == nil {
				t.Fatalf("generated OpenAPI document = %#v", document)
			}
		}
		if !strings.HasSuffix(output.Path, ".go") {
			continue
		}
		formatted, err := format.Source(output.Content)
		if err != nil {
			t.Fatalf("generated %q is invalid Go: %v", output.Path, err)
		}
		if !bytes.Equal(formatted, output.Content) {
			t.Fatalf("generated %q is not gofmt formatted\n--- got ---\n%s\n--- want ---\n%s", output.Path, output.Content, formatted)
		}
	}
	if !openAPIFound {
		t.Fatal("Generate() did not produce api/openapi.yaml")
	}
}

func TestGenerateRejectsUnsupportedStackSelections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*spec.Project)
	}{
		{name: "MySQL", mutate: func(project *spec.Project) { project.Spec.Database.Engine = "mysql" }},
		{name: "admin UI", mutate: func(project *spec.Project) { project.Spec.Stack.AdminUI = "element-plus" }},
		{name: "single auth mode", mutate: func(project *spec.Project) { project.Spec.Auth.Modes = []string{"session"} }},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			project := validProject()
			test.mutate(&project)
			if _, err := New().Generate(context.Background(), project); err == nil {
				t.Fatal("Generate() error = nil, want unsupported selection error")
			}
		})
	}
}

func validProject() spec.Project {
	return spec.Project{
		Metadata: spec.Metadata{Name: "demo"},
		Spec: spec.ProjectSpec{
			Stack:    spec.StackSpec{Backend: "go", AdminUI: "none", Storefront: "none"},
			Database: spec.DatabaseSpec{Engine: "postgresql"},
			Auth:     spec.AuthSpec{Modes: []string{"session", "token"}},
		},
	}
}
