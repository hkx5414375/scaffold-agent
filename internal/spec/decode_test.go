package spec

import (
	"strings"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/canonicaljson"
)

func TestDecodeProjectJSONAndYAMLHaveSameHash(t *testing.T) {
	t.Parallel()

	jsonDocument := `{
  "api_version":"scaffold-agent.io/v1alpha1",
  "kind":"Project",
  "metadata":{"name":"demo"},
  "spec":{"stack":{"backend":"go"},"database":{"engine":"postgresql"},"auth":{"modes":["session","token"]}}
}`
	yamlDocument := `
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
`
	jsonProject, err := DecodeProject([]byte(jsonDocument), FormatJSON)
	if err != nil {
		t.Fatalf("DecodeProject(JSON) error = %v", err)
	}
	yamlProject, err := DecodeProject([]byte(yamlDocument), FormatYAML)
	if err != nil {
		t.Fatalf("DecodeProject(YAML) error = %v", err)
	}
	jsonHash, err := canonicaljson.Hash(jsonProject)
	if err != nil {
		t.Fatalf("Hash(JSON project) error = %v", err)
	}
	yamlHash, err := canonicaljson.Hash(yamlProject)
	if err != nil {
		t.Fatalf("Hash(YAML project) error = %v", err)
	}
	if jsonHash != yamlHash {
		t.Fatalf("project hashes differ: %s != %s", jsonHash, yamlHash)
	}
}

func TestDecodeProjectRejectsUnknownField(t *testing.T) {
	t.Parallel()

	document := `
api_version: scaffold-agent.io/v1alpha1
kind: Project
metadata: {name: demo}
spec:
  stack: {backend: go}
  database: {engine: postgresql}
  auth: {modes: [session]}
  unexpected: true
`
	_, err := DecodeProject([]byte(document), FormatYAML)
	if err == nil || !strings.Contains(err.Error(), "field unexpected not found") {
		t.Fatalf("DecodeProject() error = %v, want unknown field error", err)
	}
}

func TestDecodeProjectRejectsMultipleDocuments(t *testing.T) {
	t.Parallel()

	document := "api_version: scaffold-agent.io/v1alpha1\n---\napi_version: scaffold-agent.io/v1alpha1\n"
	_, err := DecodeProject([]byte(document), FormatYAML)
	if err == nil || !strings.Contains(err.Error(), "multiple documents") {
		t.Fatalf("DecodeProject() error = %v, want multiple document error", err)
	}
}

func TestDecodeProjectRejectsDuplicateJSONKeys(t *testing.T) {
	t.Parallel()

	document := `{"api_version":"scaffold-agent.io/v1alpha1","api_version":"duplicate"}`
	_, err := DecodeProject([]byte(document), FormatJSON)
	if err == nil || !strings.Contains(err.Error(), "duplicate JSON object key") {
		t.Fatalf("DecodeProject() error = %v, want duplicate key error", err)
	}
}
