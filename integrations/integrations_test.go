// Package integrations verifies checked-in Agent host configuration examples.
package integrations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestJSONExamplesAreValid(t *testing.T) {
	t.Parallel()

	paths, err := filepath.Glob(filepath.Join("*", "mcp.json.example"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	if len(paths) != 2 {
		t.Fatalf("JSON examples = %d, want 2", len(paths))
	}
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%q) error = %v", path, err)
		}
		var document map[string]any
		if err := json.Unmarshal(content, &document); err != nil {
			t.Fatalf("example %q is invalid JSON: %v", path, err)
		}
		if _, exists := document["mcpServers"]; !exists {
			t.Fatalf("example %q has no mcpServers object", path)
		}
	}
}
