package plan

import (
	"fmt"
	"sort"

	"github.com/hkx5414375/scaffold-agent/internal/canonicaljson"
)

type identity struct {
	APIVersion     string            `json:"api_version"`
	Action         Action            `json:"action"`
	BlueprintHash  string            `json:"blueprint_hash"`
	ProjectHash    string            `json:"project_hash"`
	CapabilityLock map[string]string `json:"capability_lock"`
	Changes        []Change          `json:"changes"`
}

// ComputeID returns the immutable content address for a plan.
func ComputeID(value Plan) (string, error) {
	if value.APIVersion != APIVersionV1Alpha1 {
		return "", fmt.Errorf("plan api_version must be %s", APIVersionV1Alpha1)
	}
	changes := append([]Change(nil), value.Changes...)
	sort.Slice(changes, func(i, j int) bool {
		if changes[i].Path != changes[j].Path {
			return changes[i].Path < changes[j].Path
		}
		return changes[i].Operation < changes[j].Operation
	})
	hash, err := canonicaljson.Hash(identity{
		APIVersion:     value.APIVersion,
		Action:         value.Action,
		BlueprintHash:  value.BlueprintHash,
		ProjectHash:    value.ProjectHash,
		CapabilityLock: value.CapabilityLock,
		Changes:        changes,
	})
	if err != nil {
		return "", fmt.Errorf("hash plan identity: %w", err)
	}
	return "plan_" + hash, nil
}

// WithComputedID returns a copy of a plan with its content address assigned.
func WithComputedID(value Plan) (Plan, error) {
	id, err := ComputeID(value)
	if err != nil {
		return Plan{}, err
	}
	value.ID = id
	return value, nil
}
