# Agent 调用接口

Scaffold Agent 通过 JSON CLI 和按行分隔的 JSON-RPC MCP 服务暴露同一组应用服务。传输层不调用任何 AI 模型，也不改变 Blueprint 的含义。

## MCP 工具

| 工具 | 用途 | 是否写入项目代码 |
| --- | --- | --- |
| `scaffold_query` | 返回精简的支持范围、工作流或已托管项目信息 | 否 |
| `scaffold_plan` | 校验 Blueprint 并保存不可变的生成计划 | 不写生成文件 |
| `scaffold_preview` | 分页读取变更并获得精确的 `apply_token` | 否 |
| `scaffold_apply` | 在归属与哈希校验后应用已预览的计划 | 是 |
| `scaffold_verify` | 校验托管文件哈希并保存可分页问题 | 不写生成文件 |
| `scaffold_result` | 读取已保存结果的一个有界分页 | 否 |

必须按以下流程调用：

```text
scaffold_query -> scaffold_plan -> scaffold_preview -> scaffold_apply -> scaffold_verify
```

`scaffold_apply` 必须收到该不可变 Plan 对应的 `apply_token`。大型变更集和验证问题使用不透明 Cursor 分页，避免一次性占用模型上下文。

六个工具和存储协议已实现。在 M4 的第一个 Go 适配器完成前，`scaffold_plan` 会校验 Blueprint，然后返回稳定诊断 `generator.adapter.unavailable`，不会生成一套不完整的应用。

## STDIO 协议

启动服务：

```bash
scaffold-agent mcp
```

服务支持 MCP 版本 `2025-11-25`、`2025-06-18`、`2025-03-26` 和 `2024-11-05`。每条 STDIO 消息是一个以换行结束的 UTF-8 JSON-RPC 对象。标准输出只能写协议消息，运行错误写到标准错误。

Codex 可以用下列命令注册本地可执行文件：

```bash
codex mcp add scaffold-agent -- /absolute/path/to/scaffold-agent mcp
```

参考 [OpenAI Codex MCP 文档](https://developers.openai.com/codex/mcp) 和 [MCP 传输规范](https://modelcontextprotocol.io/specification/2025-11-25/basic/transports)。

## JSON CLI

所有工作流命令都输出版本化的 `scaffold-agent.io/result/v1alpha1` 封装：

```bash
scaffold-agent query --topic support
scaffold-agent validate --project-root /path/to/project --blueprint scaffold.yaml
scaffold-agent preview --project-root /path/to/project --plan-id plan_...
scaffold-agent apply --project-root /path/to/project --plan-id plan_... --apply-token apply_...
scaffold-agent verify --project-root /path/to/project
scaffold-agent result --project-root /path/to/project --result-id result_... --cursor ...
```

还提供两个仅供运维恢复的命令：

```bash
scaffold-agent rollback --project-root /path/to/project --plan-id plan_...
scaffold-agent recover --project-root /path/to/project --plan-id plan_...
```
