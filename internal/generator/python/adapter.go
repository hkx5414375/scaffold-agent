// Package python generates deterministic FastAPI services from project blueprints.
package python

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"github.com/hkx5414375/scaffold-agent/internal/change"
	"github.com/hkx5414375/scaffold-agent/internal/generator"
	"github.com/hkx5414375/scaffold-agent/internal/spec"
)

const (
	backend     = "python"
	baseOwner   = "python-service"
	baseVersion = "0.1.0"
)

//go:embed all:templates
var templateFS embed.FS

type databaseData struct {
	Engine           string
	DisplayName      string
	DriverDependency string
	ExampleURL       string
	LockTemplate     string
}

var databases = map[string]databaseData{
	"postgresql": {
		Engine:           "postgresql",
		DisplayName:      "PostgreSQL",
		DriverDependency: `"psycopg[binary]==3.3.5"`,
		ExampleURL:       "postgresql+psycopg://postgres:replace-me@localhost:5432/app",
		LockTemplate:     "templates/uv_postgresql.lock.tmpl",
	},
	"mysql": {
		Engine:           "mysql",
		DisplayName:      "MySQL",
		DriverDependency: `"pymysql==1.2.0"`,
		ExampleURL:       "mysql+pymysql://root:replace-me@localhost:3306/app?charset=utf8mb4",
		LockTemplate:     "templates/uv_mysql.lock.tmpl",
	},
}

var pythonKeywords = map[string]struct{}{
	"and": {}, "as": {}, "assert": {}, "async": {}, "await": {}, "break": {},
	"class": {}, "continue": {}, "def": {}, "del": {}, "elif": {}, "else": {},
	"except": {}, "finally": {}, "for": {}, "from": {}, "global": {}, "if": {},
	"import": {}, "in": {}, "is": {}, "lambda": {}, "nonlocal": {}, "not": {},
	"or": {}, "pass": {}, "raise": {}, "return": {}, "try": {}, "while": {},
	"with": {}, "yield": {},
}

type templateData struct {
	ProjectName string
	PackageName string
	Database    databaseData
}

var baseTemplates = map[string]string{
	".gitignore":                                        "templates/gitignore.tmpl",
	"README.md":                                         "templates/README.md.tmpl",
	"alembic.ini":                                       "templates/alembic.ini.tmpl",
	"api/openapi.yaml":                                  "templates/openapi.yaml.tmpl",
	"pyproject.toml":                                    "templates/pyproject.toml.tmpl",
	"src/package/__init__.py":                           "templates/package_init.py.tmpl",
	"src/package/config.py":                             "templates/config.py.tmpl",
	"src/package/database.py":                           "templates/database.py.tmpl",
	"src/package/errors.py":                             "templates/errors.py.tmpl",
	"src/package/health.py":                             "templates/health.py.tmpl",
	"src/package/main.py":                               "templates/main.py.tmpl",
	"src/package/migrations.py":                         "templates/migrations.py.tmpl",
	"src/package/identity/__init__.py":                  "templates/identity_init.py.tmpl",
	"src/package/identity/http.py":                      "templates/identity_http.py.tmpl",
	"src/package/identity/models.py":                    "templates/identity_models.py.tmpl",
	"src/package/identity/passwords.py":                 "templates/identity_passwords.py.tmpl",
	"src/package/identity/repository.py":                "templates/identity_repository.py.tmpl",
	"src/package/identity/service.py":                   "templates/identity_service.py.tmpl",
	"src/package/identity/tokens.py":                    "templates/identity_tokens.py.tmpl",
	"src/package/migration/__init__.py":                 "templates/migration_init.py.tmpl",
	"src/package/migration/env.py":                      "templates/migration_env.py.tmpl",
	"src/package/migration/versions/__init__.py":        "templates/migration_versions_init.py.tmpl",
	"src/package/migration/versions/000001_identity.py": "templates/identity_migration.py.tmpl",
	"tests/test_architecture.py":                        "templates/test_architecture.py.tmpl",
	"tests/test_health.py":                              "templates/test_health.py.tmpl",
	"tests/test_identity.py":                            "templates/test_identity.py.tmpl",
	"tests/test_identity_database.py":                   "templates/test_identity_database.py.tmpl",
	"tests/test_passwords.py":                           "templates/test_passwords.py.tmpl",
}

// Adapter generates the Python backend.
type Adapter struct{}

// New returns the built-in Python generator.
func New() Adapter {
	return Adapter{}
}

// Backend returns the Blueprint backend name.
func (Adapter) Backend() string {
	return backend
}

// Generate returns deterministic foundation outputs without touching the project filesystem.
func (Adapter) Generate(ctx context.Context, project spec.Project) (generator.Result, error) {
	if err := ctx.Err(); err != nil {
		return generator.Result{}, err
	}
	if project.Spec.Stack.Backend != backend {
		return generator.Result{}, fmt.Errorf("backend must be %q for the Python adapter", backend)
	}
	database, supported := databases[project.Spec.Database.Engine]
	if !supported {
		return generator.Result{}, fmt.Errorf("the Python adapter supports only PostgreSQL and MySQL")
	}
	if enabled(project.Spec.Stack.AdminUI) {
		return generator.Result{}, fmt.Errorf("the Python foundation does not generate an administration UI yet")
	}
	if enabled(project.Spec.Stack.Storefront) {
		return generator.Result{}, fmt.Errorf("the Python adapter does not generate a storefront yet")
	}
	if !hasExactAuthModes(project.Spec.Auth.Modes, "session", "token") {
		return generator.Result{}, fmt.Errorf("the Python adapter requires both session and token authentication")
	}
	if len(project.Spec.Capabilities) > 0 {
		return generator.Result{}, fmt.Errorf("the Python foundation does not support capability selections yet")
	}
	if len(project.Spec.Modules) > 0 {
		return generator.Result{}, fmt.Errorf("the Python foundation does not generate business modules yet")
	}

	data := templateData{
		ProjectName: project.Metadata.Name,
		PackageName: pythonIdentifier(project.Metadata.Name),
		Database:    database,
	}
	targets := make(map[string]string, len(baseTemplates)+1)
	for path, templatePath := range baseTemplates {
		targets[replacePackage(path, data.PackageName)] = templatePath
	}
	targets["uv.lock"] = database.LockTemplate

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
		content, err := render(targets[path], data)
		if err != nil {
			return generator.Result{}, fmt.Errorf("render %q: %w", path, err)
		}
		outputs = append(outputs, change.Output{Path: path, Owner: baseOwner, Content: content})
	}
	return generator.Result{
		CapabilityLock: map[string]string{baseOwner: baseVersion},
		Outputs:        outputs,
	}, nil
}

func render(path string, data templateData) ([]byte, error) {
	content, err := templateFS.ReadFile(path)
	if err != nil {
		return nil, err
	}
	parsed, err := template.New(path).Option("missingkey=error").Parse(string(content))
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := parsed.Execute(&output, data); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func replacePackage(path, packageName string) string {
	return strings.Replace(path, "package", packageName, 1)
}

func enabled(value string) bool {
	return value != "" && value != "none"
}

func hasExactAuthModes(values []string, expected ...string) bool {
	if len(values) != len(expected) {
		return false
	}
	actual := make(map[string]struct{}, len(values))
	for _, value := range values {
		actual[value] = struct{}{}
	}
	for _, value := range expected {
		if _, exists := actual[value]; !exists {
			return false
		}
	}
	return true
}

func pythonIdentifier(value string) string {
	var output strings.Builder
	for _, character := range strings.ToLower(value) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) || character == '_' {
			output.WriteRune(character)
		} else {
			output.WriteByte('_')
		}
	}
	identifier := strings.Trim(output.String(), "_")
	if identifier == "" {
		return "generated_service"
	}
	if _, reserved := pythonKeywords[identifier]; reserved {
		return "service_" + identifier
	}
	first, _ := utf8.DecodeRuneInString(identifier)
	if unicode.IsDigit(first) {
		return "service_" + identifier
	}
	return identifier
}
