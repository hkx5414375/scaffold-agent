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

六个工具和存储协议已实现。Go 适配器已能生成 PostgreSQL 或 MySQL、内嵌迁移、Session 与 Token 双认证、权限码 RBAC、审计事件、完整 CRUD、OpenAPI 3.1、Vue/Element Plus 管理端及 M5 平台能力。Java 适配器现可生成 Java 21/Spring Boot 4.1 + Maven 的 PostgreSQL 或 MySQL 服务，包含有界数据库就绪检查、PBKDF2-SHA256 密码、HttpOnly 浏览器会话、只保存摘要的 Bearer Token、权限 RBAC、事务安全审计、蓝图驱动 CRUD、有界游标分页、乐观锁、OpenAPI 和锁定的质量门禁；尚未实现的能力包和前端选择会被明确拒绝。Python 当前仍返回稳定诊断 `generator.adapter.unavailable`。所有不支持的选择都会失败，不会生成残缺项目。

Go 业务模块写法参见 `examples/task-service/scaffold.yaml`，Java 对等写法参见 `examples/task-service-java/scaffold.yaml`。
每个生成项目都会包含 `api/openapi.yaml`，其中给出稳定的操作 ID、认证方式、所需权限扩展、请求与响应结构、分页参数和乐观锁输入。AI 在读取 HTTP 实现代码前应优先读取这份契约。

选择 `organization-tenancy` `0.1.0` 后，生成项目会加入组织创建与查询、成员身份范围内的权限校验、管理端组织选择，以及所有业务读写的 `X-Organization-ID` 租户条件。`0.2.0` 进一步加入成员列表、72 小时且绑定邮箱的邀请、邀请接受、角色变更、成员移除和最后管理员保护。邀请明文只在创建响应中返回一次，数据库只保存摘要，AI 不应尝试从数据库恢复邀请明文。`0.3.0` 再加入独立所有者、组织改名、原子所有权转移、可逆停用和重新启用；所有权只能转给现有成员且会自动把新所有者提升为管理员。AI 不得绕过所有者保护或自行补一个级联删除组织的接口。停用组织仍可查询，但不能再授权租户业务请求。

项目能力选择必须锁定精确版本；传递依赖可以声明语义版本范围。Engine 会在全部约束下确定性选择最高兼容版本，并把精确结果写入能力锁。AI 应复用该锁，不要自行重新推导依赖版本。

选择 `background-jobs` `0.1.0` 后，生成项目会加入数据库可靠队列和独立的 `cmd/worker`。业务能力应通过 `jobs.Service` 提交有大小限制的 JSON 载荷和稳定幂等键；Handler 可以从 `jobs.Job` 读取已经确认的组织标识，必须响应 Context 取消，并且不得记录载荷内容。任务抢占、租约续期、指数重试、死信和过期租约恢复均由能力包提供，AI 不需要重复编写。

选择 `notifications` `0.1.0` 会自动解析并锁定 `background-jobs` `0.1.x`。业务代码通过 `notifications.Service.EnqueueEmail` 传入稳定幂等键，不得额外暴露一个无需业务授权的“任意发邮件”接口。Worker 只接受启用 TLS 的 SMTP 运行时配置。排队中的邮件正文属于数据库内的敏感持久化数据；AI 不得把 SMTP 凭据写入 Blueprint、源码、任务载荷或模型上下文。

选择 `file-assets` `0.1.0` 后，生成项目会加入租户范围内的文件元数据、流式上传、有界游标列表、附件下载、可恢复的元数据删除，以及原子本地对象存储。业务代码必须调用 `files.Service`，不得用用户文件名拼接对象路径，也不得暴露 `Asset.StorageKey`。`FILE_STORAGE_ROOT` 只能放在运行时配置中。AI 可以替换 `BlobStore` 适配器，但必须保留 10 MiB 请求上限、不覆盖发布、SHA-256 元数据、失败补偿，以及跨租户标识统一返回未找到的行为。

选择 `application-cache` `0.1.0` 后，生成项目会加入跨实例一致的数据库 TTL 缓存和有大小限制的 JSON 值。业务代码应给键增加业务命名空间并选择稳定的 TTL，把 `cache.ErrMiss` 统一理解为不存在、已过期或跨租户不可见。AI 不得增加通用缓存 HTTP 接口、缓存秘密、绕过组织范围或执行无界清理。维护任务可调用带批次上限的 `cache.Service.PurgeExpired`；以后替换为 Redis 适配器时也必须保持同一服务契约。

选择 `job-administration` `0.1.0` 会自动解析 `background-jobs` `0.1.x`。管理员可以查看有界的任务元数据，并且只能重试死信任务。所有公开类型都故意排除了载荷和幂等键。AI 不得增加载荷查看或编辑、重试运行中任务、绕过组织范围，或在生成的事务服务之外直接更新任务表。

选择 `observability` `0.1.0` 后，生成项目会加入经过校验或自动生成的请求 ID、不记录查询字符串与正文的结构化访问日志、低基数 Prometheus 计数器，以及最多等待两秒的数据库就绪检查。`GET /metrics` 只暴露进程聚合值；`GET /readyz` 在数据库不可用时只返回通用 503，不会泄露数据库错误。AI 应在已有 `X-Request-ID` 时继续传递，但不得把用户输入放进指标标签，也不得把请求正文、凭据、查询字符串或 Panic 内容写入日志。

选择 `csv-import-export` `0.1.0` 时，Blueprint 必须包含一个已生成的业务实体。生成项目会按字段顺序提供 CSV 表头、类型解析、空模板、原子且只新增的导入，以及带审计的导出；导入和导出使用独立的管理员权限。单个文档最多 5 MiB、1000 行，任意一行无效或冲突都会回滚整个导入，超出上限的导出不会返回半份文件。字符串导出会用可逆方式转义电子表格公式。AI 必须沿用生成的字段顺序和 RFC3339 时间，不得擅自改成无审计 Upsert，也不得绕过组织范围或大小限制。

选择 `approval-workflows` `0.1.0` 时，Blueprint 必须包含一个已生成的业务实体，并在同一模块中显式声明名为 `approval` 的工作流，状态顺序固定为 `pending`、`approved`、`rejected`、`cancelled`。生成服务会在提交时验证当前组织范围内的业务对象，同一对象同时最多有一个待审批请求；发起人不能批准或拒绝自己的请求，只有发起人能撤回待审批请求，所有终态变更都必须携带当前版本。发起和审批使用分离的权限，每次成功提交或变更都会在同一事务中写入审批请求、不可变事件和安全审计。AI 不得让审批隐式修改业务对象、绕过职责分离、覆盖事件历史，或自行增加 Blueprint 没有声明的流程状态。

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
