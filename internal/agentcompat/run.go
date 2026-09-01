package agentcompat

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/hkx5414375/scaffold-agent/internal/engine"
	"github.com/hkx5414375/scaffold-agent/internal/mcp"
)

const (
	reportAPIVersion       = "scaffold-agent.io/agent-conformance/v1alpha1"
	maximumInstructionSize = 512
	maximumContextTokens   = 4_096
)

var expectedTools = []string{
	"scaffold_apply",
	"scaffold_plan",
	"scaffold_preview",
	"scaffold_query",
	"scaffold_result",
	"scaffold_verify",
}

// Report is the deterministic, credential-free Agent compatibility result.
type Report struct {
	APIVersion string          `json:"api_version"`
	Status     string          `json:"status"`
	Profiles   []ProfileResult `json:"profiles"`
}

// ProfileResult records one complete MCP workflow against a client profile.
type ProfileResult struct {
	Profile                Profile  `json:"profile"`
	Status                 string   `json:"status"`
	Checks                 []string `json:"checks"`
	EstimatedContextTokens int      `json:"estimated_context_tokens"`
	Error                  string   `json:"error,omitempty"`
}

// Run executes a full, local MCP workflow for every supported Agent profile.
// It does not call a model, use credentials, or access the network.
func Run(ctx context.Context, version string) Report {
	report := Report{APIVersion: reportAPIVersion, Status: "ok"}
	for _, profile := range Profiles() {
		profileResult := runProfile(ctx, version, profile)
		report.Profiles = append(report.Profiles, profileResult)
		if profileResult.Status != "ok" {
			report.Status = "error"
		}
	}
	return report
}

func runProfile(ctx context.Context, version string, profile Profile) ProfileResult {
	profileResult := ProfileResult{Profile: profile, Status: "error"}
	root, err := os.MkdirTemp("", "scaffold-agent-conformance-")
	if err != nil {
		profileResult.Error = "create isolated project: " + err.Error()
		return profileResult
	}
	defer func() { _ = os.RemoveAll(root) }()
	if err := os.WriteFile(filepath.Join(root, "scaffold.yaml"), []byte(conformanceBlueprint), 0o600); err != nil {
		profileResult.Error = "write conformance Blueprint: " + err.Error()
		return profileResult
	}

	sequence := newSequence(profile)
	first, err := runSession(ctx, version, []map[string]any{
		sequence.initialize(),
		sequence.notification("notifications/initialized", map[string]any{}),
		sequence.request("ping", map[string]any{}),
		sequence.request("tools/list", map[string]any{}),
		sequence.toolCall("scaffold_query", map[string]any{"topic": "workflow"}),
		sequence.toolCall("scaffold_plan", map[string]any{
			"project_root":   root,
			"blueprint_path": "scaffold.yaml",
			"action":         "create",
		}),
	})
	if err != nil {
		profileResult.Error = err.Error()
		return profileResult
	}
	if len(first) != 5 {
		profileResult.Error = fmt.Sprintf("first MCP session returned %d responses, want 5", len(first))
		return profileResult
	}
	initialize, err := responseResult(first[0])
	if err != nil {
		profileResult.Error = "initialize: " + err.Error()
		return profileResult
	}
	if initialize["protocolVersion"] != profile.ProtocolVersion {
		profileResult.Error = fmt.Sprintf("negotiated protocol %v, want %s", initialize["protocolVersion"], profile.ProtocolVersion)
		return profileResult
	}
	instructions, _ := initialize["instructions"].(string)
	if len(instructions) == 0 || len(instructions) > maximumInstructionSize || !strings.Contains(instructions, "scaffold_query first") {
		profileResult.Error = "server instructions are missing, oversized, or not self-contained"
		return profileResult
	}
	profileResult.Checks = append(profileResult.Checks, "initialize")
	profileResult.EstimatedContextTokens += estimateTokens(initialize)

	listed, err := responseResult(first[2])
	if err != nil {
		profileResult.Error = "tools/list: " + err.Error()
		return profileResult
	}
	if err := validateToolSurface(listed); err != nil {
		profileResult.Error = err.Error()
		return profileResult
	}
	profileResult.Checks = append(profileResult.Checks, "six_tool_surface")
	profileResult.EstimatedContextTokens += estimateTokens(listed)

	workflow, tokens, err := toolEnvelope(first[3], profile.ResultMode)
	if err != nil || workflow["status"] != "ok" {
		profileResult.Error = "workflow query: " + errorText(err, workflow)
		return profileResult
	}
	profileResult.Checks = append(profileResult.Checks, "workflow_query", "dual_tool_result")
	profileResult.EstimatedContextTokens += tokens

	planned, tokens, err := toolEnvelope(first[4], profile.ResultMode)
	if err != nil || planned["status"] != "ok" {
		profileResult.Error = "plan: " + errorText(err, planned)
		return profileResult
	}
	planData, err := objectField(planned, "data")
	if err != nil {
		profileResult.Error = "plan: " + err.Error()
		return profileResult
	}
	planID, err := stringField(planData, "plan_id")
	if err != nil {
		profileResult.Error = "plan: " + err.Error()
		return profileResult
	}
	profileResult.Checks = append(profileResult.Checks, "immutable_plan")
	profileResult.EstimatedContextTokens += tokens

	second, err := runSession(ctx, version, []map[string]any{
		sequence.initialize(),
		sequence.notification("notifications/initialized", map[string]any{}),
		sequence.toolCall("scaffold_preview", map[string]any{
			"project_root": root,
			"plan_id":      planID,
			"limit":        1,
		}),
		sequence.toolCall("scaffold_apply", map[string]any{
			"project_root": root,
			"plan_id":      planID,
			"apply_token":  "invalid",
		}),
	})
	if err != nil || len(second) != 3 {
		profileResult.Error = sessionError("preview", err, len(second), 3)
		return profileResult
	}
	previewed, tokens, err := toolEnvelope(second[1], profile.ResultMode)
	if err != nil || previewed["status"] != "ok" || previewed["has_more"] != true {
		profileResult.Error = "paged preview: " + errorText(err, previewed)
		return profileResult
	}
	previewData, err := objectField(previewed, "data")
	if err != nil {
		profileResult.Error = "paged preview: " + err.Error()
		return profileResult
	}
	applyToken, err := stringField(previewData, "apply_token")
	if err != nil {
		profileResult.Error = "paged preview: " + err.Error()
		return profileResult
	}
	nextCursor, err := stringField(previewed, "next_cursor")
	if err != nil {
		profileResult.Error = "paged preview: " + err.Error()
		return profileResult
	}
	profileResult.Checks = append(profileResult.Checks, "paged_preview")
	profileResult.EstimatedContextTokens += tokens
	invalidApply, tokens, err := toolEnvelope(second[2], profile.ResultMode)
	invalidResult, resultErr := responseResult(second[2])
	if err != nil || resultErr != nil || invalidApply["status"] != "error" || invalidResult["isError"] != true {
		profileResult.Error = "apply token guard: " + errorText(err, invalidApply)
		return profileResult
	}
	profileResult.Checks = append(profileResult.Checks, "apply_token_guard")
	profileResult.EstimatedContextTokens += tokens

	third, err := runSession(ctx, version, []map[string]any{
		sequence.initialize(),
		sequence.notification("notifications/initialized", map[string]any{}),
		sequence.toolCall("scaffold_preview", map[string]any{
			"project_root": root,
			"plan_id":      planID,
			"cursor":       nextCursor,
			"limit":        1,
		}),
		sequence.toolCall("scaffold_apply", map[string]any{
			"project_root": root,
			"plan_id":      planID,
			"apply_token":  applyToken,
		}),
		sequence.toolCall("scaffold_verify", map[string]any{
			"project_root": root,
			"limit":        1,
		}),
		sequence.toolCall("scaffold_query", map[string]any{
			"topic":        "project",
			"project_root": root,
		}),
	})
	if err != nil || len(third) != 5 {
		profileResult.Error = sessionError("apply and verify", err, len(third), 5)
		return profileResult
	}
	for index, check := range []string{"cursor_resume", "transactional_apply", "managed_file_verify", "project_query"} {
		envelope, tokens, envelopeErr := toolEnvelope(third[index+1], profile.ResultMode)
		if envelopeErr != nil || envelope["status"] != "ok" {
			profileResult.Error = check + ": " + errorText(envelopeErr, envelope)
			return profileResult
		}
		profileResult.Checks = append(profileResult.Checks, check)
		profileResult.EstimatedContextTokens += tokens
	}
	if profileResult.EstimatedContextTokens > maximumContextTokens {
		profileResult.Error = fmt.Sprintf(
			"bounded workflow used %d estimated context tokens, maximum is %d",
			profileResult.EstimatedContextTokens,
			maximumContextTokens,
		)
		return profileResult
	}
	profileResult.Checks = append(profileResult.Checks, "bounded_context")
	profileResult.Status = "ok"
	return profileResult
}

type requestSequence struct {
	profile Profile
	nextID  int
}

func newSequence(profile Profile) *requestSequence {
	return &requestSequence{profile: profile, nextID: 1}
}

func (sequence *requestSequence) initialize() map[string]any {
	return sequence.request("initialize", map[string]any{
		"protocolVersion": sequence.profile.ProtocolVersion,
		"capabilities": map[string]any{
			"roots": map[string]any{"listChanged": true},
		},
		"clientInfo": map[string]any{
			"name":    sequence.profile.ID,
			"version": "conformance",
		},
	})
}

func (sequence *requestSequence) toolCall(name string, arguments map[string]any) map[string]any {
	return sequence.request("tools/call", map[string]any{
		"name":      name,
		"arguments": arguments,
		"_meta":     map[string]any{"profile": sequence.profile.ID},
	})
}

func (sequence *requestSequence) request(method string, params map[string]any) map[string]any {
	value := map[string]any{
		"jsonrpc": "2.0",
		"id":      sequence.identifier(),
		"method":  method,
		"params":  params,
	}
	sequence.nextID++
	return value
}

func (sequence *requestSequence) notification(method string, params map[string]any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "method": method, "params": params}
}

func (sequence *requestSequence) identifier() any {
	if sequence.profile.IDStyle == IDStyleString {
		return fmt.Sprintf("%s-%d", sequence.profile.ID, sequence.nextID)
	}
	return sequence.nextID
}

func runSession(ctx context.Context, version string, requests []map[string]any) ([]map[string]any, error) {
	var input bytes.Buffer
	encoder := json.NewEncoder(&input)
	encoder.SetEscapeHTML(false)
	for _, request := range requests {
		if err := encoder.Encode(request); err != nil {
			return nil, fmt.Errorf("encode MCP request: %w", err)
		}
	}
	var output bytes.Buffer
	server := mcp.New(engine.New(version), version)
	if err := server.Serve(ctx, &input, &output); err != nil {
		return nil, fmt.Errorf("serve MCP session: %w", err)
	}
	responses := make([]map[string]any, 0)
	decoder := json.NewDecoder(&output)
	for decoder.More() {
		var response map[string]any
		if err := decoder.Decode(&response); err != nil {
			return nil, fmt.Errorf("decode MCP response: %w", err)
		}
		responses = append(responses, response)
	}
	return responses, nil
}

func validateToolSurface(listed map[string]any) error {
	rawTools, ok := listed["tools"].([]any)
	if !ok || len(rawTools) != len(expectedTools) {
		return fmt.Errorf("tool surface contains %d tools, want %d", len(rawTools), len(expectedTools))
	}
	names := make([]string, 0, len(rawTools))
	for _, rawTool := range rawTools {
		tool, ok := rawTool.(map[string]any)
		if !ok {
			return fmt.Errorf("tool definition is not an object")
		}
		name, err := stringField(tool, "name")
		if err != nil {
			return err
		}
		names = append(names, name)
		schema, err := objectField(tool, "inputSchema")
		if err != nil || schema["additionalProperties"] != false {
			return fmt.Errorf("tool %s does not reject unknown arguments", name)
		}
		annotations, err := objectField(tool, "annotations")
		if err != nil {
			return fmt.Errorf("tool %s has no annotations", name)
		}
		wantDestructive := name == "scaffold_apply"
		if annotations["destructiveHint"] != wantDestructive || annotations["openWorldHint"] != false {
			return fmt.Errorf("tool %s has unsafe annotations", name)
		}
	}
	sort.Strings(names)
	if !reflect.DeepEqual(names, expectedTools) {
		return fmt.Errorf("tool names = %v, want %v", names, expectedTools)
	}
	return nil
}

func toolEnvelope(response map[string]any, mode ResultMode) (map[string]any, int, error) {
	resultValue, err := responseResult(response)
	if err != nil {
		return nil, 0, err
	}
	structured, err := objectField(resultValue, "structuredContent")
	if err != nil {
		return nil, 0, err
	}
	content, ok := resultValue["content"].([]any)
	if !ok || len(content) != 1 {
		return nil, 0, fmt.Errorf("tool result has no single text fallback")
	}
	textBlock, ok := content[0].(map[string]any)
	if !ok || textBlock["type"] != "text" {
		return nil, 0, fmt.Errorf("tool result text fallback is invalid")
	}
	textValue, err := stringField(textBlock, "text")
	if err != nil {
		return nil, 0, err
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(textValue), &decoded); err != nil {
		return nil, 0, fmt.Errorf("decode text fallback: %w", err)
	}
	if !reflect.DeepEqual(decoded, structured) {
		return nil, 0, fmt.Errorf("structured and text tool results differ")
	}
	visible := structured
	if mode == ResultModeText {
		visible = decoded
	}
	return visible, estimateTokens(visible), nil
}

func responseResult(response map[string]any) (map[string]any, error) {
	if rawError, exists := response["error"]; exists {
		return nil, fmt.Errorf("JSON-RPC error: %v", rawError)
	}
	return objectField(response, "result")
}

func objectField(value map[string]any, name string) (map[string]any, error) {
	field, ok := value[name].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("field %s is not an object", name)
	}
	return field, nil
}

func stringField(value map[string]any, name string) (string, error) {
	field, ok := value[name].(string)
	if !ok || field == "" {
		return "", fmt.Errorf("field %s is not a non-empty string", name)
	}
	return field, nil
}

func estimateTokens(value any) int {
	content, err := json.Marshal(value)
	if err != nil {
		return 0
	}
	return (len(content) + 3) / 4
}

func errorText(err error, envelope map[string]any) string {
	if err != nil {
		return err.Error()
	}
	if envelope == nil {
		return "missing result"
	}
	return fmt.Sprint(envelope["summary"])
}

func sessionError(name string, err error, got, want int) string {
	if err != nil {
		return name + ": " + err.Error()
	}
	return fmt.Sprintf("%s session returned %d responses, want %d", name, got, want)
}

const conformanceBlueprint = `api_version: scaffold-agent.io/v1alpha1
kind: Project
metadata:
  name: agent-conformance
spec:
  stack:
    backend: go
    admin_ui: none
    storefront: none
  database:
    engine: postgresql
  auth:
    modes: [session, token]
  modules:
    - name: tasks
      entities:
        - name: task
          fields:
            - {name: title, type: string, required: true, unique: true}
            - {name: completed, type: bool, required: true}
      permissions:
        - {code: "tasks:task:create"}
        - {code: "tasks:task:read"}
        - {code: "tasks:task:update"}
        - {code: "tasks:task:delete"}
`
