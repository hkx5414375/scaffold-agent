# Scaffold Agent

[English](README.md)

Scaffold Agent 是一个供 AI 编码助手调用的、模型无关的本地工具型 Agent。它把版本化项目蓝图和可复用能力包，转换成确定、可测试、可升级的全栈应用。

本项目不是大模型、聊天界面或模型网关。Codex、Claude Code、Kimi Code 以及其他支持 MCP 的编码 Agent 负责理解和推理；Scaffold Agent 负责提供稳定的工程事实、安全的文件变更和可重复验证结果。

## 当前状态

项目正在从底层开始建设。目前已完成协议内核、确定性文件事务、稳定 JSON CLI、六工具 MCP 服务，以及支持 PostgreSQL/MySQL 的 Go 生成器和 M5 平台能力套件。Java 21/Spring Boot 与 Python 3.12+/FastAPI 适配器均已完成平台能力对齐，覆盖 CRUD、组织租户、可靠任务、通知、文件资产、缓存、任务管理、可观测性、CSV、审批、OpenAPI 和共用 Vue 管理端。三种后端现在都能生成完全相同、依赖锁定的 Nuxt 4 SSR 商城底座，以及 `commerce-catalog` `0.1.0`、`customer-accounts` `0.1.0` 和 `crm-core` `0.1.0`。CRM 包含业务企业、联系人、不可变跟进记录、只前进的销售机会、事务审计、停用租户隔离和完全相同的 Vue 管理端，并由跨语言 OpenAPI 与迁移契约门禁持续保证 Go、Java、Python 一致；购物车、结算等后续能力仍在建设，Schema 在 1.0 前保持实验状态。

## 设计目标

- 复用成熟能力包，避免 AI 重复生成通用底层代码，降低 Token 消耗。
- 相同 Blueprint、能力版本和项目状态必须产生相同结果。
- 生成项目运行时不依赖 Scaffold Agent。
- 支持 Go、Java、Python 后端，共用 Vue 管理端与 Nuxt 商城契约。
- 同时支持 PostgreSQL 和 MySQL，Blueprint 不泄漏数据库方言。
- 所有写入先预览，保留用户扩展代码，升级可恢复。

## 基本调用流程

```text
query -> plan -> preview -> apply -> verify
```

当前参考流程可以生成 Go 模块化单体 + PostgreSQL 或 MySQL，包含 Session 与 Token 双认证、RBAC、审计日志、一个状态型 CRUD 模块，以及可选的管理端。

## 本地开发

前置环境：

- Go 1.27 系列
- Git
- 校验生成前端时需要 Node.js 22.12 或更高版本
- 校验生成 Nuxt 商城时需要 Node.js 24.11 或更高版本
- 校验生成 Python 服务时需要 Python 3.12 或 3.13，以及 uv 0.12.8

运行基础检查：

```bash
go test ./...
go vet ./...
go run ./cmd/scaffold-agent doctor --json
go run ./cmd/scaffold-agent query --topic support
go run ./cmd/scaffold-agent validate --project-root ./examples/minimal --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/task-service-mysql --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/tenant-task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/worker-task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/notification-task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/file-task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/cache-task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/job-admin-task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/observability-task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/csv-transfer-task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/approval-task-service --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/minimal-java --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/task-service-java --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/minimal-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/tenant-task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/worker-task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/notification-task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/file-task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/cache-task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/job-admin-task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/observability-task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/csv-transfer-task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/approval-task-service-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/storefront-foundation --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/catalog-store-go --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/catalog-store-java --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/catalog-store-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/customer-store-go --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/customer-store-java --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/customer-store-python --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/crm-service-go --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/crm-service-java --blueprint scaffold.yaml
go run ./cmd/scaffold-agent validate --project-root ./examples/crm-service-python --blueprint scaffold.yaml
```

贡献前请阅读 [CONTRIBUTING.md](CONTRIBUTING.md)、[Agent 调用接口](docs/agent-interface.zh-CN.md)、[基础架构决策](docs/adr/0001-foundation.md)和[开发路线图](docs/roadmap.md)。

## 许可证

Apache-2.0
