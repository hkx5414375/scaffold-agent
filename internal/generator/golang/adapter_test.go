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
	wantOutputs := len(outputTemplates) + len(databaseTemplates["postgresql"].Outputs)
	if len(first.Outputs) != wantOutputs || len(second.Outputs) != len(first.Outputs) {
		t.Fatalf("Generate() output counts = %d, %d, want %d", len(first.Outputs), len(second.Outputs), wantOutputs)
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

	project := businessProject()
	project.Spec.Stack.AdminUI = "element-plus"
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if generated.CapabilityLock[businessCapability] != businessVersion {
		t.Fatalf("Generate() capability lock = %#v", generated.CapabilityLock)
	}
	if generated.CapabilityLock[adminCapability] != adminVersion {
		t.Fatalf("Generate() administration capability lock = %#v", generated.CapabilityLock)
	}
	openAPIFound := false
	var entitySource, adminTypes, businessView, integrationTest, openAPISource string
	for _, output := range generated.Outputs {
		switch output.Path {
		case "internal/tasks/entity.go":
			entitySource = string(output.Content)
		case "web/admin/src/types.ts":
			adminTypes = string(output.Content)
		case "web/admin/src/views/BusinessView.vue":
			businessView = string(output.Content)
		case "internal/integration/postgres_test.go":
			integrationTest = string(output.Content)
		}
		if output.Path == "api/openapi.yaml" {
			openAPIFound = true
			openAPISource = string(output.Content)
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
	for name, source := range map[string]string{
		"Go entity version":           entitySource,
		"administration types":        adminTypes,
		"administration view":         businessView,
		"PostgreSQL integration test": integrationTest,
		"OpenAPI contract":            openAPISource,
	} {
		if source == "" {
			t.Fatalf("Generate() did not produce %s", name)
		}
	}
	if !strings.Contains(entitySource, `json:"version,string"`) ||
		!strings.Contains(adminTypes, "version: string;") ||
		!strings.Contains(businessView, "version: editing.value.version") ||
		!strings.Contains(openAPISource, `pattern: "^[1-9][0-9]*$"`) {
		t.Fatal("Generate() did not preserve int64 optimistic-lock versions as decimal strings")
	}
}

func TestGenerateMySQLBusinessModuleIsFormatted(t *testing.T) {
	t.Parallel()

	project := businessProject()
	project.Spec.Database.Engine = "mysql"
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	wantPaths := map[string]bool{
		"internal/identity/mysql/store.go":   false,
		"internal/tasks/mysql/store.go":      false,
		"internal/integration/mysql_test.go": false,
		"internal/platform/mysql/pool.go":    false,
	}
	var businessMigration, businessStore string
	for _, output := range generated.Outputs {
		if _, exists := wantPaths[output.Path]; exists {
			wantPaths[output.Path] = true
		}
		switch output.Path {
		case "internal/platform/migrate/migrations/000100_tasks_task.sql":
			businessMigration = string(output.Content)
		case "internal/tasks/mysql/store.go":
			businessStore = string(output.Content)
		}
		if strings.HasSuffix(output.Path, ".go") {
			formatted, err := format.Source(output.Content)
			if err != nil {
				t.Fatalf("generated %q is invalid Go: %v", output.Path, err)
			}
			if !bytes.Equal(formatted, output.Content) {
				t.Fatalf("generated %q is not gofmt formatted", output.Path)
			}
		}
	}
	for path, found := range wantPaths {
		if !found {
			t.Errorf("Generate() did not produce %s", path)
		}
	}
	if !strings.Contains(businessMigration, "`due_at` datetime(6)") ||
		!strings.Contains(businessStore, "`tasks_task`") {
		t.Fatal("Generate() did not render MySQL-specific types and quoted identifiers")
	}
}

func TestGenerateRejectsUnsupportedStackSelections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*spec.Project)
	}{
		{name: "unsupported database", mutate: func(project *spec.Project) { project.Spec.Database.Engine = "sqlite" }},
		{name: "storefront", mutate: func(project *spec.Project) { project.Spec.Stack.Storefront = "nuxt" }},
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

func TestGenerateRejectsMySQLUniqueText(t *testing.T) {
	t.Parallel()

	project := businessProject()
	project.Spec.Database.Engine = "mysql"
	project.Spec.Modules[0].Entities[0].Fields[1].Unique = true
	if _, err := New().Generate(context.Background(), project); err == nil || !strings.Contains(err.Error(), "portable unique constraint") {
		t.Fatalf("Generate() error = %v, want MySQL text uniqueness diagnostic", err)
	}
}

func TestQuoteSQLIdentifierUsesSelectedDialect(t *testing.T) {
	t.Parallel()

	if actual := quoteSQLIdentifier("postgresql", `order"item`); actual != `"order""item"` {
		t.Fatalf("quoteSQLIdentifier(PostgreSQL) = %q", actual)
	}
	if actual := quoteSQLIdentifier("mysql", "order`item"); actual != "`order``item`" {
		t.Fatalf("quoteSQLIdentifier(MySQL) = %q", actual)
	}
	if actual := quoteGoSQLIdentifier("mysql", "order"); actual != "`+\"`order`\"+`" {
		t.Fatalf("quoteGoSQLIdentifier(MySQL) = %q", actual)
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

func businessProject() spec.Project {
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
	return project
}
