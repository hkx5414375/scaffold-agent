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
			project.Spec.Capabilities = []spec.CapabilitySelection{{Name: "observability", Version: "0.1.0"}}
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
