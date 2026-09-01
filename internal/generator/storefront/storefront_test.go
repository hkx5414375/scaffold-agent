package storefront

import (
	"strings"
	"testing"
)

func TestBaseTemplatesContainLockedStorefrontProject(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"web/storefront/package-lock.json",
		"web/storefront/nuxt.config.ts",
		"web/storefront/app/pages/index.vue",
		"web/storefront/server/api/storefront/status.get.ts",
		"web/storefront/test/backend.test.ts",
	} {
		templatePath, exists := BaseTemplates[path]
		if !exists {
			t.Fatalf("BaseTemplates does not contain %q", path)
		}
		if _, err := templateFS.ReadFile(templatePath); err != nil {
			t.Fatalf("read %q: %v", templatePath, err)
		}
	}
}

func TestRenderUsesOnlyBackendNeutralProjectFacts(t *testing.T) {
	t.Parallel()

	content, err := Render(
		"templates/app/pages/index.vue.tmpl",
		NewData("shop-service", "Shop Service"),
	)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	value := string(content)
	if !strings.Contains(value, "Shop Service") || strings.Contains(value, "[[") {
		t.Fatalf("Render() = %q", value)
	}
}

func TestNewDataEscapesFreeFormDisplayMetadata(t *testing.T) {
	t.Parallel()

	data := NewData("shop-service", `Shop "</h1><script>alert(1)</script>`)
	page, err := Render("templates/app/pages/index.vue.tmpl", data)
	if err != nil {
		t.Fatalf("Render(page) error = %v", err)
	}
	configuration, err := Render("templates/nuxt.config.ts.tmpl", data)
	if err != nil {
		t.Fatalf("Render(configuration) error = %v", err)
	}
	if strings.Contains(string(page), "<script>") ||
		!strings.Contains(string(page), "&lt;script&gt;") {
		t.Fatalf("rendered page did not HTML-escape display metadata:\n%s", page)
	}
	if strings.Contains(string(configuration), "</script>") ||
		!strings.Contains(string(configuration), `\u003c/script\u003e`) {
		t.Fatalf("rendered configuration did not JSON-escape display metadata:\n%s", configuration)
	}
}

func TestNuxtConfigurationProtectsOnlyLongDynamicDescriptions(t *testing.T) {
	t.Parallel()

	short, err := Render(
		"templates/nuxt.config.ts.tmpl",
		NewData("shop-service", "Catalog Store"),
	)
	if err != nil {
		t.Fatalf("Render(short configuration) error = %v", err)
	}
	long, err := Render(
		"templates/nuxt.config.ts.tmpl",
		NewData("shop-service", "Generated Java Catalog Storefront"),
	)
	if err != nil {
		t.Fatalf("Render(long configuration) error = %v", err)
	}
	if strings.Contains(string(short), "prettier-ignore") ||
		!strings.Contains(string(long), "// prettier-ignore\n          content:") {
		t.Fatalf("dynamic description formatting guard is not scoped correctly")
	}
}

func TestCatalogCompositionPreservesUnselectedFoundation(t *testing.T) {
	t.Parallel()

	data := NewData("shop-service", "Shop Service")
	application, err := Render("templates/app/app.vue", data)
	if err != nil {
		t.Fatalf("Render(application) error = %v", err)
	}
	configuration, err := Render("templates/nuxt.config.ts.tmpl", data)
	if err != nil {
		t.Fatalf("Render(configuration) error = %v", err)
	}
	readme, err := Render("templates/README.md.tmpl", data)
	if err != nil {
		t.Fatalf("Render(README) error = %v", err)
	}
	if strings.Contains(string(application), "Products") ||
		!strings.Contains(string(application), "<NuxtLink to=\"/\">Home</NuxtLink>\n      </nav>") {
		t.Fatalf("unselected catalog changed application shell:\n%s", application)
	}
	if strings.Contains(string(configuration), "organizationId") ||
		strings.Contains(string(configuration), "catalog.css") ||
		!strings.Contains(string(configuration), `css: ["~/assets/css/main.css"],`) {
		t.Fatalf("unselected catalog changed Nuxt runtime configuration:\n%s", configuration)
	}
	if strings.Contains(string(readme), "SCAFFOLD_ORGANIZATION_ID") ||
		strings.Contains(string(readme), "catalog capability") {
		t.Fatalf("unselected catalog changed storefront documentation:\n%s", readme)
	}

	data.Catalog = true
	data.Tenancy = true
	configuration, err = Render("templates/nuxt.config.ts.tmpl", data)
	if err != nil {
		t.Fatalf("Render(catalog configuration) error = %v", err)
	}
	if !strings.Contains(string(configuration), "organizationId") ||
		!strings.Contains(string(configuration), "catalog.css") {
		t.Fatalf("selected catalog did not extend Nuxt configuration:\n%s", configuration)
	}
	for path, templatePath := range CatalogTemplates {
		content, err := Render(templatePath, data)
		if err != nil {
			t.Fatalf("Render(%s) error = %v", path, err)
		}
		if strings.Contains(string(content), "[[") {
			t.Errorf("Render(%s) left a template directive", path)
		}
	}
}
