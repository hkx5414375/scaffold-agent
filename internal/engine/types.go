package engine

import (
	"encoding/json"

	"github.com/hkx5414375/scaffold-agent/internal/plan"
)

// QueryInput selects compact Engine or project facts.
type QueryInput struct {
	Topic       string `json:"topic"`
	ProjectRoot string `json:"project_root,omitempty"`
}

// ValidateInput identifies one Blueprint inside a project root.
type ValidateInput struct {
	ProjectRoot   string `json:"project_root"`
	BlueprintPath string `json:"blueprint_path"`
}

// PlanInput describes a requested generated-project change.
type PlanInput struct {
	ProjectRoot   string      `json:"project_root"`
	BlueprintPath string      `json:"blueprint_path"`
	Action        plan.Action `json:"action"`
}

// PreviewInput selects one bounded page from an immutable plan.
type PreviewInput struct {
	ProjectRoot string `json:"project_root"`
	PlanID      string `json:"plan_id"`
	Cursor      string `json:"cursor,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

// ApplyInput applies the exact plan acknowledged by a preview token.
type ApplyInput struct {
	ProjectRoot string `json:"project_root"`
	PlanID      string `json:"plan_id"`
	ApplyToken  string `json:"apply_token"`
}

// VerifyInput checks every currently managed file and stores pageable findings.
type VerifyInput struct {
	ProjectRoot string `json:"project_root"`
	Limit       int    `json:"limit,omitempty"`
}

// ResultInput selects a page from one stored result.
type ResultInput struct {
	ProjectRoot string `json:"project_root"`
	ResultID    string `json:"result_id"`
	Cursor      string `json:"cursor,omitempty"`
	Limit       int    `json:"limit,omitempty"`
}

// TransactionInput selects an applied or interrupted transaction.
type TransactionInput struct {
	ProjectRoot string `json:"project_root"`
	PlanID      string `json:"plan_id"`
}

type validationData struct {
	BlueprintHash   string `json:"blueprint_hash"`
	ProjectName     string `json:"project_name"`
	Backend         string `json:"backend"`
	Database        string `json:"database"`
	CapabilityCount int    `json:"capability_count"`
	ModuleCount     int    `json:"module_count"`
}

type planData struct {
	PlanID         string            `json:"plan_id"`
	Action         plan.Action       `json:"action"`
	Backend        string            `json:"backend"`
	BlueprintHash  string            `json:"blueprint_hash"`
	ProjectHash    string            `json:"project_hash"`
	CapabilityLock map[string]string `json:"capability_lock"`
	ChangeCount    int               `json:"change_count"`
}

type previewData struct {
	PlanID         string            `json:"plan_id"`
	Action         plan.Action       `json:"action"`
	BlueprintHash  string            `json:"blueprint_hash"`
	ProjectHash    string            `json:"project_hash"`
	CapabilityLock map[string]string `json:"capability_lock"`
	ApplyToken     string            `json:"apply_token"`
	TotalChanges   int               `json:"total_changes"`
	Changes        []plan.Change     `json:"changes"`
}

type verificationFinding struct {
	Path         string `json:"path"`
	Owner        string `json:"owner"`
	Problem      string `json:"problem"`
	ExpectedHash string `json:"expected_hash"`
	CurrentHash  string `json:"current_hash,omitempty"`
}

type supportData struct {
	EngineVersion       string              `json:"engine_version"`
	MCPProtocolVersions []string            `json:"mcp_protocol_versions"`
	Implemented         []string            `json:"implemented"`
	ContractTargets     map[string][]string `json:"contract_targets"`
}

type workflowData struct {
	Steps []string `json:"steps"`
	Rule  string   `json:"rule"`
}

type projectData struct {
	BlueprintHash    string            `json:"blueprint_hash"`
	CapabilityLock   map[string]string `json:"capability_lock"`
	ManagedFileCount int               `json:"managed_file_count"`
}

type resultPageData struct {
	Metadata map[string]any    `json:"metadata,omitempty"`
	Items    []json.RawMessage `json:"items,omitempty"`
}
