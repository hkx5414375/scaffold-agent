// Package admin owns the model-neutral Vue administration templates shared by
// every first-party backend adapter.
package admin

import (
	"bytes"
	"embed"
	"fmt"
	"text/template"
)

const (
	// Owner is the managed-output owner recorded for the administration project.
	Owner = "vue-admin"
	// Version is the administration template contract version.
	Version = "0.2.0"
	// BusinessViewTemplate is added when a Blueprint declares a business entity.
	BusinessViewTemplate = "templates/src/views/BusinessView.vue.tmpl"
)

//go:embed all:templates
var templateFS embed.FS

// BaseTemplates maps generated output paths to shared template resources.
var BaseTemplates = map[string]string{
	"web/admin/.prettierignore":             "templates/.prettierignore",
	"web/admin/.prettierrc.json":            "templates/.prettierrc.json",
	"web/admin/eslint.config.js":            "templates/eslint.config.js",
	"web/admin/index.html":                  "templates/index.html",
	"web/admin/package-lock.json":           "templates/package-lock.json",
	"web/admin/package.json":                "templates/package.json",
	"web/admin/tsconfig.json":               "templates/tsconfig.json",
	"web/admin/vite.config.ts":              "templates/vite.config.ts",
	"web/admin/vitest.config.ts":            "templates/vitest.config.ts",
	"web/admin/src/App.vue":                 "templates/src/App.vue",
	"web/admin/src/api/client.test.ts":      "templates/src/api/client.test.ts",
	"web/admin/src/api/client.ts":           "templates/src/api/client.ts",
	"web/admin/src/env.d.ts":                "templates/src/env.d.ts",
	"web/admin/src/main.ts":                 "templates/src/main.ts",
	"web/admin/src/stores/session.ts":       "templates/src/stores/session.ts",
	"web/admin/src/styles.css":              "templates/src/styles.css",
	"web/admin/src/types.ts":                "templates/src/types.ts",
	"web/admin/src/views/DashboardView.vue": "templates/src/views/DashboardView.vue",
	"web/admin/src/views/LoginView.vue":     "templates/src/views/LoginView.vue",
}

// Render executes one shared template against a backend-neutral data shape.
func Render(templatePath string, data any) ([]byte, error) {
	content, err := templateFS.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("read administration template: %w", err)
	}
	parsed, err := template.New(templatePath).Option("missingkey=error").Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse administration template: %w", err)
	}
	var output bytes.Buffer
	if err := parsed.Execute(&output, data); err != nil {
		return nil, fmt.Errorf("execute administration template: %w", err)
	}
	return output.Bytes(), nil
}
