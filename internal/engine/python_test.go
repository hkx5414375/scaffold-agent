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
	runGeneratedPythonReference(t, "postgresql", false)
}

func TestPythonMySQLIdentityPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReference(t, "mysql", false)
}

func TestPythonPostgreSQLCRUDPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReference(t, "postgresql", true)
}

func TestPythonMySQLCRUDPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReference(t, "mysql", true)
}

func runGeneratedPythonReference(t *testing.T, database string, business bool) {
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
    admin_ui: none
    storefront: none
  database:
    engine: DATABASE_ENGINE
  auth:
    modes: [session, token]
MODULES
`
	blueprint = strings.ReplaceAll(blueprint, "DATABASE_ENGINE", database)
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
	if plannedData.ChangeCount != wantChanges || plannedData.CapabilityLock["python-service"] != "0.1.0" {
		t.Fatalf("Plan() data = %#v", plannedData)
	}
	if business && plannedData.CapabilityLock["python-crud"] != "0.1.0" {
		t.Fatalf("Plan() CRUD lock = %#v", plannedData.CapabilityLock)
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
	verified := application.Verify(ctx, VerifyInput{ProjectRoot: root})
	if verified.Status != result.StatusOK || !strings.Contains(verified.Summary, "no findings") {
		t.Fatalf("Verify() = %#v, want no findings", verified)
	}
}
