# Cross-Agent conformance

Scaffold Agent has one model-neutral Engine and MCP boundary. Models do not
launch local executables directly; an MCP-capable coding host runs the model and
starts `scaffold-agent mcp` over STDIO. The Engine receives no model name or API
key and calls no model service.

`scaffold-agent conformance --json` executes six credential-free profiles:
OpenAI GPT through Codex, Claude through Claude Code, Kimi K3 through Kimi Code,
GLM through a generic MCP coding host, DeepSeek through a generic MCP coding
host, and a minimum generic MCP client. The profile protocol versions exercise
all four supported negotiation branches; they are compatibility baselines, not
model limitations.

Every profile verifies initialization instructions, the six strict tool
schemas and safety annotations, query, immutable plan, cursor pagination,
invalid apply-token rejection, transactional apply, managed-file verification,
project query, and equivalent structured/text results. The complete visible
workflow must remain below 4,096 provider-neutral estimated tokens.

This offline gate proves transport and contract compatibility. It cannot prove
that every closed model will make the same natural-language decision on every
run. Optional real-host smoke tests may use existing accounts for release
candidates, but credentials and paid calls are not repository or contributor
requirements. See [`integrations/`](../integrations/README.md).
