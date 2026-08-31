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

	"github.com/hkx5414375/scaffold-agent/internal/change"
	"github.com/hkx5414375/scaffold-agent/internal/generator"
	"github.com/hkx5414375/scaffold-agent/internal/spec"
)

const (
	backend     = "java"
	baseOwner   = "java-service"
	baseVersion = "0.2.0"
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
	Engine                    string
	DisplayName               string
	DriverGroupID             string
	DriverArtifactID          string
	FlywayGroupID             string
	FlywayArtifactID          string
	IdentityMigrationTemplate string
}

var databases = map[string]databaseData{
	"postgresql": {
		Engine:                    "postgresql",
		DisplayName:               "PostgreSQL",
		DriverGroupID:             "org.postgresql",
		DriverArtifactID:          "postgresql",
		FlywayGroupID:             "org.flywaydb",
		FlywayArtifactID:          "flyway-database-postgresql",
		IdentityMigrationTemplate: "templates/identity_postgresql.sql.tmpl",
	},
	"mysql": {
		Engine:                    "mysql",
		DisplayName:               "MySQL",
		DriverGroupID:             "com.mysql",
		DriverArtifactID:          "mysql-connector-j",
		FlywayGroupID:             "org.flywaydb",
		FlywayArtifactID:          "flyway-mysql",
		IdentityMigrationTemplate: "templates/identity_mysql.sql.tmpl",
	},
}

type templateData struct {
	ProjectName string
	ArtifactID  string
	PackageName string
	PackagePath string
	Database    databaseData
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
	if enabled(project.Spec.Stack.AdminUI) || enabled(project.Spec.Stack.Storefront) {
		return generator.Result{}, fmt.Errorf("the Java foundation adapter does not generate frontends yet")
	}
	if !hasExactAuthModes(project.Spec.Auth.Modes, "session", "token") {
		return generator.Result{}, fmt.Errorf("the Java adapter requires both session and token authentication")
	}
	if len(project.Spec.Capabilities) > 0 {
		return generator.Result{}, fmt.Errorf("Java capability packs are not available in the foundation slice")
	}
	if len(project.Spec.Modules) > 0 {
		return generator.Result{}, fmt.Errorf("Java business modules are not available in the foundation slice")
	}

	packageSegment := javaIdentifier(project.Metadata.Name)
	data := templateData{
		ProjectName: project.Metadata.Name,
		ArtifactID:  project.Metadata.Name,
		PackageName: "com.scaffold.generated." + packageSegment,
		PackagePath: "com/scaffold/generated/" + packageSegment,
		Database:    database,
	}
	targets := make(map[string]string, len(outputTemplates)+20)
	for path, templatePath := range outputTemplates {
		targets[path] = templatePath
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
