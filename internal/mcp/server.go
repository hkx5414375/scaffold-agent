// Package mcp implements the model-neutral newline-delimited JSON-RPC STDIO adapter.
package mcp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"slices"

	"github.com/hkx5414375/scaffold-agent/internal/engine"
	"github.com/hkx5414375/scaffold-agent/internal/projectmeta"
	"github.com/hkx5414375/scaffold-agent/internal/result"
)

const (
	parseErrorCode     = -32700
	invalidRequestCode = -32600
	methodNotFoundCode = -32601
	invalidParamsCode  = -32602
	internalErrorCode  = -32603
	maxMessageSize     = 8 << 20
)

const instructions = "Use scaffold_query first, then scaffold_plan -> scaffold_preview -> scaffold_apply -> scaffold_verify. Never call scaffold_apply without the apply_token from scaffold_preview. Existing unowned files and user-modified managed files are protected. Use scaffold_result cursors instead of loading large results into model context."

// Server exposes one Engine over MCP STDIO.
type Server struct {
	engine  *engine.Engine
	version string
}

// New returns a server with a stable six-tool surface.
func New(application *engine.Engine, version string) *Server {
	return &Server{engine: application, version: version}
}

// Serve processes newline-delimited UTF-8 JSON-RPC messages until EOF or cancellation.
func (server *Server) Serve(ctx context.Context, input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	scanner.Buffer(make([]byte, 64<<10), maxMessageSize)
	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(false)
	initialized := false
	for scanner.Scan() {
		if err := ctx.Err(); err != nil {
			return err
		}
		message := append([]byte(nil), scanner.Bytes()...)
		if len(bytes.TrimSpace(message)) == 0 {
			continue
		}
		value, respond, nextInitialized := server.handle(ctx, message, initialized)
		initialized = nextInitialized
		if respond {
			if err := encoder.Encode(value); err != nil {
				return fmt.Errorf("encode MCP response: %w", err)
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read MCP message: %w", err)
	}
	return nil
}

func (server *Server) handle(ctx context.Context, message []byte, initialized bool) (response, bool, bool) {
	var requestValue request
	if err := json.Unmarshal(message, &requestValue); err != nil {
		return errorResponse(nil, parseErrorCode, "Parse error", nil), true, initialized
	}
	id := requestValue.ID
	if len(id) == 0 {
		id = json.RawMessage("null")
	}
	if requestValue.JSONRPC != "2.0" || requestValue.Method == "" {
		return errorResponse(id, invalidRequestCode, "Invalid Request", nil), len(requestValue.ID) > 0, initialized
	}
	isNotification := len(requestValue.ID) == 0
	if isNotification {
		switch requestValue.Method {
		case "notifications/initialized":
			return response{}, false, initialized
		case "notifications/cancelled":
			return response{}, false, initialized
		default:
			return response{}, false, initialized
		}
	}
	switch requestValue.Method {
	case "initialize":
		if initialized {
			return errorResponse(id, invalidRequestCode, "Server is already initialized", nil), true, initialized
		}
		var params initializeParams
		if err := decodeArguments(requestValue.Params, &params); err != nil || params.ProtocolVersion == "" || params.ClientInfo.Name == "" {
			return errorResponse(id, invalidParamsCode, "Invalid initialize parameters", nil), true, initialized
		}
		protocolVersion := engine.MCPProtocolVersions[0]
		if slices.Contains(engine.MCPProtocolVersions, params.ProtocolVersion) {
			protocolVersion = params.ProtocolVersion
		}
		return successResponse(id, initializeResult{
			ProtocolVersion: protocolVersion,
			Capabilities:    map[string]any{"tools": map[string]any{}},
			ServerInfo:      implementation{Name: "scaffold-agent", Version: server.version},
			Instructions:    instructions,
		}), true, true
	case "ping":
		return successResponse(id, map[string]any{}), true, initialized
	}
	if !initialized {
		return errorResponse(id, invalidRequestCode, "Server is not initialized", nil), true, initialized
	}
	switch requestValue.Method {
	case "tools/list":
		return successResponse(id, map[string]any{"tools": tools()}), true, initialized
	case "tools/call":
		toolResult, rpcErr := server.callTool(ctx, requestValue.Params)
		if rpcErr != nil {
			return errorResponse(id, rpcErr.Code, rpcErr.Message, rpcErr.Data), true, initialized
		}
		return successResponse(id, toolResult), true, initialized
	default:
		return errorResponse(id, methodNotFoundCode, "Method not found", requestValue.Method), true, initialized
	}
}

func (server *Server) callTool(ctx context.Context, rawParams json.RawMessage) (callToolResult, *rpcError) {
	var params callToolParams
	if err := decodeArguments(rawParams, &params); err != nil || params.Name == "" {
		return callToolResult{}, &rpcError{Code: invalidParamsCode, Message: "Invalid tools/call parameters"}
	}
	if len(params.Task) > 0 && string(params.Task) != "null" {
		return callToolResult{}, &rpcError{Code: methodNotFoundCode, Message: "Task-augmented tool calls are not supported"}
	}
	var envelope result.Envelope
	switch params.Name {
	case "scaffold_query":
		var input engine.QueryInput
		if err := decodeArguments(params.Arguments, &input); err != nil {
			return callToolResult{}, invalidToolArguments(params.Name, err)
		}
		envelope = server.engine.Query(ctx, input)
	case "scaffold_plan":
		var input engine.PlanInput
		if err := decodeArguments(params.Arguments, &input); err != nil {
			return callToolResult{}, invalidToolArguments(params.Name, err)
		}
		envelope = server.engine.Plan(ctx, input)
	case "scaffold_preview":
		var input engine.PreviewInput
		if err := decodeArguments(params.Arguments, &input); err != nil {
			return callToolResult{}, invalidToolArguments(params.Name, err)
		}
		envelope = server.engine.Preview(ctx, input)
	case "scaffold_apply":
		var input engine.ApplyInput
		if err := decodeArguments(params.Arguments, &input); err != nil {
			return callToolResult{}, invalidToolArguments(params.Name, err)
		}
		envelope = server.engine.Apply(ctx, input)
	case "scaffold_verify":
		var input engine.VerifyInput
		if err := decodeArguments(params.Arguments, &input); err != nil {
			return callToolResult{}, invalidToolArguments(params.Name, err)
		}
		envelope = server.engine.Verify(ctx, input)
	case "scaffold_result":
		var input engine.ResultInput
		if err := decodeArguments(params.Arguments, &input); err != nil {
			return callToolResult{}, invalidToolArguments(params.Name, err)
		}
		envelope = server.engine.Result(ctx, input)
	default:
		return callToolResult{}, &rpcError{Code: invalidParamsCode, Message: "Unknown tool", Data: params.Name}
	}
	content, err := json.Marshal(envelope)
	if err != nil {
		return callToolResult{}, &rpcError{Code: internalErrorCode, Message: "Unable to encode tool result"}
	}
	var structured map[string]any
	if err := json.Unmarshal(content, &structured); err != nil {
		return callToolResult{}, &rpcError{Code: internalErrorCode, Message: "Unable to structure tool result"}
	}
	return callToolResult{
		Content:           []textContent{{Type: "text", Text: string(content)}},
		StructuredContent: structured,
		IsError:           envelope.Status == result.StatusError,
	}, nil
}

func decodeArguments(content []byte, target any) error {
	if len(content) == 0 {
		content = []byte("{}")
	}
	return projectmeta.DecodeStrict(content, target)
}

func invalidToolArguments(name string, err error) *rpcError {
	return &rpcError{Code: invalidParamsCode, Message: "Invalid tool arguments", Data: map[string]any{"tool": name, "detail": err.Error()}}
}

func successResponse(id json.RawMessage, value any) response {
	return response{JSONRPC: "2.0", ID: normalizeID(id), Result: value}
}

func errorResponse(id json.RawMessage, code int, message string, data any) response {
	return response{JSONRPC: "2.0", ID: normalizeID(id), Error: &rpcError{Code: code, Message: message, Data: data}}
}

func normalizeID(id json.RawMessage) json.RawMessage {
	if len(id) == 0 {
		return json.RawMessage("null")
	}
	return id
}
