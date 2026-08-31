package java

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/generator"
	"github.com/hkx5414375/scaffold-agent/internal/spec"
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
			if first.CapabilityLock[baseOwner] != baseVersion || len(first.Outputs) != 11 {
				t.Fatalf("Generate() result = %#v", first)
			}
			for _, path := range []string{
				"pom.xml",
				"api/openapi.yaml",
				"src/main/java/com/scaffold/generated/demoservice/Application.java",
				"src/main/java/com/scaffold/generated/demoservice/http/HealthController.java",
				"src/test/java/com/scaffold/generated/demoservice/architecture/ArchitectureTest.java",
			} {
				if outputContent(first, path) == nil {
					t.Errorf("Generate() did not produce %s", path)
				}
			}
			pom := string(outputContent(first, "pom.xml"))
			if !strings.Contains(pom, databases[database].DriverArtifactID) ||
				!strings.Contains(pom, databases[database].FlywayArtifactID) {
				t.Fatalf("generated pom does not select %s dependencies", database)
			}
		})
	}
}

func TestGenerateRejectsIncompleteFoundationSelections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*spec.Project)
		want   string
	}{
		{name: "frontend", mutate: func(project *spec.Project) { project.Spec.Stack.AdminUI = "element-plus" }, want: "does not generate frontends"},
		{name: "auth", mutate: func(project *spec.Project) { project.Spec.Auth.Modes = []string{"session"} }, want: "requires both session and token"},
		{name: "capability", mutate: func(project *spec.Project) {
			project.Spec.Capabilities = []spec.CapabilitySelection{{Name: "observability", Version: "0.1.0"}}
		}, want: "capability packs"},
		{name: "module", mutate: func(project *spec.Project) { project.Spec.Modules = []spec.Module{{Name: "tasks"}} }, want: "business modules"},
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

func validProject() spec.Project {
	return spec.Project{
		Metadata: spec.Metadata{Name: "demo-service"},
		Spec: spec.ProjectSpec{
			Stack:    spec.StackSpec{Backend: "java", AdminUI: "none", Storefront: "none"},
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
