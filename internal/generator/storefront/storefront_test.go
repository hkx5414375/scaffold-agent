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
