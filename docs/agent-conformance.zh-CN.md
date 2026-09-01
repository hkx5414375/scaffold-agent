# 跨 Agent 兼容性

Scaffold Agent 只实现一次模型无关的 Engine 和 MCP 边界，不为不同模型复制
生成逻辑。兼容性的实际分层如下：

```text
GPT / Claude / Kimi / GLM / DeepSeek
                ↓ 由宿主完成模型工具调用
Codex / Claude Code / Kimi Code / 其他 MCP 编码宿主
                ↓ 本地 STDIO MCP
          Scaffold Agent Engine
```

模型 API 本身不能“安装本地 Engine”。必须由能启动本地 STDIO MCP 进程的
编码 Agent 承载模型，再把 `scaffold-agent mcp` 注册为工具。Engine 不读取模型
名称，不接收模型 API Key，也不调用任何模型服务。

## 自动兼容画像

`scaffold-agent conformance --json` 会在隔离临时项目内逐个运行以下保守画像：

| 画像 | 模型族 | MCP 宿主 | 协议基线 | 结果读取 |
| --- | --- | --- | --- | --- |
| `openai-codex` | OpenAI GPT | Codex / ChatGPT 桌面端的 Codex 宿主 | `2025-11-25` | 结构化 + 文本 |
| `anthropic-claude-code` | Claude | Claude Code | `2025-06-18` | 结构化 + 文本 |
| `moonshot-kimi-code` | Kimi K3 | Kimi Code | `2025-03-26` | 文本保底 |
| `zhipu-glm-mcp-host` | 智谱 GLM | 支持 MCP 的编码宿主 | `2024-11-05` | 文本保底 |
| `deepseek-mcp-host` | DeepSeek | 支持 MCP 的编码宿主 | `2024-11-05` | 文本保底 |
| `generic-mcp` | 任意模型 | 通用 MCP 宿主 | `2024-11-05` | 文本保底 |

协议版本是用于覆盖兼容分支的测试基线，不表示某个模型只能使用该版本。实际
版本由 MCP 客户端和服务器在初始化时协商。

每个画像都会检查：初始化说明、六工具 Schema 与安全注解、工作流查询、不可变
Plan、游标分页、错误 `apply_token` 拒绝、正确令牌事务应用、托管文件哈希校验、
项目查询，以及 `structuredContent` 与文本回退完全等价。全过程不联网、不使用
模型账号、不消耗模型额度。

## Token 边界

完整画像的模型可见内容采用统一的 `ceil(UTF-8 bytes / 4)` 估算。每个画像必须
不超过 4096 个估算 token。它是跨供应商回归门禁，不是任一模型厂商的计费值。
大结果必须继续通过 `scaffold_preview` 和 `scaffold_result` 的游标分页读取。

## 真实宿主验收

离线画像证明协议、参数、结果和安全流程可兼容，但不能证明某个闭源模型在每次
自然语言请求中都会做出相同决策。发布候选可在已有账号中额外做真实宿主冒烟，
但不得把 API Key 写入仓库或 CI，也不得把付费模型调用作为开源贡献者的必跑门禁。

各宿主安装入口见 [`integrations/`](../integrations/README.md)。
