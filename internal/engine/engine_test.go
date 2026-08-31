package engine

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/artifactstore"
	"github.com/hkx5414375/scaffold-agent/internal/change"
	"github.com/hkx5414375/scaffold-agent/internal/plan"
	"github.com/hkx5414375/scaffold-agent/internal/result"
)

func TestValidateBlueprint(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	writeBlueprint(t, root, `
api_version: scaffold-agent.io/v1alpha1
kind: Project
metadata:
  name: demo
spec:
  stack:
    backend: go
  database:
    engine: postgresql
  auth:
    modes: [session, token]
`)
	envelope := New("test").Validate(context.Background(), ValidateInput{ProjectRoot: root, BlueprintPath: "scaffold.yaml"})
	if envelope.Status != result.StatusOK {
		t.Fatalf("Validate() = %#v, want ok", envelope)
	}
	data := envelope.Data.(validationData)
	if data.Backend != "go" || data.BlueprintHash == "" {
		t.Fatalf("Validate() data = %#v", data)
	}
}

func TestPreviewRequiresStoredArtifactAndApplyToken(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	artifact, err := change.Build(root, plan.ActionCreate, testHash("a"), nil, []change.Output{{Path: "file.txt", Owner: "test", Content: []byte("content")}})
	if err != nil {
		t.Fatalf("change.Build() error = %v", err)
	}
	if err := artifactstore.Save(root, artifact); err != nil {
		t.Fatalf("artifactstore.Save() error = %v", err)
	}
	engine := New("test")
	preview := engine.Preview(context.Background(), PreviewInput{ProjectRoot: root, PlanID: artifact.Plan.ID})
	if preview.Status != result.StatusOK {
		t.Fatalf("Preview() = %#v, want ok", preview)
	}
	data := preview.Data.(previewData)
	invalid := engine.Apply(context.Background(), ApplyInput{ProjectRoot: root, PlanID: artifact.Plan.ID, ApplyToken: "invalid"})
	if invalid.Status != result.StatusError {
		t.Fatalf("Apply(invalid token) status = %q, want error", invalid.Status)
	}
	applied := engine.Apply(context.Background(), ApplyInput{ProjectRoot: root, PlanID: artifact.Plan.ID, ApplyToken: data.ApplyToken})
	if applied.Status != result.StatusOK {
		t.Fatalf("Apply() = %#v, want ok", applied)
	}
}

func TestVerifyStoresPageableFinding(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	artifact, err := change.Build(root, plan.ActionCreate, testHash("a"), nil, []change.Output{{Path: "file.txt", Owner: "test", Content: []byte("content")}})
	if err != nil {
		t.Fatalf("change.Build() error = %v", err)
	}
	if _, err := change.Apply(artifact); err != nil {
		t.Fatalf("change.Apply() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("changed"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	envelope := New("test").Verify(context.Background(), VerifyInput{ProjectRoot: root})
	if envelope.Status != result.StatusError || envelope.ResultID == "" {
		t.Fatalf("Verify() = %#v, want stored error result", envelope)
	}
	data := envelope.Data.(resultPageData)
	if len(data.Items) != 1 {
		t.Fatalf("Verify() findings = %d, want 1", len(data.Items))
	}
}

func writeBlueprint(t *testing.T, root, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(root, "scaffold.yaml"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
}

func testHash(character string) string {
	value := ""
	for range 64 {
		value += character
	}
	return value
}
