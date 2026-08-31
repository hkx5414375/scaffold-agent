package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/hkx5414375/scaffold-agent/internal/engine"
)

func TestServerInitializationAndToolList(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}`,
	}, "\n") + "\n"
	responses := runServer(t, input)
	if len(responses) != 2 {
		t.Fatalf("responses = %d, want 2", len(responses))
	}
	initialize := responses[0]["result"].(map[string]any)
	if initialize["protocolVersion"] != "2025-11-25" || initialize["instructions"] == "" {
		t.Fatalf("initialize result = %#v", initialize)
	}
	listed := responses[1]["result"].(map[string]any)
	tools := listed["tools"].([]any)
	if len(tools) != 6 {
		t.Fatalf("tools/list count = %d, want 6", len(tools))
	}
}

func TestServerCallsQueryWithStructuredAndTextContent(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":"init","method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"1","title":"Test Client","description":"integration test"},"_meta":{"request":"test"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":"call","method":"tools/call","params":{"name":"scaffold_query","arguments":{"topic":"workflow"},"_meta":{"progressToken":"progress"}}}`,
	}, "\n") + "\n"
	responses := runServer(t, input)
	call := responses[1]["result"].(map[string]any)
	structured := call["structuredContent"].(map[string]any)
	if structured["status"] != "ok" {
		t.Fatalf("structured status = %v, want ok", structured["status"])
	}
	content := call["content"].([]any)
	if len(content) != 1 || content[0].(map[string]any)["type"] != "text" {
		t.Fatalf("content = %#v", content)
	}
}

func TestServerReturnsProtocolErrorForUnknownTool(t *testing.T) {
	t.Parallel()

	input := strings.Join([]string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"test","version":"1"}}}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"unknown","arguments":{}}}`,
	}, "\n") + "\n"
	responses := runServer(t, input)
	errorValue := responses[1]["error"].(map[string]any)
	if int(errorValue["code"].(float64)) != invalidParamsCode {
		t.Fatalf("error code = %v, want %d", errorValue["code"], invalidParamsCode)
	}
}

func TestServerReturnsParseError(t *testing.T) {
	t.Parallel()

	responses := runServer(t, "{invalid}\n")
	errorValue := responses[0]["error"].(map[string]any)
	if int(errorValue["code"].(float64)) != parseErrorCode {
		t.Fatalf("error code = %v, want %d", errorValue["code"], parseErrorCode)
	}
}

func runServer(t *testing.T, input string) []map[string]any {
	t.Helper()
	var output bytes.Buffer
	server := New(engine.New("test"), "test")
	if err := server.Serve(context.Background(), strings.NewReader(input), &output); err != nil {
		t.Fatalf("Serve() error = %v", err)
	}
	var responses []map[string]any
	for _, line := range strings.Split(strings.TrimSpace(output.String()), "\n") {
		var value map[string]any
		if err := json.Unmarshal([]byte(line), &value); err != nil {
			t.Fatalf("response is not one-line JSON: %q: %v", line, err)
		}
		responses = append(responses, value)
	}
	return responses
}
