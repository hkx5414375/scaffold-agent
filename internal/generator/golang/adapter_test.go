package gogen

import (
	"bytes"
	"context"
	"go/format"
	"strings"
	"testing"

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
