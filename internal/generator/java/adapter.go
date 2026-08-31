// Package java generates deterministic Spring Boot services from project blueprints.
package java

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"
	"text/template"
	"unicode"

	"github.com/hkx5414375/scaffold-agent/internal/capability"
	"github.com/hkx5414375/scaffold-agent/internal/change"
	"github.com/hkx5414375/scaffold-agent/internal/generator"
	adminui "github.com/hkx5414375/scaffold-agent/internal/generator/admin"
	"github.com/hkx5414375/scaffold-agent/internal/spec"
)

const (
	backend               = "java"
	baseOwner             = "java-service"
	baseVersion           = "0.3.0"
	businessOwner         = "java-crud"
	businessVersion       = "0.1.0"
	adminOwner            = adminui.Owner
	adminVersion          = adminui.Version
	tenancyOwner          = "organization-tenancy"
	tenancyVersion        = "0.1.0"
	tenancyMembersVersion = "0.2.0"
)

//go:embed all:templates
var templateFS embed.FS

var outputTemplates = map[string]string{
	".editorconfig":                       "templates/editorconfig.tmpl",
	".gitignore":                          "templates/gitignore.tmpl",
	"README.md":                           "templates/README.md.tmpl",
	"checkstyle.xml":                      "templates/checkstyle.xml.tmpl",
	"pom.xml":                             "templates/pom.xml.tmpl",
	"api/openapi.yaml":                    "templates/openapi.yaml.tmpl",
	"src/main/resources/application.yaml": "templates/application.yaml.tmpl",
}

type databaseData struct {
	Engine                          string
	DisplayName                     string
	DriverGroupID                   string
	DriverArtifactID                string
	FlywayGroupID                   string
	FlywayArtifactID                string
	IdentityMigrationTemplate       string
	BusinessMigrationTemplate       string
	TenancyMigrationTemplate        string
	TenancyMembersMigrationTemplate string
}

var databases = map[string]databaseData{
	"postgresql": {
		Engine:                          "postgresql",
		DisplayName:                     "PostgreSQL",
		DriverGroupID:                   "org.postgresql",
		DriverArtifactID:                "postgresql",
		FlywayGroupID:                   "org.flywaydb",
		FlywayArtifactID:                "flyway-database-postgresql",
		IdentityMigrationTemplate:       "templates/identity_postgresql.sql.tmpl",
		BusinessMigrationTemplate:       "templates/business_postgresql.sql.tmpl",
		TenancyMigrationTemplate:        "templates/tenancy_postgresql.sql.tmpl",
		TenancyMembersMigrationTemplate: "templates/tenancy_members_postgresql.sql.tmpl",
	},
	"mysql": {
		Engine:                          "mysql",
		DisplayName:                     "MySQL",
		DriverGroupID:                   "com.mysql",
		DriverArtifactID:                "mysql-connector-j",
		FlywayGroupID:                   "org.flywaydb",
		FlywayArtifactID:                "flyway-mysql",
		IdentityMigrationTemplate:       "templates/identity_mysql.sql.tmpl",
		BusinessMigrationTemplate:       "templates/business_mysql.sql.tmpl",
		TenancyMigrationTemplate:        "templates/tenancy_mysql.sql.tmpl",
		TenancyMembersMigrationTemplate: "templates/tenancy_members_mysql.sql.tmpl",
	},
}

var javaCapabilityCatalog = capability.NewCatalog(
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
	spec.CapabilityPack{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindCapabilityPack,
		Metadata:   spec.Metadata{Name: tenancyOwner, Version: tenancyMembersVersion},
		Spec: spec.CapabilityPackSpec{
			Description: "Organization invitations, member administration, and tenant data isolation.",
			Backends:    []string{backend},
			Databases:   []string{"postgresql", "mysql"},
		},
	},
)

type templateData struct {
	ProjectName      string
	ArtifactID       string
	PackageName      string
	PackagePath      string
	Database         databaseData
	Business         *businessData
	Admin            bool
	Tenancy          bool
	TenancyMembers   bool
	TenancyLifecycle bool
	Files            bool
	JobAdmin         bool
	CSVTransfer      bool
	Approvals        bool
	MigrationCount   int
}

type businessData struct {
	ModuleName       string
	EntityName       string
	EntityClass      string
	EntityType       string
	PackageName      string
	PackagePath      string
	TableName        string
	RoutePath        string
	PermissionPrefix string
	Fields           []businessField
	RequiredFields   []string
}

type businessField struct {
	Name              string
	JavaName          string
	JavaType          string
	EntityType        string
	SQLType           string
	Required          bool
	Unique            bool
	StringLike        bool
	MaximumLength     int
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

// Adapter generates the first-party Java service.
type Adapter struct{}

// New returns the first-party Java generator.
func New() Adapter {
	return Adapter{}
}

// Backend identifies this adapter.
func (Adapter) Backend() string {
	return backend
}

// Generate renders a deterministic Maven service with no runtime Engine dependency.
func (Adapter) Generate(ctx context.Context, project spec.Project) (generator.Result, error) {
	if err := ctx.Err(); err != nil {
		return generator.Result{}, err
	}
	if project.Spec.Stack.Backend != backend {
		return generator.Result{}, fmt.Errorf("backend must be %q for the Java adapter", backend)
	}
	database, supported := databases[project.Spec.Database.Engine]
	if !supported {
		return generator.Result{}, fmt.Errorf("the Java adapter supports only PostgreSQL and MySQL")
	}
	if enabled(project.Spec.Stack.Storefront) {
		return generator.Result{}, fmt.Errorf("the Java adapter does not generate a storefront yet")
	}
	adminEnabled := enabled(project.Spec.Stack.AdminUI)
	if adminEnabled && project.Spec.Stack.AdminUI != "element-plus" {
		return generator.Result{}, fmt.Errorf("the Java adapter supports only the Element Plus administration UI")
	}
	if !hasExactAuthModes(project.Spec.Auth.Modes, "session", "token") {
		return generator.Result{}, fmt.Errorf("the Java adapter requires both session and token authentication")
	}
	resolvedCapabilities, diagnostics := capability.Resolve(
		javaCapabilityCatalog, project.Spec.Capabilities,
	)
	if len(diagnostics) > 0 {
		return generator.Result{}, fmt.Errorf(
			"resolve Java capabilities: %s", diagnostics[0].Message,
		)
	}
	for _, selection := range project.Spec.Capabilities {
		if len(selection.Config) > 0 {
			return generator.Result{}, fmt.Errorf(
				"Java capability %q does not accept configuration in this version",
				selection.Name,
			)
		}
	}
	tenancyEnabled := false
	tenancyMembersEnabled := false
	selectedTenancyVersion := ""
	for _, pack := range resolvedCapabilities {
		if pack.Metadata.Name == tenancyOwner {
			tenancyEnabled = true
			selectedTenancyVersion = pack.Metadata.Version
			tenancyMembersEnabled = pack.Metadata.Version == tenancyMembersVersion
		}
	}
	business, err := buildBusinessData(project.Spec.Modules, database.Engine)
	if err != nil {
		return generator.Result{}, err
	}

	migrationCount := 1
	if business != nil {
		migrationCount++
	}
	if tenancyEnabled {
		migrationCount++
	}
	if tenancyMembersEnabled {
		migrationCount++
	}
	packageSegment := javaIdentifier(project.Metadata.Name)
	data := templateData{
		ProjectName:    project.Metadata.Name,
		ArtifactID:     project.Metadata.Name,
		PackageName:    "com.scaffold.generated." + packageSegment,
		PackagePath:    "com/scaffold/generated/" + packageSegment,
		Database:       database,
		Business:       business,
		Admin:          adminEnabled,
		Tenancy:        tenancyEnabled,
		TenancyMembers: tenancyMembersEnabled,
		MigrationCount: migrationCount,
	}
	targets := make(map[string]string, len(outputTemplates)+20)
	for path, templatePath := range outputTemplates {
		targets[path] = templatePath
	}
	businessPaths := make(map[string]struct{}, 9)
	adminPaths := make(map[string]struct{}, len(adminui.BaseTemplates)+1)
	tenancyPaths := make(map[string]struct{}, 19)
	addBusinessTarget := func(path, templatePath string) {
		targets[path] = templatePath
		businessPaths[path] = struct{}{}
	}
	addAdminTarget := func(path, templatePath string) {
		targets[path] = templatePath
		adminPaths[path] = struct{}{}
	}
	addTenancyTarget := func(path, templatePath string) {
		targets[path] = templatePath
		tenancyPaths[path] = struct{}{}
	}
	addTenancyAdminTarget := func(path, templatePath string) {
		targets[path] = templatePath
		adminPaths[path] = struct{}{}
		tenancyPaths[path] = struct{}{}
	}
	mainRoot := "src/main/java/" + data.PackagePath
	testRoot := "src/test/java/" + data.PackagePath
	targets[mainRoot+"/Application.java"] = "templates/Application.java.tmpl"
	targets[mainRoot+"/config/BootstrapAdmin.java"] = "templates/BootstrapAdmin.java.tmpl"
	targets[mainRoot+"/config/WebConfiguration.java"] = "templates/WebConfiguration.java.tmpl"
	targets[mainRoot+"/http/ApiError.java"] = "templates/ApiError.java.tmpl"
	targets[mainRoot+"/http/ApiExceptionHandler.java"] = "templates/ApiExceptionHandler.java.tmpl"
	targets[mainRoot+"/http/AuthController.java"] = "templates/AuthController.java.tmpl"
	targets[mainRoot+"/http/HealthController.java"] = "templates/HealthController.java.tmpl"
	targets[mainRoot+"/identity/AuditEvent.java"] = "templates/AuditEvent.java.tmpl"
	targets[mainRoot+"/identity/AuthenticationInterceptor.java"] = "templates/AuthenticationInterceptor.java.tmpl"
	targets[mainRoot+"/identity/IdentityException.java"] = "templates/IdentityException.java.tmpl"
	targets[mainRoot+"/identity/IdentityRepository.java"] = "templates/IdentityRepository.java.tmpl"
	targets[mainRoot+"/identity/IdentityService.java"] = "templates/IdentityService.java.tmpl"
	targets[mainRoot+"/identity/JdbcIdentityRepository.java"] = "templates/JdbcIdentityRepository.java.tmpl"
	targets[mainRoot+"/identity/PasswordHasher.java"] = "templates/PasswordHasher.java.tmpl"
	targets[mainRoot+"/identity/Principal.java"] = "templates/Principal.java.tmpl"
	targets[mainRoot+"/identity/RequirePermission.java"] = "templates/RequirePermission.java.tmpl"
	targets[mainRoot+"/identity/TokenCodec.java"] = "templates/TokenCodec.java.tmpl"
	targets[mainRoot+"/identity/User.java"] = "templates/User.java.tmpl"
	targets[testRoot+"/http/HealthControllerTest.java"] = "templates/HealthControllerTest.java.tmpl"
	targets[testRoot+"/identity/IdentityServiceTest.java"] = "templates/IdentityServiceTest.java.tmpl"
	targets[testRoot+"/identity/IdentityDatabaseIntegrationTest.java"] = "templates/IdentityDatabaseIntegrationTest.java.tmpl"
	targets[testRoot+"/identity/PasswordHasherTest.java"] = "templates/PasswordHasherTest.java.tmpl"
	targets[testRoot+"/architecture/ArchitectureTest.java"] = "templates/ArchitectureTest.java.tmpl"
	targets["src/main/resources/db/migration/V000001__identity.sql"] = database.IdentityMigrationTemplate
	capabilityLock := map[string]string{baseOwner: baseVersion}
	if tenancyEnabled {
		addTenancyTarget(mainRoot+"/tenancy/Organization.java", "templates/Organization.java.tmpl")
		addTenancyTarget(mainRoot+"/tenancy/TenancyException.java", "templates/TenancyException.java.tmpl")
		addTenancyTarget(mainRoot+"/tenancy/TenancyRepository.java", "templates/TenancyRepository.java.tmpl")
		addTenancyTarget(mainRoot+"/tenancy/TenancyService.java", "templates/TenancyService.java.tmpl")
		addTenancyTarget(mainRoot+"/tenancy/JdbcTenancyRepository.java", "templates/JdbcTenancyRepository.java.tmpl")
		addTenancyTarget(mainRoot+"/tenancy/OrganizationController.java", "templates/OrganizationController.java.tmpl")
		addTenancyTarget(testRoot+"/tenancy/TenancyServiceTest.java", "templates/TenancyServiceTest.java.tmpl")
		addTenancyTarget(testRoot+"/tenancy/TenancyDatabaseIntegrationTest.java", "templates/TenancyDatabaseIntegrationTest.java.tmpl")
		addTenancyTarget(
			"src/main/resources/db/migration/V000050__organization_tenancy.sql",
			database.TenancyMigrationTemplate,
		)
		capabilityLock[tenancyOwner] = selectedTenancyVersion
	}
	if tenancyMembersEnabled {
		addTenancyTarget(mainRoot+"/tenancy/OrganizationMember.java", "templates/OrganizationMember.java.tmpl")
		addTenancyTarget(mainRoot+"/tenancy/OrganizationInvitation.java", "templates/OrganizationInvitation.java.tmpl")
		addTenancyTarget(mainRoot+"/tenancy/TenancyMemberRepository.java", "templates/TenancyMemberRepository.java.tmpl")
		addTenancyTarget(mainRoot+"/tenancy/TenancyMemberService.java", "templates/TenancyMemberService.java.tmpl")
		addTenancyTarget(mainRoot+"/tenancy/JdbcTenancyMemberRepository.java", "templates/JdbcTenancyMemberRepository.java.tmpl")
		addTenancyTarget(mainRoot+"/tenancy/OrganizationMemberController.java", "templates/OrganizationMemberController.java.tmpl")
		addTenancyTarget(testRoot+"/tenancy/TenancyMemberServiceTest.java", "templates/TenancyMemberServiceTest.java.tmpl")
		addTenancyTarget(testRoot+"/tenancy/TenancyMemberDatabaseIntegrationTest.java", "templates/TenancyMemberDatabaseIntegrationTest.java.tmpl")
		addTenancyTarget(
			"src/main/resources/db/migration/V000060__organization_members.sql",
			database.TenancyMembersMigrationTemplate,
		)
		if adminEnabled {
			addTenancyAdminTarget(
				"web/admin/src/views/MembersView.vue",
				"templates/src/views/MembersView.vue",
			)
		}
	}
	if business != nil {
		businessRoot := mainRoot + "/" + business.PackagePath
		businessTestRoot := testRoot + "/" + business.PackagePath
		addBusinessTarget(businessRoot+"/"+business.EntityClass+".java", "templates/BusinessEntity.java.tmpl")
		addBusinessTarget(businessRoot+"/"+business.EntityClass+"Exception.java", "templates/BusinessException.java.tmpl")
		addBusinessTarget(businessRoot+"/"+business.EntityClass+"Repository.java", "templates/BusinessRepository.java.tmpl")
		addBusinessTarget(businessRoot+"/"+business.EntityClass+"Service.java", "templates/BusinessService.java.tmpl")
		addBusinessTarget(businessRoot+"/Jdbc"+business.EntityClass+"Repository.java", "templates/JdbcBusinessRepository.java.tmpl")
		addBusinessTarget(businessRoot+"/"+business.EntityClass+"Controller.java", "templates/BusinessController.java.tmpl")
		addBusinessTarget(businessTestRoot+"/"+business.EntityClass+"ServiceTest.java", "templates/BusinessServiceTest.java.tmpl")
		addBusinessTarget(businessTestRoot+"/"+business.EntityClass+"DatabaseIntegrationTest.java", "templates/BusinessDatabaseIntegrationTest.java.tmpl")
		addBusinessTarget("src/main/resources/db/migration/V000100__"+business.ModuleName+"_"+business.EntityName+".sql", database.BusinessMigrationTemplate)
		capabilityLock[businessOwner] = businessVersion
	}
	if adminEnabled {
		for path, templatePath := range adminui.BaseTemplates {
			addAdminTarget(path, templatePath)
		}
		if business != nil {
			addAdminTarget("web/admin/src/views/BusinessView.vue", adminui.BusinessViewTemplate)
		}
		capabilityLock[adminOwner] = adminVersion
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
		var content []byte
		var err error
		if _, isAdminPath := adminPaths[path]; isAdminPath {
			content, err = adminui.Render(targets[path], data)
		} else {
			content, err = render(targets[path], data)
		}
		if err != nil {
			return generator.Result{}, fmt.Errorf("render %q: %w", path, err)
		}
		owner := baseOwner
		if _, isBusinessPath := businessPaths[path]; isBusinessPath {
			owner = businessOwner
		}
		if _, isAdminPath := adminPaths[path]; isAdminPath {
			owner = adminOwner
		}
		if _, isTenancyPath := tenancyPaths[path]; isTenancyPath {
			owner = tenancyOwner
		}
		outputs = append(outputs, change.Output{Path: path, Owner: owner, Content: content})
	}
	return generator.Result{
		CapabilityLock: capabilityLock,
		Outputs:        outputs,
	}, nil
}

func render(path string, data templateData) ([]byte, error) {
	content, err := templateFS.ReadFile(path)
	if err != nil {
		return nil, err
	}
	functions := template.FuncMap{
		"sql": func(value string) string { return quoteSQLIdentifier(data.Database.Engine, value) },
		"javaSQL": func(value string) string {
			quoted := quoteSQLIdentifier(data.Database.Engine, value)
			return strings.ReplaceAll(strings.ReplaceAll(quoted, `\`, `\\`), `"`, `\"`)
		},
	}
	parsed, err := template.New(path).Funcs(functions).Option("missingkey=error").Parse(string(content))
	if err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := parsed.Execute(&output, data); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func buildBusinessData(modules []spec.Module, databaseEngine string) (*businessData, error) {
	if len(modules) == 0 {
		return nil, nil
	}
	if len(modules) != 1 {
		return nil, fmt.Errorf("the first Java CRUD slice supports exactly one business module")
	}
	module := modules[0]
	if strings.Contains(module.Name, "-") {
		return nil, fmt.Errorf("Java business module names must use lowercase letters and digits without hyphens")
	}
	if _, reserved := javaKeywords[module.Name]; reserved {
		return nil, fmt.Errorf("Java business module name %q is a language keyword", module.Name)
	}
	if len(module.Entities) != 1 {
		return nil, fmt.Errorf("the first Java CRUD slice supports exactly one entity")
	}
	if len(module.Pages) > 0 || len(module.Workflows) > 0 {
		return nil, fmt.Errorf("Java generated pages and workflows require a selected capability")
	}
	entity := module.Entities[0]
	if _, reserved := javaKeywords[entity.Name]; reserved {
		return nil, fmt.Errorf("Java business entity name %q is a language keyword", entity.Name)
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
		prefix + ":create": {},
		prefix + ":read":   {},
		prefix + ":update": {},
		prefix + ":delete": {},
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
	fields := make([]businessField, 0, len(entity.Fields))
	requiredFields := make([]string, 0, len(entity.Fields))
	reservedFields := map[string]struct{}{"id": {}, "version": {}, "created_at": {}, "updated_at": {}}
	for _, field := range entity.Fields {
		if _, reserved := reservedFields[field.Name]; reserved {
			return nil, fmt.Errorf("business field %q is reserved", field.Name)
		}
		javaName := lowerCamel(field.Name)
		if _, reserved := javaKeywords[javaName]; reserved {
			return nil, fmt.Errorf("Java business field name %q is a language keyword", field.Name)
		}
		fieldData, supported := javaBusinessField(field.Type, databaseEngine)
		if !supported {
			return nil, fmt.Errorf("business field %q uses unsupported Java CRUD type %q", field.Name, field.Type)
		}
		if databaseEngine == "mysql" && field.Type == "text" && field.Unique {
			return nil, fmt.Errorf("MySQL text field %q cannot use the portable unique constraint", field.Name)
		}
		fieldData.Name = field.Name
		fieldData.JavaName = javaName
		fieldData.GoName = upperCamel(field.Name)
		fieldData.Required = field.Required
		fieldData.Unique = field.Unique
		if field.Required {
			fieldData.EntityType = requiredJavaType(fieldData.JavaType)
			requiredFields = append(requiredFields, field.Name)
		} else {
			fieldData.EntityType = fieldData.JavaType
		}
		fields = append(fields, fieldData)
	}
	entityClass := upperCamel(entity.Name)
	return &businessData{
		ModuleName: module.Name, EntityName: entity.Name, EntityClass: entityClass,
		EntityType:  entityClass,
		PackageName: module.Name, PackagePath: module.Name,
		TableName: tableName, RoutePath: "/api/v1/" + module.Name,
		PermissionPrefix: prefix, Fields: fields, RequiredFields: requiredFields,
	}, nil
}

func javaBusinessField(fieldType, databaseEngine string) (businessField, bool) {
	switch fieldType {
	case "string":
		return businessField{JavaType: "String", SQLType: "varchar(255)", StringLike: true, MaximumLength: 255, SampleOne: `"first"`, SampleTwo: `"second"`, JSONSampleOne: `\"first\"`, OpenAPIType: "string", TypeScriptType: "string", TypeScriptDefault: `""`, InputKind: "text"}, true
	case "text":
		return businessField{JavaType: "String", SQLType: "text", StringLike: true, MaximumLength: 4000, SampleOne: `"first text"`, SampleTwo: `"second text"`, JSONSampleOne: `\"first text\"`, OpenAPIType: "string", TypeScriptType: "string", TypeScriptDefault: `""`, InputKind: "textarea"}, true
	case "bool":
		return businessField{JavaType: "Boolean", SQLType: "boolean", SampleOne: "true", SampleTwo: "false", JSONSampleOne: "true", OpenAPIType: "boolean", TypeScriptType: "boolean", TypeScriptDefault: "false", InputKind: "boolean"}, true
	case "int64":
		return businessField{JavaType: "Long", SQLType: "bigint", SampleOne: "1L", SampleTwo: "2L", JSONSampleOne: `\"1\"`, OpenAPIType: "string", OpenAPIPattern: `"^-?[0-9]+$"`, TypeScriptType: "string", TypeScriptDefault: `"0"`, InputKind: "text"}, true
	case "datetime":
		sqlType := "timestamptz"
		if databaseEngine == "mysql" {
			sqlType = "datetime(6)"
		}
		return businessField{JavaType: "Instant", SQLType: sqlType, SampleOne: `Instant.parse("2026-09-01T00:00:00Z")`, SampleTwo: `Instant.parse("2026-09-02T00:00:00Z")`, JSONSampleOne: `\"2026-09-01T00:00:00Z\"`, OpenAPIType: "string", OpenAPIFormat: "date-time", TypeScriptType: "string", TypeScriptDefault: `""`, InputKind: "datetime"}, true
	default:
		return businessField{}, false
	}
}

func requiredJavaType(javaType string) string {
	switch javaType {
	case "Boolean":
		return "boolean"
	case "Long":
		return "long"
	default:
		return javaType
	}
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

func lowerCamel(value string) string {
	upper := upperCamel(value)
	if upper == "" {
		return upper
	}
	return strings.ToLower(upper[:1]) + upper[1:]
}

func quoteSQLIdentifier(databaseEngine, value string) string {
	if databaseEngine == "mysql" {
		return "`" + strings.ReplaceAll(value, "`", "``") + "`"
	}
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

var javaKeywords = map[string]struct{}{
	"abstract": {}, "assert": {}, "boolean": {}, "break": {}, "byte": {},
	"case": {}, "catch": {}, "char": {}, "class": {}, "const": {}, "continue": {},
	"default": {}, "do": {}, "double": {}, "else": {}, "enum": {}, "extends": {},
	"final": {}, "finally": {}, "float": {}, "for": {}, "goto": {}, "if": {},
	"implements": {}, "import": {}, "instanceof": {}, "int": {}, "interface": {},
	"long": {}, "native": {}, "new": {}, "package": {}, "private": {}, "protected": {},
	"public": {}, "return": {}, "short": {}, "static": {}, "strictfp": {}, "super": {},
	"switch": {}, "synchronized": {}, "this": {}, "throw": {}, "throws": {},
	"transient": {}, "try": {}, "void": {}, "volatile": {}, "while": {},
	"false": {}, "null": {}, "true": {}, "_": {}, "record": {}, "sealed": {},
	"permits": {}, "var": {}, "when": {}, "yield": {},
}

func javaIdentifier(value string) string {
	var result strings.Builder
	for _, character := range strings.ToLower(value) {
		if unicode.IsLetter(character) || unicode.IsDigit(character) {
			result.WriteRune(character)
		}
	}
	if result.Len() == 0 {
		return "application"
	}
	return result.String()
}

func enabled(value string) bool {
	return value != "" && value != "none"
}

func hasExactAuthModes(actual []string, expected ...string) bool {
	if len(actual) != len(expected) {
		return false
	}
	actualSet := make(map[string]struct{}, len(actual))
	for _, value := range actual {
		actualSet[value] = struct{}{}
	}
	for _, value := range expected {
		if _, exists := actualSet[value]; !exists {
			return false
		}
	}
	return true
}
