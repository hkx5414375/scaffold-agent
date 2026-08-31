package change

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/plan"
	"github.com/hkx5414375/scaffold-agent/internal/projectfs"
)

func TestApplyAndRollbackTransaction(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	seedManagedFiles(t, root, []Output{
		{Path: "modify.txt", Owner: "test", Content: []byte("before")},
		{Path: "delete.txt", Owner: "test", Content: []byte("delete")},
	})
	manifestBefore, err := os.ReadFile(filepath.Join(root, ".scaffold-agent", "manifest.json"))
	if err != nil {
		t.Fatalf("ReadFile(manifest before) error = %v", err)
	}
	artifact, err := Build(root, plan.ActionModify, testBlueprintHash, nil, []Output{
		{Path: "create.txt", Owner: "test", Content: []byte("created")},
		{Path: "modify.txt", Owner: "test", Content: []byte("after")},
		{Path: "delete.txt", Owner: "test", Delete: true},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	receipt, err := Apply(artifact)
	if err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if receipt.ChangedFiles != 4 {
		t.Fatalf("Apply() changed files = %d, want 4 including manifest", receipt.ChangedFiles)
	}
	assertFileContent(t, root, "create.txt", "created")
	assertFileContent(t, root, "modify.txt", "after")
	assertFileMissing(t, root, "delete.txt")

	rollbackReceipt, err := Rollback(root, artifact.Plan.ID)
	if err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	if !rollbackReceipt.RolledBack {
		t.Fatal("Rollback() RolledBack = false, want true")
	}
	assertFileMissing(t, root, "create.txt")
	assertFileContent(t, root, "modify.txt", "before")
	assertFileContent(t, root, "delete.txt", "delete")
	manifestAfterRollback, err := os.ReadFile(filepath.Join(root, ".scaffold-agent", "manifest.json"))
	if err != nil {
		t.Fatalf("ReadFile(manifest after rollback) error = %v", err)
	}
	if string(manifestAfterRollback) != string(manifestBefore) {
		t.Fatal("rollback did not restore the ownership manifest")
	}
}

func TestApplyRejectsStalePlan(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	seedManagedFiles(t, root, []Output{{Path: "file.txt", Owner: "test", Content: []byte("before")}})
	artifact, err := Build(root, plan.ActionModify, testBlueprintHash, nil, []Output{{Path: "file.txt", Owner: "test", Content: []byte("planned")}})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	writeTestFile(t, root, "file.txt", "user-change")
	if _, err := Apply(artifact); err == nil || !strings.Contains(err.Error(), "precondition failed") {
		t.Fatalf("Apply() error = %v, want stale precondition error", err)
	}
	assertFileContent(t, root, "file.txt", "user-change")
}

func TestApplyIsIdempotentAfterSuccess(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	artifact, err := Build(root, plan.ActionCreate, testBlueprintHash, nil, []Output{{Path: "file.txt", Owner: "test", Content: []byte("content")}})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if _, err := Apply(artifact); err != nil {
		t.Fatalf("Apply(first) error = %v", err)
	}
	receipt, err := Apply(artifact)
	if err != nil {
		t.Fatalf("Apply(second) error = %v", err)
	}
	if !receipt.AlreadyApplied {
		t.Fatal("Apply(second) AlreadyApplied = false, want true")
	}
}

func TestRollbackFirstGenerationRemovesManifestAndGeneratedFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	artifact, err := Build(root, plan.ActionCreate, testBlueprintHash, nil, []Output{{Path: "file.txt", Owner: "test", Content: []byte("content")}})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if _, err := Apply(artifact); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	if _, err := Rollback(root, artifact.Plan.ID); err != nil {
		t.Fatalf("Rollback() error = %v", err)
	}
	assertFileMissing(t, root, "file.txt")
	assertFileMissing(t, root, ".scaffold-agent/manifest.json")
}

func TestRollbackRejectsUserChangeAfterApply(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	artifact, err := Build(root, plan.ActionCreate, testBlueprintHash, nil, []Output{{Path: "file.txt", Owner: "test", Content: []byte("content")}})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	if _, err := Apply(artifact); err != nil {
		t.Fatalf("Apply() error = %v", err)
	}
	writeTestFile(t, root, "file.txt", "user-change")
	if _, err := Rollback(root, artifact.Plan.ID); err == nil || !strings.Contains(err.Error(), "refuse rollback") {
		t.Fatalf("Rollback() error = %v, want user change error", err)
	}
	assertFileContent(t, root, "file.txt", "user-change")
}

func TestBuildRejectsReservedMetadataOutput(t *testing.T) {
	t.Parallel()

	_, err := Build(t.TempDir(), plan.ActionCreate, testBlueprintHash, nil, []Output{{Path: ".scaffold-agent/manifest.json", Owner: "test"}})
	if err == nil || !strings.Contains(err.Error(), "reserved") {
		t.Fatalf("Build() error = %v, want reserved metadata error", err)
	}
}

func TestRecoverRestoresModifyInterruptedAfterBackup(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	seedManagedFiles(t, rootPath, []Output{{Path: "file.txt", Owner: "test", Content: []byte("before")}})
	artifact, err := Build(rootPath, plan.ActionModify, testBlueprintHash, nil, []Output{{Path: "file.txt", Owner: "test", Content: []byte("after")}})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	root := mustProjectRoot(t, rootPath)
	backupRoot, err := root.Resolve(".scaffold-agent/backups/" + artifact.Plan.ID)
	if err != nil {
		t.Fatalf("Resolve(backup) error = %v", err)
	}
	entries := journalEntries(artifact.Plan.Changes)
	backup, err := backupTarget(backupRoot, entries[0])
	if err != nil {
		t.Fatalf("backupTarget() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(backup), 0o700); err != nil {
		t.Fatalf("MkdirAll(backup) error = %v", err)
	}
	if err := os.Rename(filepath.Join(rootPath, "file.txt"), backup); err != nil {
		t.Fatalf("Rename(to backup) error = %v", err)
	}
	value := journal{Version: journalVersion, PlanID: artifact.Plan.ID, Status: journalApplying, Entries: entries}
	if err := writeJournalSnapshot(backupRoot, 0, value); err != nil {
		t.Fatalf("writeJournalSnapshot() error = %v", err)
	}
	if _, err := Recover(rootPath, artifact.Plan.ID); err != nil {
		t.Fatalf("Recover() error = %v", err)
	}
	assertFileContent(t, rootPath, "file.txt", "before")
}

func TestRecoverRemovesCreateInterruptedBeforeJournalAdvance(t *testing.T) {
	t.Parallel()

	rootPath := t.TempDir()
	artifact, err := Build(rootPath, plan.ActionCreate, testBlueprintHash, nil, []Output{{Path: "nested/file.txt", Owner: "test", Content: []byte("created")}})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	writeTestFile(t, rootPath, "nested/file.txt", "created")
	root := mustProjectRoot(t, rootPath)
	backupRoot, err := root.Resolve(".scaffold-agent/backups/" + artifact.Plan.ID)
	if err != nil {
		t.Fatalf("Resolve(backup) error = %v", err)
	}
	value := journal{Version: journalVersion, PlanID: artifact.Plan.ID, Status: journalApplying, Entries: journalEntries(artifact.Plan.Changes)}
	if err := writeJournalSnapshot(backupRoot, 0, value); err != nil {
		t.Fatalf("writeJournalSnapshot() error = %v", err)
	}
	if _, err := Recover(rootPath, artifact.Plan.ID); err != nil {
		t.Fatalf("Recover() error = %v", err)
	}
	assertFileMissing(t, rootPath, "nested/file.txt")
	if _, err := os.Lstat(filepath.Join(rootPath, "nested")); !os.IsNotExist(err) {
		t.Fatalf("nested directory remains after recovery: %v", err)
	}
}

func mustProjectRoot(t *testing.T, rootPath string) projectfs.Root {
	t.Helper()
	root, err := projectfs.Open(rootPath)
	if err != nil {
		t.Fatalf("projectfs.Open() error = %v", err)
	}
	return root
}

func assertFileContent(t *testing.T, root, relativePath, want string) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(relativePath)))
	if err != nil {
		t.Fatalf("ReadFile(%q) error = %v", relativePath, err)
	}
	if string(content) != want {
		t.Fatalf("ReadFile(%q) = %q, want %q", relativePath, content, want)
	}
}

func assertFileMissing(t *testing.T, root, relativePath string) {
	t.Helper()
	_, err := os.Lstat(filepath.Join(root, filepath.FromSlash(relativePath)))
	if !os.IsNotExist(err) {
		t.Fatalf("Lstat(%q) error = %v, want not exist", relativePath, err)
	}
}
