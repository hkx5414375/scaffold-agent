// Package spec defines versioned Scaffold Agent project and capability contracts.
package spec

const (
	APIVersionV1Alpha1 = "scaffold-agent.io/v1alpha1"
	KindProject        = "Project"
	KindCapabilityPack = "CapabilityPack"
)

// Metadata identifies a versioned project or capability pack.
type Metadata struct {
	Name        string `json:"name" yaml:"name"`
	DisplayName string `json:"display_name,omitempty" yaml:"display_name,omitempty"`
	Version     string `json:"version,omitempty" yaml:"version,omitempty"`
}

// Project is the model-neutral source of truth for a generated application.
type Project struct {
	APIVersion string      `json:"api_version" yaml:"api_version"`
	Kind       string      `json:"kind" yaml:"kind"`
	Metadata   Metadata    `json:"metadata" yaml:"metadata"`
	Spec       ProjectSpec `json:"spec" yaml:"spec"`
}

// ProjectSpec contains stack choices and business modules.
type ProjectSpec struct {
	Stack        StackSpec             `json:"stack" yaml:"stack"`
	Database     DatabaseSpec          `json:"database" yaml:"database"`
	Auth         AuthSpec              `json:"auth" yaml:"auth"`
	Capabilities []CapabilitySelection `json:"capabilities,omitempty" yaml:"capabilities,omitempty"`
	Modules      []Module              `json:"modules,omitempty" yaml:"modules,omitempty"`
}

// StackSpec selects one backend and the supported first-party frontends.
type StackSpec struct {
	Backend    string `json:"backend" yaml:"backend"`
	AdminUI    string `json:"admin_ui,omitempty" yaml:"admin_ui,omitempty"`
	Storefront string `json:"storefront,omitempty" yaml:"storefront,omitempty"`
}

// DatabaseSpec selects the portable SQL target.
type DatabaseSpec struct {
	Engine string `json:"engine" yaml:"engine"`
}

// AuthSpec lists authentication modes enabled by the generated project.
type AuthSpec struct {
	Modes []string `json:"modes" yaml:"modes"`
}

// CapabilitySelection pins a capability version and its project inputs.
type CapabilitySelection struct {
	Name    string         `json:"name" yaml:"name"`
	Version string         `json:"version" yaml:"version"`
	Config  map[string]any `json:"config,omitempty" yaml:"config,omitempty"`
}

// Module describes one business boundary in a modular monolith.
type Module struct {
	Name        string       `json:"name" yaml:"name"`
	Entities    []Entity     `json:"entities,omitempty" yaml:"entities,omitempty"`
	Workflows   []Workflow   `json:"workflows,omitempty" yaml:"workflows,omitempty"`
	Permissions []Permission `json:"permissions,omitempty" yaml:"permissions,omitempty"`
	Pages       []Page       `json:"pages,omitempty" yaml:"pages,omitempty"`
	Acceptance  []string     `json:"acceptance,omitempty" yaml:"acceptance,omitempty"`
}

// Entity is a portable domain and persistence model.
type Entity struct {
	Name   string  `json:"name" yaml:"name"`
	Fields []Field `json:"fields" yaml:"fields"`
}

// Field describes a database-neutral entity field.
type Field struct {
	Name     string `json:"name" yaml:"name"`
	Type     string `json:"type" yaml:"type"`
	Required bool   `json:"required,omitempty" yaml:"required,omitempty"`
	Unique   bool   `json:"unique,omitempty" yaml:"unique,omitempty"`
}

// Workflow describes a portable state or approval workflow.
type Workflow struct {
	Name   string   `json:"name" yaml:"name"`
	States []string `json:"states" yaml:"states"`
}

// Permission declares a stable permission code.
type Permission struct {
	Code        string `json:"code" yaml:"code"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// Page declares a generated UI surface without coupling to a component library.
type Page struct {
	Name string `json:"name" yaml:"name"`
	Type string `json:"type" yaml:"type"`
}

// CapabilityPack is a reusable vertical software capability.
type CapabilityPack struct {
	APIVersion string             `json:"api_version" yaml:"api_version"`
	Kind       string             `json:"kind" yaml:"kind"`
	Metadata   Metadata           `json:"metadata" yaml:"metadata"`
	Spec       CapabilityPackSpec `json:"spec" yaml:"spec"`
}

// CapabilityPackSpec defines dependency, compatibility, and input contracts.
type CapabilityPackSpec struct {
	Description string           `json:"description" yaml:"description"`
	Requires    []PackDependency `json:"requires,omitempty" yaml:"requires,omitempty"`
	Conflicts   []string         `json:"conflicts,omitempty" yaml:"conflicts,omitempty"`
	Backends    []string         `json:"backends" yaml:"backends"`
	Databases   []string         `json:"databases" yaml:"databases"`
	InputSchema map[string]any   `json:"input_schema,omitempty" yaml:"input_schema,omitempty"`
	Outputs     []ManagedOutput  `json:"outputs,omitempty" yaml:"outputs,omitempty"`
}

// PackDependency references a compatible capability version.
type PackDependency struct {
	Name       string `json:"name" yaml:"name"`
	Constraint string `json:"constraint" yaml:"constraint"`
}

// ManagedOutput declares a generated ownership boundary.
type ManagedOutput struct {
	Path  string `json:"path" yaml:"path"`
	Owner string `json:"owner" yaml:"owner"`
}
