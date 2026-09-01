package python

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/generator"
	adminui "github.com/hkx5414375/scaffold-agent/internal/generator/admin"
	storefrontui "github.com/hkx5414375/scaffold-agent/internal/generator/storefront"
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

func TestGenerateSharedNuxtStorefrontFoundation(t *testing.T) {
	t.Parallel()

	project := validProject()
	project.Metadata.DisplayName = "Demo Store"
	project.Spec.Stack.Storefront = "nuxt"
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(generated.Outputs) != 29+len(storefrontui.BaseTemplates) ||
		generated.CapabilityLock[storefrontOwner] != storefrontVersion {
		t.Fatalf("Generate() result = %#v", generated)
	}
	for _, path := range []string{
		"web/storefront/package-lock.json",
		"web/storefront/nuxt.config.ts",
		"web/storefront/app/pages/index.vue",
		"web/storefront/server/api/storefront/status.get.ts",
	} {
		if outputContent(generated, path) == nil || outputOwner(generated, path) != storefrontOwner {
			t.Errorf("Generate() storefront output %s is missing or has the wrong owner", path)
		}
	}
	if !strings.Contains(string(outputContent(generated, "web/storefront/app/pages/index.vue")), "Demo Store") {
		t.Fatal("generated storefront does not use the project display name")
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
		{name: "storefront", mutate: func(project *spec.Project) { project.Spec.Stack.Storefront = "unsupported" }, want: "Nuxt"},
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

func TestGenerateOrganizationMemberAdministrationForBothDatabases(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			project := validBusinessProject()
			project.Spec.Database.Engine = database
			project.Spec.Stack.AdminUI = "element-plus"
			project.Spec.Capabilities = []spec.CapabilitySelection{{
				Name: tenancyOwner, Version: tenancyMembersVersion,
			}}
			generated, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate(%s) error = %v", database, err)
			}
			if generated.CapabilityLock[tenancyOwner] != tenancyMembersVersion || len(generated.Outputs) != 74 {
				t.Fatalf("Generate(%s) result = %#v", database, generated)
			}
			for _, path := range []string{
				"src/demo_service/tenancy/member_http.py",
				"src/demo_service/tenancy/member_models.py",
				"src/demo_service/tenancy/member_repository.py",
				"src/demo_service/tenancy/member_service.py",
				"src/demo_service/migration/versions/000060_tenancy_members.py",
				"tests/test_tenancy_members.py",
				"tests/test_tenancy_members_database.py",
				"web/admin/src/views/MembersView.vue",
			} {
				if outputContent(generated, path) == nil || outputOwner(generated, path) != tenancyOwner {
					t.Errorf("Generate(%s) did not produce member-owned %s", database, path)
				}
			}
			if !strings.Contains(string(outputContent(generated, "api/openapi.yaml")), "organization-invitations/accept") {
				t.Errorf("Generate(%s) stable OpenAPI omits invitation acceptance", database)
			}
		})
	}
}

func TestOrganizationMemberUpgradePreservesTenancyMigration(t *testing.T) {
	t.Parallel()

	baseline := validProject()
	baseline.Spec.Capabilities = []spec.CapabilitySelection{{
		Name: tenancyOwner, Version: tenancyVersion,
	}}
	upgraded := baseline
	upgraded.Spec.Capabilities = []spec.CapabilitySelection{{
		Name: tenancyOwner, Version: tenancyMembersVersion,
	}}
	baselineResult, err := New().Generate(context.Background(), baseline)
	if err != nil {
		t.Fatalf("Generate(baseline) error = %v", err)
	}
	upgradedResult, err := New().Generate(context.Background(), upgraded)
	if err != nil {
		t.Fatalf("Generate(upgraded) error = %v", err)
	}
	path := "src/demo_service/migration/versions/000050_tenancy.py"
	if !reflect.DeepEqual(outputContent(baselineResult, path), outputContent(upgradedResult, path)) {
		t.Fatal("organization member upgrade rewrote the 0.1.0 tenancy migration")
	}
	if len(upgradedResult.Outputs) != 44 {
		t.Fatalf("Generate(upgraded) output count = %d, want 44", len(upgradedResult.Outputs))
	}
}

func TestGenerateOrganizationLifecycleForBothDatabases(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			project := validBusinessProject()
			project.Spec.Database.Engine = database
			project.Spec.Stack.AdminUI = "element-plus"
			project.Spec.Capabilities = []spec.CapabilitySelection{{
				Name: tenancyOwner, Version: tenancyLifecycleVersion,
			}}
			generated, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate(%s) error = %v", database, err)
			}
			if generated.CapabilityLock[tenancyOwner] != tenancyLifecycleVersion || len(generated.Outputs) != 81 {
				t.Fatalf("Generate(%s) result = %#v", database, generated)
			}
			for _, path := range []string{
				"src/demo_service/tenancy/lifecycle_http.py",
				"src/demo_service/tenancy/lifecycle_repository.py",
				"src/demo_service/tenancy/lifecycle_service.py",
				"src/demo_service/migration/versions/000070_tenancy_lifecycle.py",
				"tests/test_tenancy_lifecycle.py",
				"tests/test_tenancy_lifecycle_database.py",
				"web/admin/src/views/OrganizationSettingsView.vue",
			} {
				if outputContent(generated, path) == nil || outputOwner(generated, path) != tenancyOwner {
					t.Errorf("Generate(%s) did not produce lifecycle-owned %s", database, path)
				}
			}
		})
	}
}

func TestOrganizationLifecycleUpgradePreservesPriorMigrations(t *testing.T) {
	t.Parallel()

	members := validProject()
	members.Spec.Capabilities = []spec.CapabilitySelection{{
		Name: tenancyOwner, Version: tenancyMembersVersion,
	}}
	lifecycle := members
	lifecycle.Spec.Capabilities = []spec.CapabilitySelection{{
		Name: tenancyOwner, Version: tenancyLifecycleVersion,
	}}
	membersResult, err := New().Generate(context.Background(), members)
	if err != nil {
		t.Fatalf("Generate(members) error = %v", err)
	}
	lifecycleResult, err := New().Generate(context.Background(), lifecycle)
	if err != nil {
		t.Fatalf("Generate(lifecycle) error = %v", err)
	}
	for _, path := range []string{
		"src/demo_service/migration/versions/000050_tenancy.py",
		"src/demo_service/migration/versions/000060_tenancy_members.py",
	} {
		if !reflect.DeepEqual(outputContent(membersResult, path), outputContent(lifecycleResult, path)) {
			t.Fatalf("organization lifecycle upgrade rewrote %s", path)
		}
	}
	if len(lifecycleResult.Outputs) != 50 {
		t.Fatalf("Generate(lifecycle) output count = %d, want 50", len(lifecycleResult.Outputs))
	}
}

func TestGenerateDurableJobsForBothDatabases(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			project := validProject()
			project.Spec.Database.Engine = database
			project.Spec.Capabilities = []spec.CapabilitySelection{{
				Name: jobsOwner, Version: jobsVersion,
			}}
			generated, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate(%s) error = %v", database, err)
			}
			if generated.CapabilityLock[jobsOwner] != jobsVersion || len(generated.Outputs) != 38 {
				t.Fatalf("Generate(%s) result = %#v", database, generated)
			}
			for _, path := range []string{
				"src/demo_service/jobs/models.py",
				"src/demo_service/jobs/repository.py",
				"src/demo_service/jobs/service.py",
				"src/demo_service/jobs/worker.py",
				"src/demo_service/migration/versions/000200_background_jobs.py",
				"tests/test_jobs.py",
				"tests/test_jobs_database.py",
				"tests/test_jobs_worker.py",
			} {
				if outputContent(generated, path) == nil || outputOwner(generated, path) != jobsOwner {
					t.Errorf("Generate(%s) did not produce jobs-owned %s", database, path)
				}
			}
			readme := string(outputContent(generated, "README.md"))
			if !strings.Contains(readme, "python -m demo_service.jobs.worker") ||
				!strings.Contains(readme, "JobService.enqueue") {
				t.Fatalf("Generate(%s) README does not document the independent worker:\n%s", database, readme)
			}
		})
	}
}

func TestGenerateDurableJobsComposesWithTenantLifecycle(t *testing.T) {
	t.Parallel()

	project := validBusinessProject()
	project.Spec.Stack.AdminUI = "element-plus"
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyOwner, Version: tenancyLifecycleVersion},
		{Name: jobsOwner, Version: jobsVersion},
	}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(generated.Outputs) != 90 {
		t.Fatalf("Generate() output count = %d, want 90", len(generated.Outputs))
	}
	migration := string(outputContent(
		generated,
		"src/demo_service/migration/versions/000200_background_jobs.py",
	))
	if !strings.Contains(migration, `down_revision: str | None = "000070_tenancy_lifecycle"`) ||
		!strings.Contains(migration, "organizations.id") {
		t.Fatalf("generated tenant job migration is not lifecycle-aware:\n%s", migration)
	}
}

func TestGenerateNotificationsResolveJobsForBothDatabases(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			project := validProject()
			project.Spec.Database.Engine = database
			project.Spec.Capabilities = []spec.CapabilitySelection{{
				Name: notificationsOwner, Version: notificationsVersion,
			}}
			generated, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate(%s) error = %v", database, err)
			}
			if generated.CapabilityLock[jobsOwner] != jobsVersion ||
				generated.CapabilityLock[notificationsOwner] != notificationsVersion ||
				len(generated.Outputs) != 47 {
				t.Fatalf("Generate(%s) result = %#v", database, generated)
			}
			for _, path := range []string{
				"src/demo_service/notifications/handler.py",
				"src/demo_service/notifications/service.py",
				"src/demo_service/notifications/smtp.py",
				"tests/test_notification_handler.py",
				"tests/test_notification_smtp.py",
				"tests/test_notifications.py",
				"tests/test_notifications_database.py",
				"tests/test_notifications_worker_configuration.py",
			} {
				if outputContent(generated, path) == nil ||
					outputOwner(generated, path) != notificationsOwner {
					t.Errorf("Generate(%s) did not produce notification-owned %s", database, path)
				}
			}
			mainSource := string(outputContent(generated, "src/demo_service/main.py"))
			if !strings.Contains(mainSource, "NotificationService(job_service)") ||
				strings.Contains(mainSource, "SmtpEmailSender") {
				t.Fatalf("Generate(%s) web composition initialized SMTP:\n%s", database, mainSource)
			}
			workerSource := string(outputContent(generated, "src/demo_service/jobs/worker.py"))
			if !strings.Contains(workerSource, "SmtpEmailSender.from_environment()") {
				t.Fatalf("Generate(%s) worker does not initialize SMTP:\n%s", database, workerSource)
			}
			readme := string(outputContent(generated, "README.md"))
			if !strings.Contains(readme, "SMTP_TLS_MODE") ||
				!strings.Contains(readme, "NotificationService.enqueue_email") {
				t.Fatalf("Generate(%s) README does not document notifications:\n%s", database, readme)
			}
		})
	}
}

func TestGenerateNotificationsComposeWithTenantLifecycle(t *testing.T) {
	t.Parallel()

	project := validBusinessProject()
	project.Spec.Stack.AdminUI = "element-plus"
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyOwner, Version: tenancyLifecycleVersion},
		{Name: notificationsOwner, Version: notificationsVersion},
	}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(generated.Outputs) != 99 ||
		generated.CapabilityLock[jobsOwner] != jobsVersion ||
		generated.CapabilityLock[notificationsOwner] != notificationsVersion {
		t.Fatalf("Generate() result = %#v", generated)
	}
}

func TestGenerateFileAssetsForBothDatabases(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			project := validProject()
			project.Spec.Database.Engine = database
			project.Spec.Capabilities = []spec.CapabilitySelection{{
				Name: filesOwner, Version: filesVersion,
			}}
			generated, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate(%s) error = %v", database, err)
			}
			if generated.CapabilityLock[filesOwner] != filesVersion || len(generated.Outputs) != 40 {
				t.Fatalf("Generate(%s) result = %#v", database, generated)
			}
			for _, path := range []string{
				"src/demo_service/files/http.py",
				"src/demo_service/files/models.py",
				"src/demo_service/files/repository.py",
				"src/demo_service/files/service.py",
				"src/demo_service/files/storage.py",
				"src/demo_service/migration/versions/000210_file_assets.py",
				"tests/test_file_assets.py",
				"tests/test_file_assets_database.py",
				"tests/test_file_assets_http.py",
				"tests/test_file_assets_storage.py",
			} {
				if outputContent(generated, path) == nil || outputOwner(generated, path) != filesOwner {
					t.Errorf("Generate(%s) did not produce file-owned %s", database, path)
				}
			}
			contract := string(outputContent(generated, "api/openapi.yaml"))
			for _, fragment := range []string{
				"/api/v1/files:",
				"operationId: uploadFileAsset",
				"operationId: downloadFileAsset",
				`"413":`,
			} {
				if !strings.Contains(contract, fragment) {
					t.Errorf("Generate(%s) OpenAPI does not contain %q", database, fragment)
				}
			}
			if strings.Contains(contract, "storage_key") {
				t.Errorf("Generate(%s) OpenAPI exposes an internal storage key", database)
			}
			configuration := string(outputContent(generated, "pyproject.toml"))
			if !strings.Contains(configuration, `"python-multipart==0.0.32"`) {
				t.Errorf("Generate(%s) does not pin multipart parsing", database)
			}
			readme := string(outputContent(generated, "README.md"))
			if !strings.Contains(readme, "FILE_STORAGE_ROOT") ||
				!strings.Contains(readme, "10 MiB") {
				t.Errorf("Generate(%s) README does not document file storage", database)
			}
		})
	}
}

func TestGenerateFileAssetsComposeWithTenantLifecycleAndAdministration(t *testing.T) {
	t.Parallel()

	project := validBusinessProject()
	project.Spec.Stack.AdminUI = "element-plus"
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyOwner, Version: tenancyLifecycleVersion},
		{Name: filesOwner, Version: filesVersion},
	}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(generated.Outputs) != 93 ||
		generated.CapabilityLock[filesOwner] != filesVersion {
		t.Fatalf("Generate() result = %#v", generated)
	}
	viewPath := "web/admin/src/views/FilesView.vue"
	if outputContent(generated, viewPath) == nil || outputOwner(generated, viewPath) != filesOwner {
		t.Fatalf("Generate() did not produce the file administration view")
	}
	migration := string(outputContent(
		generated,
		"src/demo_service/migration/versions/000210_file_assets.py",
	))
	if !strings.Contains(migration, `down_revision: str | None = "000070_tenancy_lifecycle"`) ||
		!strings.Contains(migration, "organizations.id") {
		t.Fatalf("generated file migration is not tenant lifecycle aware:\n%s", migration)
	}
}

func TestGenerateApplicationCacheForBothDatabases(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			project := validProject()
			project.Spec.Database.Engine = database
			project.Spec.Capabilities = []spec.CapabilitySelection{{
				Name: cacheOwner, Version: cacheVersion,
			}}
			generated, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate(%s) error = %v", database, err)
			}
			if generated.CapabilityLock[cacheOwner] != cacheVersion || len(generated.Outputs) != 36 {
				t.Fatalf("Generate(%s) result = %#v", database, generated)
			}
			for _, path := range []string{
				"src/demo_service/cache/models.py",
				"src/demo_service/cache/repository.py",
				"src/demo_service/cache/service.py",
				"src/demo_service/migration/versions/000220_application_cache.py",
				"tests/test_application_cache.py",
				"tests/test_application_cache_database.py",
			} {
				if outputContent(generated, path) == nil || outputOwner(generated, path) != cacheOwner {
					t.Errorf("Generate(%s) did not produce cache-owned %s", database, path)
				}
			}
			mainSource := string(outputContent(generated, "src/demo_service/main.py"))
			if !strings.Contains(mainSource, "CacheService(SQLAlchemyCacheRepository(engine))") {
				t.Errorf("Generate(%s) does not compose the cache service", database)
			}
			contract := string(outputContent(generated, "api/openapi.yaml"))
			if strings.Contains(contract, "/api/v1/cache") {
				t.Errorf("Generate(%s) exposes a generic cache endpoint", database)
			}
		})
	}
}

func TestGenerateApplicationCacheComposesAfterFileAssets(t *testing.T) {
	t.Parallel()

	project := validBusinessProject()
	project.Spec.Stack.AdminUI = "element-plus"
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyOwner, Version: tenancyLifecycleVersion},
		{Name: filesOwner, Version: filesVersion},
		{Name: cacheOwner, Version: cacheVersion},
	}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(generated.Outputs) != 100 || generated.CapabilityLock[cacheOwner] != cacheVersion {
		t.Fatalf("Generate() result = %#v", generated)
	}
	migration := string(outputContent(
		generated,
		"src/demo_service/migration/versions/000220_application_cache.py",
	))
	if !strings.Contains(migration, `down_revision: str | None = "000210_file_assets"`) ||
		!strings.Contains(migration, "organizations.id") {
		t.Fatalf("generated cache migration does not follow tenant files:\n%s", migration)
	}
}

func TestGenerateJobAdministrationForBothDatabases(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			project := validProject()
			project.Spec.Database.Engine = database
			project.Spec.Capabilities = []spec.CapabilitySelection{{
				Name: jobAdminOwner, Version: jobAdminVersion,
			}}
			generated, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate(%s) error = %v", database, err)
			}
			if generated.CapabilityLock[jobsOwner] != jobsVersion ||
				generated.CapabilityLock[jobAdminOwner] != jobAdminVersion ||
				len(generated.Outputs) != 47 {
				t.Fatalf("Generate(%s) result = %#v", database, generated)
			}
			for _, path := range []string{
				"src/demo_service/jobadmin/http.py",
				"src/demo_service/jobadmin/models.py",
				"src/demo_service/jobadmin/repository.py",
				"src/demo_service/jobadmin/service.py",
				"src/demo_service/migration/versions/000230_job_administration.py",
				"tests/test_job_administration.py",
				"tests/test_job_administration_database.py",
				"tests/test_job_administration_http.py",
			} {
				if outputContent(generated, path) == nil || outputOwner(generated, path) != jobAdminOwner {
					t.Errorf("Generate(%s) did not produce job-administration-owned %s", database, path)
				}
			}
			mainSource := string(outputContent(generated, "src/demo_service/main.py"))
			if !strings.Contains(mainSource, "JobAdministrationService(") ||
				!strings.Contains(mainSource, "application.include_router(job_administration_router)") {
				t.Errorf("Generate(%s) does not compose job administration", database)
			}
			publicSource := string(outputContent(generated, "src/demo_service/jobadmin/models.py"))
			contract := string(outputContent(generated, "api/openapi.yaml"))
			for _, sensitive := range []string{"payload_json", "scope_key", "dedupe_key"} {
				if strings.Contains(publicSource, sensitive) || strings.Contains(contract, sensitive) {
					t.Errorf("Generate(%s) exposes sensitive job field %q", database, sensitive)
				}
			}
			var openAPI map[string]any
			if err := yaml.Unmarshal(outputContent(generated, "api/openapi.yaml"), &openAPI); err != nil {
				t.Fatalf("generated job administration OpenAPI is invalid YAML: %v", err)
			}
		})
	}
}

func TestGenerateJobAdministrationComposesWithTenantAdministration(t *testing.T) {
	t.Parallel()

	project := validBusinessProject()
	project.Spec.Stack.AdminUI = "element-plus"
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyOwner, Version: tenancyLifecycleVersion},
		{Name: jobAdminOwner, Version: jobAdminVersion},
	}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(generated.Outputs) != 100 ||
		generated.CapabilityLock[jobAdminOwner] != jobAdminVersion ||
		generated.CapabilityLock[jobsOwner] != jobsVersion {
		t.Fatalf("Generate() result = %#v", generated)
	}
	viewPath := "web/admin/src/views/JobsView.vue"
	if outputContent(generated, viewPath) == nil || outputOwner(generated, viewPath) != jobAdminOwner {
		t.Fatalf("Generate() did not produce the shared job administration view")
	}
	migration := string(outputContent(
		generated,
		"src/demo_service/migration/versions/000230_job_administration.py",
	))
	if !strings.Contains(migration, `down_revision: str | None = "000200_background_jobs"`) ||
		!strings.Contains(migration, `"role_code": "admin"`) {
		t.Fatalf("generated job administration migration is invalid:\n%s", migration)
	}
	if strings.Contains(migration, `"role_code": "user"`) {
		t.Fatalf("generated job administration permissions are not administrator-only:\n%s", migration)
	}
}

func TestGenerateJobAdministrationFollowsCacheMigration(t *testing.T) {
	t.Parallel()

	project := validProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: cacheOwner, Version: cacheVersion},
		{Name: jobAdminOwner, Version: jobAdminVersion},
	}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	migration := string(outputContent(
		generated,
		"src/demo_service/migration/versions/000230_job_administration.py",
	))
	if !strings.Contains(migration, `down_revision: str | None = "000220_application_cache"`) {
		t.Fatalf("generated job administration migration does not follow cache:\n%s", migration)
	}
}

func TestGenerateObservabilityForBothDatabases(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			project := validProject()
			project.Spec.Database.Engine = database
			project.Spec.Capabilities = []spec.CapabilitySelection{{
				Name: observabilityOwner, Version: observabilityVersion,
			}}
			generated, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate(%s) error = %v", database, err)
			}
			if generated.CapabilityLock[observabilityOwner] != observabilityVersion ||
				len(generated.Outputs) != 34 {
				t.Fatalf("Generate(%s) result = %#v", database, generated)
			}
			for _, path := range []string{
				"src/demo_service/observability/http.py",
				"src/demo_service/observability/middleware.py",
				"tests/test_observability.py",
				"tests/test_observability_database.py",
			} {
				if outputContent(generated, path) == nil || outputOwner(generated, path) != observabilityOwner {
					t.Errorf("Generate(%s) did not produce observability-owned %s", database, path)
				}
			}
			mainSource := string(outputContent(generated, "src/demo_service/main.py"))
			if !strings.Contains(mainSource, "HttpObservabilityMiddleware") ||
				!strings.Contains(mainSource, "application.include_router(observability_router)") {
				t.Errorf("Generate(%s) does not compose observability", database)
			}
			databaseSource := string(outputContent(generated, "src/demo_service/database.py"))
			if !strings.Contains(databaseSource, "timeout_seconds: float = 2.0") ||
				!strings.Contains(databaseSource, "future.result(timeout=self._timeout_seconds)") {
				t.Errorf("Generate(%s) does not bound the whole readiness probe", database)
			}
			contract := outputContent(generated, "api/openapi.yaml")
			var openAPI map[string]any
			if err := yaml.Unmarshal(contract, &openAPI); err != nil {
				t.Fatalf("generated observability OpenAPI is invalid YAML: %v", err)
			}
			if !strings.Contains(string(contract), "/metrics:") ||
				!strings.Contains(string(contract), "enum: [ok, ready, unavailable]") {
				t.Errorf("Generate(%s) observability OpenAPI is incomplete", database)
			}
		})
	}
}

func TestGenerateCSVTransferRequiresBusinessEntity(t *testing.T) {
	t.Parallel()

	project := validProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{{
		Name: csvTransferOwner, Version: csvTransferVersion,
	}}
	_, err := New().Generate(context.Background(), project)
	if err == nil || !strings.Contains(err.Error(), "requires exactly one generated business entity") {
		t.Fatalf("Generate() error = %v", err)
	}
}

func TestGenerateCSVTransferForBothDatabases(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			project := validBusinessProject()
			project.Spec.Database.Engine = database
			project.Spec.Capabilities = []spec.CapabilitySelection{{
				Name: csvTransferOwner, Version: csvTransferVersion,
			}}
			generated, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate(%s) error = %v", database, err)
			}
			if generated.CapabilityLock[csvTransferOwner] != csvTransferVersion ||
				len(generated.Outputs) != 46 {
				t.Fatalf("Generate(%s) result = %#v", database, generated)
			}
			for _, path := range []string{
				"src/demo_service/tasks/transfer/http.py",
				"src/demo_service/tasks/transfer/models.py",
				"src/demo_service/tasks/transfer/repository.py",
				"src/demo_service/tasks/transfer/service.py",
				"src/demo_service/migration/versions/000240_csv_import_export.py",
				"tests/test_csv_transfer.py",
				"tests/test_csv_transfer_database.py",
				"tests/test_csv_transfer_http.py",
			} {
				if outputContent(generated, path) == nil || outputOwner(generated, path) != csvTransferOwner {
					t.Errorf("Generate(%s) did not produce CSV-owned %s", database, path)
				}
			}
			mainSource := string(outputContent(generated, "src/demo_service/main.py"))
			csvRouter := strings.Index(mainSource, "application.include_router(csv_transfer_router)")
			businessRouter := strings.Index(mainSource, "application.include_router(business_router)")
			if csvRouter < 0 || businessRouter < 0 || csvRouter > businessRouter {
				t.Errorf("Generate(%s) does not compose the CSV router before dynamic IDs", database)
			}
			migration := string(outputContent(
				generated,
				"src/demo_service/migration/versions/000240_csv_import_export.py",
			))
			for _, permission := range []string{"tasks:task:import", "tasks:task:export"} {
				if !strings.Contains(migration, permission) {
					t.Errorf("Generate(%s) does not grant %s", database, permission)
				}
			}
			if strings.Contains(migration, `"role_code": "user"`) {
				t.Errorf("Generate(%s) grants CSV transfer to non-administrators", database)
			}
			contract := outputContent(generated, "api/openapi.yaml")
			var openAPI map[string]any
			if err := yaml.Unmarshal(contract, &openAPI); err != nil {
				t.Fatalf("generated CSV OpenAPI is invalid YAML: %v", err)
			}
			for _, fragment := range []string{
				"/api/v1/tasks/import:",
				"/api/v1/tasks/import-template:",
				"/api/v1/tasks/export:",
				"CSVImportResult:",
			} {
				if !strings.Contains(string(contract), fragment) {
					t.Errorf("Generate(%s) OpenAPI does not contain %q", database, fragment)
				}
			}
		})
	}
}

func TestGenerateCSVTransferComposesWithTenantAdministration(t *testing.T) {
	t.Parallel()

	project := validBusinessProject()
	project.Spec.Stack.AdminUI = "element-plus"
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyOwner, Version: tenancyLifecycleVersion},
		{Name: csvTransferOwner, Version: csvTransferVersion},
	}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(generated.Outputs) != 90 ||
		generated.CapabilityLock[csvTransferOwner] != csvTransferVersion {
		t.Fatalf("Generate() result = %#v", generated)
	}
	migration := string(outputContent(
		generated,
		"src/demo_service/migration/versions/000240_csv_import_export.py",
	))
	if !strings.Contains(migration, `down_revision: str | None = "000070_tenancy_lifecycle"`) {
		t.Fatalf("generated CSV migration does not follow tenant lifecycle:\n%s", migration)
	}
	view := string(outputContent(generated, "web/admin/src/views/BusinessView.vue"))
	for _, fragment := range []string{"Import CSV", "Export CSV", "import-template", "5 * 1024 * 1024"} {
		if !strings.Contains(view, fragment) {
			t.Errorf("generated administration view does not contain %q", fragment)
		}
	}
}

func TestGenerateApprovalWorkflowsRequiresBusinessAndExactWorkflow(t *testing.T) {
	t.Parallel()

	missingBusiness := validProject()
	missingBusiness.Spec.Capabilities = []spec.CapabilitySelection{{
		Name: approvalsOwner, Version: approvalsVersion,
	}}
	_, err := New().Generate(context.Background(), missingBusiness)
	if err == nil || !strings.Contains(err.Error(), "requires exactly one generated business entity") {
		t.Fatalf("Generate(missing business) error = %v", err)
	}

	invalidWorkflow := validBusinessProject()
	invalidWorkflow.Spec.Modules[0].Workflows = []spec.Workflow{{
		Name: "approval", States: []string{"pending", "approved"},
	}}
	invalidWorkflow.Spec.Capabilities = []spec.CapabilitySelection{{
		Name: approvalsOwner, Version: approvalsVersion,
	}}
	_, err = New().Generate(context.Background(), invalidWorkflow)
	if err == nil || !strings.Contains(err.Error(), "requires workflow approval") {
		t.Fatalf("Generate(invalid workflow) error = %v", err)
	}
}

func TestGenerateApprovalWorkflowsForBothDatabases(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			project := validApprovalProject()
			project.Spec.Database.Engine = database
			generated, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate(%s) error = %v", database, err)
			}
			if generated.CapabilityLock[approvalsOwner] != approvalsVersion ||
				len(generated.Outputs) != 46 {
				t.Fatalf("Generate(%s) result = %#v", database, generated)
			}
			for _, path := range []string{
				"src/demo_service/approvals/http.py",
				"src/demo_service/approvals/models.py",
				"src/demo_service/approvals/repository.py",
				"src/demo_service/approvals/service.py",
				"src/demo_service/migration/versions/000250_approval_workflows.py",
				"tests/test_approval_workflows.py",
				"tests/test_approval_workflows_database.py",
				"tests/test_approval_workflows_http.py",
			} {
				if outputContent(generated, path) == nil || outputOwner(generated, path) != approvalsOwner {
					t.Errorf("Generate(%s) did not produce approval-owned %s", database, path)
				}
			}
			mainSource := string(outputContent(generated, "src/demo_service/main.py"))
			if !strings.Contains(mainSource, "app.state.approval_service = ApprovalService(") ||
				!strings.Contains(mainSource, "SQLAlchemyApprovalRepository(engine)") ||
				!strings.Contains(mainSource, "application.include_router(approval_router)") {
				t.Errorf("Generate(%s) does not compose approvals", database)
			}
			migration := string(outputContent(
				generated,
				"src/demo_service/migration/versions/000250_approval_workflows.py",
			))
			for _, fragment := range []string{
				`{"role_code": "admin", "permission_code": row["code"]}`,
				`{"role_code": "user", "permission_code": "approvals:mine"}`,
				`"approvals:decide"`,
				"approval_pending_subject_unique",
				"pending_subject_id",
			} {
				if !strings.Contains(migration, fragment) {
					t.Errorf("Generate(%s) migration does not contain %q", database, fragment)
				}
			}
			contract := outputContent(generated, "api/openapi.yaml")
			var openAPI map[string]any
			if err := yaml.Unmarshal(contract, &openAPI); err != nil {
				t.Fatalf("generated approval OpenAPI is invalid YAML: %v", err)
			}
			for _, fragment := range []string{
				"/api/v1/approvals/{id}/approve:",
				"/api/v1/approvals/{id}/reject:",
				"x-required-permission: approvals:decide",
				"ApprovalEventPage:",
			} {
				if !strings.Contains(string(contract), fragment) {
					t.Errorf("Generate(%s) OpenAPI does not contain %q", database, fragment)
				}
			}
		})
	}
}

func TestGenerateApprovalWorkflowsComposeWithTenantAdministration(t *testing.T) {
	t.Parallel()

	project := validApprovalProject()
	project.Spec.Stack.AdminUI = "element-plus"
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyOwner, Version: tenancyLifecycleVersion},
		{Name: approvalsOwner, Version: approvalsVersion},
	}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(generated.Outputs) != 91 ||
		generated.CapabilityLock[approvalsOwner] != approvalsVersion {
		t.Fatalf("Generate() result = %#v", generated)
	}
	viewPath := "web/admin/src/views/ApprovalsView.vue"
	view := string(outputContent(generated, viewPath))
	if outputOwner(generated, viewPath) != approvalsOwner ||
		!strings.Contains(view, "requesters cannot decide") && !strings.Contains(view, "Approve") {
		t.Fatalf("generated approval administration view is incomplete")
	}
	migration := string(outputContent(
		generated,
		"src/demo_service/migration/versions/000250_approval_workflows.py",
	))
	if !strings.Contains(migration, `down_revision: str | None = "000070_tenancy_lifecycle"`) ||
		!strings.Contains(migration, `sa.ForeignKey("organizations.id"`) {
		t.Fatalf("generated approval migration is not tenant-aware:\n%s", migration)
	}
}

func TestGenerateCSVAndApprovalMigrationsCompose(t *testing.T) {
	t.Parallel()

	project := validApprovalProject()
	project.Spec.Stack.AdminUI = "element-plus"
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyOwner, Version: tenancyLifecycleVersion},
		{Name: csvTransferOwner, Version: csvTransferVersion},
		{Name: approvalsOwner, Version: approvalsVersion},
	}
	generated, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(generated.Outputs) != 100 {
		t.Fatalf("Generate() output count = %d, want 100", len(generated.Outputs))
	}
	migration := string(outputContent(
		generated,
		"src/demo_service/migration/versions/000250_approval_workflows.py",
	))
	if !strings.Contains(migration, `down_revision: str | None = "000240_csv_import_export"`) {
		t.Fatalf("approval migration does not follow CSV transfer:\n%s", migration)
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

func validApprovalProject() spec.Project {
	project := validBusinessProject()
	project.Spec.Modules[0].Workflows = []spec.Workflow{{
		Name: "approval",
		States: []string{
			"pending",
			"approved",
			"rejected",
			"cancelled",
		},
	}}
	project.Spec.Capabilities = []spec.CapabilitySelection{{
		Name: approvalsOwner, Version: approvalsVersion,
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
