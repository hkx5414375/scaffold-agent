package plan

import "testing"

func TestComputeIDIsIndependentOfMapAndChangeOrder(t *testing.T) {
	t.Parallel()

	first := Plan{
		APIVersion:     APIVersionV1Alpha1,
		Action:         ActionCreate,
		BlueprintHash:  "blueprint",
		ProjectHash:    "project",
		CapabilityLock: map[string]string{"rbac": "1.0.0", "auth": "1.0.0"},
		Changes: []Change{
			{Operation: OperationCreate, Path: "z.go", Owner: "rbac", AfterHash: "z"},
			{Operation: OperationCreate, Path: "a.go", Owner: "auth", AfterHash: "a"},
		},
	}
	second := first
	second.CapabilityLock = map[string]string{"auth": "1.0.0", "rbac": "1.0.0"}
	second.Changes = []Change{first.Changes[1], first.Changes[0]}

	firstID, err := ComputeID(first)
	if err != nil {
		t.Fatalf("ComputeID(first) error = %v", err)
	}
	secondID, err := ComputeID(second)
	if err != nil {
		t.Fatalf("ComputeID(second) error = %v", err)
	}
	if firstID != secondID {
		t.Fatalf("plan IDs differ: %s != %s", firstID, secondID)
	}
}

func TestComputeIDExcludesProjectRoot(t *testing.T) {
	t.Parallel()

	first := Plan{APIVersion: APIVersionV1Alpha1, Action: ActionCreate, ProjectRoot: "first"}
	second := first
	second.ProjectRoot = "second"
	firstID, err := ComputeID(first)
	if err != nil {
		t.Fatalf("ComputeID(first) error = %v", err)
	}
	secondID, err := ComputeID(second)
	if err != nil {
		t.Fatalf("ComputeID(second) error = %v", err)
	}
	if firstID != secondID {
		t.Fatalf("project root changed plan ID: %s != %s", firstID, secondID)
	}
}
