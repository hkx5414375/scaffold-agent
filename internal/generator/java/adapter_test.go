package java

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
			if first.CapabilityLock[baseOwner] != baseVersion || len(first.Outputs) != 31 {
				t.Fatalf("Generate() result = %#v", first)
			}
			for _, path := range []string{
				"pom.xml",
				"api/openapi.yaml",
				"src/main/java/com/scaffold/generated/demoservice/Application.java",
				"src/main/java/com/scaffold/generated/demoservice/http/HealthController.java",
				"src/main/java/com/scaffold/generated/demoservice/http/AuthController.java",
				"src/main/java/com/scaffold/generated/demoservice/identity/IdentityService.java",
				"src/main/resources/db/migration/V000001__identity.sql",
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
		{name: "module", mutate: func(project *spec.Project) { project.Spec.Modules = []spec.Module{{Name: "tasks"}} }, want: "exactly one entity"},
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

func TestGeneratePortableBusinessCRUD(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			project := validProject()
			project.Spec.Database.Engine = database
			project.Spec.Modules = []spec.Module{businessModule()}
			result, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if result.CapabilityLock[baseOwner] != baseVersion ||
				result.CapabilityLock[businessOwner] != businessVersion || len(result.Outputs) != 40 {
				t.Fatalf("Generate() result = %#v", result)
			}
			for _, path := range []string{
				"src/main/java/com/scaffold/generated/demoservice/tasks/Task.java",
				"src/main/java/com/scaffold/generated/demoservice/tasks/TaskController.java",
				"src/main/java/com/scaffold/generated/demoservice/tasks/JdbcTaskRepository.java",
				"src/test/java/com/scaffold/generated/demoservice/tasks/TaskServiceTest.java",
				"src/test/java/com/scaffold/generated/demoservice/tasks/TaskDatabaseIntegrationTest.java",
				"src/main/resources/db/migration/V000100__tasks_task.sql",
			} {
				if outputContent(result, path) == nil {
					t.Errorf("Generate() did not produce %s", path)
				}
			}
			if owner := outputOwner(result,
				"src/main/java/com/scaffold/generated/demoservice/tasks/TaskService.java"); owner != businessOwner {
				t.Fatalf("generated CRUD owner = %q, want %q", owner, businessOwner)
			}
			migration := string(outputContent(result,
				"src/main/resources/db/migration/V000100__tasks_task.sql"))
			quote := `"tasks_task"`
			if database == "mysql" {
				quote = "`tasks_task`"
			}
			if !strings.Contains(migration, quote) ||
				!strings.Contains(migration, "tasks:task:create") {
				t.Fatalf("generated migration does not contain portable CRUD contract:\n%s", migration)
			}
			openAPI := outputContent(result, "api/openapi.yaml")
			var contract map[string]any
			if err := yaml.Unmarshal(openAPI, &contract); err != nil {
				t.Fatalf("generated OpenAPI is not valid YAML: %v\n%s", err, openAPI)
			}
			if !strings.Contains(string(openAPI), "/api/v1/tasks/{id}:") ||
				!strings.Contains(string(openAPI), "tasks:task:update") {
				t.Fatalf("generated OpenAPI does not contain CRUD routes:\n%s", openAPI)
			}
		})
	}
}

func TestGenerateRejectsMySQLUniqueTextField(t *testing.T) {
	t.Parallel()

	project := validProject()
	project.Spec.Database.Engine = "mysql"
	module := businessModule()
	module.Entities[0].Fields[1].Unique = true
	project.Spec.Modules = []spec.Module{module}
	_, err := New().Generate(context.Background(), project)
	if err == nil || !strings.Contains(err.Error(), "cannot use the portable unique constraint") {
		t.Fatalf("Generate() error = %v", err)
	}
}

func businessModule() spec.Module {
	return spec.Module{
		Name: "tasks",
		Entities: []spec.Entity{{
			Name: "task",
			Fields: []spec.Field{
				{Name: "title", Type: "string", Required: true, Unique: true},
				{Name: "description", Type: "text"},
				{Name: "completed", Type: "bool", Required: true},
				{Name: "priority", Type: "int64"},
				{Name: "due_at", Type: "datetime"},
			},
		}},
		Permissions: []spec.Permission{
			{Code: "tasks:task:create"},
			{Code: "tasks:task:read"},
			{Code: "tasks:task:update"},
			{Code: "tasks:task:delete"},
		},
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

func outputOwner(result generator.Result, path string) string {
	for _, output := range result.Outputs {
		if output.Path == path {
			return output.Owner
		}
	}
	return ""
}
