// Package agentcompat verifies model-host compatibility at the MCP boundary.
package agentcompat

// ResultMode describes the most conservative tool-result representation used
// by a client profile. Scaffold Agent always returns both representations.
type ResultMode string

const (
	ResultModeStructured ResultMode = "structured_and_text"
	ResultModeText       ResultMode = "text"
)

// IDStyle describes the JSON-RPC request identifier shape used by a profile.
type IDStyle string

const (
	IDStyleNumber IDStyle = "number"
	IDStyleString IDStyle = "string"
)

// Profile is a conservative protocol shape, not a claim that a model API can
// launch MCP processes by itself. Host identifies the Agent that owns MCP.
type Profile struct {
	ID              string     `json:"id"`
	DisplayName     string     `json:"display_name"`
	ModelFamily     string     `json:"model_family"`
	Host            string     `json:"host"`
	ProtocolVersion string     `json:"protocol_version"`
	ResultMode      ResultMode `json:"result_mode"`
	IDStyle         IDStyle    `json:"id_style"`
}

var profiles = []Profile{
	{
		ID:              "openai-codex",
		DisplayName:     "OpenAI GPT through Codex or ChatGPT desktop",
		ModelFamily:     "OpenAI GPT",
		Host:            "Codex",
		ProtocolVersion: "2025-11-25",
		ResultMode:      ResultModeStructured,
		IDStyle:         IDStyleString,
	},
	{
		ID:              "anthropic-claude-code",
		DisplayName:     "Claude through Claude Code",
		ModelFamily:     "Anthropic Claude",
		Host:            "Claude Code",
		ProtocolVersion: "2025-06-18",
		ResultMode:      ResultModeStructured,
		IDStyle:         IDStyleNumber,
	},
	{
		ID:              "moonshot-kimi-code",
		DisplayName:     "Kimi K3 through Kimi Code",
		ModelFamily:     "Moonshot Kimi",
		Host:            "Kimi Code",
		ProtocolVersion: "2025-03-26",
		ResultMode:      ResultModeText,
		IDStyle:         IDStyleString,
	},
	{
		ID:              "zhipu-glm-mcp-host",
		DisplayName:     "GLM through an MCP-capable coding host",
		ModelFamily:     "Zhipu GLM",
		Host:            "Generic MCP coding host",
		ProtocolVersion: "2024-11-05",
		ResultMode:      ResultModeText,
		IDStyle:         IDStyleNumber,
	},
	{
		ID:              "deepseek-mcp-host",
		DisplayName:     "DeepSeek through an MCP-capable coding host",
		ModelFamily:     "DeepSeek",
		Host:            "Generic MCP coding host",
		ProtocolVersion: "2024-11-05",
		ResultMode:      ResultModeText,
		IDStyle:         IDStyleString,
	},
	{
		ID:              "generic-mcp",
		DisplayName:     "Generic MCP client",
		ModelFamily:     "Model independent",
		Host:            "Generic MCP host",
		ProtocolVersion: "2024-11-05",
		ResultMode:      ResultModeText,
		IDStyle:         IDStyleNumber,
	},
}

// Profiles returns the release conformance profiles in stable order.
func Profiles() []Profile {
	return append([]Profile(nil), profiles...)
}
