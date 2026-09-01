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
	runGeneratedPythonReference(t, "postgresql")
}

func TestPythonMySQLIdentityPlanApplyVerifyEndToEnd(t *testing.T) {
	runGeneratedPythonReference(t, "mysql")
}

func runGeneratedPythonReference(t *testing.T, database string) {
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
`
	writeBlueprint(t, root, strings.ReplaceAll(blueprint, "DATABASE_ENGINE", database))
	application := New("test")
	planned := application.Plan(ctx, PlanInput{
		ProjectRoot: root, BlueprintPath: "scaffold.yaml", Action: plan.ActionCreate,
	})
	if planned.Status != result.StatusOK {
		t.Fatalf("Plan() = %#v, want ok", planned)
	}
	plannedData := planned.Data.(planData)
	if plannedData.ChangeCount != 30 || plannedData.CapabilityLock["python-service"] != "0.1.0" {
		t.Fatalf("Plan() data = %#v", plannedData)
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
