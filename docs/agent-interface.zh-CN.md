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

六个工具和存储协议已实现。Go 适配器已能生成 PostgreSQL 或 MySQL、内嵌迁移、Session 与 Token 双认证、权限码 RBAC、审计事件、完整 CRUD、OpenAPI 3.1 和 Vue/Element Plus 管理端。对尚未支持的数据库、前端、认证组合、字段类型、工作流、页面和更大的模块形态，它会明确拒绝，不会静默漏生代码。Java 和 Python 目前会返回稳定诊断 `generator.adapter.unavailable`。

当前支持的业务模块写法参见 `examples/task-service/scaffold.yaml`。
每个生成项目都会包含 `api/openapi.yaml`，其中给出稳定的操作 ID、认证方式、所需权限扩展、请求与响应结构、分页参数和乐观锁输入。AI 在读取 HTTP 实现代码前应优先读取这份契约。

选择 `organization-tenancy` `0.1.0` 后，生成项目会加入组织创建与查询、成员身份范围内的权限校验、管理端组织选择，以及所有业务读写的 `X-Organization-ID` 租户条件。`0.2.0` 进一步加入成员列表、72 小时且绑定邮箱的邀请、邀请接受、角色变更、成员移除和最后管理员保护。邀请明文只在创建响应中返回一次，数据库只保存摘要，AI 不应尝试从数据库恢复邀请明文。

项目能力选择必须锁定精确版本；传递依赖可以声明语义版本范围。Engine 会在全部约束下确定性选择最高兼容版本，并把精确结果写入能力锁。AI 应复用该锁，不要自行重新推导依赖版本。

选择 `background-jobs` `0.1.0` 后，生成项目会加入数据库可靠队列和独立的 `cmd/worker`。业务能力应通过 `jobs.Service` 提交有大小限制的 JSON 载荷和稳定幂等键；Handler 可以从 `jobs.Job` 读取已经确认的组织标识，必须响应 Context 取消，并且不得记录载荷内容。任务抢占、租约续期、指数重试、死信和过期租约恢复均由能力包提供，AI 不需要重复编写。

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
