// Package integrations verifies checked-in Agent host configuration examples.
package integrations

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestJSONExamplesAreValid(t *testing.T) {
	t.Parallel()

	paths, err := filepath.Glob(filepath.Join("*", "mcp.json.example"))
	if err != nil {
		t.Fatalf("Glob() error = %v", err)
	}
	wantPaths := []string{
		filepath.Join("claude-code", "mcp.json.example"),
		filepath.Join("deepseek", "mcp.json.example"),
		filepath.Join("generic", "mcp.json.example"),
		filepath.Join("glm", "mcp.json.example"),
		filepath.Join("kimi-code", "mcp.json.example"),
	}
	sort.Strings(paths)
	if !reflect.DeepEqual(paths, wantPaths) {
		t.Fatalf("JSON examples = %v, want %v", paths, wantPaths)
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
		servers, ok := document["mcpServers"].(map[string]any)
		if !ok {
			t.Fatalf("example %q has no mcpServers object", path)
		}
		server, ok := servers["scaffold-agent"].(map[string]any)
		if !ok || server["command"] != "/absolute/path/to/scaffold-agent" {
			t.Fatalf("example %q has no portable Scaffold Agent command", path)
		}
		args, ok := server["args"].([]any)
		if !ok || len(args) != 1 || args[0] != "mcp" {
			t.Fatalf("example %q has invalid arguments", path)
		}
		if env, exists := server["env"]; exists {
			values, ok := env.(map[string]any)
			if !ok || len(values) != 0 {
				t.Fatalf("example %q must not contain model credentials", path)
			}
		}
	}
}
