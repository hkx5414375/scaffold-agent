package engine

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/plan"
	"github.com/hkx5414375/scaffold-agent/internal/result"
)

func TestPythonPostgreSQLIdentityPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReference(t, "postgresql", false, false, "")
}

func TestPythonMySQLIdentityPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReference(t, "mysql", false, false, "")
}

func TestPythonPostgreSQLCRUDPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReference(t, "postgresql", true, false, "")
}

func TestPythonMySQLCRUDPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReference(t, "mysql", true, false, "")
}

func TestPythonPostgreSQLAdminPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReference(t, "postgresql", true, true, "")
}

func TestPythonPostgreSQLTenantCRUDPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReference(t, "postgresql", true, false, "0.1.0")
}

func TestPythonPostgreSQLTenancyPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReference(t, "postgresql", false, false, "0.1.0")
}

func TestPythonMySQLTenancyPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReference(t, "mysql", false, false, "0.1.0")
}

func TestPythonMySQLTenantCRUDPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReference(t, "mysql", true, false, "0.1.0")
}

func TestPythonPostgreSQLTenantAdminPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReference(t, "postgresql", true, true, "0.1.0")
}

func TestPythonPostgreSQLTenancyMembersPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReference(t, "postgresql", true, true, "0.2.0")
}

func TestPythonMySQLTenancyMembersPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReference(t, "mysql", true, false, "0.2.0")
}

func TestPythonPostgreSQLTenancyLifecyclePlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReference(t, "postgresql", true, true, "0.3.0")
}

func TestPythonMySQLTenancyLifecyclePlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReference(t, "mysql", true, false, "0.3.0")
}

func TestPythonPostgreSQLJobsPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReferenceOptions(t, "postgresql", false, false, "", true, false, false, false)
}

func TestPythonMySQLJobsPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReferenceOptions(t, "mysql", false, false, "", true, false, false, false)
}

func TestPythonPostgreSQLTenantJobsPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReferenceOptions(t, "postgresql", true, false, "0.3.0", true, false, false, false)
}

func TestPythonMySQLTenantJobsPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReferenceOptions(t, "mysql", true, false, "0.3.0", true, false, false, false)
}

func TestPythonPostgreSQLNotificationsPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReferenceOptions(t, "postgresql", false, false, "", false, true, false, false)
}

func TestPythonMySQLNotificationsPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReferenceOptions(t, "mysql", false, false, "", false, true, false, false)
}

func TestPythonPostgreSQLTenantNotificationsPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReferenceOptions(t, "postgresql", true, false, "0.3.0", false, true, false, false)
}

func TestPythonMySQLTenantNotificationsPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReferenceOptions(t, "mysql", true, false, "0.3.0", false, true, false, false)
}

func TestPythonPostgreSQLFilesPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReferenceOptions(t, "postgresql", false, false, "", false, false, true, false)
}

func TestPythonMySQLFilesPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReferenceOptions(t, "mysql", false, false, "", false, false, true, false)
}

func TestPythonPostgreSQLTenantFilesPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReferenceOptions(t, "postgresql", true, true, "0.3.0", false, false, true, false)
}

func TestPythonMySQLTenantFilesPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReferenceOptions(t, "mysql", true, false, "0.3.0", false, false, true, false)
}

func TestPythonPostgreSQLCachePlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReferenceOptions(t, "postgresql", false, false, "", false, false, false, true)
}

func TestPythonMySQLCachePlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReferenceOptions(t, "mysql", false, false, "", false, false, false, true)
}

func TestPythonPostgreSQLTenantCachePlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReferenceOptions(t, "postgresql", true, false, "0.3.0", false, false, false, true)
}

func TestPythonMySQLTenantCachePlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReferenceOptions(t, "mysql", true, false, "0.3.0", false, false, false, true)
}

func TestPythonPostgreSQLTenantFilesCachePlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReferenceOptions(t, "postgresql", true, false, "0.3.0", false, false, true, true)
}

func runGeneratedPythonReference(
	t *testing.T,
	database string,
	business bool,
	admin bool,
	organizationTenancyVersion string,
) {
	t.Helper()
	runGeneratedPythonReferenceOptions(
		t,
		database,
		business,
		admin,
		organizationTenancyVersion,
		false,
		false,
		false,
		false,
	)
}

func runGeneratedPythonReferenceOptions(
	t *testing.T,
	database string,
	business bool,
	admin bool,
	organizationTenancyVersion string,
	jobs bool,
	notifications bool,
	files bool,
	cache bool,
) {
	t.Helper()
	root := t.TempDir()
	if captureRoot := os.Getenv("SCAFFOLD_AGENT_CAPTURE_PYTHON_ROOT"); captureRoot != "" {
		root = filepath.Join(captureRoot, database)
		if err := os.MkdirAll(root, 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
	}
	ctx := context.Background()
	blueprint := `
api_version: scaffold-agent.io/v1alpha1
kind: Project
metadata:
  name: generated-python
spec:
  stack:
    backend: python
    admin_ui: ADMIN_UI
    storefront: none
  database:
    engine: DATABASE_ENGINE
  auth:
    modes: [session, token]
MODULES
CAPABILITIES
`
	blueprint = strings.ReplaceAll(blueprint, "DATABASE_ENGINE", database)
	adminUI := "none"
	if admin {
		adminUI = "element-plus"
	}
	blueprint = strings.ReplaceAll(blueprint, "ADMIN_UI", adminUI)
	capabilities := ""
	if organizationTenancyVersion != "" || jobs || notifications || files || cache {
		capabilities = "  capabilities:\n"
	}
	if organizationTenancyVersion != "" {
		capabilities += `    - name: organization-tenancy
      version: TENANCY_VERSION
`
		capabilities = strings.ReplaceAll(
			capabilities,
			"TENANCY_VERSION",
			organizationTenancyVersion,
		)
	}
	if jobs {
		capabilities += `    - name: background-jobs
      version: 0.1.0
`
	}
	if notifications {
		capabilities += `    - name: notifications
      version: 0.1.0
`
	}
	if files {
		capabilities += `    - name: file-assets
      version: 0.1.0
`
	}
	if cache {
		capabilities += `    - name: application-cache
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
	writeBlueprint(t, root, strings.ReplaceAll(blueprint, "MODULES", modules))
	application := New("test")
	planned := application.Plan(ctx, PlanInput{
		ProjectRoot: root, BlueprintPath: "scaffold.yaml", Action: plan.ActionCreate,
	})
	if planned.Status != result.StatusOK {
		t.Fatalf("Plan() = %#v, want ok", planned)
	}
	plannedData := planned.Data.(planData)
	wantChanges := 30
	if business {
		wantChanges = 38
	}
	if admin {
		wantChanges += 19
		if business {
			wantChanges++
		}
	}
	if organizationTenancyVersion != "" {
		wantChanges += 8
		if business {
			wantChanges++
		}
		if organizationTenancyVersion == "0.2.0" || organizationTenancyVersion == "0.3.0" {
			wantChanges += 7
			if admin {
				wantChanges++
			}
		}
		if organizationTenancyVersion == "0.3.0" {
			wantChanges += 6
			if admin {
				wantChanges++
			}
		}
	}
	if jobs || notifications {
		wantChanges += 9
	}
	if notifications {
		wantChanges += 9
	}
	if files {
		wantChanges += 11
		if admin {
			wantChanges++
		}
	}
	if cache {
		wantChanges += 7
	}
	if plannedData.ChangeCount != wantChanges || plannedData.CapabilityLock["python-service"] != "0.1.0" {
		t.Fatalf("Plan() data = %#v", plannedData)
	}
	if business && plannedData.CapabilityLock["python-crud"] != "0.1.0" {
		t.Fatalf("Plan() CRUD lock = %#v", plannedData.CapabilityLock)
	}
	if admin && plannedData.CapabilityLock["vue-admin"] != "0.2.0" {
		t.Fatalf("Plan() administration lock = %#v", plannedData.CapabilityLock)
	}
	if organizationTenancyVersion != "" &&
		plannedData.CapabilityLock["organization-tenancy"] != organizationTenancyVersion {
		t.Fatalf("Plan() tenancy lock = %#v", plannedData.CapabilityLock)
	}
	if (jobs || notifications) && plannedData.CapabilityLock["background-jobs"] != "0.1.0" {
		t.Fatalf("Plan() jobs lock = %#v", plannedData.CapabilityLock)
	}
	if notifications && plannedData.CapabilityLock["notifications"] != "0.1.0" {
		t.Fatalf("Plan() notifications lock = %#v", plannedData.CapabilityLock)
	}
	if files && plannedData.CapabilityLock["file-assets"] != "0.1.0" {
		t.Fatalf("Plan() file assets lock = %#v", plannedData.CapabilityLock)
	}
	if cache && plannedData.CapabilityLock["application-cache"] != "0.1.0" {
		t.Fatalf("Plan() cache lock = %#v", plannedData.CapabilityLock)
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
	if os.Getenv("SCAFFOLD_AGENT_RUN_PYTHON_BUILD") == "1" {
		commands := [][]string{
			{"uv", "sync", "--frozen", "--all-groups"},
			{"uv", "lock", "--check"},
			{"uv", "run", "ruff", "format", "--check", "."},
			{"uv", "run", "ruff", "check", "."},
			{"uv", "run", "mypy", "src", "tests"},
			{"uv", "run", "bandit", "-c", "pyproject.toml", "-r", "src"},
			{"uv", "run", "pytest"},
		}
		for _, arguments := range commands {
			command := exec.CommandContext(ctx, arguments[0], arguments[1:]...)
			command.Dir = root
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("generated %s failed: %v\n%s", strings.Join(arguments, " "), err, output)
			}
		}
	}
	if admin && os.Getenv("SCAFFOLD_AGENT_RUN_ADMIN_BUILD") == "1" {
		adminRoot := filepath.Join(root, "web", "admin")
		commands := [][]string{
			{"npm", "ci", "--ignore-scripts", "--no-audit", "--no-fund"},
			{"npm", "run", "lint"},
			{"npm", "test"},
			{"npm", "run", "build"},
			{"npm", "run", "format:check"},
		}
		for _, arguments := range commands {
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
