// Package python generates deterministic FastAPI services from project blueprints.
package python

import (
	"bytes"
	"context"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"unicode"
	"unicode/utf8"

	"github.com/hkx5414375/scaffold-agent/internal/capability"
	"github.com/hkx5414375/scaffold-agent/internal/change"
	"github.com/hkx5414375/scaffold-agent/internal/generator"
	adminui "github.com/hkx5414375/scaffold-agent/internal/generator/admin"
	"github.com/hkx5414375/scaffold-agent/internal/spec"
)

const (
	backend        = "python"
	baseOwner      = "python-service"
	baseVersion    = "0.1.0"
	crudOwner      = "python-crud"
	crudVersion    = "0.1.0"
	tenancyOwner   = "organization-tenancy"
	tenancyVersion = "0.1.0"
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

var pythonCapabilityCatalog = capability.NewCatalog(
	spec.CapabilityPack{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindCapabilityPack,
		Metadata:   spec.Metadata{Name: tenancyOwner, Version: tenancyVersion},
		Spec: spec.CapabilityPackSpec{
			Description: "Organization creation, membership-scoped RBAC, and tenant data isolation.",
			Backends:    []string{backend},
			Databases:   []string{"postgresql", "mysql"},
		},
	},
)

var pythonKeywords = map[string]struct{}{
	"and": {}, "as": {}, "assert": {}, "async": {}, "await": {}, "break": {},
	"class": {}, "continue": {}, "def": {}, "del": {}, "elif": {}, "else": {},
	"except": {}, "finally": {}, "for": {}, "from": {}, "global": {}, "if": {},
	"import": {}, "in": {}, "is": {}, "lambda": {}, "nonlocal": {}, "not": {},
	"or": {}, "pass": {}, "raise": {}, "return": {}, "try": {}, "while": {},
	"with": {}, "yield": {},
}

type templateData struct {
	ProjectName           string
	PackageName           string
	Database              databaseData
	Business              *businessData
	BusinessBeforeTenancy bool
	Admin                 bool
	Tenancy               bool
	TenancyMembers        bool
	TenancyLifecycle      bool
	Files                 bool
	JobAdmin              bool
	Approvals             bool
	CSVTransfer           bool
}

type businessData struct {
	ModuleName           string
	EntityName           string
	EntityClass          string
	EntityType           string
	PackageName          string
	TableName            string
	IndexName            string
	VersionCheckName     string
	PrimaryKeyName       string
	TenantForeignKeyName string
	RoutePath            string
	PermissionPrefix     string
	Fields               []businessField
	HasDateTime          bool
	HasInt64             bool
	HasUnique            bool
	HasInputChecks       bool
	DateTimeFields       []string
}

type businessField struct {
	Name              string
	PythonType        string
	SQLAlchemyType    string
	Required          bool
	Unique            bool
	UniqueName        string
	StringLike        bool
	MaximumLength     int
	Int64             bool
	DateTime          bool
	SampleOne         string
	SampleTwo         string
	JSONSampleOne     string
	OpenAPIType       string
	OpenAPIFormat     string
	OpenAPIPattern    string
	GoName            string
	TypeScriptType    string
	TypeScriptDefault string
	InputKind         string
}

type renderTarget struct {
	Template    string
	Owner       string
	SharedAdmin bool
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

var businessTemplates = map[string]string{
	"src/package/business/__init__.py":                  "templates/business_init.py.tmpl",
	"src/package/business/http.py":                      "templates/business_http.py.tmpl",
	"src/package/business/models.py":                    "templates/business_models.py.tmpl",
	"src/package/business/repository.py":                "templates/business_repository.py.tmpl",
	"src/package/business/service.py":                   "templates/business_service.py.tmpl",
	"src/package/migration/versions/000002_business.py": "templates/business_migration.py.tmpl",
	"tests/test_business.py":                            "templates/test_business.py.tmpl",
	"tests/test_business_database.py":                   "templates/test_business_database.py.tmpl",
}

var tenancyTemplates = map[string]string{
	"src/package/tenancy/__init__.py":                  "templates/tenancy_init.py.tmpl",
	"src/package/tenancy/http.py":                      "templates/tenancy_http.py.tmpl",
	"src/package/tenancy/models.py":                    "templates/tenancy_models.py.tmpl",
	"src/package/tenancy/repository.py":                "templates/tenancy_repository.py.tmpl",
	"src/package/tenancy/service.py":                   "templates/tenancy_service.py.tmpl",
	"src/package/migration/versions/000050_tenancy.py": "templates/tenancy_migration.py.tmpl",
	"tests/test_tenancy.py":                            "templates/test_tenancy.py.tmpl",
	"tests/test_tenancy_database.py":                   "templates/test_tenancy_database.py.tmpl",
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
	adminEnabled := enabled(project.Spec.Stack.AdminUI)
	if adminEnabled && project.Spec.Stack.AdminUI != "element-plus" {
		return generator.Result{}, fmt.Errorf("the Python adapter supports only the Element Plus administration UI")
	}
	if enabled(project.Spec.Stack.Storefront) {
		return generator.Result{}, fmt.Errorf("the Python adapter does not generate a storefront yet")
	}
	if !hasExactAuthModes(project.Spec.Auth.Modes, "session", "token") {
		return generator.Result{}, fmt.Errorf("the Python adapter requires both session and token authentication")
	}
	resolvedCapabilities, diagnostics := capability.Resolve(
		pythonCapabilityCatalog, project.Spec.Capabilities,
	)
	if len(diagnostics) > 0 {
		return generator.Result{}, fmt.Errorf(
			"resolve Python capabilities: %s", diagnostics[0].Message,
		)
	}
	for _, selection := range project.Spec.Capabilities {
		if len(selection.Config) > 0 {
			return generator.Result{}, fmt.Errorf(
				"Python capability %q does not accept configuration in this version",
				selection.Name,
			)
		}
	}
	tenancyEnabled := false
	for _, pack := range resolvedCapabilities {
		if pack.Metadata.Name == tenancyOwner {
			tenancyEnabled = true
		}
	}
	if len(project.Spec.Modules) > 1 {
		return generator.Result{}, fmt.Errorf("the Python CRUD slice supports at most one business module")
	}
	var business *businessData
	if len(project.Spec.Modules) == 1 {
		var err error
		business, err = buildBusiness(project.Spec.Modules[0], database.Engine)
		if err != nil {
			return generator.Result{}, err
		}
	}

	data := templateData{
		ProjectName:           project.Metadata.Name,
		PackageName:           pythonIdentifier(project.Metadata.Name),
		Database:              database,
		Business:              business,
		BusinessBeforeTenancy: business != nil && business.PackageName < "tenancy",
		Admin:                 adminEnabled,
		Tenancy:               tenancyEnabled,
	}
	targets := make(map[string]renderTarget, len(baseTemplates)+len(businessTemplates)+len(tenancyTemplates)+2)
	for path, templatePath := range baseTemplates {
		targets[replacePackage(path, data.PackageName)] = renderTarget{Template: templatePath, Owner: baseOwner}
	}
	targets["uv.lock"] = renderTarget{Template: database.LockTemplate, Owner: baseOwner}
	if business != nil {
		targets[replacePackage("src/package/migration/env.py", data.PackageName)] = renderTarget{
			Template: "templates/migration_env_business.py.tmpl",
			Owner:    baseOwner,
		}
		for path, templatePath := range businessTemplates {
			path = replacePackage(path, data.PackageName)
			path = strings.Replace(path, "business", business.PackageName, 1)
			targets[path] = renderTarget{Template: templatePath, Owner: crudOwner}
		}
	}
	if tenancyEnabled {
		mainTemplate := "templates/main_tenancy_foundation.py.tmpl"
		if business != nil {
			mainTemplate = "templates/main_tenancy.py.tmpl"
		}
		targets[replacePackage("src/package/main.py", data.PackageName)] = renderTarget{
			Template: mainTemplate,
			Owner:    baseOwner,
		}
		for path, templatePath := range tenancyTemplates {
			targets[replacePackage(path, data.PackageName)] = renderTarget{
				Template: templatePath,
				Owner:    tenancyOwner,
			}
		}
		if business != nil {
			targets[replacePackage("src/package/migration/versions/000051_tenant_business.py", data.PackageName)] = renderTarget{
				Template: "templates/tenant_business_migration.py.tmpl",
				Owner:    tenancyOwner,
			}
		}
	}
	if adminEnabled {
		for path, templatePath := range adminui.BaseTemplates {
			targets[path] = renderTarget{
				Template: templatePath, Owner: adminui.Owner, SharedAdmin: true,
			}
		}
		if business != nil {
			targets["web/admin/src/views/BusinessView.vue"] = renderTarget{
				Template: adminui.BusinessViewTemplate, Owner: adminui.Owner, SharedAdmin: true,
			}
		}
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
		var content []byte
		var err error
		if target.SharedAdmin {
			content, err = adminui.Render(target.Template, data)
		} else {
			content, err = render(target.Template, data)
		}
		if err != nil {
			return generator.Result{}, fmt.Errorf("render %q: %w", path, err)
		}
		outputs = append(outputs, change.Output{Path: path, Owner: target.Owner, Content: content})
	}
	capabilityLock := map[string]string{baseOwner: baseVersion}
	if business != nil {
		capabilityLock[crudOwner] = crudVersion
	}
	if adminEnabled {
		capabilityLock[adminui.Owner] = adminui.Version
	}
	if tenancyEnabled {
		capabilityLock[tenancyOwner] = tenancyVersion
	}
	return generator.Result{
		CapabilityLock: capabilityLock,
		Outputs:        outputs,
	}, nil
}

func buildBusiness(module spec.Module, databaseEngine string) (*businessData, error) {
	if _, reserved := pythonKeywords[module.Name]; reserved {
		return nil, fmt.Errorf("Python business module name %q is a language keyword", module.Name)
	}
	if len(module.Entities) != 1 {
		return nil, fmt.Errorf("the Python CRUD slice requires exactly one entity")
	}
	if len(module.Workflows) > 0 || len(module.Pages) > 0 {
		return nil, fmt.Errorf("the Python CRUD slice does not support workflows or pages yet")
	}
	entity := module.Entities[0]
	if _, reserved := pythonKeywords[entity.Name]; reserved {
		return nil, fmt.Errorf("Python business entity name %q is a language keyword", entity.Name)
	}
	tableName := module.Name + "_" + entity.Name
	maximumIdentifierLength := 63
	if databaseEngine == "mysql" {
		maximumIdentifierLength = 64
	}
	if len(tableName) > maximumIdentifierLength {
		return nil, fmt.Errorf("business table name %q exceeds the %s identifier limit", tableName, databaseEngine)
	}
	prefix := module.Name + ":" + entity.Name
	wantedPermissions := map[string]struct{}{
		prefix + ":create": {}, prefix + ":read": {}, prefix + ":update": {}, prefix + ":delete": {},
	}
	if len(module.Permissions) != len(wantedPermissions) {
		return nil, fmt.Errorf("business module permissions must declare create, read, update, and delete codes for its entity")
	}
	for _, permission := range module.Permissions {
		if _, exists := wantedPermissions[permission.Code]; !exists {
			return nil, fmt.Errorf("unsupported business permission %q", permission.Code)
		}
		delete(wantedPermissions, permission.Code)
	}
	reservedFields := map[string]struct{}{
		"id": {}, "version": {}, "created_at": {}, "updated_at": {}, "model_config": {},
	}
	fields := make([]businessField, 0, len(entity.Fields))
	hasDateTime := false
	hasInt64 := false
	hasUnique := false
	hasInputChecks := false
	dateTimeFields := make([]string, 0, len(entity.Fields))
	for _, field := range entity.Fields {
		if _, reserved := reservedFields[field.Name]; reserved {
			return nil, fmt.Errorf("business field %q is reserved", field.Name)
		}
		if _, reserved := pythonKeywords[field.Name]; reserved {
			return nil, fmt.Errorf("Python business field name %q is a language keyword", field.Name)
		}
		fieldData, supported := pythonBusinessField(field.Type)
		if !supported {
			return nil, fmt.Errorf("business field %q uses unsupported Python CRUD type %q", field.Name, field.Type)
		}
		if databaseEngine == "mysql" && field.Type == "text" && field.Unique {
			return nil, fmt.Errorf("MySQL text field %q cannot use the portable unique constraint", field.Name)
		}
		fieldData.Name = field.Name
		fieldData.GoName = upperCamel(field.Name)
		fieldData.Required = field.Required
		fieldData.Unique = field.Unique
		fieldData.UniqueName = compactIdentifier("uq", tableName, field.Name)
		hasDateTime = hasDateTime || fieldData.DateTime
		hasInt64 = hasInt64 || fieldData.Int64
		hasUnique = hasUnique || fieldData.Unique
		hasInputChecks = hasInputChecks || fieldData.StringLike || fieldData.Int64
		if fieldData.DateTime {
			dateTimeFields = append(dateTimeFields, field.Name)
		}
		fields = append(fields, fieldData)
	}
	return &businessData{
		ModuleName: module.Name, EntityName: entity.Name, EntityClass: upperCamel(entity.Name),
		EntityType:  upperCamel(entity.Name),
		PackageName: module.Name, TableName: tableName,
		IndexName:            compactIdentifier("ix", tableName, "created_at_id"),
		VersionCheckName:     compactIdentifier("ck", tableName, "version_positive"),
		PrimaryKeyName:       compactIdentifier("pk", tableName, "id"),
		TenantForeignKeyName: compactIdentifier("fk", tableName, "organization_id"),
		RoutePath:            "/api/v1/" + module.Name, PermissionPrefix: prefix, Fields: fields,
		HasDateTime: hasDateTime, HasInt64: hasInt64, HasUnique: hasUnique,
		HasInputChecks: hasInputChecks, DateTimeFields: dateTimeFields,
	}, nil
}

func pythonBusinessField(fieldType string) (businessField, bool) {
	switch fieldType {
	case "string":
		return businessField{PythonType: "str", SQLAlchemyType: "sa.String(255)", StringLike: true, MaximumLength: 255, SampleOne: `"first"`, SampleTwo: `"second"`, JSONSampleOne: `"first"`, OpenAPIType: "string", TypeScriptType: "string", TypeScriptDefault: `""`, InputKind: "text"}, true
	case "text":
		return businessField{PythonType: "str", SQLAlchemyType: "sa.Text()", StringLike: true, MaximumLength: 4000, SampleOne: `"first text"`, SampleTwo: `"second text"`, JSONSampleOne: `"first text"`, OpenAPIType: "string", TypeScriptType: "string", TypeScriptDefault: `""`, InputKind: "textarea"}, true
	case "bool":
		return businessField{PythonType: "bool", SQLAlchemyType: "sa.Boolean()", SampleOne: "True", SampleTwo: "False", JSONSampleOne: "true", OpenAPIType: "boolean", TypeScriptType: "boolean", TypeScriptDefault: "false", InputKind: "boolean"}, true
	case "int64":
		return businessField{PythonType: "str", SQLAlchemyType: "sa.BigInteger()", Int64: true, SampleOne: `"1"`, SampleTwo: `"2"`, JSONSampleOne: `"1"`, OpenAPIType: "string", OpenAPIPattern: `"^-?[0-9]+$"`, TypeScriptType: "string", TypeScriptDefault: `"0"`, InputKind: "text"}, true
	case "datetime":
		return businessField{PythonType: "datetime", SQLAlchemyType: "sa.DateTime(timezone=True)", DateTime: true, SampleOne: `datetime(2026, 9, 1, tzinfo=UTC)`, SampleTwo: `datetime(2026, 9, 2, tzinfo=UTC)`, JSONSampleOne: `"2026-09-01T00:00:00Z"`, OpenAPIType: "string", OpenAPIFormat: "date-time", TypeScriptType: "string", TypeScriptDefault: `""`, InputKind: "datetime"}, true
	default:
		return businessField{}, false
	}
}

func compactIdentifier(kind, tableName, suffix string) string {
	digest := sha256.Sum256([]byte(kind + ":" + tableName + ":" + suffix))
	return kind + "_" + hex.EncodeToString(digest[:8])
}

func upperCamel(value string) string {
	parts := strings.Split(value, "_")
	for index, part := range parts {
		if part != "" {
			parts[index] = strings.ToUpper(part[:1]) + part[1:]
		}
	}
	return strings.Join(parts, "")
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
