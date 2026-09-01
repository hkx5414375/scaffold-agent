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
		{name: "admin", mutate: func(project *spec.Project) { project.Spec.Stack.AdminUI = "ant-design" }, want: "Element Plus"},
		{name: "storefront", mutate: func(project *spec.Project) { project.Spec.Stack.Storefront = "nuxt" }, want: "storefront"},
		{name: "auth", mutate: func(project *spec.Project) { project.Spec.Auth.Modes = []string{"session"} }, want: "requires both session and token"},
		{name: "capability", mutate: func(project *spec.Project) {
			project.Spec.Capabilities = []spec.CapabilitySelection{{Name: "unknown-capability", Version: "0.1.0"}}
		}, want: "not present in the catalog"},
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

func TestGenerateSharedAdministrationProject(t *testing.T) {
	t.Parallel()

	project := validProject()
	project.Spec.Stack.AdminUI = "element-plus"
	project.Spec.Modules = []spec.Module{businessModule()}
	result, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(result.Outputs) != 60 || result.CapabilityLock[adminOwner] != adminVersion {
		t.Fatalf("Generate() result = %#v", result)
	}
	for _, path := range []string{
		"web/admin/package-lock.json",
		"web/admin/src/App.vue",
		"web/admin/src/types.ts",
		"web/admin/src/views/BusinessView.vue",
	} {
		if outputContent(result, path) == nil || outputOwner(result, path) != adminOwner {
			t.Errorf("Generate() administration output %s is missing or has the wrong owner", path)
		}
	}
	types := string(outputContent(result, "web/admin/src/types.ts"))
	if !strings.Contains(types, "priority?: string") ||
		!strings.Contains(types, "version: string") {
		t.Fatalf("generated administration types lose the decimal-string contract:\n%s", types)
	}
}

func TestGenerateOrganizationTenancyForBothDatabases(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			project := validProject()
			project.Spec.Database.Engine = database
			project.Spec.Stack.AdminUI = "element-plus"
			project.Spec.Capabilities = []spec.CapabilitySelection{{
				Name: tenancyOwner, Version: tenancyVersion,
			}}
			project.Spec.Modules = []spec.Module{businessModule()}
			result, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if len(result.Outputs) != 69 ||
				result.CapabilityLock[tenancyOwner] != tenancyVersion {
				t.Fatalf("Generate() result = %#v", result)
			}
			for _, path := range []string{
				"src/main/java/com/scaffold/generated/demoservice/tenancy/TenancyService.java",
				"src/main/java/com/scaffold/generated/demoservice/tenancy/OrganizationController.java",
				"src/test/java/com/scaffold/generated/demoservice/tenancy/TenancyDatabaseIntegrationTest.java",
				"src/main/resources/db/migration/V000050__organization_tenancy.sql",
			} {
				if outputContent(result, path) == nil || outputOwner(result, path) != tenancyOwner {
					t.Errorf("Generate() tenancy output %s is missing or has the wrong owner", path)
				}
			}
			businessMigration := string(outputContent(
				result, "src/main/resources/db/migration/V000100__tasks_task.sql",
			))
			if !strings.Contains(businessMigration, "organization_id") ||
				!strings.Contains(businessMigration, "organizations") {
				t.Fatalf("generated business migration is not tenant scoped:\n%s", businessMigration)
			}
			client := string(outputContent(result, "web/admin/src/api/client.ts"))
			if !strings.Contains(client, "X-Organization-ID") {
				t.Fatalf("generated administration client is not tenant aware:\n%s", client)
			}
			openAPI := outputContent(result, "api/openapi.yaml")
			var contract map[string]any
			if err := yaml.Unmarshal(openAPI, &contract); err != nil {
				t.Fatalf("generated tenant OpenAPI is not valid YAML: %v\n%s", err, openAPI)
			}
			if !strings.Contains(string(openAPI), "/api/v1/organizations:") ||
				!strings.Contains(string(openAPI), "OrganizationHeader") {
				t.Fatalf("generated OpenAPI is not tenant aware:\n%s", openAPI)
			}
		})
	}
}

func TestGenerateOrganizationMemberAdministrationForBothDatabases(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			project := validProject()
			project.Spec.Database.Engine = database
			project.Spec.Stack.AdminUI = "element-plus"
			project.Spec.Capabilities = []spec.CapabilitySelection{{
				Name: tenancyOwner, Version: tenancyMembersVersion,
			}}
			project.Spec.Modules = []spec.Module{businessModule()}
			result, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if len(result.Outputs) != 79 ||
				result.CapabilityLock[tenancyOwner] != tenancyMembersVersion {
				t.Fatalf("Generate() result = %#v", result)
			}
			for _, path := range []string{
				"src/main/java/com/scaffold/generated/demoservice/tenancy/TenancyMemberService.java",
				"src/main/java/com/scaffold/generated/demoservice/tenancy/OrganizationMemberController.java",
				"src/test/java/com/scaffold/generated/demoservice/tenancy/TenancyMemberDatabaseIntegrationTest.java",
				"src/main/resources/db/migration/V000060__organization_members.sql",
				"web/admin/src/views/MembersView.vue",
			} {
				if outputContent(result, path) == nil || outputOwner(result, path) != tenancyOwner {
					t.Errorf("Generate() member output %s is missing or has the wrong owner", path)
				}
			}
			migration := string(outputContent(
				result, "src/main/resources/db/migration/V000060__organization_members.sql",
			))
			if !strings.Contains(migration, "tenancy:members:manage") ||
				!strings.Contains(migration, "token_hash") {
				t.Fatalf("generated member migration is incomplete:\n%s", migration)
			}
			application := string(outputContent(result, "web/admin/src/App.vue"))
			if !strings.Contains(application, "MembersView") ||
				!strings.Contains(application, "acceptInvitation") {
				t.Fatalf("generated administration UI lacks member workflows:\n%s", application)
			}
			openAPI := outputContent(result, "api/openapi.yaml")
			var contract map[string]any
			if err := yaml.Unmarshal(openAPI, &contract); err != nil {
				t.Fatalf("generated member OpenAPI is not valid YAML: %v\n%s", err, openAPI)
			}
			if !strings.Contains(string(openAPI),
				"/api/v1/organizations/{organizationId}/members:") ||
				!strings.Contains(string(openAPI), "tenancy:members:manage") {
				t.Fatalf("generated OpenAPI lacks member workflows:\n%s", openAPI)
			}
		})
	}
}

func TestOrganizationMemberUpgradePreservesFoundationMigration(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			versionOne := validProject()
			versionOne.Spec.Database.Engine = database
			versionOne.Spec.Capabilities = []spec.CapabilitySelection{{
				Name: tenancyOwner, Version: tenancyVersion,
			}}
			versionTwo := validProject()
			versionTwo.Spec.Database.Engine = database
			versionTwo.Spec.Capabilities = []spec.CapabilitySelection{{
				Name: tenancyOwner, Version: tenancyMembersVersion,
			}}
			base, err := New().Generate(context.Background(), versionOne)
			if err != nil {
				t.Fatalf("Generate(0.1.0) error = %v", err)
			}
			members, err := New().Generate(context.Background(), versionTwo)
			if err != nil {
				t.Fatalf("Generate(0.2.0) error = %v", err)
			}
			const migration = "src/main/resources/db/migration/V000050__organization_tenancy.sql"
			if !reflect.DeepEqual(outputContent(base, migration), outputContent(members, migration)) {
				t.Fatal("organization-tenancy 0.2.0 rewrites the already-applied 0.1.0 migration")
			}
		})
	}
}

func TestGenerateOrganizationLifecycleForBothDatabases(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			project := validProject()
			project.Spec.Database.Engine = database
			project.Spec.Stack.AdminUI = "element-plus"
			project.Spec.Capabilities = []spec.CapabilitySelection{{
				Name: tenancyOwner, Version: tenancyLifecycleVersion,
			}}
			project.Spec.Modules = []spec.Module{businessModule()}
			result, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if len(result.Outputs) != 87 ||
				result.CapabilityLock[tenancyOwner] != tenancyLifecycleVersion {
				t.Fatalf("Generate() result = %#v", result)
			}
			for _, path := range []string{
				"src/main/java/com/scaffold/generated/demoservice/tenancy/TenancyLifecycleService.java",
				"src/main/java/com/scaffold/generated/demoservice/tenancy/OrganizationLifecycleController.java",
				"src/test/java/com/scaffold/generated/demoservice/tenancy/TenancyLifecycleDatabaseIntegrationTest.java",
				"src/main/resources/db/migration/V000070__organization_lifecycle.sql",
				"web/admin/src/views/OrganizationSettingsView.vue",
			} {
				if outputContent(result, path) == nil || outputOwner(result, path) != tenancyOwner {
					t.Errorf("Generate() lifecycle output %s is missing or has the wrong owner", path)
				}
			}
			migration := string(outputContent(
				result, "src/main/resources/db/migration/V000070__organization_lifecycle.sql",
			))
			if !strings.Contains(migration, "owner_user_id") ||
				!strings.Contains(migration, "deactivated_at") {
				t.Fatalf("generated lifecycle migration is incomplete:\n%s", migration)
			}
			memberRepository := string(outputContent(result,
				"src/main/java/com/scaffold/generated/demoservice/tenancy/JdbcTenancyMemberRepository.java"))
			if !strings.Contains(memberRepository, "OWNER_PROTECTED") {
				t.Fatalf("generated member repository does not protect the owner:\n%s", memberRepository)
			}
			openAPI := outputContent(result, "api/openapi.yaml")
			var contract map[string]any
			if err := yaml.Unmarshal(openAPI, &contract); err != nil {
				t.Fatalf("generated lifecycle OpenAPI is not valid YAML: %v\n%s", err, openAPI)
			}
			if !strings.Contains(string(openAPI),
				"/api/v1/organizations/{organizationId}/reactivation:") ||
				!strings.Contains(string(openAPI), "owner_user_id") {
				t.Fatalf("generated OpenAPI lacks lifecycle workflows:\n%s", openAPI)
			}
		})
	}
}

func TestOrganizationLifecycleUpgradePreservesEarlierMigrations(t *testing.T) {
	t.Parallel()

	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			t.Parallel()
			versionTwo := validProject()
			versionTwo.Spec.Database.Engine = database
			versionTwo.Spec.Capabilities = []spec.CapabilitySelection{{
				Name: tenancyOwner, Version: tenancyMembersVersion,
			}}
			versionThree := validProject()
			versionThree.Spec.Database.Engine = database
			versionThree.Spec.Capabilities = []spec.CapabilitySelection{{
				Name: tenancyOwner, Version: tenancyLifecycleVersion,
			}}
			members, err := New().Generate(context.Background(), versionTwo)
			if err != nil {
				t.Fatalf("Generate(0.2.0) error = %v", err)
			}
			lifecycle, err := New().Generate(context.Background(), versionThree)
			if err != nil {
				t.Fatalf("Generate(0.3.0) error = %v", err)
			}
			for _, migration := range []string{
				"src/main/resources/db/migration/V000050__organization_tenancy.sql",
				"src/main/resources/db/migration/V000060__organization_members.sql",
			} {
				if !reflect.DeepEqual(
					outputContent(members, migration), outputContent(lifecycle, migration),
				) {
					t.Fatalf("organization-tenancy 0.3.0 rewrites migration %s", migration)
				}
			}
		})
	}
}

func TestGenerateDurableBackgroundJobsForBothDatabases(t *testing.T) {
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
			result, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if len(result.Outputs) != 42 || result.CapabilityLock[jobsOwner] != jobsVersion {
				t.Fatalf("Generate() result = %#v", result)
			}
			for _, path := range []string{
				"src/main/java/com/scaffold/generated/demoservice/jobs/JobService.java",
				"src/main/java/com/scaffold/generated/demoservice/jobs/JdbcJobRepository.java",
				"src/main/java/com/scaffold/generated/demoservice/jobs/WorkerRunner.java",
				"src/test/java/com/scaffold/generated/demoservice/jobs/JobDatabaseIntegrationTest.java",
				"src/main/resources/db/migration/V000200__background_jobs.sql",
			} {
				if outputContent(result, path) == nil || outputOwner(result, path) != jobsOwner {
					t.Errorf("Generate() jobs output %s is missing or has the wrong owner", path)
				}
			}
			migration := string(outputContent(
				result, "src/main/resources/db/migration/V000200__background_jobs.sql",
			))
			if !strings.Contains(migration, "skip locked") &&
				!strings.Contains(string(outputContent(result,
					"src/main/java/com/scaffold/generated/demoservice/jobs/JdbcJobRepository.java")),
					"skip locked") {
				t.Fatal("generated background jobs do not use skip-locked leasing")
			}
		})
	}
}

func TestGenerateTenantScopedBackgroundJobs(t *testing.T) {
	t.Parallel()

	project := validProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyOwner, Version: tenancyLifecycleVersion},
		{Name: jobsOwner, Version: jobsVersion},
	}
	result, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	migration := string(outputContent(
		result, "src/main/resources/db/migration/V000200__background_jobs.sql",
	))
	if !strings.Contains(migration, "organization_id text not null") ||
		!strings.Contains(migration, "references organizations") {
		t.Fatalf("generated jobs are not tenant scoped:\n%s", migration)
	}
}

func TestGenerateEmailNotificationsResolvesBackgroundJobs(t *testing.T) {
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
			result, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if len(result.Outputs) != 53 ||
				result.CapabilityLock[notificationsOwner] != notificationsVersion ||
				result.CapabilityLock[jobsOwner] != jobsVersion {
				t.Fatalf("Generate() result = %#v", result)
			}
			for _, path := range []string{
				"src/main/java/com/scaffold/generated/demoservice/notifications/NotificationService.java",
				"src/main/java/com/scaffold/generated/demoservice/notifications/SmtpEmailSender.java",
				"src/main/java/com/scaffold/generated/demoservice/notifications/EmailNotificationHandler.java",
				"src/test/java/com/scaffold/generated/demoservice/notifications/NotificationDatabaseIntegrationTest.java",
			} {
				if outputContent(result, path) == nil || outputOwner(result, path) != notificationsOwner {
					t.Errorf("Generate() notification output %s is missing or has the wrong owner", path)
				}
			}
			pom := string(outputContent(result, "pom.xml"))
			if !strings.Contains(pom, "spring-boot-starter-mail") {
				t.Fatalf("generated notification project lacks its mail dependency:\n%s", pom)
			}
			openAPI := string(outputContent(result, "api/openapi.yaml"))
			if strings.Contains(openAPI, "notifications.email.deliver") {
				t.Fatal("generated notification capability exposes an arbitrary email endpoint")
			}
		})
	}
}

func TestGenerateTenantScopedEmailNotifications(t *testing.T) {
	t.Parallel()

	project := validProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyOwner, Version: tenancyLifecycleVersion},
		{Name: notificationsOwner, Version: notificationsVersion},
	}
	result, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(result.Outputs) != 78 {
		t.Fatalf("Generate() output count = %d, want 78", len(result.Outputs))
	}
	databaseTest := string(outputContent(
		result,
		"src/test/java/com/scaffold/generated/demoservice/notifications/NotificationDatabaseIntegrationTest.java",
	))
	if !strings.Contains(databaseTest, "organization.id()") {
		t.Fatalf("generated notification integration does not use tenant scope:\n%s", databaseTest)
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
			result, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if len(result.Outputs) != 43 || result.CapabilityLock[filesOwner] != filesVersion {
				t.Fatalf("Generate() result = %#v", result)
			}
			for _, path := range []string{
				"src/main/java/com/scaffold/generated/demoservice/files/FileAssetService.java",
				"src/main/java/com/scaffold/generated/demoservice/files/LocalBlobStore.java",
				"src/main/java/com/scaffold/generated/demoservice/files/FileAssetController.java",
				"src/test/java/com/scaffold/generated/demoservice/files/FileAssetDatabaseIntegrationTest.java",
				"src/main/resources/db/migration/V000210__file_assets.sql",
			} {
				if outputContent(result, path) == nil || outputOwner(result, path) != filesOwner {
					t.Errorf("Generate() file output %s is missing or has the wrong owner", path)
				}
			}
			migration := string(outputContent(
				result, "src/main/resources/db/migration/V000210__file_assets.sql",
			))
			if !strings.Contains(migration, "files:create") ||
				!strings.Contains(migration, "storage_key") ||
				!strings.Contains(migration, "10485760") {
				t.Fatalf("generated file migration is incomplete:\n%s", migration)
			}
			openAPI := string(outputContent(result, "api/openapi.yaml"))
			var contract map[string]any
			if err := yaml.Unmarshal([]byte(openAPI), &contract); err != nil {
				t.Fatalf("generated file OpenAPI is not valid YAML: %v\n%s", err, openAPI)
			}
			if !strings.Contains(openAPI, "/api/v1/files/{id}/content:") ||
				strings.Contains(openAPI, "storage_key") {
				t.Fatalf("generated file OpenAPI is incomplete or leaks storage keys:\n%s", openAPI)
			}
		})
	}
}

func TestGenerateTenantFileAdministration(t *testing.T) {
	t.Parallel()

	project := validProject()
	project.Spec.Stack.AdminUI = "element-plus"
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyOwner, Version: tenancyLifecycleVersion},
		{Name: filesOwner, Version: filesVersion},
	}
	result, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(result.Outputs) != 90 {
		t.Fatalf("Generate() output count = %d, want 90", len(result.Outputs))
	}
	view := "web/admin/src/views/FilesView.vue"
	if outputContent(result, view) == nil || outputOwner(result, view) != filesOwner {
		t.Fatalf("Generate() file administration view is missing or has the wrong owner")
	}
	application := string(outputContent(result, "web/admin/src/App.vue"))
	if !strings.Contains(application, "FilesView") {
		t.Fatalf("generated administration application lacks file workflows:\n%s", application)
	}
	migration := string(outputContent(
		result, "src/main/resources/db/migration/V000210__file_assets.sql",
	))
	if !strings.Contains(migration, "organization_id text not null") ||
		!strings.Contains(migration, "references organizations") {
		t.Fatalf("generated file assets are not tenant scoped:\n%s", migration)
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
			result, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if len(result.Outputs) != 39 || result.CapabilityLock[cacheOwner] != cacheVersion {
				t.Fatalf("Generate() result = %#v", result)
			}
			for _, path := range []string{
				"src/main/java/com/scaffold/generated/demoservice/cache/CacheService.java",
				"src/main/java/com/scaffold/generated/demoservice/cache/JdbcCacheRepository.java",
				"src/test/java/com/scaffold/generated/demoservice/cache/CacheDatabaseIntegrationTest.java",
				"src/main/resources/db/migration/V000220__application_cache.sql",
			} {
				if outputContent(result, path) == nil || outputOwner(result, path) != cacheOwner {
					t.Errorf("Generate() cache output %s is missing or has the wrong owner", path)
				}
			}
			migration := string(outputContent(
				result, "src/main/resources/db/migration/V000220__application_cache.sql",
			))
			if !strings.Contains(migration, "application_cache") ||
				!strings.Contains(migration, "expires_at") {
				t.Fatalf("generated cache migration is incomplete:\n%s", migration)
			}
			if strings.Contains(string(outputContent(result, "api/openapi.yaml")), "application_cache") {
				t.Fatal("generated application cache exposes a generic HTTP endpoint")
			}
		})
	}
}

func TestGenerateTenantScopedApplicationCache(t *testing.T) {
	t.Parallel()

	project := validProject()
	project.Spec.Capabilities = []spec.CapabilitySelection{
		{Name: tenancyOwner, Version: tenancyLifecycleVersion},
		{Name: cacheOwner, Version: cacheVersion},
	}
	result, err := New().Generate(context.Background(), project)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	if len(result.Outputs) != 64 {
		t.Fatalf("Generate() output count = %d, want 64", len(result.Outputs))
	}
	migration := string(outputContent(
		result, "src/main/resources/db/migration/V000220__application_cache.sql",
	))
	if !strings.Contains(migration, "organization_id text not null") ||
		!strings.Contains(migration, "references organizations") {
		t.Fatalf("generated cache is not tenant scoped:\n%s", migration)
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
			result, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if len(result.Outputs) != 51 ||
				result.CapabilityLock[jobAdminOwner] != jobAdminVersion ||
				result.CapabilityLock[jobsOwner] != jobsVersion {
				t.Fatalf("Generate() result = %#v", result)
			}
			for _, path := range []string{
				"src/main/java/com/scaffold/generated/demoservice/jobadmin/JobAdministrationItem.java",
				"src/main/java/com/scaffold/generated/demoservice/jobadmin/JdbcJobAdministrationRepository.java",
				"src/main/java/com/scaffold/generated/demoservice/jobadmin/JobAdministrationController.java",
				"src/test/java/com/scaffold/generated/demoservice/jobadmin/JobAdministrationDatabaseIntegrationTest.java",
				"src/main/resources/db/migration/V000230__job_administration.sql",
			} {
				if outputContent(result, path) == nil || outputOwner(result, path) != jobAdminOwner {
					t.Errorf("Generate() job administration output %s is missing or has the wrong owner", path)
				}
			}
			item := string(outputContent(result,
				"src/main/java/com/scaffold/generated/demoservice/jobadmin/JobAdministrationItem.java"))
			if strings.Contains(item, "payload") || strings.Contains(item, "scopeKey") ||
				strings.Contains(item, "dedupeKey") {
				t.Fatalf("generated administration DTO exposes private job data:\n%s", item)
			}
			migration := string(outputContent(
				result, "src/main/resources/db/migration/V000230__job_administration.sql",
			))
			if !strings.Contains(migration, "jobs:read") ||
				!strings.Contains(migration, "jobs:manage") {
				t.Fatalf("generated job administration permissions are incomplete:\n%s", migration)
			}
			openAPI := string(outputContent(result, "api/openapi.yaml"))
			var contract map[string]any
			if err := yaml.Unmarshal([]byte(openAPI), &contract); err != nil {
				t.Fatalf("generated job administration OpenAPI is not valid YAML: %v\n%s", err, openAPI)
			}
			if !strings.Contains(openAPI, "/api/v1/jobs/{id}/retry:") ||
				strings.Contains(openAPI, "payload_json") || strings.Contains(openAPI, "dedupe_key") {
				t.Fatalf("generated job administration contract is unsafe:\n%s", openAPI)
			}
		})
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
			result, err := New().Generate(context.Background(), project)
			if err != nil {
				t.Fatalf("Generate() error = %v", err)
			}
			if len(result.Outputs) != 37 ||
				result.CapabilityLock[observabilityOwner] != observabilityVersion {
				t.Fatalf("Generate() result = %#v", result)
			}
			for _, path := range []string{
				"src/main/java/com/scaffold/generated/demoservice/observability/HttpObservabilityFilter.java",
				"src/main/java/com/scaffold/generated/demoservice/observability/MetricsController.java",
				"src/main/java/com/scaffold/generated/demoservice/http/BoundedReadinessProbe.java",
				"src/test/java/com/scaffold/generated/demoservice/observability/HttpObservabilityFilterTest.java",
				"src/test/java/com/scaffold/generated/demoservice/http/BoundedReadinessProbeTest.java",
				"src/test/java/com/scaffold/generated/demoservice/http/ObservabilityDatabaseIntegrationTest.java",
			} {
				if outputContent(result, path) == nil || outputOwner(result, path) != observabilityOwner {
					t.Errorf("Generate() observability output %s is missing or has the wrong owner", path)
				}
			}
			openAPI := outputContent(result, "api/openapi.yaml")
			var contract map[string]any
			if err := yaml.Unmarshal(openAPI, &contract); err != nil {
				t.Fatalf("generated observability OpenAPI is not valid YAML: %v\n%s", err, openAPI)
			}
			if !strings.Contains(string(openAPI), "/metrics:") ||
				!strings.Contains(string(openAPI), "ready") {
				t.Fatalf("generated observability OpenAPI is incomplete:\n%s", openAPI)
			}
			filter := string(outputContent(result,
				"src/main/java/com/scaffold/generated/demoservice/observability/HttpObservabilityFilter.java"))
			if strings.Contains(filter, "getQueryString") || strings.Contains(filter, "getParameter") {
				t.Fatalf("generated access logging reads unsafe request data:\n%s", filter)
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
