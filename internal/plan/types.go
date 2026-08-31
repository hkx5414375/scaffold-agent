// Package plan defines immutable, content-addressed project change plans.
package plan

import "github.com/hkx5414375/scaffold-agent/internal/spec"

const APIVersionV1Alpha1 = "scaffold-agent.io/plan/v1alpha1"

// Action is the user-visible intent represented by a plan.
type Action string

const (
	ActionCreate  Action = "create"
	ActionModify  Action = "modify"
	ActionExtend  Action = "extend"
	ActionReduce  Action = "reduce"
	ActionRepair  Action = "repair"
	ActionUpgrade Action = "upgrade"
)

// Operation is one deterministic filesystem operation.
type Operation string

const (
	OperationCreate Operation = "create"
	OperationModify Operation = "modify"
	OperationDelete Operation = "delete"
)

// Plan is immutable after its ID is calculated.
type Plan struct {
	APIVersion     string            `json:"api_version"`
	ID             string            `json:"id"`
	Action         Action            `json:"action"`
	ProjectRoot    string            `json:"project_root"`
	BlueprintHash  string            `json:"blueprint_hash"`
	ProjectHash    string            `json:"project_hash"`
	CapabilityLock map[string]string `json:"capability_lock"`
	Changes        []Change          `json:"changes"`
	Diagnostics    []spec.Diagnostic `json:"diagnostics,omitempty"`
}

// Change declares a preconditioned filesystem mutation.
type Change struct {
	Operation  Operation `json:"operation"`
	Path       string    `json:"path"`
	Owner      string    `json:"owner"`
	BeforeHash string    `json:"before_hash,omitempty"`
	AfterHash  string    `json:"after_hash,omitempty"`
}
