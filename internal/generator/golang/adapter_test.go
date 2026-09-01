package gogen

import (
	"bytes"
	"context"
	"go/format"
	"strings"
	"testing"

	"go.yaml.in/yaml/v3"

	"github.com/hkx5414375/scaffold-agent/internal/generator"
	storefrontui "github.com/hkx5414375/scaffold-agent/internal/generator/storefront"
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

func TestGenerateSharedNuxtStorefrontFoundation(t *testing.T) {
	t.Parallel()

	project := validProject()
	project.Metadata.DisplayName = "Demo Store"
	project.Spec.Stack.Storefront = "nuxt"
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	wantOutputs := len(outputTemplates) + len(databaseTemplates["postgresql"].Outputs) + len(storefrontui.BaseTemplates)
	if len(generated.Outputs) != wantOutputs ||
		generated.CapabilityLock[storefrontCapability] != storefrontVersion {
		t.Fatalf("Generate() result = %#v", generated)
	}
	for _, path := range []string{
		"web/storefront/package-lock.json",
		"web/storefront/nuxt.config.ts",
		"web/storefront/app/pages/index.vue",
		"web/storefront/server/api/storefront/status.get.ts",
	} {
		if outputContent(generated, path) == nil || outputOwner(generated, path) != storefrontCapability {
			t.Errorf("Generate() storefront output %s is missing or has the wrong owner", path)
		}
	}
	if !strings.Contains(string(outputContent(generated, "web/storefront/app/pages/index.vue")), "Demo Store") {
		t.Fatal("generated storefront does not use the project display name")
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

func TestGenerateJobAdministrationResolvesBackgroundJobs(t *testing.T) {
	t.Parallel()

	project := businessProject()
	project.Spec.Stack.AdminUI = "element-plus"
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyCapability, Version: tenancyLifecycleVersion},
		{Name: jobAdminCapability, Version: jobAdminVersion},
	}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if generated.CapabilityLock[jobAdminCapability] != jobAdminVersion || generated.CapabilityLock[jobsCapability] != jobsVersion {
		t.Fatalf("Generate() capability lock = %#v", generated.CapabilityLock)
	}
	for _, path := range []string{
		"cmd/worker/main.go",
		"internal/jobadmin/service.go",
		"internal/jobadmin/httpapi/handler.go",
		"internal/jobs/postgres/admin.go",
		"internal/platform/migrate/migrations/000230_job_administration.sql",
		"web/admin/src/views/JobsView.vue",
	} {
		if outputContent(generated, path) == nil {
			t.Errorf("Generate() did not produce %s", path)
		}
	}
	openAPI := string(outputContent(generated, "api/openapi.yaml"))
	if !strings.Contains(openAPI, "/api/v1/jobs/{id}/retry:") || strings.Contains(openAPI, "payload:") {
		t.Fatal("Generate() did not expose the payload-free job administration contract")
	}
	var document map[string]any
	if err := yaml.Unmarshal([]byte(openAPI), &document); err != nil {
		t.Fatalf("generated job administration OpenAPI is invalid YAML: %v", err)
	}
}

func TestGenerateObservabilityCapability(t *testing.T) {
	t.Parallel()

	project := businessProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{{Name: observabilityCapability, Version: observabilityVersion}}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if generated.CapabilityLock[observabilityCapability] != observabilityVersion {
		t.Fatalf("Generate() capability lock = %#v", generated.CapabilityLock)
	}
	for _, path := range []string{
		"internal/platform/observability/observability.go",
		"internal/platform/observability/observability_test.go",
	} {
		if outputContent(generated, path) == nil {
			t.Errorf("Generate() did not produce %s", path)
		}
	}
	mainSource := string(outputContent(generated, "cmd/server/main.go"))
	openAPI := string(outputContent(generated, "api/openapi.yaml"))
	if !strings.Contains(mainSource, `mux.HandleFunc("GET /readyz", readiness)`) ||
		!strings.Contains(openAPI, "/metrics:") || !strings.Contains(openAPI, "/readyz:") {
		t.Fatal("Generate() did not wire observability routes and readiness")
	}
	var document map[string]any
	if err := yaml.Unmarshal([]byte(openAPI), &document); err != nil {
		t.Fatalf("generated observability OpenAPI is invalid YAML: %v", err)
	}
}

func TestGenerateCSVImportExportCapability(t *testing.T) {
	t.Parallel()

	project := businessProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{{Name: csvTransferCapability, Version: csvTransferVersion}}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if generated.CapabilityLock[csvTransferCapability] != csvTransferVersion {
		t.Fatalf("Generate() capability lock = %#v", generated.CapabilityLock)
	}
	for _, path := range []string{
		"internal/tasks/transfer/service.go",
		"internal/tasks/transfer/service_test.go",
		"internal/tasks/transfer/httpapi/handler.go",
		"internal/tasks/transfer/httpapi/handler_test.go",
		"internal/tasks/transfer/postgres/store.go",
		"internal/platform/migrate/migrations/000240_csv_import_export.sql",
	} {
		if outputContent(generated, path) == nil {
			t.Errorf("Generate() did not produce %s", path)
		}
	}
	mainSource := string(outputContent(generated, "cmd/server/main.go"))
	openAPI := string(outputContent(generated, "api/openapi.yaml"))
	if !strings.Contains(mainSource, "tasksTransferAPI.Register") ||
		!strings.Contains(openAPI, "/api/v1/tasks/import:") ||
		!strings.Contains(openAPI, "/api/v1/tasks/export:") {
		t.Fatal("Generate() did not wire CSV transfer routes")
	}
	var document map[string]any
	if err := yaml.Unmarshal([]byte(openAPI), &document); err != nil {
		t.Fatalf("generated CSV transfer OpenAPI is invalid YAML: %v", err)
	}
}

func TestGenerateCSVImportExportRequiresBusinessEntity(t *testing.T) {
	t.Parallel()

	project := businessProject()
	project.Spec.Modules = nil
	project.Spec.Capabilities = []spec.CapabilitySelection{{Name: csvTransferCapability, Version: csvTransferVersion}}
	if _, err := New().Generate(context.Background(), project); err == nil || !strings.Contains(err.Error(), "requires one generated business entity") {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestGenerateApprovalWorkflowsCapability(t *testing.T) {
	t.Parallel()

	project := businessProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{{Name: approvalsCapability, Version: approvalsVersion}}
	project.Spec.Modules[0].Workflows = []spec.Workflow{{
		Name:   "approval",
		States: []string{"pending", "approved", "rejected", "cancelled"},
	}}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if generated.CapabilityLock[approvalsCapability] != approvalsVersion {
		t.Fatalf("Generate() capability lock = %#v", generated.CapabilityLock)
	}
	for _, path := range []string{
		"internal/approvals/service.go",
		"internal/approvals/service_test.go",
		"internal/approvals/httpapi/handler.go",
		"internal/approvals/httpapi/handler_test.go",
		"internal/approvals/postgres/store.go",
		"internal/platform/migrate/migrations/000250_approval_workflows.sql",
	} {
		if outputContent(generated, path) == nil {
			t.Errorf("Generate() did not produce %s", path)
		}
	}
	mainSource := string(outputContent(generated, "cmd/server/main.go"))
	openAPI := string(outputContent(generated, "api/openapi.yaml"))
	if !strings.Contains(mainSource, "approvalAPI.Register") ||
		!strings.Contains(openAPI, "/api/v1/approvals:") ||
		!strings.Contains(openAPI, "/api/v1/approvals/{id}/approve:") {
		t.Fatal("Generate() did not wire approval workflow routes")
	}
	var document map[string]any
	if err := yaml.Unmarshal([]byte(openAPI), &document); err != nil {
		t.Fatalf("generated approval workflow OpenAPI is invalid YAML: %v", err)
	}
}

func TestGenerateApprovalWorkflowsRequiresBusinessEntityAndExactWorkflow(t *testing.T) {
	t.Parallel()

	withoutBusiness := businessProject()
	withoutBusiness.Spec.Modules = nil
	withoutBusiness.Spec.Capabilities = []spec.CapabilitySelection{{Name: approvalsCapability, Version: approvalsVersion}}
	if _, err := New().Generate(context.Background(), withoutBusiness); err == nil ||
		!strings.Contains(err.Error(), "requires one generated business entity") {
		t.Fatalf("Generate() without business error = %v", err)
	}

	withoutWorkflow := businessProject()
	withoutWorkflow.Spec.Capabilities = []spec.CapabilitySelection{{Name: approvalsCapability, Version: approvalsVersion}}
	if _, err := New().Generate(context.Background(), withoutWorkflow); err == nil ||
		!strings.Contains(err.Error(), "requires workflow approval") {
		t.Fatalf("Generate() without workflow error = %v", err)
	}

	wrongStates := businessProject()
	wrongStates.Spec.Capabilities = []spec.CapabilitySelection{{Name: approvalsCapability, Version: approvalsVersion}}
	wrongStates.Spec.Modules[0].Workflows = []spec.Workflow{{Name: "approval", States: []string{"pending", "approved"}}}
	if _, err := New().Generate(context.Background(), wrongStates); err == nil ||
		!strings.Contains(err.Error(), "pending, approved, rejected, cancelled") {
		t.Fatalf("Generate() wrong workflow error = %v", err)
	}
}

func TestGenerateCommerceCatalogAcrossDatabasesAndSurfaces(t *testing.T) {
	t.Parallel()

	for _, engine := range []string{"postgresql", "mysql"} {
		engine := engine
		t.Run(engine, func(t *testing.T) {
			t.Parallel()
			project := validProject()
			project.Spec.Database.Engine = engine
			project.Spec.Stack.AdminUI = "element-plus"
			project.Spec.Stack.Storefront = "nuxt"
			project.Spec.Capabilities = []spec.CapabilitySelection{
				{Name: tenancyCapability, Version: tenancyLifecycleVersion},
				{Name: catalogCapability, Version: catalogVersion},
			}
			generated, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if generated.CapabilityLock[catalogCapability] != catalogVersion {
				t.Fatalf("Generate() capability lock = %#v", generated.CapabilityLock)
			}
			storePath := "internal/catalog/" + databaseTemplates[engine].Data.PackageName + "/store.go"
			for _, path := range []string{
				"internal/catalog/catalog.go",
				"internal/catalog/catalog_test.go",
				"internal/catalog/httpapi/handler.go",
				"internal/catalog/httpapi/handler_test.go",
				storePath,
				"internal/platform/migrate/migrations/000260_commerce_catalog.sql",
				"web/admin/src/views/CatalogView.vue",
				"web/storefront/app/pages/products/index.vue",
				"web/storefront/app/pages/products/[id].vue",
				"web/storefront/server/api/storefront/products.get.ts",
			} {
				if outputContent(generated, path) == nil || outputOwner(generated, path) != catalogCapability {
					t.Errorf("Generate() catalog output %s is missing or has the wrong owner", path)
				}
			}
			for _, output := range generated.Outputs {
				if !strings.HasSuffix(output.Path, ".go") {
					continue
				}
				formatted, err := format.Source(output.Content)
				if err != nil {
					t.Fatalf("generated %q is invalid Go: %v\n%s", output.Path, err, output.Content)
				}
				if !bytes.Equal(formatted, output.Content) {
					t.Fatalf("generated %q is not gofmt formatted", output.Path)
				}
			}
			mainSource := string(outputContent(generated, "cmd/server/main.go"))
			if !strings.Contains(mainSource, "catalogAPI.Register") {
				t.Fatal("Generate() did not wire catalog routes")
			}
			openAPI := outputContent(generated, "api/openapi.yaml")
			var document struct {
				Paths map[string]any `yaml:"paths"`
			}
			if err := yaml.Unmarshal(openAPI, &document); err != nil {
				t.Fatalf("generated catalog OpenAPI is invalid YAML: %v\n%s", err, openAPI)
			}
			for _, path := range []string{
				"/api/v1/catalog/products", "/api/v1/catalog/products/{id}/publish",
				"/api/v1/storefront/products", "/api/v1/storefront/products/{id}",
			} {
				if document.Paths[path] == nil {
					t.Errorf("generated catalog OpenAPI is missing %s", path)
				}
			}
			storefrontConfig := string(outputContent(generated, "web/storefront/nuxt.config.ts"))
			if !strings.Contains(storefrontConfig, "SCAFFOLD_ORGANIZATION_ID") {
				t.Fatal("Generate() did not keep tenant storefront scope server-only")
			}
		})
	}
}

func TestCommerceCatalogRequiresLifecycleTenancyWhenScoped(t *testing.T) {
	t.Parallel()

	project := validProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyCapability, Version: tenancyMembersVersion},
		{Name: catalogCapability, Version: catalogVersion},
	}
	if _, err := New().Generate(context.Background(), project); err == nil ||
		!strings.Contains(err.Error(), "requires organization-tenancy 0.3.0") {
		t.Fatalf("Generate() error = %v, want lifecycle tenancy requirement", err)
	}
}

func TestGenerateCustomerAccountsAcrossDatabases(t *testing.T) {
	t.Parallel()

	for _, engine := range []string{"postgresql", "mysql"} {
		engine := engine
		t.Run(engine, func(t *testing.T) {
			t.Parallel()
			project := validProject()
			project.Spec.Database.Engine = engine
			project.Spec.Stack.AdminUI = "element-plus"
			project.Spec.Stack.Storefront = "nuxt"
			project.Spec.Capabilities = []spec.CapabilitySelection{
				{Name: tenancyCapability, Version: tenancyLifecycleVersion},
				{Name: customerCapability, Version: customerVersion},
			}
			generated, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if generated.CapabilityLock[customerCapability] != customerVersion {
				t.Fatalf("Generate() capability lock = %#v", generated.CapabilityLock)
			}
			storePath := "internal/customeraccounts/" + databaseTemplates[engine].Data.PackageName + "/store.go"
			for _, path := range []string{
				"internal/customeraccounts/accounts.go",
				"internal/customeraccounts/accounts_test.go",
				"internal/customeraccounts/httpapi/handler.go",
				"internal/customeraccounts/httpapi/handler_test.go",
				storePath,
				"internal/platform/migrate/migrations/000270_customer_accounts.sql",
				"web/admin/src/views/CustomerAccountsView.vue",
				"web/storefront/app/pages/account/index.vue",
				"web/storefront/app/pages/account/login.vue",
				"web/storefront/server/api/storefront/account/login.post.ts",
				"web/storefront/server/utils/customer.ts",
			} {
				if outputContent(generated, path) == nil || outputOwner(generated, path) != customerCapability {
					t.Errorf("Generate() customer output %s is missing or has the wrong owner", path)
				}
			}
			for _, output := range generated.Outputs {
				if !strings.HasSuffix(output.Path, ".go") {
					continue
				}
				formatted, err := format.Source(output.Content)
				if err != nil {
					t.Fatalf("generated %q is invalid Go: %v\n%s", output.Path, err, output.Content)
				}
				if !bytes.Equal(formatted, output.Content) {
					t.Fatalf("generated %q is not gofmt formatted", output.Path)
				}
			}
			mainSource := string(outputContent(generated, "cmd/server/main.go"))
			if !strings.Contains(mainSource, "customerAPI.Register") || !strings.Contains(mainSource, "secureCookies") {
				t.Fatal("Generate() did not wire customer routes with the shared secure cookie setting")
			}
			openAPI := outputContent(generated, "api/openapi.yaml")
			var document struct {
				Paths map[string]any `yaml:"paths"`
			}
			if err := yaml.Unmarshal(openAPI, &document); err != nil {
				t.Fatalf("generated customer OpenAPI is invalid YAML: %v\n%s", err, openAPI)
			}
			for _, path := range []string{
				"/api/v1/storefront/account/register", "/api/v1/storefront/account/password",
				"/api/v1/customers", "/api/v1/customers/{id}/suspend",
			} {
				if document.Paths[path] == nil {
					t.Errorf("generated customer OpenAPI is missing %s", path)
				}
			}
		})
	}
}

func TestCustomerAccountsRequiresLifecycleTenancyWhenScoped(t *testing.T) {
	t.Parallel()

	project := validProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyCapability, Version: tenancyMembersVersion},
		{Name: customerCapability, Version: customerVersion},
	}
	if _, err := New().Generate(context.Background(), project); err == nil ||
		!strings.Contains(err.Error(), "customer-accounts with organization-tenancy requires organization-tenancy 0.3.0") {
		t.Fatalf("Generate() error = %v, want lifecycle tenancy requirement", err)
	}
}

func TestGenerateCRMCoreAcrossDatabases(t *testing.T) {
	t.Parallel()

	for _, engine := range []string{"postgresql", "mysql"} {
		engine := engine
		t.Run(engine, func(t *testing.T) {
			t.Parallel()
			project := validProject()
			project.Spec.Database.Engine = engine
			project.Spec.Stack.AdminUI = "element-plus"
			project.Spec.Capabilities = []spec.CapabilitySelection{
				{Name: tenancyCapability, Version: tenancyLifecycleVersion},
				{Name: crmCapability, Version: crmVersion},
			}
			generated, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if generated.CapabilityLock[crmCapability] != crmVersion {
				t.Fatalf("Generate() capability lock = %#v", generated.CapabilityLock)
			}
			storePath := "internal/crm/" + databaseTemplates[engine].Data.PackageName + "/store.go"
			for _, path := range []string{
				"internal/crm/crm.go",
				"internal/crm/crm_test.go",
				"internal/crm/httpapi/handler.go",
				"internal/crm/httpapi/handler_test.go",
				storePath,
				"internal/platform/migrate/migrations/000280_crm_core.sql",
				"web/admin/src/views/CRMView.vue",
			} {
				if outputContent(generated, path) == nil || outputOwner(generated, path) != crmCapability {
					t.Errorf("Generate() CRM output %s is missing or has the wrong owner", path)
				}
			}
			for _, output := range generated.Outputs {
				if !strings.HasSuffix(output.Path, ".go") {
					continue
				}
				formatted, err := format.Source(output.Content)
				if err != nil {
					t.Fatalf("generated %q is invalid Go: %v\n%s", output.Path, err, output.Content)
				}
				if !bytes.Equal(formatted, output.Content) {
					t.Fatalf("generated %q is not gofmt formatted", output.Path)
				}
			}
			if mainSource := string(outputContent(generated, "cmd/server/main.go")); !strings.Contains(mainSource, "crmAPI.Register") {
				t.Fatal("Generate() did not wire CRM routes")
			}
			var document struct {
				Paths map[string]any `yaml:"paths"`
			}
			openAPI := outputContent(generated, "api/openapi.yaml")
			if err := yaml.Unmarshal(openAPI, &document); err != nil {
				t.Fatalf("generated CRM OpenAPI is invalid YAML: %v\n%s", err, openAPI)
			}
			for _, path := range []string{
				"/api/v1/crm/accounts", "/api/v1/crm/contacts",
				"/api/v1/crm/activities", "/api/v1/crm/opportunities/{id}/advance",
			} {
				if document.Paths[path] == nil {
					t.Errorf("generated CRM OpenAPI is missing %s", path)
				}
			}
		})
	}
}

func TestCRMCoreRequiresLifecycleTenancyWhenScoped(t *testing.T) {
	t.Parallel()

	project := validProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyCapability, Version: tenancyMembersVersion},
		{Name: crmCapability, Version: crmVersion},
	}
	if _, err := New().Generate(context.Background(), project); err == nil ||
		!strings.Contains(err.Error(), "crm-core with organization-tenancy requires organization-tenancy 0.3.0") {
		t.Fatalf("Generate() error = %v, want lifecycle tenancy requirement", err)
	}
}

func TestGenerateERPInventoryCapability(t *testing.T) {
	t.Parallel()

	for _, engine := range []string{"postgresql", "mysql"} {
		engine := engine
		t.Run(engine, func(t *testing.T) {
			t.Parallel()
			project := validProject()
			project.Spec.Database.Engine = engine
			project.Spec.Stack.AdminUI = "element-plus"
			project.Spec.Capabilities = []spec.CapabilitySelection{
				{Name: tenancyCapability, Version: tenancyLifecycleVersion},
				{Name: inventoryCapability, Version: inventoryVersion},
			}
			generated, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if generated.CapabilityLock[inventoryCapability] != inventoryVersion {
				t.Fatalf("Generate() capability lock = %#v", generated.CapabilityLock)
			}
			storePath := "internal/inventory/" + databaseTemplates[engine].Data.PackageName + "/store.go"
			for _, path := range []string{
				"internal/inventory/inventory.go",
				"internal/inventory/inventory_test.go",
				"internal/inventory/httpapi/handler.go",
				"internal/inventory/httpapi/handler_test.go",
				storePath,
				"internal/platform/migrate/migrations/000290_erp_inventory.sql",
				"web/admin/src/views/InventoryView.vue",
			} {
				if outputContent(generated, path) == nil || outputOwner(generated, path) != inventoryCapability {
					t.Errorf("Generate() inventory output %s is missing or has the wrong owner", path)
				}
			}
			if mainSource := string(outputContent(generated, "cmd/server/main.go")); !strings.Contains(mainSource, "inventoryAPI.Register") {
				t.Fatal("Generate() did not wire inventory routes")
			}
			var document struct {
				Paths map[string]any `yaml:"paths"`
			}
			openAPI := outputContent(generated, "api/openapi.yaml")
			if err := yaml.Unmarshal(openAPI, &document); err != nil {
				t.Fatalf("generated inventory OpenAPI is invalid YAML: %v\n%s", err, openAPI)
			}
			for _, path := range []string{
				"/api/v1/inventory/items", "/api/v1/inventory/balances",
				"/api/v1/inventory/reservations", "/api/v1/inventory/purchase-orders/{id}/receive",
			} {
				if document.Paths[path] == nil {
					t.Errorf("generated inventory OpenAPI is missing %s", path)
				}
			}
		})
	}
}

func TestERPInventoryRequiresLifecycleTenancyWhenScoped(t *testing.T) {
	t.Parallel()
	project := validProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyCapability, Version: tenancyMembersVersion},
		{Name: inventoryCapability, Version: inventoryVersion},
	}
	if _, err := New().Generate(context.Background(), project); err == nil ||
		!strings.Contains(err.Error(), "erp-inventory with organization-tenancy requires organization-tenancy 0.3.0") {
		t.Fatalf("Generate() error = %v, want lifecycle tenancy requirement", err)
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

func TestGenerateCommerceOperationsForBothDatabases(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			project := validProject()
			project.Spec.Database.Engine = database
			project.Spec.Capabilities = []spec.CapabilitySelection{
				{Name: catalogCapability, Version: catalogVersion},
				{Name: customerCapability, Version: customerVersion},
				{Name: commerceCapability, Version: commerceVersion},
			}
			generated, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if generated.CapabilityLock[commerceCapability] != commerceVersion {
				t.Fatalf("capability lock = %#v", generated.CapabilityLock)
			}
			for _, path := range []string{
				"internal/commerce/commerce.go",
				"internal/commerce/httpapi/handler.go",
				"internal/commerce/sandbox/gateway.go",
				"internal/platform/migrate/migrations/000300_commerce_operations.sql",
				"api/openapi.yaml",
			} {
				if outputContent(generated, path) == nil {
					t.Errorf("commerce output %s is missing", path)
				}
			}
			var document struct {
				Paths map[string]any `yaml:"paths"`
			}
			if err := yaml.Unmarshal(outputContent(generated, "api/openapi.yaml"), &document); err != nil {
				t.Fatalf("generated OpenAPI is invalid YAML: %v", err)
			}
			if document.Paths["/api/v1/storefront/checkout"] == nil || document.Paths["/api/v1/commerce/orders/{id}/refund"] == nil {
				t.Fatalf("commerce paths are incomplete")
			}
		})
	}
}

func TestGenerateRejectsUnsupportedStackSelections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*spec.Project)
	}{
		{name: "unsupported database", mutate: func(project *spec.Project) { project.Spec.Database.Engine = "sqlite" }},
		{name: "storefront", mutate: func(project *spec.Project) { project.Spec.Stack.Storefront = "unsupported" }},
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

func outputOwner(generated generator.Result, path string) string {
	for _, output := range generated.Outputs {
		if output.Path == path {
			return output.Owner
		}
	}
	return ""
}
