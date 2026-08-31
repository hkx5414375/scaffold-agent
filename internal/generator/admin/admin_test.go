package admin

import "testing"

func TestBaseTemplatesContainLockedAdministrationProject(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"web/admin/package-lock.json",
		"web/admin/src/App.vue",
		"web/admin/src/api/client.test.ts",
		"web/admin/src/stores/session.ts",
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
