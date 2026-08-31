// Package gogen implements the first-party Go modular-monolith adapter.
package gogen

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"go/format"
	"sort"
	"strings"
	"text/template"

	"github.com/hkx5414375/scaffold-agent/internal/change"
	"github.com/hkx5414375/scaffold-agent/internal/generator"
	"github.com/hkx5414375/scaffold-agent/internal/spec"
)

const (
	backend            = "go"
	baseCapability     = "go-service"
	baseVersion        = "0.1.0"
	businessCapability = "go-crud"
	businessVersion    = "0.1.0"
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
	"internal/platform/httpjson/httpjson.go":                   "templates/httpjson.go.tmpl",
	"internal/platform/migrate/migrate.go":                     "templates/migrate.go.tmpl",
	"internal/platform/migrate/migrations/000001_identity.sql": "templates/000001_identity.sql.tmpl",
	"internal/platform/postgres/pool.go":                       "templates/postgres_pool.go.tmpl",
}

var businessTemplates = []struct {
	PathSuffix   string
	TemplatePath string
}{
	{PathSuffix: "entity.go", TemplatePath: "templates/business_entity.go.tmpl"},
	{PathSuffix: "entity_test.go", TemplatePath: "templates/business_entity_test.go.tmpl"},
	{PathSuffix: "httpapi/handler.go", TemplatePath: "templates/business_handler.go.tmpl"},
	{PathSuffix: "postgres/store.go", TemplatePath: "templates/business_postgres_store.go.tmpl"},
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
	if len(project.Spec.Capabilities) > 0 {
		return generator.Result{}, fmt.Errorf("go capability packs are not available in the base adapter yet")
	}
	business, err := buildBusinessData(project.Spec.Modules)
	if err != nil {
		return generator.Result{}, err
	}
	data := templateData{
		ProjectName: project.Metadata.Name,
		ModulePath:  "example.com/" + project.Metadata.Name,
		Business:    business,
	}
	targets := make(map[string]renderTarget, len(outputTemplates)+len(businessTemplates)+1)
	for path, templatePath := range outputTemplates {
		targets[path] = renderTarget{TemplatePath: templatePath, Owner: baseCapability}
	}
	capabilityLock := map[string]string{baseCapability: baseVersion}
	if business != nil {
		for _, target := range businessTemplates {
			path := "internal/" + business.ModuleName + "/" + target.PathSuffix
			targets[path] = renderTarget{TemplatePath: target.TemplatePath, Owner: businessCapability}
		}
		migrationPath := "internal/platform/migrate/migrations/000100_" + business.ModuleName + "_" + business.EntityName + ".sql"
		targets[migrationPath] = renderTarget{TemplatePath: "templates/business.sql.tmpl", Owner: businessCapability}
		capabilityLock[businessCapability] = businessVersion
	}
	paths := make([]string, 0, len(targets))
	for path := range targets {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	outputs := make([]change.Output, 0, len(paths))
	for _, path := range paths {
		if err := ctx.Err(); err != nil {
			return generator.Result{}, err
		}
		target := targets[path]
		content, err := render(target.TemplatePath, data)
		if err != nil {
			return generator.Result{}, fmt.Errorf("render %q: %w", path, err)
		}
		if strings.HasSuffix(path, ".go") {
			content, err = format.Source(content)
			if err != nil {
				return generator.Result{}, fmt.Errorf("format generated %q: %w", path, err)
			}
		}
		outputs = append(outputs, change.Output{Path: path, Owner: target.Owner, Content: content})
	}
	return generator.Result{
		CapabilityLock: capabilityLock,
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
	Business    *businessData
}

type renderTarget struct {
	TemplatePath string
	Owner        string
}

type businessData struct {
	ModuleName       string
	PackageName      string
	EntityName       string
	EntityType       string
	TableName        string
	RoutePath        string
	PermissionPrefix string
	Fields           []businessField
}

type businessField struct {
	Name       string
	GoName     string
	GoType     string
	EntityType string
	SQLType    string
	Required   bool
	Unique     bool
	StringLike bool
	SampleOne  string
	SampleTwo  string
}

func buildBusinessData(modules []spec.Module) (*businessData, error) {
	if len(modules) == 0 {
		return nil, nil
	}
	if len(modules) != 1 {
		return nil, fmt.Errorf("the first Go CRUD slice supports exactly one business module")
	}
	module := modules[0]
	if strings.Contains(module.Name, "-") {
		return nil, fmt.Errorf("Go business module names must use lowercase letters and digits without hyphens")
	}
	if _, reserved := goKeywords[module.Name]; reserved {
		return nil, fmt.Errorf("Go business module name %q is a language keyword", module.Name)
	}
	if len(module.Entities) != 1 {
		return nil, fmt.Errorf("the first Go CRUD slice supports exactly one entity")
	}
	if len(module.Workflows) > 0 || len(module.Pages) > 0 {
		return nil, fmt.Errorf("Go workflows and generated pages are not available in the first CRUD slice")
	}
	entity := module.Entities[0]
	prefix := module.Name + ":" + entity.Name
	wantPermissions := map[string]struct{}{
		prefix + ":create": {},
		prefix + ":read":   {},
		prefix + ":update": {},
		prefix + ":delete": {},
	}
	if len(module.Permissions) != len(wantPermissions) {
		return nil, fmt.Errorf("business module permissions must declare create, read, update, and delete codes for its entity")
	}
	for _, permission := range module.Permissions {
		if _, exists := wantPermissions[permission.Code]; !exists {
			return nil, fmt.Errorf("unsupported business permission %q", permission.Code)
		}
		delete(wantPermissions, permission.Code)
	}
	fields := make([]businessField, 0, len(entity.Fields))
	reserved := map[string]struct{}{"id": {}, "version": {}, "created_at": {}, "updated_at": {}}
	for _, field := range entity.Fields {
		if _, exists := reserved[field.Name]; exists {
			return nil, fmt.Errorf("business field %q is reserved", field.Name)
		}
		goType, sqlType, stringLike, sampleOne, sampleTwo, ok := businessFieldType(field.Type)
		if !ok {
			return nil, fmt.Errorf("business field %q uses unsupported Go CRUD type %q", field.Name, field.Type)
		}
		entityType := goType
		if !field.Required {
			entityType = "*" + goType
		}
		fields = append(fields, businessField{
			Name: field.Name, GoName: exportedName(field.Name), GoType: goType,
			EntityType: entityType, SQLType: sqlType, Required: field.Required,
			Unique: field.Unique, StringLike: stringLike, SampleOne: sampleOne, SampleTwo: sampleTwo,
		})
	}
	return &businessData{
		ModuleName: module.Name, PackageName: module.Name, EntityName: entity.Name,
		EntityType: exportedName(entity.Name), TableName: module.Name + "_" + entity.Name,
		RoutePath: "/api/v1/" + module.Name, PermissionPrefix: prefix, Fields: fields,
	}, nil
}

func businessFieldType(value string) (goType, sqlType string, stringLike bool, sampleOne, sampleTwo string, ok bool) {
	switch value {
	case "string":
		return "string", "varchar(255)", true, `"first"`, `"second"`, true
	case "text":
		return "string", "text", true, `"first text"`, `"second text"`, true
	case "bool":
		return "bool", "boolean", false, "true", "false", true
	case "int64":
		return "int64", "bigint", false, "int64(1)", "int64(2)", true
	case "datetime":
		return "time.Time", "timestamptz", false, "time.Unix(1_700_000_000, 0).UTC()", "time.Unix(1_700_000_100, 0).UTC()", true
	default:
		return "", "", false, "", "", false
	}
}

func exportedName(value string) string {
	parts := strings.Split(value, "_")
	for index, part := range parts {
		if part != "" {
			parts[index] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
}

var goKeywords = map[string]struct{}{
	"break": {}, "default": {}, "func": {}, "interface": {}, "select": {},
	"case": {}, "defer": {}, "go": {}, "map": {}, "struct": {}, "chan": {},
	"else": {}, "goto": {}, "package": {}, "switch": {}, "const": {}, "fallthrough": {},
	"if": {}, "range": {}, "type": {}, "continue": {}, "for": {}, "import": {},
	"return": {}, "var": {},
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
