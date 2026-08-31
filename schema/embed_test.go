package schema

import (
	"encoding/json"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/spec"
)

func TestEmbeddedSchemasAreValidJSON(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"project.schema.json", "capability-pack.schema.json", "plan.schema.json", "result.schema.json"} {
		content, err := Read("v1alpha1", name)
		if err != nil {
			t.Fatalf("Read(%q) error = %v", name, err)
		}
		var document map[string]any
		if err := json.Unmarshal(content, &document); err != nil {
			t.Fatalf("schema %q is invalid JSON: %v", name, err)
		}
		if document["$schema"] != "https://json-schema.org/draft/2020-12/schema" {
			t.Fatalf("schema %q has unexpected $schema = %v", name, document["$schema"])
		}
	}
}

func TestProjectSchemaAcceptsValidProject(t *testing.T) {
	t.Parallel()

	project := spec.Project{
		APIVersion: spec.APIVersionV1Alpha1,
		Kind:       spec.KindProject,
		Metadata:   spec.Metadata{Name: "demo"},
		Spec: spec.ProjectSpec{
			Stack:    spec.StackSpec{Backend: "go"},
			Database: spec.DatabaseSpec{Engine: "postgresql"},
			Auth:     spec.AuthSpec{Modes: []string{"session", "token"}},
		},
	}
	if err := Validate("v1alpha1", "project.schema.json", project); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func TestProjectSchemaRejectsUnknownBackend(t *testing.T) {
	t.Parallel()

	project := map[string]any{
		"api_version": "scaffold-agent.io/v1alpha1",
		"kind":        "Project",
		"metadata":    map[string]any{"name": "demo"},
		"spec": map[string]any{
			"stack":    map[string]any{"backend": "unknown"},
			"database": map[string]any{"engine": "postgresql"},
			"auth":     map[string]any{"modes": []any{"session"}},
		},
	}
	if err := Validate("v1alpha1", "project.schema.json", project); err == nil {
		t.Fatal("Validate() error = nil, want backend validation error")
	}
}
