package artifactstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/change"
	"github.com/hkx5414375/scaffold-agent/internal/plan"
)

func TestSaveLoadRoundTripAndRebindRoot(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	artifact := buildTestArtifact(t, root)
	if err := Save(root, artifact); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := Load(root, artifact.Plan.ID)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.Plan.ID != artifact.Plan.ID {
		t.Fatalf("Load() plan ID = %q, want %q", loaded.Plan.ID, artifact.Plan.ID)
	}
	if string(loaded.Content["file.txt"]) != "content" {
		t.Fatalf("Load() content = %q, want content", loaded.Content["file.txt"])
	}
}

func TestSaveIsIdempotentAndRejectsChangedArtifactFile(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	artifact := buildTestArtifact(t, root)
	if err := Save(root, artifact); err != nil {
		t.Fatalf("Save(first) error = %v", err)
	}
	if err := Save(root, artifact); err != nil {
		t.Fatalf("Save(second) error = %v", err)
	}
	target := filepath.Join(root, ".scaffold-agent", "plans", artifact.Plan.ID+".json")
	if err := os.WriteFile(target, []byte("{}"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := Save(root, artifact); err == nil || !strings.Contains(err.Error(), "different content") {
		t.Fatalf("Save(changed) error = %v, want immutable conflict", err)
	}
}

func TestLoadRejectsInvalidIDAndTamperedContent(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if _, err := Load(root, "../outside"); err == nil {
		t.Fatal("Load(invalid ID) error = nil, want error")
	}
	artifact := buildTestArtifact(t, root)
	if err := Save(root, artifact); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	target := filepath.Join(root, ".scaffold-agent", "plans", artifact.Plan.ID+".json")
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}
	content = []byte(strings.Replace(string(content), "Y29udGVudA==", "dGFtcGVyZWQ=", 1))
	if err := os.WriteFile(target, content, 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if _, err := Load(root, artifact.Plan.ID); err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("Load(tampered) error = %v, want hash mismatch", err)
	}
}

func buildTestArtifact(t *testing.T, root string) change.Artifact {
	t.Helper()
	artifact, err := change.Build(root, plan.ActionCreate, strings.Repeat("a", 64), nil, []change.Output{{
		Path:    "file.txt",
		Owner:   "test",
		Content: []byte("content"),
	}})
	if err != nil {
		t.Fatalf("change.Build() error = %v", err)
	}
	return artifact
}
