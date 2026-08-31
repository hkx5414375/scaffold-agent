package spec

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

var (
	resourceNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{0,62}$`)
	fieldNamePattern    = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)
	permissionPattern   = regexp.MustCompile(`^[a-z][a-z0-9-]*(?::[a-z][a-z0-9-]*)+$`)
	semverPattern       = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$`)
)

var fieldTypes = setOf("string", "text", "bool", "int64", "decimal", "money", "date", "time", "datetime", "json", "ulid", "enum")

// ValidateProject performs model-neutral semantic validation after schema decoding.
func ValidateProject(project Project) []Diagnostic {
	var diagnostics []Diagnostic
	if project.APIVersion != APIVersionV1Alpha1 {
		diagnostics = append(diagnostics, errorDiagnostic("project.api_version.unsupported", "api_version", "api_version must be "+APIVersionV1Alpha1))
	}
	if project.Kind != KindProject {
		diagnostics = append(diagnostics, errorDiagnostic("project.kind.invalid", "kind", "kind must be "+KindProject))
	}
	diagnostics = append(diagnostics, validateResourceName(project.Metadata.Name, "metadata.name")...)
	if !contains(setOf("go", "java", "python"), project.Spec.Stack.Backend) {
		diagnostics = append(diagnostics, errorDiagnostic("project.stack.backend.invalid", "spec.stack.backend", "backend must be go, java, or python"))
	}
	if project.Spec.Stack.AdminUI != "" && !contains(setOf("none", "element-plus"), project.Spec.Stack.AdminUI) {
		diagnostics = append(diagnostics, errorDiagnostic("project.stack.admin_ui.invalid", "spec.stack.admin_ui", "admin_ui must be none or element-plus"))
	}
	if project.Spec.Stack.Storefront != "" && !contains(setOf("none", "nuxt"), project.Spec.Stack.Storefront) {
		diagnostics = append(diagnostics, errorDiagnostic("project.stack.storefront.invalid", "spec.stack.storefront", "storefront must be none or nuxt"))
	}
	if !contains(setOf("postgresql", "mysql"), project.Spec.Database.Engine) {
		diagnostics = append(diagnostics, errorDiagnostic("project.database.engine.invalid", "spec.database.engine", "database engine must be postgresql or mysql"))
	}
	diagnostics = append(diagnostics, validateAuth(project.Spec.Auth)...)
	diagnostics = append(diagnostics, validateCapabilities(project.Spec.Capabilities)...)
	diagnostics = append(diagnostics, validateModules(project.Spec.Modules)...)
	return diagnostics
}

// ValidateCapabilityPack performs model-neutral pack validation.
func ValidateCapabilityPack(pack CapabilityPack) []Diagnostic {
	var diagnostics []Diagnostic
	if pack.APIVersion != APIVersionV1Alpha1 {
		diagnostics = append(diagnostics, errorDiagnostic("pack.api_version.unsupported", "api_version", "api_version must be "+APIVersionV1Alpha1))
	}
	if pack.Kind != KindCapabilityPack {
		diagnostics = append(diagnostics, errorDiagnostic("pack.kind.invalid", "kind", "kind must be "+KindCapabilityPack))
	}
	diagnostics = append(diagnostics, validateResourceName(pack.Metadata.Name, "metadata.name")...)
	if !semverPattern.MatchString(pack.Metadata.Version) {
		diagnostics = append(diagnostics, errorDiagnostic("pack.version.invalid", "metadata.version", "version must be a semantic version without a v prefix"))
	}
	if strings.TrimSpace(pack.Spec.Description) == "" {
		diagnostics = append(diagnostics, errorDiagnostic("pack.description.required", "spec.description", "description is required"))
	}
	if len(pack.Spec.Backends) == 0 {
		diagnostics = append(diagnostics, errorDiagnostic("pack.backends.required", "spec.backends", "at least one backend is required"))
	}
	if len(pack.Spec.Databases) == 0 {
		diagnostics = append(diagnostics, errorDiagnostic("pack.databases.required", "spec.databases", "at least one database is required"))
	}
	diagnostics = append(diagnostics, validateStringSet(pack.Spec.Backends, setOf("go", "java", "python"), "pack.backend.invalid", "spec.backends")...)
	diagnostics = append(diagnostics, validateStringSet(pack.Spec.Databases, setOf("postgresql", "mysql"), "pack.database.invalid", "spec.databases")...)
	diagnostics = append(diagnostics, validatePackDependencies(pack)...)
	diagnostics = append(diagnostics, validatePackConflicts(pack)...)
	diagnostics = append(diagnostics, validatePackOutputs(pack.Metadata.Name, pack.Spec.Outputs)...)
	return diagnostics
}

func validateAuth(auth AuthSpec) []Diagnostic {
	if len(auth.Modes) == 0 {
		return []Diagnostic{errorDiagnostic("project.auth.modes.required", "spec.auth.modes", "at least one authentication mode is required")}
	}
	allowed := setOf("session", "token")
	seen := make(map[string]struct{}, len(auth.Modes))
	var diagnostics []Diagnostic
	for index, mode := range auth.Modes {
		path := fmt.Sprintf("spec.auth.modes[%d]", index)
		if !contains(allowed, mode) {
			diagnostics = append(diagnostics, errorDiagnostic("project.auth.mode.invalid", path, "authentication mode must be session or token"))
		}
		if _, exists := seen[mode]; exists {
			diagnostics = append(diagnostics, errorDiagnostic("project.auth.mode.duplicate", path, "authentication modes must be unique"))
		}
		seen[mode] = struct{}{}
	}
	return diagnostics
}

func validateCapabilities(capabilities []CapabilitySelection) []Diagnostic {
	seen := make(map[string]struct{}, len(capabilities))
	var diagnostics []Diagnostic
	for index, capability := range capabilities {
		basePath := fmt.Sprintf("spec.capabilities[%d]", index)
		diagnostics = append(diagnostics, validateResourceName(capability.Name, basePath+".name")...)
		if !semverPattern.MatchString(capability.Version) {
			diagnostics = append(diagnostics, errorDiagnostic("project.capability.version.invalid", basePath+".version", "capability version must be a semantic version without a v prefix"))
		}
		if _, exists := seen[capability.Name]; exists {
			diagnostics = append(diagnostics, errorDiagnostic("project.capability.duplicate", basePath+".name", "capability names must be unique"))
		}
		seen[capability.Name] = struct{}{}
	}
	return diagnostics
}

func validateModules(modules []Module) []Diagnostic {
	seen := make(map[string]struct{}, len(modules))
	var diagnostics []Diagnostic
	for moduleIndex, module := range modules {
		basePath := fmt.Sprintf("spec.modules[%d]", moduleIndex)
		diagnostics = append(diagnostics, validateResourceName(module.Name, basePath+".name")...)
		if _, exists := seen[module.Name]; exists {
			diagnostics = append(diagnostics, errorDiagnostic("project.module.duplicate", basePath+".name", "module names must be unique"))
		}
		seen[module.Name] = struct{}{}
		diagnostics = append(diagnostics, validateEntities(module.Entities, basePath+".entities")...)
		diagnostics = append(diagnostics, validateWorkflows(module.Workflows, basePath+".workflows")...)
		diagnostics = append(diagnostics, validatePermissions(module.Permissions, basePath+".permissions")...)
		diagnostics = append(diagnostics, validatePages(module.Pages, basePath+".pages")...)
		diagnostics = append(diagnostics, validateAcceptance(module.Acceptance, basePath+".acceptance")...)
	}
	return diagnostics
}

func validateEntities(entities []Entity, path string) []Diagnostic {
	seen := make(map[string]struct{}, len(entities))
	var diagnostics []Diagnostic
	for entityIndex, entity := range entities {
		basePath := fmt.Sprintf("%s[%d]", path, entityIndex)
		if !fieldNamePattern.MatchString(entity.Name) {
			diagnostics = append(diagnostics, errorDiagnostic("project.entity.name.invalid", basePath+".name", "entity name must use lower snake_case"))
		}
		if _, exists := seen[entity.Name]; exists {
			diagnostics = append(diagnostics, errorDiagnostic("project.entity.duplicate", basePath+".name", "entity names must be unique within a module"))
		}
		seen[entity.Name] = struct{}{}
		if len(entity.Fields) == 0 {
			diagnostics = append(diagnostics, errorDiagnostic("project.entity.fields.required", basePath+".fields", "entity must declare at least one field"))
		}
		fieldSeen := make(map[string]struct{}, len(entity.Fields))
		for fieldIndex, field := range entity.Fields {
			fieldPath := fmt.Sprintf("%s.fields[%d]", basePath, fieldIndex)
			if !fieldNamePattern.MatchString(field.Name) {
				diagnostics = append(diagnostics, errorDiagnostic("project.field.name.invalid", fieldPath+".name", "field name must use lower snake_case"))
			}
			if !contains(fieldTypes, field.Type) {
				diagnostics = append(diagnostics, errorDiagnostic("project.field.type.invalid", fieldPath+".type", "field type is not portable"))
			}
			if _, exists := fieldSeen[field.Name]; exists {
				diagnostics = append(diagnostics, errorDiagnostic("project.field.duplicate", fieldPath+".name", "field names must be unique within an entity"))
			}
			fieldSeen[field.Name] = struct{}{}
		}
	}
	return diagnostics
}

func validatePackDependencies(pack CapabilityPack) []Diagnostic {
	seen := make(map[string]struct{}, len(pack.Spec.Requires))
	var diagnostics []Diagnostic
	for index, dependency := range pack.Spec.Requires {
		basePath := fmt.Sprintf("spec.requires[%d]", index)
		diagnostics = append(diagnostics, validateResourceName(dependency.Name, basePath+".name")...)
		if dependency.Name == pack.Metadata.Name {
			diagnostics = append(diagnostics, errorDiagnostic("pack.dependency.self", basePath+".name", "a capability cannot require itself"))
		}
		if !validConstraint(dependency.Constraint) {
			diagnostics = append(diagnostics, errorDiagnostic("pack.dependency.constraint.invalid", basePath+".constraint", "constraint must be *, an exact semantic version, ^version, or >=version"))
		}
		if _, exists := seen[dependency.Name]; exists {
			diagnostics = append(diagnostics, errorDiagnostic("pack.dependency.duplicate", basePath+".name", "dependencies must be unique"))
		}
		seen[dependency.Name] = struct{}{}
	}
	return diagnostics
}

func validatePackConflicts(pack CapabilityPack) []Diagnostic {
	seen := make(map[string]struct{}, len(pack.Spec.Conflicts))
	var diagnostics []Diagnostic
	for index, conflict := range pack.Spec.Conflicts {
		itemPath := fmt.Sprintf("spec.conflicts[%d]", index)
		diagnostics = append(diagnostics, validateResourceName(conflict, itemPath)...)
		if conflict == pack.Metadata.Name {
			diagnostics = append(diagnostics, errorDiagnostic("pack.conflict.self", itemPath, "a capability cannot conflict with itself"))
		}
		if _, exists := seen[conflict]; exists {
			diagnostics = append(diagnostics, errorDiagnostic("pack.conflict.duplicate", itemPath, "conflicts must be unique"))
		}
		seen[conflict] = struct{}{}
	}
	return diagnostics
}

func validatePackOutputs(packName string, outputs []ManagedOutput) []Diagnostic {
	seen := make(map[string]struct{}, len(outputs))
	var diagnostics []Diagnostic
	for index, output := range outputs {
		basePath := fmt.Sprintf("spec.outputs[%d]", index)
		cleaned := path.Clean(strings.ReplaceAll(output.Path, "\\", "/"))
		if output.Path == "" || cleaned == "." || strings.HasPrefix(cleaned, "../") || strings.HasPrefix(cleaned, "/") {
			diagnostics = append(diagnostics, errorDiagnostic("pack.output.path.invalid", basePath+".path", "output path must be contained by the generated project"))
		}
		if _, exists := seen[cleaned]; exists {
			diagnostics = append(diagnostics, errorDiagnostic("pack.output.path.duplicate", basePath+".path", "output paths must be unique"))
		}
		seen[cleaned] = struct{}{}
		diagnostics = append(diagnostics, validateResourceName(output.Owner, basePath+".owner")...)
		if output.Owner != packName {
			diagnostics = append(diagnostics, errorDiagnostic("pack.output.owner.invalid", basePath+".owner", "output owner must match the capability pack name"))
		}
	}
	return diagnostics
}

func validateWorkflows(workflows []Workflow, pathPrefix string) []Diagnostic {
	seen := make(map[string]struct{}, len(workflows))
	var diagnostics []Diagnostic
	for index, workflow := range workflows {
		basePath := fmt.Sprintf("%s[%d]", pathPrefix, index)
		if !fieldNamePattern.MatchString(workflow.Name) {
			diagnostics = append(diagnostics, errorDiagnostic("project.workflow.name.invalid", basePath+".name", "workflow name must use lower snake_case"))
		}
		if _, exists := seen[workflow.Name]; exists {
			diagnostics = append(diagnostics, errorDiagnostic("project.workflow.duplicate", basePath+".name", "workflow names must be unique within a module"))
		}
		seen[workflow.Name] = struct{}{}
		if len(workflow.States) < 2 {
			diagnostics = append(diagnostics, errorDiagnostic("project.workflow.states.insufficient", basePath+".states", "workflow must declare at least two states"))
		}
		stateSeen := make(map[string]struct{}, len(workflow.States))
		for stateIndex, state := range workflow.States {
			statePath := fmt.Sprintf("%s.states[%d]", basePath, stateIndex)
			if !fieldNamePattern.MatchString(state) {
				diagnostics = append(diagnostics, errorDiagnostic("project.workflow.state.invalid", statePath, "workflow state must use lower snake_case"))
			}
			if _, exists := stateSeen[state]; exists {
				diagnostics = append(diagnostics, errorDiagnostic("project.workflow.state.duplicate", statePath, "workflow states must be unique"))
			}
			stateSeen[state] = struct{}{}
		}
	}
	return diagnostics
}

func validatePermissions(permissions []Permission, pathPrefix string) []Diagnostic {
	seen := make(map[string]struct{}, len(permissions))
	var diagnostics []Diagnostic
	for index, permission := range permissions {
		itemPath := fmt.Sprintf("%s[%d].code", pathPrefix, index)
		if !permissionPattern.MatchString(permission.Code) {
			diagnostics = append(diagnostics, errorDiagnostic("project.permission.code.invalid", itemPath, "permission code must use colon-separated lowercase segments"))
		}
		if _, exists := seen[permission.Code]; exists {
			diagnostics = append(diagnostics, errorDiagnostic("project.permission.duplicate", itemPath, "permission codes must be unique within a module"))
		}
		seen[permission.Code] = struct{}{}
	}
	return diagnostics
}

func validatePages(pages []Page, pathPrefix string) []Diagnostic {
	allowedTypes := setOf("list", "create", "edit", "detail", "dashboard", "storefront")
	seen := make(map[string]struct{}, len(pages))
	var diagnostics []Diagnostic
	for index, page := range pages {
		basePath := fmt.Sprintf("%s[%d]", pathPrefix, index)
		diagnostics = append(diagnostics, validateResourceName(page.Name, basePath+".name")...)
		if !contains(allowedTypes, page.Type) {
			diagnostics = append(diagnostics, errorDiagnostic("project.page.type.invalid", basePath+".type", "page type is not supported"))
		}
		if _, exists := seen[page.Name]; exists {
			diagnostics = append(diagnostics, errorDiagnostic("project.page.duplicate", basePath+".name", "page names must be unique within a module"))
		}
		seen[page.Name] = struct{}{}
	}
	return diagnostics
}

func validateAcceptance(acceptance []string, pathPrefix string) []Diagnostic {
	seen := make(map[string]struct{}, len(acceptance))
	var diagnostics []Diagnostic
	for index, criterion := range acceptance {
		itemPath := fmt.Sprintf("%s[%d]", pathPrefix, index)
		normalized := strings.TrimSpace(criterion)
		if normalized == "" {
			diagnostics = append(diagnostics, errorDiagnostic("project.acceptance.empty", itemPath, "acceptance criterion cannot be empty"))
		}
		if _, exists := seen[normalized]; exists {
			diagnostics = append(diagnostics, errorDiagnostic("project.acceptance.duplicate", itemPath, "acceptance criteria must be unique within a module"))
		}
		seen[normalized] = struct{}{}
	}
	return diagnostics
}

func validateStringSet(values []string, allowed map[string]struct{}, code, pathPrefix string) []Diagnostic {
	seen := make(map[string]struct{}, len(values))
	var diagnostics []Diagnostic
	for index, value := range values {
		itemPath := fmt.Sprintf("%s[%d]", pathPrefix, index)
		if !contains(allowed, value) {
			diagnostics = append(diagnostics, errorDiagnostic(code, itemPath, "value is not supported"))
		}
		if _, exists := seen[value]; exists {
			diagnostics = append(diagnostics, errorDiagnostic(code+".duplicate", itemPath, "values must be unique"))
		}
		seen[value] = struct{}{}
	}
	return diagnostics
}

func validateResourceName(name, path string) []Diagnostic {
	if resourceNamePattern.MatchString(name) {
		return nil
	}
	return []Diagnostic{errorDiagnostic("metadata.name.invalid", path, "name must start with a lowercase letter and contain only lowercase letters, digits, or hyphens")}
}

func validConstraint(value string) bool {
	if value == "*" {
		return true
	}
	value = strings.TrimPrefix(value, "^")
	value = strings.TrimPrefix(value, ">=")
	return semverPattern.MatchString(value)
}

func errorDiagnostic(code, path, message string) Diagnostic {
	return Diagnostic{Code: code, Severity: SeverityError, Path: path, Message: message}
}

func setOf(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}

func contains(set map[string]struct{}, value string) bool {
	_, exists := set[value]
	return exists
}
