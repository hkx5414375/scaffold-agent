// Package result defines compact, pageable command and MCP results.
package result

import "github.com/hkx5414375/scaffold-agent/internal/spec"

const APIVersionV1Alpha1 = "scaffold-agent.io/result/v1alpha1"

// Status is the machine-readable result state.
type Status string

const (
	StatusOK      Status = "ok"
	StatusWarning Status = "warning"
	StatusError   Status = "error"
)

// Envelope is shared by JSON CLI and MCP adapters.
type Envelope struct {
	APIVersion      string            `json:"api_version"`
	Status          Status            `json:"status"`
	Summary         string            `json:"summary"`
	ResultID        string            `json:"result_id,omitempty"`
	NextCursor      string            `json:"next_cursor,omitempty"`
	HasMore         bool              `json:"has_more,omitempty"`
	EstimatedTokens int               `json:"estimated_tokens,omitempty"`
	Diagnostics     []spec.Diagnostic `json:"diagnostics,omitempty"`
	Data            any               `json:"data,omitempty"`
}
