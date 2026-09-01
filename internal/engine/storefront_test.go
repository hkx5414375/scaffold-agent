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

func TestSharedNuxtStorefrontPlanApplyVerifyAllBackends(t *testing.T) {
	backends := []struct {
		name        string
		changeCount int
	}{
		{name: "go", changeCount: 39},
		{name: "java", changeCount: 50},
		{name: "python", changeCount: 48},
	}
	var reference map[string]string
	for _, backend := range backends {
		backend := backend
		t.Run(backend.name, func(t *testing.T) {
			root := t.TempDir()
			blueprint := `
api_version: scaffold-agent.io/v1alpha1
kind: Project
metadata:
  name: generated-storefront
  display_name: Generated Storefront
spec:
  stack:
    backend: BACKEND
    admin_ui: none
    storefront: nuxt
  database:
    engine: postgresql
  auth:
    modes: [session, token]
`
			writeBlueprint(t, root, strings.ReplaceAll(blueprint, "BACKEND", backend.name))
			application := New("test")
			ctx := context.Background()
			planned := application.Plan(ctx, PlanInput{
				ProjectRoot:   root,
				BlueprintPath: "scaffold.yaml",
				Action:        plan.ActionCreate,
			})
			if planned.Status != result.StatusOK {
				t.Fatalf("Plan() = %#v, want ok", planned)
			}
			plannedData := planned.Data.(planData)
			if plannedData.ChangeCount != backend.changeCount ||
				plannedData.CapabilityLock["nuxt-storefront"] != "0.1.0" {
				t.Fatalf("Plan() data = %#v", plannedData)
			}
			previewed := application.Preview(ctx, PreviewInput{
				ProjectRoot: root,
				PlanID:      plannedData.PlanID,
			})
			if previewed.Status != result.StatusOK {
				t.Fatalf("Preview() = %#v, want ok", previewed)
			}
			applied := application.Apply(ctx, ApplyInput{
				ProjectRoot: root,
				PlanID:      plannedData.PlanID,
				ApplyToken:  previewed.Data.(previewData).ApplyToken,
			})
			if applied.Status != result.StatusOK {
				t.Fatalf("Apply() = %#v, want ok", applied)
			}

			storefrontFiles := make(map[string]string, 18)
			for _, path := range []string{
				"package.json",
				"package-lock.json",
				"nuxt.config.ts",
				"app/pages/index.vue",
				"server/api/storefront/status.get.ts",
				"server/utils/backend.ts",
				"test/backend.test.ts",
			} {
				content, err := os.ReadFile(filepath.Join(root, "web", "storefront", filepath.FromSlash(path)))
				if err != nil {
					t.Fatalf("read storefront %s: %v", path, err)
				}
				storefrontFiles[path] = string(content)
			}
			if reference == nil {
				reference = storefrontFiles
			} else {
				for path, content := range reference {
					if storefrontFiles[path] != content {
						t.Errorf("%s storefront %s differs from the shared contract", backend.name, path)
					}
				}
			}

			if backend.name == "go" && os.Getenv("SCAFFOLD_AGENT_RUN_STOREFRONT_BUILD") == "1" {
				runGeneratedStorefrontQualityGate(t, filepath.Join(root, "web", "storefront"))
			}
			verified := application.Verify(ctx, VerifyInput{ProjectRoot: root})
			if verified.Status != result.StatusOK || !strings.Contains(verified.Summary, "no findings") {
				t.Fatalf("Verify() = %#v, want no findings", verified)
			}
		})
	}
}

func runGeneratedStorefrontQualityGate(t *testing.T, root string) {
	t.Helper()
	commands := [][]string{
		{"npm", "ci", "--no-audit", "--no-fund"},
		{"npm", "run", "lint"},
		{"npm", "run", "test"},
		{"npm", "run", "typecheck"},
		{"npm", "run", "build"},
		{"npm", "run", "format"},
	}
	for _, arguments := range commands {
		command := exec.Command(arguments[0], arguments[1:]...)
		command.Dir = root
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("generated storefront %s failed: %v\n%s", strings.Join(arguments, " "), err, output)
		}
	}
}
