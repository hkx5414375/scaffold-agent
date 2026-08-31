package change

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/plan"
)

var testBlueprintHash = strings.Repeat("a", 64)

func TestBuildPlansCreateModifyDeleteAndSkipsUnchanged(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	seedManagedFiles(t, root, []Output{
		{Path: "modify.txt", Owner: "test", Content: []byte("before")},
		{Path: "delete.txt", Owner: "test", Content: []byte("delete")},
		{Path: "same.txt", Owner: "test", Content: []byte("same")},
	})

	artifact, err := Build(root, plan.ActionModify, testBlueprintHash, map[string]string{"test": "1.0.0"}, []Output{
		{Path: "create.txt", Owner: "test", Content: []byte("created")},
		{Path: "modify.txt", Owner: "test", Content: []byte("after")},
		{Path: "delete.txt", Owner: "test", Delete: true},
		{Path: "same.txt", Owner: "test", Content: []byte("same")},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if len(artifact.Plan.Changes) != 4 {
		t.Fatalf("Build() changes = %#v, want 4 including manifest", artifact.Plan.Changes)
	}
	wantOperations := []plan.Operation{plan.OperationCreate, plan.OperationDelete, plan.OperationModify}
	for index, operation := range wantOperations {
		if artifact.Plan.Changes[index].Operation != operation {
			t.Fatalf("change %d operation = %q, want %q", index, artifact.Plan.Changes[index].Operation, operation)
		}
	}
	if len(artifact.Content) != 3 {
		t.Fatalf("Build() content entries = %d, want 3 including manifest", len(artifact.Content))
	}
}

func TestBuildIsIndependentOfOutputOrder(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	first, err := Build(root, plan.ActionCreate, testBlueprintHash, nil, []Output{
		{Path: "b.txt", Owner: "test", Content: []byte("b")},
		{Path: "a.txt", Owner: "test", Content: []byte("a")},
	})
	if err != nil {
		t.Fatalf("Build(first) error = %v", err)
	}
	second, err := Build(root, plan.ActionCreate, testBlueprintHash, nil, []Output{
		{Path: "a.txt", Owner: "test", Content: []byte("a")},
		{Path: "b.txt", Owner: "test", Content: []byte("b")},
	})
	if err != nil {
		t.Fatalf("Build(second) error = %v", err)
	}
	if first.Plan.ID != second.Plan.ID {
		t.Fatalf("plan IDs differ: %s != %s", first.Plan.ID, second.Plan.ID)
	}
}

func TestBuildRejectsEscapingOutput(t *testing.T) {
	t.Parallel()

	_, err := Build(t.TempDir(), plan.ActionCreate, testBlueprintHash, nil, []Output{{Path: "../outside", Owner: "test"}})
	if err == nil {
		t.Fatal("Build() error = nil, want containment error")
	}
}

func TestBuildRejectsOverwriteAndDeleteOfUnmanagedFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeTestFile(t, root, "user.txt", "user-owned")
	for _, output := range []Output{
		{Path: "user.txt", Owner: "test", Content: []byte("generated")},
		{Path: "user.txt", Owner: "test", Delete: true},
	} {
		_, err := Build(root, plan.ActionModify, testBlueprintHash, nil, []Output{output})
		if err == nil || !strings.Contains(err.Error(), "unmanaged file") {
			t.Fatalf("Build(%+v) error = %v, want unmanaged file error", output, err)
		}
	}
}

func TestBuildRejectsManagedFileChangedOutsideEngine(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	seedManagedFiles(t, root, []Output{{Path: "file.txt", Owner: "test", Content: []byte("generated")}})
	writeTestFile(t, root, "file.txt", "user-change")
	_, err := Build(root, plan.ActionModify, testBlueprintHash, nil, []Output{{Path: "file.txt", Owner: "test", Content: []byte("next")}})
	if err == nil || !strings.Contains(err.Error(), "changed outside Scaffold Agent") {
		t.Fatalf("Build() error = %v, want managed file change error", err)
	}
}

func TestRepairRecreatesMissingManagedFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	output := Output{Path: "file.txt", Owner: "test", Content: []byte("generated")}
	seedManagedFiles(t, root, []Output{output})
	if err := os.Remove(filepath.Join(root, "file.txt")); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}
	if _, err := Build(root, plan.ActionModify, testBlueprintHash, nil, []Output{output}); err == nil || !strings.Contains(err.Error(), "action \"repair\"") {
		t.Fatalf("Build(modify) error = %v, want repair direction", err)
	}
	artifact, err := Build(root, plan.ActionRepair, testBlueprintHash, nil, []Output{output})
	if err != nil {
		t.Fatalf("Build(repair) error = %v", err)
	}
	if _, err := Apply(artifact); err != nil {
		t.Fatalf("Apply(repair) error = %v", err)
	}
	assertFileContent(t, root, "file.txt", "generated")
}

func writeTestFile(t *testing.T, root, relativePath, content string) {
	t.Helper()
	target := filepath.Join(root, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func seedManagedFiles(t *testing.T, root string, outputs []Output) {
	t.Helper()
	artifact, err := Build(root, plan.ActionCreate, testBlueprintHash, nil, outputs)
	if err != nil {
		t.Fatalf("Build(seed) error = %v", err)
	}
	if _, err := Apply(artifact); err != nil {
		t.Fatalf("Apply(seed) error = %v", err)
	}
}
