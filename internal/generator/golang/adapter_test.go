package gogen

import (
	"bytes"
	"context"
	"go/format"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/hkx5414375/scaffold-agent/internal/generator"
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

func TestGenerateOrganizationTenancyCapability(t *testing.T) {
	t.Parallel()

	project := businessProject()
	project.Spec.Stack.AdminUI = "element-plus"
	project.Spec.Capabilities = []spec.CapabilitySelection{{Name: tenancyCapability, Version: tenancyVersion}}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if generated.CapabilityLock[tenancyCapability] != tenancyVersion {
		t.Fatalf("Generate() capability lock = %#v", generated.CapabilityLock)
	}
	wantPaths := map[string]bool{
		"internal/tenancy/service.go":                                          false,
		"internal/tenancy/httpapi/handler.go":                                  false,
		"internal/tenancy/postgres/store.go":                                   false,
		"internal/platform/migrate/migrations/000050_organization_tenancy.sql": false,
	}
	var businessMigration, openAPI string
	for _, output := range generated.Outputs {
		if _, exists := wantPaths[output.Path]; exists {
			wantPaths[output.Path] = true
		}
		switch output.Path {
		case "internal/platform/migrate/migrations/000100_tasks_task.sql":
			businessMigration = string(output.Content)
		case "api/openapi.yaml":
			openAPI = string(output.Content)
		}
	}
	for path, found := range wantPaths {
		if !found {
			t.Errorf("Generate() did not produce %s", path)
		}
	}
	if !strings.Contains(businessMigration, "organization_id") ||
		!strings.Contains(openAPI, "/api/v1/organizations:") ||
		!strings.Contains(openAPI, "X-Organization-ID") {
		t.Fatal("Generate() did not scope persistence and the HTTP contract by organization")
	}
	var document map[string]any
	if err := yaml.Unmarshal([]byte(openAPI), &document); err != nil {
		t.Fatalf("generated tenant OpenAPI is invalid YAML: %v", err)
	}
}

func TestGenerateOrganizationMemberCapabilityVersion(t *testing.T) {
	t.Parallel()

	project := businessProject()
	project.Spec.Stack.AdminUI = "element-plus"
	project.Spec.Capabilities = []spec.CapabilitySelection{{Name: tenancyCapability, Version: tenancyMembersVersion}}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if generated.CapabilityLock[tenancyCapability] != tenancyMembersVersion {
		t.Fatalf("Generate() capability lock = %#v", generated.CapabilityLock)
	}
	wantPaths := map[string]bool{
		"internal/tenancy/members.go":                                          false,
		"internal/tenancy/httpapi/members_handler.go":                          false,
		"internal/tenancy/postgres/members.go":                                 false,
		"internal/platform/migrate/migrations/000060_organization_members.sql": false,
		"web/admin/src/views/MembersView.vue":                                  false,
	}
	var openAPI string
	for _, output := range generated.Outputs {
		if _, exists := wantPaths[output.Path]; exists {
			wantPaths[output.Path] = true
		}
		if output.Path == "api/openapi.yaml" {
			openAPI = string(output.Content)
		}
	}
	for path, found := range wantPaths {
		if !found {
			t.Errorf("Generate() did not produce %s", path)
		}
	}
	if !strings.Contains(openAPI, "/api/v1/organization-invitations/accept:") ||
		!strings.Contains(openAPI, "tenancy:members:manage") {
		t.Fatal("Generate() did not expose the member administration contract")
	}
	var document map[string]any
	if err := yaml.Unmarshal([]byte(openAPI), &document); err != nil {
		t.Fatalf("generated member OpenAPI is invalid YAML: %v", err)
	}
}

func TestOrganizationMemberUpgradePreservesBaseMigration(t *testing.T) {
	t.Parallel()

	versionOne := businessProject()
	versionOne.Spec.Capabilities = []spec.CapabilitySelection{{Name: tenancyCapability, Version: tenancyVersion}}
	versionTwo := businessProject()
	versionTwo.Spec.Capabilities = []spec.CapabilitySelection{{Name: tenancyCapability, Version: tenancyMembersVersion}}
	first, err := New().Generate(context.Background(), versionOne)
	if err != nil {
		t.Fatalf("Generate(0.1.0) error = %v", err)
	}
	second, err := New().Generate(context.Background(), versionTwo)
	if err != nil {
		t.Fatalf("Generate(0.2.0) error = %v", err)
	}
	const baseMigration = "internal/platform/migrate/migrations/000050_organization_tenancy.sql"
	if !bytes.Equal(outputContent(first, baseMigration), outputContent(second, baseMigration)) {
		t.Fatal("organization-tenancy 0.2.0 rewrites the already-applied 0.1.0 migration")
	}
	const memberMigration = "internal/platform/migrate/migrations/000060_organization_members.sql"
	if outputContent(first, memberMigration) != nil || outputContent(second, memberMigration) == nil {
		t.Fatal("organization member migration is not isolated to 0.2.0")
	}
}

func TestGenerateOrganizationLifecycleCapabilityVersion(t *testing.T) {
	t.Parallel()

	project := businessProject()
	project.Spec.Stack.AdminUI = "element-plus"
	project.Spec.Capabilities = []spec.CapabilitySelection{{Name: tenancyCapability, Version: tenancyLifecycleVersion}}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if generated.CapabilityLock[tenancyCapability] != tenancyLifecycleVersion {
		t.Fatalf("Generate() capability lock = %#v", generated.CapabilityLock)
	}
	wantPaths := map[string]bool{
		"internal/tenancy/lifecycle.go":                                          false,
		"internal/tenancy/httpapi/lifecycle_handler.go":                          false,
		"internal/tenancy/postgres/lifecycle.go":                                 false,
		"internal/platform/migrate/migrations/000070_organization_lifecycle.sql": false,
		"web/admin/src/views/OrganizationSettingsView.vue":                       false,
	}
	var openAPI string
	for _, output := range generated.Outputs {
		if _, exists := wantPaths[output.Path]; exists {
			wantPaths[output.Path] = true
		}
		if output.Path == "api/openapi.yaml" {
			openAPI = string(output.Content)
		}
	}
	for path, found := range wantPaths {
		if !found {
			t.Errorf("Generate() did not produce %s", path)
		}
	}
	if !strings.Contains(openAPI, "/api/v1/organizations/{organizationID}/ownership-transfers:") ||
		!strings.Contains(openAPI, "deactivateOrganization") ||
		!strings.Contains(openAPI, "is_owner") {
		t.Fatal("Generate() did not expose the organization lifecycle contract")
	}
	var document map[string]any
	if err := yaml.Unmarshal([]byte(openAPI), &document); err != nil {
		t.Fatalf("generated lifecycle OpenAPI is invalid YAML: %v", err)
	}
}

func TestOrganizationLifecycleUpgradePreservesEarlierMigrations(t *testing.T) {
	t.Parallel()

	versionTwo := businessProject()
	versionTwo.Spec.Capabilities = []spec.CapabilitySelection{{Name: tenancyCapability, Version: tenancyMembersVersion}}
	versionThree := businessProject()
	versionThree.Spec.Capabilities = []spec.CapabilitySelection{{Name: tenancyCapability, Version: tenancyLifecycleVersion}}
	second, err := New().Generate(context.Background(), versionTwo)
	if err != nil {
		t.Fatalf("Generate(0.2.0) error = %v", err)
	}
	third, err := New().Generate(context.Background(), versionThree)
	if err != nil {
		t.Fatalf("Generate(0.3.0) error = %v", err)
	}
	for _, path := range []string{
		"internal/platform/migrate/migrations/000050_organization_tenancy.sql",
		"internal/platform/migrate/migrations/000060_organization_members.sql",
	} {
		if !bytes.Equal(outputContent(second, path), outputContent(third, path)) {
			t.Fatalf("organization-tenancy 0.3.0 rewrites already-applied migration %s", path)
		}
	}
	const lifecycleMigration = "internal/platform/migrate/migrations/000070_organization_lifecycle.sql"
	if outputContent(second, lifecycleMigration) != nil || outputContent(third, lifecycleMigration) == nil {
		t.Fatal("organization lifecycle migration is not isolated to 0.3.0")
	}
}

func TestGenerateBackgroundJobsCapability(t *testing.T) {
	t.Parallel()

	project := businessProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyCapability, Version: tenancyMembersVersion},
		{Name: jobsCapability, Version: jobsVersion},
	}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if generated.CapabilityLock[jobsCapability] != jobsVersion {
		t.Fatalf("Generate() capability lock = %#v", generated.CapabilityLock)
	}
	wantPaths := map[string]bool{
		"cmd/worker/main.go":                                              false,
		"internal/jobhandlers/registry.go":                                false,
		"internal/jobs/jobs.go":                                           false,
		"internal/jobs/worker.go":                                         false,
		"internal/jobs/postgres/store.go":                                 false,
		"internal/platform/migrate/migrations/000200_background_jobs.sql": false,
	}
	for _, output := range generated.Outputs {
		if _, exists := wantPaths[output.Path]; exists {
			wantPaths[output.Path] = true
		}
	}
	for path, found := range wantPaths {
		if !found {
			t.Errorf("Generate() did not produce %s", path)
		}
	}
}

func TestGenerateBackgroundJobsWithoutTenancy(t *testing.T) {
	t.Parallel()

	project := businessProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{{Name: jobsCapability, Version: jobsVersion}}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	migration := string(outputContent(generated, "internal/platform/migrate/migrations/000200_background_jobs.sql"))
	jobService := string(outputContent(generated, "internal/jobs/jobs.go"))
	if strings.Contains(migration, "references organizations") || !strings.Contains(jobService, `organizationID = ""`) {
		t.Fatal("Generate() did not use an explicit global background-job scope")
	}
}

func TestGenerateNotificationsResolvesBackgroundJobs(t *testing.T) {
	t.Parallel()

	project := businessProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyCapability, Version: tenancyLifecycleVersion},
		{Name: notificationsCapability, Version: notificationsVersion},
	}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if generated.CapabilityLock[notificationsCapability] != notificationsVersion ||
		generated.CapabilityLock[jobsCapability] != jobsVersion {
		t.Fatalf("Generate() capability lock = %#v", generated.CapabilityLock)
	}
	wantPaths := map[string]bool{
		"cmd/worker/main.go":                      false,
		"internal/jobs/jobs.go":                   false,
		"internal/jobhandlers/notifications.go":   false,
		"internal/notifications/notifications.go": false,
		"internal/notifications/smtp/sender.go":   false,
		"internal/jobs/postgres/store.go":         false,
	}
	for _, output := range generated.Outputs {
		if _, exists := wantPaths[output.Path]; exists {
			wantPaths[output.Path] = true
		}
	}
	for path, found := range wantPaths {
		if !found {
			t.Errorf("Generate() did not produce %s", path)
		}
	}
}

func TestGenerateTenantFileAssetsCapability(t *testing.T) {
	t.Parallel()

	project := businessProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyCapability, Version: tenancyLifecycleVersion},
		{Name: filesCapability, Version: filesVersion},
	}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if generated.CapabilityLock[filesCapability] != filesVersion {
		t.Fatalf("Generate() capability lock = %#v", generated.CapabilityLock)
	}
	wantPaths := map[string]bool{
		"internal/files/files.go":                                     false,
		"internal/files/httpapi/handler.go":                           false,
		"internal/files/local/store.go":                               false,
		"internal/files/postgres/store.go":                            false,
		"internal/platform/migrate/migrations/000210_file_assets.sql": false,
	}
	for _, output := range generated.Outputs {
		if _, exists := wantPaths[output.Path]; exists {
			wantPaths[output.Path] = true
		}
	}
	for path, found := range wantPaths {
		if !found {
			t.Errorf("Generate() did not produce %s", path)
		}
	}
	migration := string(outputContent(generated, "internal/platform/migrate/migrations/000210_file_assets.sql"))
	openAPI := string(outputContent(generated, "api/openapi.yaml"))
	if !strings.Contains(migration, "references organizations") ||
		!strings.Contains(openAPI, "/api/v1/files/{id}/content:") ||
		!strings.Contains(openAPI, "files:create") {
		t.Fatal("Generate() did not produce the tenant file persistence and HTTP contracts")
	}
	var document map[string]any
	if err := yaml.Unmarshal([]byte(openAPI), &document); err != nil {
		t.Fatalf("generated file OpenAPI is invalid YAML: %v", err)
	}
}

func TestGenerateGlobalMySQLFileAssetsCapability(t *testing.T) {
	t.Parallel()

	project := businessProject()
	project.Spec.Database.Engine = "mysql"
	project.Spec.Capabilities = []spec.CapabilitySelection{{Name: filesCapability, Version: filesVersion}}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	migration := string(outputContent(generated, "internal/platform/migrate/migrations/000210_file_assets.sql"))
	mainSource := string(outputContent(generated, "cmd/server/main.go"))
	if strings.Contains(migration, "references organizations") ||
		!strings.Contains(mainSource, `os.Getenv("FILE_STORAGE_ROOT")`) {
		t.Fatal("Generate() did not produce an explicit global MySQL file scope")
	}
}

func TestGenerateTenantApplicationCacheCapability(t *testing.T) {
	t.Parallel()

	project := businessProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyCapability, Version: tenancyLifecycleVersion},
		{Name: cacheCapability, Version: cacheVersion},
	}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if generated.CapabilityLock[cacheCapability] != cacheVersion {
		t.Fatalf("Generate() capability lock = %#v", generated.CapabilityLock)
	}
	for _, path := range []string{
		"internal/cache/cache.go",
		"internal/cache/cache_test.go",
		"internal/cache/postgres/store.go",
		"internal/platform/migrate/migrations/000220_application_cache.sql",
	} {
		if outputContent(generated, path) == nil {
			t.Errorf("Generate() did not produce %s", path)
		}
	}
	migration := string(outputContent(generated, "internal/platform/migrate/migrations/000220_application_cache.sql"))
	if !strings.Contains(migration, "references organizations") {
		t.Fatal("Generate() did not scope application cache persistence by organization")
	}
}

func TestGenerateGlobalMySQLApplicationCacheCapability(t *testing.T) {
	t.Parallel()

	project := businessProject()
	project.Spec.Database.Engine = "mysql"
	project.Spec.Capabilities = []spec.CapabilitySelection{{Name: cacheCapability, Version: cacheVersion}}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	migration := string(outputContent(generated, "internal/platform/migrate/migrations/000220_application_cache.sql"))
	service := string(outputContent(generated, "internal/cache/cache.go"))
	if strings.Contains(migration, "references organizations") || !strings.Contains(service, `return "global", key, nil`) {
		t.Fatal("Generate() did not produce an explicit global application cache scope")
	}
}

func TestGenerateRejectsCapabilityConfiguration(t *testing.T) {
	t.Parallel()

	project := validProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{{
		Name: tenancyCapability, Version: tenancyVersion, Config: map[string]any{"header": "X-Tenant"},
	}}
	if _, err := New().Generate(context.Background(), project); err == nil || !strings.Contains(err.Error(), "does not accept configuration") {
		t.Fatalf("Generate() error = %v, want unsupported configuration diagnostic", err)
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

func outputContent(generated generator.Result, path string) []byte {
	for _, output := range generated.Outputs {
		if output.Path == path {
			return output.Content
		}
	}
	return nil
}
