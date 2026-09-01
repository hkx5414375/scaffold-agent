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

func TestGoTenantJobAdministrationPostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", tenancyJobAdminSelection, "0.3.0", 82, os.Getenv("SCAFFOLD_AGENT_RUN_ADMIN_BUILD") == "1")
}

func TestGoTenantJobAdministrationMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", tenancyJobAdminSelection, "0.3.0", 83, false)
}

func TestGoJobAdministrationWithoutTenancyPostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", jobAdminSelection, "", 62, os.Getenv("SCAFFOLD_AGENT_RUN_ADMIN_BUILD") == "1")
}

func TestGoJobAdministrationWithoutTenancyMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", jobAdminSelection, "", 63, false)
}

func TestGoNotificationsAndJobAdministrationCompose(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", notificationsJobAdminSelection, "0.3.0", 88, false)
}

func TestGoTenantObservabilityPostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", tenancyObservabilitySelection, "0.3.0", 69, false)
}

func TestGoTenantObservabilityMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", tenancyObservabilitySelection, "0.3.0", 70, false)
}

func TestGoObservabilityWithoutTenancyPostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", observabilitySelection, "", 49, false)
}

func TestGoObservabilityWithoutTenancyMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", observabilitySelection, "", 50, false)
}

func TestGoTenantCSVImportExportPostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", tenancyCSVTransferSelection, "0.3.0", 73, os.Getenv("SCAFFOLD_AGENT_RUN_ADMIN_BUILD") == "1")
}

func TestGoTenantCSVImportExportMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", tenancyCSVTransferSelection, "0.3.0", 74, false)
}

func TestGoCSVImportExportWithoutTenancyPostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", csvTransferSelection, "", 53, false)
}

func TestGoCSVImportExportWithoutTenancyMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", csvTransferSelection, "", 54, false)
}

func TestGoTenantApprovalWorkflowsPostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", tenancyApprovalsSelection, "0.3.0", 74, os.Getenv("SCAFFOLD_AGENT_RUN_ADMIN_BUILD") == "1")
}

func TestGoTenantApprovalWorkflowsMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", tenancyApprovalsSelection, "0.3.0", 75, false)
}

func TestGoApprovalWorkflowsWithoutTenancyPostgreSQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "postgresql", "element-plus", approvalsSelection, "", 54, false)
}

func TestGoApprovalWorkflowsWithoutTenancyMySQLPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedGoEndToEnd(t, "mysql", "element-plus", approvalsSelection, "", 55, false)
}

func TestJavaPostgreSQLIdentityPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaReference(t, "postgresql", false, false, "")
}

func TestJavaMySQLIdentityPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaReference(t, "mysql", false, false, "")
}

func TestJavaPostgreSQLCRUDPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaReference(t, "postgresql", true, false, "")
}

func TestJavaMySQLCRUDPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaReference(t, "mysql", true, false, "")
}

func TestJavaPostgreSQLAdminPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaReference(t, "postgresql", true, true, "")
}

func TestJavaMySQLAdminPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaReference(t, "mysql", true, true, "")
}

func TestJavaPostgreSQLTenancyPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaReference(t, "postgresql", true, true, "0.1.0")
}

func TestJavaMySQLTenancyPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaReference(t, "mysql", true, true, "0.1.0")
}

func TestJavaPostgreSQLTenancyMembersPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaReference(t, "postgresql", true, true, "0.2.0")
}

func TestJavaMySQLTenancyMembersPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaReference(t, "mysql", true, true, "0.2.0")
}

func TestJavaPostgreSQLTenancyLifecyclePlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaReference(t, "postgresql", true, true, "0.3.0")
}

func TestJavaMySQLTenancyLifecyclePlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaReference(t, "mysql", true, true, "0.3.0")
}

func TestJavaPostgreSQLJobsPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaJobsReference(t, "postgresql", true, false, "")
}

func TestJavaPostgreSQLTenantJobsPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaJobsReference(t, "postgresql", true, true, "0.3.0")
}

func TestJavaMySQLTenantJobsPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaJobsReference(t, "mysql", true, true, "0.3.0")
}

func TestJavaPostgreSQLNotificationsPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaNotificationsReference(t, "postgresql", true, false, "")
}

func TestJavaPostgreSQLTenantNotificationsPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaNotificationsReference(t, "postgresql", true, true, "0.3.0")
}

func TestJavaMySQLTenantNotificationsPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaNotificationsReference(t, "mysql", true, true, "0.3.0")
}

func TestJavaPostgreSQLFilesPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaFilesReference(t, "postgresql", true, true, "")
}

func TestJavaPostgreSQLTenantFilesPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaFilesReference(t, "postgresql", true, true, "0.3.0")
}

func TestJavaMySQLTenantFilesPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaFilesReference(t, "mysql", true, true, "0.3.0")
}

func TestJavaPostgreSQLCachePlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaCacheReference(t, "postgresql", true, false, "")
}

func TestJavaPostgreSQLTenantCachePlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaCacheReference(t, "postgresql", true, false, "0.3.0")
}

func TestJavaMySQLTenantCachePlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaCacheReference(t, "mysql", true, false, "0.3.0")
}

func TestJavaPostgreSQLJobAdministrationPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaJobAdministrationReference(t, "postgresql", true, true, "")
}

func TestJavaPostgreSQLTenantJobAdministrationPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaJobAdministrationReference(t, "postgresql", true, true, "0.3.0")
}

func TestJavaMySQLTenantJobAdministrationPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaJobAdministrationReference(t, "mysql", true, true, "0.3.0")
}

func TestJavaPostgreSQLPlatformCapabilitiesCompose(t *testing.T) {
	runGeneratedJavaCapabilities(
		t, "postgresql", true, true, "0.3.0", false, true, true, true, true, true, true,
	)
}

func TestJavaPostgreSQLObservabilityPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaObservabilityReference(t, "postgresql", true, true, "")
}

func TestJavaMySQLObservabilityPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaObservabilityReference(t, "mysql", true, false, "")
}

func TestJavaPostgreSQLCSVImportExportPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaCSVTransferReference(t, "postgresql", true, "")
}

func TestJavaPostgreSQLTenantCSVImportExportPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaCSVTransferReference(t, "postgresql", true, "0.3.0")
}

func TestJavaMySQLTenantCSVImportExportPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedJavaCSVTransferReference(t, "mysql", true, "0.3.0")
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

const tenancyJobAdminSelection = `  capabilities:
    - name: organization-tenancy
      version: 0.3.0
    - name: job-administration
      version: 0.1.0
`

const jobAdminSelection = `  capabilities:
    - name: job-administration
      version: 0.1.0
`

const notificationsJobAdminSelection = `  capabilities:
    - name: organization-tenancy
      version: 0.3.0
    - name: notifications
      version: 0.1.0
    - name: job-administration
      version: 0.1.0
`

const tenancyObservabilitySelection = `  capabilities:
    - name: organization-tenancy
      version: 0.3.0
    - name: observability
      version: 0.1.0
`

const observabilitySelection = `  capabilities:
    - name: observability
      version: 0.1.0
`

const tenancyCSVTransferSelection = `  capabilities:
    - name: organization-tenancy
      version: 0.3.0
    - name: csv-import-export
      version: 0.1.0
`

const csvTransferSelection = `  capabilities:
    - name: csv-import-export
      version: 0.1.0
`

const tenancyApprovalsSelection = `  capabilities:
    - name: organization-tenancy
      version: 0.3.0
    - name: approval-workflows
      version: 0.1.0
`

const approvalsSelection = `  capabilities:
    - name: approval-workflows
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
WORKFLOWS
`
	blueprint = strings.ReplaceAll(blueprint, "ADMIN_UI", adminUI)
	blueprint = strings.ReplaceAll(blueprint, "DATABASE_ENGINE", database)
	blueprint = strings.ReplaceAll(blueprint, "CAPABILITIES", capabilities)
	workflows := ""
	if strings.Contains(capabilities, "approval-workflows") {
		workflows = `      workflows:
        - name: approval
          states: [pending, approved, rejected, cancelled]`
	}
	blueprint = strings.ReplaceAll(blueprint, "WORKFLOWS", workflows)
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
	if adminUI == "element-plus" && plannedData.CapabilityLock["vue-admin"] != "0.2.0" {
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
	if strings.Contains(capabilities, "job-administration") {
		if plannedData.CapabilityLock["job-administration"] != "0.1.0" || plannedData.CapabilityLock["background-jobs"] != "0.1.0" {
			t.Fatalf("Plan() job administration lock = %#v", plannedData.CapabilityLock)
		}
	}
	if strings.Contains(capabilities, "observability") && plannedData.CapabilityLock["observability"] != "0.1.0" {
		t.Fatalf("Plan() observability lock = %#v", plannedData.CapabilityLock)
	}
	if strings.Contains(capabilities, "csv-import-export") && plannedData.CapabilityLock["csv-import-export"] != "0.1.0" {
		t.Fatalf("Plan() CSV import/export lock = %#v", plannedData.CapabilityLock)
	}
	if strings.Contains(capabilities, "approval-workflows") && plannedData.CapabilityLock["approval-workflows"] != "0.1.0" {
		t.Fatalf("Plan() approval workflows lock = %#v", plannedData.CapabilityLock)
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

func runGeneratedJavaReference(
	t *testing.T, database string, business, admin bool, tenancyVersion string,
) {
	t.Helper()
	runGeneratedJavaCapabilities(t, database, business, admin, tenancyVersion, false, false, false, false, false, false, false)
}

func runGeneratedJavaJobsReference(
	t *testing.T, database string, business, admin bool, tenancyVersion string,
) {
	t.Helper()
	runGeneratedJavaCapabilities(t, database, business, admin, tenancyVersion, true, false, false, false, false, false, false)
}

func runGeneratedJavaNotificationsReference(
	t *testing.T, database string, business, admin bool, tenancyVersion string,
) {
	t.Helper()
	runGeneratedJavaCapabilities(t, database, business, admin, tenancyVersion, false, true, false, false, false, false, false)
}

func runGeneratedJavaFilesReference(
	t *testing.T, database string, business, admin bool, tenancyVersion string,
) {
	t.Helper()
	runGeneratedJavaCapabilities(t, database, business, admin, tenancyVersion, false, false, true, false, false, false, false)
}

func runGeneratedJavaCacheReference(
	t *testing.T, database string, business, admin bool, tenancyVersion string,
) {
	t.Helper()
	runGeneratedJavaCapabilities(t, database, business, admin, tenancyVersion, false, false, false, true, false, false, false)
}

func runGeneratedJavaJobAdministrationReference(
	t *testing.T, database string, business, admin bool, tenancyVersion string,
) {
	t.Helper()
	runGeneratedJavaCapabilities(t, database, business, admin, tenancyVersion, false, false, false, false, true, false, false)
}

func runGeneratedJavaObservabilityReference(
	t *testing.T, database string, business, admin bool, tenancyVersion string,
) {
	t.Helper()
	runGeneratedJavaCapabilities(t, database, business, admin, tenancyVersion, false, false, false, false, false, true, false)
}

func runGeneratedJavaCSVTransferReference(
	t *testing.T, database string, admin bool, tenancyVersion string,
) {
	t.Helper()
	runGeneratedJavaCapabilities(t, database, true, admin, tenancyVersion, false, false, false, false, false, false, true)
}

func runGeneratedJavaCapabilities(
	t *testing.T,
	database string,
	business bool,
	admin bool,
	tenancyVersion string,
	jobs bool,
	notifications bool,
	files bool,
	cache bool,
	jobAdmin bool,
	observability bool,
	csvTransfer bool,
) {
	t.Helper()
	root := t.TempDir()
	ctx := context.Background()
	blueprint := `
api_version: scaffold-agent.io/v1alpha1
kind: Project
metadata:
  name: generated-java
spec:
  stack:
    backend: java
    admin_ui: ADMIN_UI
    storefront: none
  database:
    engine: DATABASE_ENGINE
  auth:
    modes: [session, token]
CAPABILITIES
MODULES
`
	blueprint = strings.ReplaceAll(blueprint, "DATABASE_ENGINE", database)
	adminUI := "none"
	if admin {
		adminUI = "element-plus"
	}
	blueprint = strings.ReplaceAll(blueprint, "ADMIN_UI", adminUI)
	capabilities := ""
	if tenancyVersion != "" {
		capabilities = `  capabilities:
    - name: organization-tenancy
      version: TENANCY_VERSION
`
		capabilities = strings.ReplaceAll(capabilities, "TENANCY_VERSION", tenancyVersion)
	}
	if jobs {
		if capabilities == "" {
			capabilities = "  capabilities:\n"
		}
		capabilities += `    - name: background-jobs
      version: 0.1.0
`
	}
	if notifications {
		if capabilities == "" {
			capabilities = "  capabilities:\n"
		}
		capabilities += `    - name: notifications
      version: 0.1.0
`
	}
	if files {
		if capabilities == "" {
			capabilities = "  capabilities:\n"
		}
		capabilities += `    - name: file-assets
      version: 0.1.0
`
	}
	if cache {
		if capabilities == "" {
			capabilities = "  capabilities:\n"
		}
		capabilities += `    - name: application-cache
      version: 0.1.0
`
	}
	if jobAdmin {
		if capabilities == "" {
			capabilities = "  capabilities:\n"
		}
		capabilities += `    - name: job-administration
      version: 0.1.0
`
	}
	if observability {
		if capabilities == "" {
			capabilities = "  capabilities:\n"
		}
		capabilities += `    - name: observability
      version: 0.1.0
`
	}
	if csvTransfer {
		if capabilities == "" {
			capabilities = "  capabilities:\n"
		}
		capabilities += `    - name: csv-import-export
      version: 0.1.0
`
	}
	blueprint = strings.ReplaceAll(blueprint, "CAPABILITIES", capabilities)
	modules := ""
	if business {
		modules = `  modules:
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
	}
	blueprint = strings.ReplaceAll(blueprint, "MODULES", modules)
	writeBlueprint(t, root, blueprint)
	application := New("test")
	planned := application.Plan(ctx, PlanInput{ProjectRoot: root, BlueprintPath: "scaffold.yaml", Action: plan.ActionCreate})
	if planned.Status != result.StatusOK {
		t.Fatalf("Plan() = %#v, want ok", planned)
	}
	plannedData := planned.Data.(planData)
	wantChanges := 32
	if business {
		wantChanges = 41
	}
	if admin {
		wantChanges += 20
	}
	if tenancyVersion != "" {
		wantChanges += 9
	}
	if tenancyVersion == "0.2.0" || tenancyVersion == "0.3.0" {
		wantChanges += 9
		if admin {
			wantChanges++
		}
	}
	if tenancyVersion == "0.3.0" {
		wantChanges += 7
		if admin {
			wantChanges++
		}
	}
	resolvedJobs := jobs || notifications || jobAdmin
	if resolvedJobs {
		wantChanges += 11
	}
	if notifications {
		wantChanges += 11
	}
	if files {
		wantChanges += 12
		if admin {
			wantChanges++
		}
	}
	if cache {
		wantChanges += 8
	}
	if jobAdmin {
		wantChanges += 9
		if admin {
			wantChanges++
		}
	}
	if observability {
		wantChanges += 6
	}
	if csvTransfer {
		wantChanges += 8
	}
	if plannedData.ChangeCount != wantChanges || plannedData.CapabilityLock["java-service"] != "0.3.0" {
		t.Fatalf("Plan() data = %#v", plannedData)
	}
	if business && plannedData.CapabilityLock["java-crud"] != "0.1.0" {
		t.Fatalf("Plan() CRUD lock = %#v", plannedData.CapabilityLock)
	}
	if admin && plannedData.CapabilityLock["vue-admin"] != "0.2.0" {
		t.Fatalf("Plan() administration lock = %#v", plannedData.CapabilityLock)
	}
	if tenancyVersion != "" &&
		plannedData.CapabilityLock["organization-tenancy"] != tenancyVersion {
		t.Fatalf("Plan() tenancy lock = %#v", plannedData.CapabilityLock)
	}
	if resolvedJobs && plannedData.CapabilityLock["background-jobs"] != "0.1.0" {
		t.Fatalf("Plan() jobs lock = %#v", plannedData.CapabilityLock)
	}
	if notifications &&
		(plannedData.CapabilityLock["notifications"] != "0.1.0" ||
			plannedData.CapabilityLock["background-jobs"] != "0.1.0") {
		t.Fatalf("Plan() notifications lock = %#v", plannedData.CapabilityLock)
	}
	if files && plannedData.CapabilityLock["file-assets"] != "0.1.0" {
		t.Fatalf("Plan() file assets lock = %#v", plannedData.CapabilityLock)
	}
	if cache && plannedData.CapabilityLock["application-cache"] != "0.1.0" {
		t.Fatalf("Plan() application cache lock = %#v", plannedData.CapabilityLock)
	}
	if jobAdmin && plannedData.CapabilityLock["job-administration"] != "0.1.0" {
		t.Fatalf("Plan() job administration lock = %#v", plannedData.CapabilityLock)
	}
	if observability && plannedData.CapabilityLock["observability"] != "0.1.0" {
		t.Fatalf("Plan() observability lock = %#v", plannedData.CapabilityLock)
	}
	if csvTransfer && plannedData.CapabilityLock["csv-import-export"] != "0.1.0" {
		t.Fatalf("Plan() CSV transfer lock = %#v", plannedData.CapabilityLock)
	}
	previewed := application.Preview(ctx, PreviewInput{ProjectRoot: root, PlanID: plannedData.PlanID})
	if previewed.Status != result.StatusOK {
		t.Fatalf("Preview() = %#v, want ok", previewed)
	}
	previewedData := previewed.Data.(previewData)
	applied := application.Apply(ctx, ApplyInput{
		ProjectRoot: root, PlanID: plannedData.PlanID, ApplyToken: previewedData.ApplyToken,
	})
	if applied.Status != result.StatusOK {
		t.Fatalf("Apply() = %#v, want ok", applied)
	}
	if os.Getenv("SCAFFOLD_AGENT_RUN_JAVA_BUILD") == "1" {
		command := exec.CommandContext(ctx, "mvn", "-B", "-ntp", "verify")
		command.Dir = root
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("generated mvn verify failed: %v\n%s", err, output)
		}
	}
	if admin && os.Getenv("SCAFFOLD_AGENT_RUN_ADMIN_BUILD") == "1" {
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
				t.Fatalf("generated Java admin %s failed: %v\n%s", strings.Join(arguments, " "), err, output)
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
