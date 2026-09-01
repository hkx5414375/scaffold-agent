package app

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/result"
)

func TestRunHelp(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"help"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	if stdout.String() != Usage() {
		t.Fatalf("Run() stdout = %q, want usage", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("Run() stderr = %q, want empty", stderr.String())
	}
}

func TestRunQueryReturnsStableEnvelope(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"query", "--topic", "workflow"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	var envelope result.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil {
		t.Fatalf("query output is not JSON: %v", err)
	}
	if envelope.APIVersion != result.APIVersionV1Alpha1 || envelope.Status != result.StatusOK {
		t.Fatalf("query envelope = %#v", envelope)
	}
}

func TestRunValidateBlueprint(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	blueprint := `
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
	if err := os.WriteFile(filepath.Join(root, "scaffold.yaml"), []byte(blueprint), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"validate", "--project-root", root}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
	var envelope result.Envelope
	if err := json.Unmarshal(stdout.Bytes(), &envelope); err != nil || envelope.Status != result.StatusOK {
		t.Fatalf("validate envelope = %#v, error = %v", envelope, err)
	}
}

func TestRunIOMCPDoesNotMixProtocolAndLogs(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"scaffold_query","arguments":{"topic":"support"}}}`,
	}, "\n") + "\n"
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := RunIO([]string{"mcp"}, strings.NewReader(input), &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("RunIO() code = %d, stderr = %q", code, stderr.String())
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("MCP response lines = %d, want 2", len(lines))
	}
	for _, line := range lines {
		var value map[string]any
		if err := json.Unmarshal([]byte(line), &value); err != nil {
			t.Fatalf("MCP stdout contains non-protocol text %q: %v", line, err)
		}
	}
}

func TestRunUnknownCommand(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"unknown"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("Run() code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") {
		t.Fatalf("Run() stderr = %q, want unknown command diagnostic", stderr.String())
	}
}

func TestRunDoctorJSON(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"doctor", "--json"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("Run() code = %d, want 0; stderr = %q", code, stderr.String())
	}
	var result doctorResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("doctor output is not JSON: %v", err)
	}
	if len(result.Checks) != 3 {
		t.Fatalf("doctor checks = %d, want 3", len(result.Checks))
	}
}

func TestRunConformanceJSON(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"conformance", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
	}
	var report struct {
		Status   string `json:"status"`
		Profiles []any  `json:"profiles"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatalf("conformance output is not JSON: %v", err)
	}
	if report.Status != "ok" || len(report.Profiles) != 6 {
		t.Fatalf("conformance report = %#v", report)
	}
}

func TestRunRejectsUnexpectedFlags(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := Run([]string{"version", "--verbose"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("Run() code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "only --json") {
		t.Fatalf("Run() stderr = %q, want flag diagnostic", stderr.String())
	}
}
