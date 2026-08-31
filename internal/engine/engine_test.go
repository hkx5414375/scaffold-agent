package engine

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/artifactstore"
	"github.com/hkx5414375/scaffold-agent/internal/change"
	"github.com/hkx5414375/scaffold-agent/internal/plan"
	"github.com/hkx5414375/scaffold-agent/internal/result"
)

func TestValidateBlueprint(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeBlueprint(t, root, `
api_version: scaffold-agent.io/v1alpha1
kind: Project
metadata:
  name: demo
spec:
  stack:
    backend: go
  database:
    engine: postgresql
  auth:
    modes: [session, token]
  modules:
    - name: tasks
      entities:
        - name: task
          fields:
            - {name: title, type: string, required: true, unique: true}
            - {name: description, type: text}
            - {name: done, type: bool, required: true}
            - {name: priority, type: int64, required: true}
            - {name: due_at, type: datetime}
      permissions:
        - {code: "tasks:task:create"}
        - {code: "tasks:task:read"}
        - {code: "tasks:task:update"}
        - {code: "tasks:task:delete"}
`)
	envelope := New("test").Validate(context.Background(), ValidateInput{ProjectRoot: root, BlueprintPath: "scaffold.yaml"})
	if envelope.Status != result.StatusOK {
		t.Fatalf("Validate() = %#v, want ok", envelope)
	}
	data := envelope.Data.(validationData)
	if data.Backend != "go" || data.BlueprintHash == "" {
		t.Fatalf("Validate() data = %#v", data)
	}
}

func TestPreviewRequiresStoredArtifactAndApplyToken(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	artifact, err := change.Build(root, plan.ActionCreate, testHash("a"), nil, []change.Output{{Path: "file.txt", Owner: "test", Content: []byte("content")}})
	if err != nil {
		t.Fatalf("change.Build() error = %v", err)
	}
	if err := artifactstore.Save(root, artifact); err != nil {
		t.Fatalf("artifactstore.Save() error = %v", err)
	}
	engine := New("test")
	preview := engine.Preview(context.Background(), PreviewInput{ProjectRoot: root, PlanID: artifact.Plan.ID})
	if preview.Status != result.StatusOK {
		t.Fatalf("Preview() = %#v, want ok", preview)
	}
	data := preview.Data.(previewData)
	invalid := engine.Apply(context.Background(), ApplyInput{ProjectRoot: root, PlanID: artifact.Plan.ID, ApplyToken: "invalid"})
	if invalid.Status != result.StatusError {
		t.Fatalf("Apply(invalid token) status = %q, want error", invalid.Status)
	}
	applied := engine.Apply(context.Background(), ApplyInput{ProjectRoot: root, PlanID: artifact.Plan.ID, ApplyToken: data.ApplyToken})
	if applied.Status != result.StatusOK {
		t.Fatalf("Apply() = %#v, want ok", applied)
	}
}

func TestVerifyStoresPageableFinding(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	artifact, err := change.Build(root, plan.ActionCreate, testHash("a"), nil, []change.Output{{Path: "file.txt", Owner: "test", Content: []byte("content")}})
	if err != nil {
		t.Fatalf("change.Build() error = %v", err)
	}
	if _, err := change.Apply(artifact); err != nil {
		t.Fatalf("change.Apply() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("changed"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	envelope := New("test").Verify(context.Background(), VerifyInput{ProjectRoot: root})
	if envelope.Status != result.StatusError || envelope.ResultID == "" {
		t.Fatalf("Verify() = %#v, want stored error result", envelope)
	}
	data := envelope.Data.(resultPageData)
	if len(data.Items) != 1 {
		t.Fatalf("Verify() findings = %d, want 1", len(data.Items))
	}
}

func TestGoPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", "", "", 47, os.Getenv("SCAFFOLD_AGENT_RUN_ADMIN_BUILD") == "1")
}

func TestGoMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", "", "", 48, false)
}

func TestGoTenantPostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", tenancySelectionV1, "0.1.0", 53, os.Getenv("SCAFFOLD_AGENT_RUN_ADMIN_BUILD") == "1")
}

func TestGoTenantMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", tenancySelectionV1, "0.1.0", 54, false)
}

func TestGoTenantMembersPostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", tenancySelectionV2, "0.2.0", 60, os.Getenv("SCAFFOLD_AGENT_RUN_ADMIN_BUILD") == "1")
}

func TestGoTenantMembersMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", tenancySelectionV2, "0.2.0", 61, false)
}

func TestGoTenantLifecyclePostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", tenancySelectionV3, "0.3.0", 67, os.Getenv("SCAFFOLD_AGENT_RUN_ADMIN_BUILD") == "1")
}

func TestGoTenantLifecycleMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", tenancySelectionV3, "0.3.0", 68, false)
}

func TestGoTenantJobsPostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", tenancyJobsSelection, "0.3.0", 75, false)
}

func TestGoTenantJobsMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", tenancyJobsSelection, "0.3.0", 76, false)
}

func TestGoTenantNotificationsPostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", tenancyNotificationsSelection, "0.3.0", 81, false)
}

func TestGoTenantNotificationsMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", tenancyNotificationsSelection, "0.3.0", 82, false)
}

func TestGoJobsWithoutTenancyPostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", jobsSelection, "", 55, false)
}

func TestGoJobsWithoutTenancyMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", jobsSelection, "", 56, false)
}

func TestGoNotificationsWithoutTenancyPostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", notificationsSelection, "", 61, false)
}

func TestGoNotificationsWithoutTenancyMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", notificationsSelection, "", 62, false)
}

func TestGoTenantFilesPostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", tenancyFilesSelection, "0.3.0", 76, os.Getenv("SCAFFOLD_AGENT_RUN_ADMIN_BUILD") == "1")
}

func TestGoTenantFilesMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", tenancyFilesSelection, "0.3.0", 77, false)
}

func TestGoFilesWithoutTenancyPostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", filesSelection, "", 56, os.Getenv("SCAFFOLD_AGENT_RUN_ADMIN_BUILD") == "1")
}

func TestGoFilesWithoutTenancyMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", filesSelection, "", 57, false)
}

func TestGoTenantCachePostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", tenancyCacheSelection, "0.3.0", 71, false)
}

func TestGoTenantCacheMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", tenancyCacheSelection, "0.3.0", 72, false)
}

func TestGoCacheWithoutTenancyPostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", cacheSelection, "", 51, false)
}

func TestGoCacheWithoutTenancyMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", cacheSelection, "", 52, false)
}

const tenancySelectionV1 = `  capabilities:
    - name: organization-tenancy
      version: 0.1.0
`

const tenancySelectionV2 = `  capabilities:
    - name: organization-tenancy
      version: 0.2.0
`

const tenancySelectionV3 = `  capabilities:
    - name: organization-tenancy
      version: 0.3.0
`

const tenancyJobsSelection = `  capabilities:
    - name: organization-tenancy
      version: 0.3.0
    - name: background-jobs
      version: 0.1.0
`

const tenancyNotificationsSelection = `  capabilities:
    - name: organization-tenancy
      version: 0.3.0
    - name: notifications
      version: 0.1.0
`

const jobsSelection = `  capabilities:
    - name: background-jobs
      version: 0.1.0
`

const notificationsSelection = `  capabilities:
    - name: notifications
      version: 0.1.0
`

const tenancyFilesSelection = `  capabilities:
    - name: organization-tenancy
      version: 0.3.0
    - name: file-assets
      version: 0.1.0
`

const filesSelection = `  capabilities:
    - name: file-assets
      version: 0.1.0
`

const tenancyCacheSelection = `  capabilities:
    - name: organization-tenancy
      version: 0.3.0
    - name: application-cache
      version: 0.1.0
`

const cacheSelection = `  capabilities:
    - name: application-cache
      version: 0.1.0
`

func runGeneratedGoEndToEnd(t *testing.T, database, adminUI, capabilities, expectedTenancyVersion string, wantChanges int, runAdminBuild bool) {
	t.Helper()
	root := t.TempDir()
	ctx := context.Background()
	blueprint := `
api_version: scaffold-agent.io/v1alpha1
kind: Project
metadata:
  name: generated-demo
spec:
  stack:
    backend: go
    admin_ui: ADMIN_UI
    storefront: none
  database:
    engine: DATABASE_ENGINE
  auth:
    modes: [session, token]
CAPABILITIES
  modules:
    - name: tasks
      entities:
        - name: task
          fields:
            - {name: title, type: string, required: true, unique: true}
            - {name: description, type: text}
            - {name: done, type: bool, required: true}
            - {name: priority, type: int64, required: true}
            - {name: due_at, type: datetime}
      permissions:
        - {code: "tasks:task:create"}
        - {code: "tasks:task:read"}
        - {code: "tasks:task:update"}
        - {code: "tasks:task:delete"}
`
	blueprint = strings.ReplaceAll(blueprint, "ADMIN_UI", adminUI)
	blueprint = strings.ReplaceAll(blueprint, "DATABASE_ENGINE", database)
	blueprint = strings.ReplaceAll(blueprint, "CAPABILITIES", capabilities)
	writeBlueprint(t, root, blueprint)
	application := New("test")
	planned := application.Plan(ctx, PlanInput{ProjectRoot: root, BlueprintPath: "scaffold.yaml", Action: plan.ActionCreate})
	if planned.Status != result.StatusOK {
		t.Fatalf("Plan() = %#v, want ok", planned)
	}
	plannedData := planned.Data.(planData)
	if plannedData.ChangeCount != wantChanges || plannedData.CapabilityLock["go-service"] != "0.3.0" || plannedData.CapabilityLock["go-crud"] != "0.3.0" {
		t.Fatalf("Plan() data = %#v", plannedData)
	}
	if adminUI == "element-plus" && plannedData.CapabilityLock["vue-admin"] != "0.1.0" {
		t.Fatalf("Plan() administration lock = %#v", plannedData.CapabilityLock)
	}
	if strings.Contains(capabilities, "organization-tenancy") && plannedData.CapabilityLock["organization-tenancy"] != expectedTenancyVersion {
		t.Fatalf("Plan() tenancy lock = %#v", plannedData.CapabilityLock)
	}
	if strings.Contains(capabilities, "background-jobs") && plannedData.CapabilityLock["background-jobs"] != "0.1.0" {
		t.Fatalf("Plan() background jobs lock = %#v", plannedData.CapabilityLock)
	}
	if strings.Contains(capabilities, "notifications") {
		if plannedData.CapabilityLock["notifications"] != "0.1.0" || plannedData.CapabilityLock["background-jobs"] != "0.1.0" {
			t.Fatalf("Plan() notifications lock = %#v", plannedData.CapabilityLock)
		}
	}
	if strings.Contains(capabilities, "file-assets") && plannedData.CapabilityLock["file-assets"] != "0.1.0" {
		t.Fatalf("Plan() file assets lock = %#v", plannedData.CapabilityLock)
	}
	if strings.Contains(capabilities, "application-cache") && plannedData.CapabilityLock["application-cache"] != "0.1.0" {
		t.Fatalf("Plan() application cache lock = %#v", plannedData.CapabilityLock)
	}
	previewed := application.Preview(ctx, PreviewInput{ProjectRoot: root, PlanID: plannedData.PlanID})
	if previewed.Status != result.StatusOK {
		t.Fatalf("Preview() = %#v, want ok", previewed)
	}
	previewedData := previewed.Data.(previewData)
	applied := application.Apply(ctx, ApplyInput{ProjectRoot: root, PlanID: plannedData.PlanID, ApplyToken: previewedData.ApplyToken})
	if applied.Status != result.StatusOK {
		t.Fatalf("Apply() = %#v, want ok", applied)
	}
	commands := [][]string{
		{"go", "mod", "verify"},
		{"go", "test", "./..."},
		{"go", "vet", "./..."},
	}
	for _, arguments := range commands {
		command := exec.CommandContext(ctx, arguments[0], arguments[1:]...)
		command.Dir = root
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("generated %s failed: %v\n%s", strings.Join(arguments, " "), err, output)
		}
	}
	if runAdminBuild {
		adminRoot := filepath.Join(root, "web", "admin")
		adminCommands := [][]string{
			{"npm", "ci", "--ignore-scripts", "--no-audit", "--no-fund"},
			{"npm", "run", "lint"},
			{"npm", "test"},
			{"npm", "run", "build"},
			{"npm", "run", "format:check"},
		}
		for _, arguments := range adminCommands {
			command := exec.CommandContext(ctx, arguments[0], arguments[1:]...)
			command.Dir = adminRoot
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("generated admin %s failed: %v\n%s", strings.Join(arguments, " "), err, output)
			}
		}
	}
	verified := application.Verify(ctx, VerifyInput{ProjectRoot: root})
	if verified.Status != result.StatusOK || !strings.Contains(verified.Summary, "no findings") {
		t.Fatalf("Verify() = %#v, want no findings", verified)
	}
}

func writeBlueprint(t *testing.T, root, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "scaffold.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func testHash(character string) string {
	value := ""
	for range 64 {
		value += character
	}
	return value
}
