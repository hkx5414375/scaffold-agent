// Package gogen implements the first-party Go modular-monolith adapter.
package gogen

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"sort"
	"text/template"

	"github.com/hkx5414375/scaffold-agent/internal/change"
	"github.com/hkx5414375/scaffold-agent/internal/generator"
	"github.com/hkx5414375/scaffold-agent/internal/spec"
)

const (
	backend        = "go"
	baseCapability = "go-service"
	baseVersion    = "0.1.0"
)

//go:embed templates
var templateFS embed.FS

var outputTemplates = map[string]string{
	".gitignore":                           "templates/gitignore.tmpl",
	"README.md":                            "templates/README.md.tmpl",
	"cmd/server/main.go":                   "templates/main.go.tmpl",
	"go.mod":                               "templates/go.mod.tmpl",
	"go.sum":                               "templates/go.sum.tmpl",
	"internal/identity/httpapi/handler.go": "templates/identity_handler.go.tmpl",
	"internal/identity/httpapi/handler_test.go":                "templates/identity_handler_test.go.tmpl",
	"internal/identity/password.go":                            "templates/password.go.tmpl",
	"internal/identity/password_test.go":                       "templates/password_test.go.tmpl",
	"internal/identity/postgres/store.go":                      "templates/identity_postgres_store.go.tmpl",
	"internal/identity/service.go":                             "templates/identity_service.go.tmpl",
	"internal/identity/service_test.go":                        "templates/identity_service_test.go.tmpl",
	"internal/identity/store.go":                               "templates/identity_store.go.tmpl",
	"internal/platform/httpserver/server.go":                   "templates/server.go.tmpl",
	"internal/platform/httpserver/server_test.go":              "templates/server_test.go.tmpl",
	"internal/platform/migrate/migrate.go":                     "templates/migrate.go.tmpl",
	"internal/platform/migrate/migrations/000001_identity.sql": "templates/000001_identity.sql.tmpl",
	"internal/platform/postgres/pool.go":                       "templates/postgres_pool.go.tmpl",
}

// Adapter generates the Go base service.
type Adapter struct{}

// New returns the first-party Go generator.
func New() Adapter {
	return Adapter{}
}

// Backend identifies this adapter.
func (Adapter) Backend() string {
	return backend
}

// Generate renders a deterministic service with no runtime Engine dependency.
func (Adapter) Generate(ctx context.Context, project spec.Project) (generator.Result, error) {
	if err := ctx.Err(); err != nil {
		return generator.Result{}, err
	}
	if project.Spec.Stack.Backend != backend {
		return generator.Result{}, fmt.Errorf("backend must be %q for the Go adapter", backend)
	}
	if project.Spec.Database.Engine != "postgresql" {
		return generator.Result{}, fmt.Errorf("the Go adapter currently supports only PostgreSQL")
	}
	if enabled(project.Spec.Stack.AdminUI) || enabled(project.Spec.Stack.Storefront) {
		return generator.Result{}, fmt.Errorf("the Go adapter does not generate frontends yet")
	}
	if !hasExactAuthModes(project.Spec.Auth.Modes, "session", "token") {
		return generator.Result{}, fmt.Errorf("the Go adapter currently requires both session and token authentication")
	}
	if len(project.Spec.Capabilities) > 0 || len(project.Spec.Modules) > 0 {
		return generator.Result{}, fmt.Errorf("go business capabilities and modules are not available in the base adapter yet")
	}
	data := templateData{
		ProjectName: project.Metadata.Name,
		ModulePath:  "example.com/" + project.Metadata.Name,
	}
	paths := make([]string, 0, len(outputTemplates))
	for path := range outputTemplates {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	outputs := make([]change.Output, 0, len(paths))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return generator.Result{}, err
		}
		content, err := render(outputTemplates[path], data)
		if err != nil {
			return generator.Result{}, fmt.Errorf("render %q: %w", path, err)
		}
		outputs = append(outputs, change.Output{Path: path, Owner: baseCapability, Content: content})
	}
	return generator.Result{
		CapabilityLock: map[string]string{baseCapability: baseVersion},
		Outputs:        outputs,
	}, nil
}

func enabled(value string) bool {
	return value != "" && value != "none"
}

func hasExactAuthModes(actual []string, expected ...string) bool {
	if len(actual) != len(expected) {
		return false
	}
	want := make(map[string]struct{}, len(expected))
	for _, mode := range expected {
		want[mode] = struct{}{}
	}
	for _, mode := range actual {
		if _, exists := want[mode]; !exists {
			return false
		}
		delete(want, mode)
	}
	return len(want) == 0
}

type templateData struct {
	ProjectName string
	ModulePath  string
}

func render(templatePath string, data templateData) ([]byte, error) {
	content, err := templateFS.ReadFile(templatePath)
	if err != nil {
		return nil, err
	}
	parsed, err := template.New(templatePath).Option("missingkey=error").Parse(string(content))
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := parsed.Execute(&output, data); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}
