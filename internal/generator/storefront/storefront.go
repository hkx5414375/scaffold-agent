// Package storefront owns the model-neutral Nuxt storefront foundation shared by
// every first-party backend adapter.
package storefront

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html"
	"text/template"
)

const (
	// Owner is the managed-output owner recorded for the storefront project.
	Owner = "nuxt-storefront"
	// Version is the storefront foundation contract version.
	Version = "0.1.0"
)

// Data is the backend-neutral storefront template input.
type Data struct {
	ProjectName     string
	DisplayNameHTML string
	DisplayNameJSON string
	DescriptionJSON string
}

// NewData normalizes optional project display metadata for shared templates.
func NewData(projectName, displayName string) Data {
	if displayName == "" {
		displayName = projectName
	}
	displayNameJSON, err := json.Marshal(displayName)
	if err != nil {
		panic("encode storefront display name: " + err.Error())
	}
	descriptionJSON, err := json.Marshal(
		"A deterministic storefront generated for " + displayName + ".",
	)
	if err != nil {
		panic("encode storefront description: " + err.Error())
	}
	return Data{
		ProjectName:     projectName,
		DisplayNameHTML: html.EscapeString(displayName),
		DisplayNameJSON: string(displayNameJSON),
		DescriptionJSON: string(descriptionJSON),
	}
}

//go:embed all:templates
var templateFS embed.FS

// BaseTemplates maps generated output paths to shared template resources.
var BaseTemplates = map[string]string{
	"web/storefront/.gitignore":                          "templates/.gitignore",
	"web/storefront/.prettierignore":                     "templates/.prettierignore",
	"web/storefront/.prettierrc.json":                    "templates/.prettierrc.json",
	"web/storefront/README.md":                           "templates/README.md.tmpl",
	"web/storefront/eslint.config.mjs":                   "templates/eslint.config.mjs",
	"web/storefront/nuxt.config.ts":                      "templates/nuxt.config.ts.tmpl",
	"web/storefront/package-lock.json":                   "templates/package-lock.json",
	"web/storefront/package.json":                        "templates/package.json",
	"web/storefront/tsconfig.json":                       "templates/tsconfig.json",
	"web/storefront/vitest.config.ts":                    "templates/vitest.config.ts",
	"web/storefront/app/app.vue":                         "templates/app/app.vue",
	"web/storefront/app/error.vue":                       "templates/app/error.vue",
	"web/storefront/app/assets/css/main.css":             "templates/app/assets/css/main.css",
	"web/storefront/app/pages/index.vue":                 "templates/app/pages/index.vue.tmpl",
	"web/storefront/server/api/storefront/status.get.ts": "templates/server/api/storefront/status.get.ts",
	"web/storefront/server/utils/backend.ts":             "templates/server/utils/backend.ts",
	"web/storefront/shared/types/storefront.ts":          "templates/shared/types/storefront.ts",
	"web/storefront/test/backend.test.ts":                "templates/test/backend.test.ts",
}

// Render executes one shared template against a backend-neutral data shape.
func Render(templatePath string, data Data) ([]byte, error) {
	content, err := templateFS.ReadFile(templatePath)
	if err != nil {
		return nil, fmt.Errorf("read storefront template: %w", err)
	}
	parsed, err := template.New(templatePath).
		Delims("[[", "]]").
		Option("missingkey=error").
		Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("parse storefront template: %w", err)
	}
	var output bytes.Buffer
	if err := parsed.Execute(&output, data); err != nil {
		return nil, fmt.Errorf("execute storefront template: %w", err)
	}
	return output.Bytes(), nil
}
