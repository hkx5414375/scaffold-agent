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
	ProjectName      string
	DisplayNameHTML  string
	DisplayNameJSON  string
	DescriptionJSON  string
	DescriptionLong  bool
	Catalog          bool
	CustomerAccounts bool
	Commerce         bool
	Tenancy          bool
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
	description := "A deterministic storefront generated for " + displayName + "."
	descriptionJSON, err := json.Marshal(description)
	if err != nil {
		panic("encode storefront description: " + err.Error())
	}
	return Data{
		ProjectName:     projectName,
		DisplayNameHTML: html.EscapeString(displayName),
		DisplayNameJSON: string(displayNameJSON),
		DescriptionJSON: string(descriptionJSON),
		DescriptionLong: len([]rune(string(descriptionJSON))) > 61,
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

// CatalogTemplates maps the composable public product catalog outputs.
var CatalogTemplates = map[string]string{
	"web/storefront/app/assets/css/catalog.css":                 "templates/app/assets/css/catalog.css",
	"web/storefront/app/pages/products/index.vue":               "templates/app/pages/products/index.vue",
	"web/storefront/app/pages/products/[id].vue":                "templates/app/pages/products/[id].vue",
	"web/storefront/server/api/storefront/products.get.ts":      "templates/server/api/storefront/products.get.ts",
	"web/storefront/server/api/storefront/products/[id].get.ts": "templates/server/api/storefront/products/[id].get.ts",
	"web/storefront/server/utils/catalog.ts":                    "templates/server/utils/catalog.ts",
	"web/storefront/shared/types/catalog.ts":                    "templates/shared/types/catalog.ts",
	"web/storefront/app/utils/catalog.ts":                       "templates/app/utils/catalog.ts",
	"web/storefront/test/catalog.test.ts":                       "templates/test/catalog.test.ts",
}

// CustomerAccountTemplates maps the composable storefront customer identity outputs.
var CustomerAccountTemplates = map[string]string{
	"web/storefront/app/assets/css/account.css":                     "templates/app/assets/css/account.css",
	"web/storefront/app/pages/account/index.vue":                    "templates/app/pages/account/index.vue",
	"web/storefront/app/pages/account/login.vue":                    "templates/app/pages/account/login.vue",
	"web/storefront/app/pages/account/register.vue":                 "templates/app/pages/account/register.vue",
	"web/storefront/server/api/storefront/account/close.post.ts":    "templates/server/api/storefront/account/close.post.ts",
	"web/storefront/server/api/storefront/account/login.post.ts":    "templates/server/api/storefront/account/login.post.ts",
	"web/storefront/server/api/storefront/account/logout.post.ts":   "templates/server/api/storefront/account/logout.post.ts",
	"web/storefront/server/api/storefront/account/me.get.ts":        "templates/server/api/storefront/account/me.get.ts",
	"web/storefront/server/api/storefront/account/password.put.ts":  "templates/server/api/storefront/account/password.put.ts",
	"web/storefront/server/api/storefront/account/profile.put.ts":   "templates/server/api/storefront/account/profile.put.ts",
	"web/storefront/server/api/storefront/account/register.post.ts": "templates/server/api/storefront/account/register.post.ts",
	"web/storefront/server/utils/customer.ts":                       "templates/server/utils/customer.ts",
	"web/storefront/shared/types/customer.ts":                       "templates/shared/types/customer.ts",
	"web/storefront/test/customer.test.ts":                          "templates/test/customer.test.ts",
}

// CommerceTemplates maps cart, checkout, and customer order storefront outputs.
var CommerceTemplates = map[string]string{
	"web/storefront/app/assets/css/commerce.css":                                   "templates/app/assets/css/commerce.css",
	"web/storefront/app/pages/cart.vue":                                            "templates/app/pages/cart.vue",
	"web/storefront/app/pages/checkout.vue":                                        "templates/app/pages/checkout.vue",
	"web/storefront/app/pages/account/orders/index.vue":                            "templates/app/pages/account/orders/index.vue",
	"web/storefront/app/pages/account/orders/[id].vue":                             "templates/app/pages/account/orders/[id].vue",
	"web/storefront/app/utils/commerce.ts":                                         "templates/app/utils/commerce.ts",
	"web/storefront/server/api/storefront/cart.get.ts":                             "templates/server/api/storefront/cart.get.ts",
	"web/storefront/server/api/storefront/cart/lines/[product_id].put.ts":          "templates/server/api/storefront/cart/lines/[product_id].put.ts",
	"web/storefront/server/api/storefront/cart/lines/[product_id]/remove.post.ts":  "templates/server/api/storefront/cart/lines/[product_id]/remove.post.ts",
	"web/storefront/server/api/storefront/checkout.post.ts":                        "templates/server/api/storefront/checkout.post.ts",
	"web/storefront/server/api/storefront/orders.get.ts":                           "templates/server/api/storefront/orders.get.ts",
	"web/storefront/server/api/storefront/orders/[id].get.ts":                      "templates/server/api/storefront/orders/[id].get.ts",
	"web/storefront/server/api/storefront/orders/[id]/return.post.ts":              "templates/server/api/storefront/orders/[id]/return.post.ts",
	"web/storefront/server/api/storefront/sandbox/payments/[provider_ref].post.ts": "templates/server/api/storefront/sandbox/payments/[provider_ref].post.ts",
	"web/storefront/server/utils/commerce.ts":                                      "templates/server/utils/commerce.ts",
	"web/storefront/shared/types/commerce.ts":                                      "templates/shared/types/commerce.ts",
	"web/storefront/test/commerce.test.ts":                                         "templates/test/commerce.test.ts",
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
