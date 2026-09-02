package engine

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
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
		{name: "python", changeCount: 49},
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

func TestCommerceOperationsStorefrontPlanApplyVerifyAllBackends(t *testing.T) {
	var reference map[string]string
	for _, backend := range []string{"go", "java", "python"} {
		backend := backend
		t.Run(backend, func(t *testing.T) {
			root := t.TempDir()
			blueprint := `
api_version: scaffold-agent.io/v1alpha1
kind: Project
metadata:
  name: generated-commerce-storefront
  display_name: Generated Commerce Storefront
spec:
  stack:
    backend: BACKEND
    admin_ui: none
    storefront: nuxt
  database:
    engine: postgresql
  auth:
    modes: [session, token]
  capabilities:
    - name: organization-tenancy
      version: 0.3.0
    - name: commerce-operations
      version: 0.1.0
`
			writeBlueprint(t, root, strings.ReplaceAll(blueprint, "BACKEND", backend))
			application := New("test")
			ctx := context.Background()
			planned := application.Plan(ctx, PlanInput{
				ProjectRoot: root, BlueprintPath: "scaffold.yaml", Action: plan.ActionCreate,
			})
			if planned.Status != result.StatusOK {
				t.Fatalf("Plan() = %#v, want ok", planned)
			}
			plannedData := planned.Data.(planData)
			for name, version := range map[string]string{
				"commerce-catalog":    "0.1.0",
				"customer-accounts":   "0.1.0",
				"commerce-operations": "0.1.0",
				"nuxt-storefront":     "0.1.0",
			} {
				if plannedData.CapabilityLock[name] != version {
					t.Errorf("Plan() capability %s = %q, want %q", name, plannedData.CapabilityLock[name], version)
				}
			}
			previewed := application.Preview(ctx, PreviewInput{
				ProjectRoot: root, PlanID: plannedData.PlanID,
			})
			if previewed.Status != result.StatusOK {
				t.Fatalf("Preview() = %#v, want ok", previewed)
			}
			applied := application.Apply(ctx, ApplyInput{
				ProjectRoot: root, PlanID: plannedData.PlanID,
				ApplyToken: previewed.Data.(previewData).ApplyToken,
			})
			if applied.Status != result.StatusOK {
				t.Fatalf("Apply() = %#v, want ok", applied)
			}

			storefrontFiles := make(map[string]string)
			for _, path := range []string{
				"app/pages/cart.vue",
				"app/pages/checkout.vue",
				"app/pages/account/orders/index.vue",
				"app/pages/account/orders/[id].vue",
				"server/api/storefront/checkout.post.ts",
				"server/api/storefront/orders/[id]/return.post.ts",
				"server/utils/commerce.ts",
				"shared/types/commerce.ts",
				"test/commerce.test.ts",
			} {
				content, err := os.ReadFile(filepath.Join(root, "web", "storefront", filepath.FromSlash(path)))
				if err != nil {
					t.Fatalf("read commerce storefront %s: %v", path, err)
				}
				storefrontFiles[path] = string(content)
			}
			if reference == nil {
				reference = storefrontFiles
			} else if !reflect.DeepEqual(storefrontFiles, reference) {
				t.Errorf("%s commerce storefront differs from the shared contract", backend)
			}
			if backend == "go" && os.Getenv("SCAFFOLD_AGENT_RUN_STOREFRONT_BUILD") == "1" {
				runGeneratedStorefrontQualityGate(t, filepath.Join(root, "web", "storefront"))
			}
			verified := application.Verify(ctx, VerifyInput{ProjectRoot: root})
			if verified.Status != result.StatusOK || !strings.Contains(verified.Summary, "no findings") {
				t.Fatalf("Verify() = %#v, want no findings", verified)
			}
		})
	}
}

func TestGoCommerceCatalogStorefrontPlanApplyVerifyBothDatabases(t *testing.T) {
	for _, database := range []struct {
		name        string
		changeCount int
	}{
		{name: "postgresql", changeCount: 72},
		{name: "mysql", changeCount: 73},
	} {
		database := database
		t.Run(database.name, func(t *testing.T) {
			root := t.TempDir()
			blueprint := `
api_version: scaffold-agent.io/v1alpha1
kind: Project
metadata:
  name: generated-catalog-storefront
  display_name: Generated Catalog Storefront
spec:
  stack:
    backend: go
    admin_ui: none
    storefront: nuxt
  database:
    engine: DATABASE
  auth:
    modes: [session, token]
  capabilities:
    - name: organization-tenancy
      version: 0.3.0
    - name: commerce-catalog
      version: 0.1.0
`
			writeBlueprint(t, root, strings.ReplaceAll(blueprint, "DATABASE", database.name))
			application := New("test")
			ctx := context.Background()
			planned := application.Plan(ctx, PlanInput{
				ProjectRoot: root, BlueprintPath: "scaffold.yaml", Action: plan.ActionCreate,
			})
			if planned.Status != result.StatusOK {
				t.Fatalf("Plan() = %#v, want ok", planned)
			}
			plannedData := planned.Data.(planData)
			if plannedData.ChangeCount != database.changeCount ||
				plannedData.CapabilityLock["commerce-catalog"] != "0.1.0" ||
				plannedData.CapabilityLock["nuxt-storefront"] != "0.1.0" {
				t.Fatalf("Plan() data = %#v", plannedData)
			}
			previewed := application.Preview(ctx, PreviewInput{ProjectRoot: root, PlanID: plannedData.PlanID})
			if previewed.Status != result.StatusOK {
				t.Fatalf("Preview() = %#v, want ok", previewed)
			}
			applied := application.Apply(ctx, ApplyInput{
				ProjectRoot: root, PlanID: plannedData.PlanID,
				ApplyToken: previewed.Data.(previewData).ApplyToken,
			})
			if applied.Status != result.StatusOK {
				t.Fatalf("Apply() = %#v, want ok", applied)
			}
			for _, path := range []string{
				"internal/catalog/catalog.go",
				"api/openapi.yaml",
				"web/storefront/app/pages/products/index.vue",
				"web/storefront/app/pages/products/[id].vue",
				"web/storefront/server/api/storefront/products.get.ts",
			} {
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
					t.Errorf("generated catalog storefront is missing %s: %v", path, err)
				}
			}
			for _, arguments := range [][]string{
				{"go", "mod", "verify"}, {"go", "test", "./..."}, {"go", "vet", "./..."},
			} {
				command := exec.CommandContext(ctx, arguments[0], arguments[1:]...)
				command.Dir = root
				if output, err := command.CombinedOutput(); err != nil {
					t.Fatalf("generated %s failed: %v\n%s", strings.Join(arguments, " "), err, output)
				}
			}
			if database.name == "postgresql" && os.Getenv("SCAFFOLD_AGENT_RUN_STOREFRONT_BUILD") == "1" {
				runGeneratedStorefrontQualityGate(t, filepath.Join(root, "web", "storefront"))
			}
			verified := application.Verify(ctx, VerifyInput{ProjectRoot: root})
			if verified.Status != result.StatusOK || !strings.Contains(verified.Summary, "no findings") {
				t.Fatalf("Verify() = %#v, want no findings", verified)
			}
		})
	}
}

func TestGoCustomerAccountsStorefrontPlanApplyVerifyBothDatabases(t *testing.T) {
	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			root := t.TempDir()
			blueprint := `
api_version: scaffold-agent.io/v1alpha1
kind: Project
metadata:
  name: generated-customer-storefront
  display_name: Generated Customer Storefront
spec:
  stack:
    backend: go
    admin_ui: none
    storefront: nuxt
  database:
    engine: DATABASE
  auth:
    modes: [session, token]
  capabilities:
    - name: organization-tenancy
      version: 0.3.0
    - name: customer-accounts
      version: 0.1.0
`
			writeBlueprint(t, root, strings.ReplaceAll(blueprint, "DATABASE", database))
			application := New("test")
			ctx := context.Background()
			planned := application.Plan(ctx, PlanInput{
				ProjectRoot: root, BlueprintPath: "scaffold.yaml", Action: plan.ActionCreate,
			})
			if planned.Status != result.StatusOK {
				t.Fatalf("Plan() = %#v, want ok", planned)
			}
			plannedData := planned.Data.(planData)
			if plannedData.ChangeCount < 1 ||
				plannedData.CapabilityLock["customer-accounts"] != "0.1.0" ||
				plannedData.CapabilityLock["nuxt-storefront"] != "0.1.0" {
				t.Fatalf("Plan() data = %#v", plannedData)
			}
			previewed := application.Preview(ctx, PreviewInput{ProjectRoot: root, PlanID: plannedData.PlanID})
			if previewed.Status != result.StatusOK {
				t.Fatalf("Preview() = %#v, want ok", previewed)
			}
			applied := application.Apply(ctx, ApplyInput{
				ProjectRoot: root, PlanID: plannedData.PlanID,
				ApplyToken: previewed.Data.(previewData).ApplyToken,
			})
			if applied.Status != result.StatusOK {
				t.Fatalf("Apply() = %#v, want ok", applied)
			}
			for _, path := range []string{
				"internal/customeraccounts/accounts.go",
				"api/openapi.yaml",
				"web/storefront/app/pages/account/index.vue",
				"web/storefront/app/pages/account/login.vue",
				"web/storefront/server/api/storefront/account/login.post.ts",
				"web/storefront/server/utils/customer.ts",
			} {
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
					t.Errorf("generated customer storefront is missing %s: %v", path, err)
				}
			}
			for _, arguments := range [][]string{
				{"go", "mod", "verify"}, {"go", "test", "./..."}, {"go", "vet", "./..."},
			} {
				command := exec.CommandContext(ctx, arguments[0], arguments[1:]...)
				command.Dir = root
				if output, err := command.CombinedOutput(); err != nil {
					t.Fatalf("generated %s failed: %v\n%s", strings.Join(arguments, " "), err, output)
				}
			}
			if database == "postgresql" && os.Getenv("SCAFFOLD_AGENT_RUN_STOREFRONT_BUILD") == "1" {
				runGeneratedStorefrontQualityGate(t, filepath.Join(root, "web", "storefront"))
			}
			verified := application.Verify(ctx, VerifyInput{ProjectRoot: root})
			if verified.Status != result.StatusOK || !strings.Contains(verified.Summary, "no findings") {
				t.Fatalf("Verify() = %#v, want no findings", verified)
			}
		})
	}
}

func TestJavaCommerceCatalogStorefrontPlanApplyVerifyBothDatabases(t *testing.T) {
	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			root := t.TempDir()
			blueprint := `
api_version: scaffold-agent.io/v1alpha1
kind: Project
metadata:
  name: generated-java-catalog-storefront
  display_name: Generated Java Catalog Storefront
spec:
  stack:
    backend: java
    admin_ui: none
    storefront: nuxt
  database:
    engine: DATABASE
  auth:
    modes: [session, token]
  capabilities:
    - name: organization-tenancy
      version: 0.3.0
    - name: commerce-catalog
      version: 0.1.0
`
			writeBlueprint(t, root, strings.ReplaceAll(blueprint, "DATABASE", database))
			application := New("test")
			ctx := context.Background()
			planned := application.Plan(ctx, PlanInput{
				ProjectRoot: root, BlueprintPath: "scaffold.yaml", Action: plan.ActionCreate,
			})
			if planned.Status != result.StatusOK {
				t.Fatalf("Plan() = %#v, want ok", planned)
			}
			plannedData := planned.Data.(planData)
			if plannedData.ChangeCount != 93 ||
				plannedData.CapabilityLock["commerce-catalog"] != "0.1.0" ||
				plannedData.CapabilityLock["nuxt-storefront"] != "0.1.0" {
				t.Fatalf("Plan() data = %#v", plannedData)
			}
			previewed := application.Preview(ctx, PreviewInput{
				ProjectRoot: root, PlanID: plannedData.PlanID,
			})
			if previewed.Status != result.StatusOK {
				t.Fatalf("Preview() = %#v, want ok", previewed)
			}
			applied := application.Apply(ctx, ApplyInput{
				ProjectRoot: root, PlanID: plannedData.PlanID,
				ApplyToken: previewed.Data.(previewData).ApplyToken,
			})
			if applied.Status != result.StatusOK {
				t.Fatalf("Apply() = %#v, want ok", applied)
			}
			for _, path := range []string{
				"src/main/java/com/scaffold/generated/generatedjavacatalogstorefront/catalog/CatalogService.java",
				"api/openapi.yaml",
				"web/storefront/app/pages/products/index.vue",
				"web/storefront/app/pages/products/[id].vue",
				"web/storefront/server/api/storefront/products.get.ts",
			} {
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
					t.Errorf("generated Java catalog storefront is missing %s: %v", path, err)
				}
			}
			if database == "postgresql" && os.Getenv("SCAFFOLD_AGENT_RUN_STOREFRONT_BUILD") == "1" {
				runGeneratedStorefrontQualityGate(t, filepath.Join(root, "web", "storefront"))
			}
			verified := application.Verify(ctx, VerifyInput{ProjectRoot: root})
			if verified.Status != result.StatusOK || !strings.Contains(verified.Summary, "no findings") {
				t.Fatalf("Verify() = %#v, want no findings", verified)
			}
		})
	}
}

func TestJavaCustomerAccountsStorefrontPlanApplyVerifyBothDatabases(t *testing.T) {
	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			root := t.TempDir()
			blueprint := `
api_version: scaffold-agent.io/v1alpha1
kind: Project
metadata:
  name: generated-java-customer-storefront
  display_name: Generated Java Customer Storefront
spec:
  stack:
    backend: java
    admin_ui: none
    storefront: nuxt
  database:
    engine: DATABASE
  auth:
    modes: [session, token]
  capabilities:
    - name: organization-tenancy
      version: 0.3.0
    - name: customer-accounts
      version: 0.1.0
`
			writeBlueprint(t, root, strings.ReplaceAll(blueprint, "DATABASE", database))
			application := New("test")
			ctx := context.Background()
			planned := application.Plan(ctx, PlanInput{
				ProjectRoot: root, BlueprintPath: "scaffold.yaml", Action: plan.ActionCreate,
			})
			if planned.Status != result.StatusOK {
				t.Fatalf("Plan() = %#v, want ok", planned)
			}
			plannedData := planned.Data.(planData)
			if plannedData.ChangeCount < 1 ||
				plannedData.CapabilityLock["customer-accounts"] != "0.1.0" ||
				plannedData.CapabilityLock["nuxt-storefront"] != "0.1.0" {
				t.Fatalf("Plan() data = %#v", plannedData)
			}
			previewed := application.Preview(ctx, PreviewInput{
				ProjectRoot: root, PlanID: plannedData.PlanID,
			})
			if previewed.Status != result.StatusOK {
				t.Fatalf("Preview() = %#v, want ok", previewed)
			}
			applied := application.Apply(ctx, ApplyInput{
				ProjectRoot: root, PlanID: plannedData.PlanID,
				ApplyToken: previewed.Data.(previewData).ApplyToken,
			})
			if applied.Status != result.StatusOK {
				t.Fatalf("Apply() = %#v, want ok", applied)
			}
			for _, path := range []string{
				"src/main/java/com/scaffold/generated/generatedjavacustomerstorefront/customeraccounts/CustomerAccountService.java",
				"api/openapi.yaml",
				"web/storefront/app/pages/account/index.vue",
				"web/storefront/app/pages/account/login.vue",
				"web/storefront/server/api/storefront/account/login.post.ts",
				"web/storefront/server/utils/customer.ts",
			} {
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
					t.Errorf("generated Java customer storefront is missing %s: %v", path, err)
				}
			}
			if database == "postgresql" && os.Getenv("SCAFFOLD_AGENT_RUN_STOREFRONT_BUILD") == "1" {
				runGeneratedStorefrontQualityGate(t, filepath.Join(root, "web", "storefront"))
			}
			verified := application.Verify(ctx, VerifyInput{ProjectRoot: root})
			if verified.Status != result.StatusOK || !strings.Contains(verified.Summary, "no findings") {
				t.Fatalf("Verify() = %#v, want no findings", verified)
			}
		})
	}
}

func TestPythonCommerceCatalogStorefrontPlanApplyVerifyBothDatabases(t *testing.T) {
	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			root := t.TempDir()
			blueprint := `
api_version: scaffold-agent.io/v1alpha1
kind: Project
metadata:
  name: generated-python-catalog-storefront
  display_name: Generated Python Catalog Storefront
spec:
  stack:
    backend: python
    admin_ui: none
    storefront: nuxt
  database:
    engine: DATABASE
  auth:
    modes: [session, token]
  capabilities:
    - name: organization-tenancy
      version: 0.3.0
    - name: commerce-catalog
      version: 0.1.0
`
			writeBlueprint(t, root, strings.ReplaceAll(blueprint, "DATABASE", database))
			application := New("test")
			ctx := context.Background()
			planned := application.Plan(ctx, PlanInput{
				ProjectRoot: root, BlueprintPath: "scaffold.yaml", Action: plan.ActionCreate,
			})
			if planned.Status != result.StatusOK {
				t.Fatalf("Plan() = %#v, want ok", planned)
			}
			plannedData := planned.Data.(planData)
			if plannedData.ChangeCount != 88 ||
				plannedData.CapabilityLock["commerce-catalog"] != "0.1.0" ||
				plannedData.CapabilityLock["nuxt-storefront"] != "0.1.0" {
				t.Fatalf("Plan() data = %#v", plannedData)
			}
			previewed := application.Preview(ctx, PreviewInput{
				ProjectRoot: root, PlanID: plannedData.PlanID,
			})
			if previewed.Status != result.StatusOK {
				t.Fatalf("Preview() = %#v, want ok", previewed)
			}
			applied := application.Apply(ctx, ApplyInput{
				ProjectRoot: root, PlanID: plannedData.PlanID,
				ApplyToken: previewed.Data.(previewData).ApplyToken,
			})
			if applied.Status != result.StatusOK {
				t.Fatalf("Apply() = %#v, want ok", applied)
			}
			for _, path := range []string{
				"src/generated_python_catalog_storefront/catalog/service.py",
				"api/openapi.yaml",
				"web/storefront/app/pages/products/index.vue",
				"web/storefront/app/pages/products/[id].vue",
				"web/storefront/server/api/storefront/products.get.ts",
			} {
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
					t.Errorf("generated Python catalog storefront is missing %s: %v", path, err)
				}
			}
			if database == "postgresql" && os.Getenv("SCAFFOLD_AGENT_RUN_STOREFRONT_BUILD") == "1" {
				runGeneratedStorefrontQualityGate(t, filepath.Join(root, "web", "storefront"))
			}
			verified := application.Verify(ctx, VerifyInput{ProjectRoot: root})
			if verified.Status != result.StatusOK || !strings.Contains(verified.Summary, "no findings") {
				t.Fatalf("Verify() = %#v, want no findings", verified)
			}
		})
	}
}

func TestPythonCustomerAccountsStorefrontPlanApplyVerifyBothDatabases(t *testing.T) {
	for _, database := range []string{"postgresql", "mysql"} {
		database := database
		t.Run(database, func(t *testing.T) {
			root := t.TempDir()
			blueprint := `
api_version: scaffold-agent.io/v1alpha1
kind: Project
metadata:
  name: generated-python-customer-storefront
  display_name: Generated Python Customer Storefront
spec:
  stack:
    backend: python
    admin_ui: none
    storefront: nuxt
  database:
    engine: DATABASE
  auth:
    modes: [session, token]
  capabilities:
    - name: organization-tenancy
      version: 0.3.0
    - name: customer-accounts
      version: 0.1.0
`
			writeBlueprint(t, root, strings.ReplaceAll(blueprint, "DATABASE", database))
			application := New("test")
			ctx := context.Background()
			planned := application.Plan(ctx, PlanInput{
				ProjectRoot: root, BlueprintPath: "scaffold.yaml", Action: plan.ActionCreate,
			})
			if planned.Status != result.StatusOK {
				t.Fatalf("Plan() = %#v, want ok", planned)
			}
			plannedData := planned.Data.(planData)
			if plannedData.ChangeCount != 93 ||
				plannedData.CapabilityLock["customer-accounts"] != "0.1.0" ||
				plannedData.CapabilityLock["nuxt-storefront"] != "0.1.0" {
				t.Fatalf("Plan() data = %#v", plannedData)
			}
			previewed := application.Preview(ctx, PreviewInput{
				ProjectRoot: root, PlanID: plannedData.PlanID,
			})
			if previewed.Status != result.StatusOK {
				t.Fatalf("Preview() = %#v, want ok", previewed)
			}
			applied := application.Apply(ctx, ApplyInput{
				ProjectRoot: root, PlanID: plannedData.PlanID,
				ApplyToken: previewed.Data.(previewData).ApplyToken,
			})
			if applied.Status != result.StatusOK {
				t.Fatalf("Apply() = %#v, want ok", applied)
			}
			for _, path := range []string{
				"src/generated_python_customer_storefront/customer_accounts/service.py",
				"api/openapi.yaml",
				"web/storefront/app/pages/account/index.vue",
				"web/storefront/app/pages/account/login.vue",
				"web/storefront/server/api/storefront/account/login.post.ts",
				"web/storefront/server/utils/customer.ts",
			} {
				if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(path))); err != nil {
					t.Errorf("generated Python customer storefront is missing %s: %v", path, err)
				}
			}
			if database == "postgresql" && os.Getenv("SCAFFOLD_AGENT_RUN_STOREFRONT_BUILD") == "1" {
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
