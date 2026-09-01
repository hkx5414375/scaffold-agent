# Scaffold Agent

[English](README.md)

Scaffold Agent 是一个供 AI 编码助手调用的、模型无关的本地工具型 Agent。它把版本化项目蓝图和可复用能力包，转换成确定、可测试、可升级的全栈应用。

本项目不是大模型、聊天界面或模型网关。Codex、Claude Code、Kimi Code 以及其他支持 MCP 的编码 Agent 负责理解和推理；Scaffold Agent 负责提供稳定的工程事实、安全的文件变更和可重复验证结果。

## 当前状态

项目正在从底层开始建设。目前已完成协议内核、确定性文件事务、稳定 JSON CLI、六工具 MCP 服务，以及支持 PostgreSQL/MySQL 的 Go 生成器和 M5 平台能力套件。Java 21/Spring Boot 适配器已完成平台能力对齐，包含 CRUD、组织租户、可靠任务、通知、文件资产、缓存、任务管理、可观测性、CSV、审批、OpenAPI 和共用 Vue 管理端。Python 3.12+/FastAPI 适配器现已能生成 PostgreSQL 或 MySQL 服务，包含确定性 uv 锁、Alembic 迁移、有界健康检查、安全 Session/Token 身份、权限 RBAC、事务审计、稳定 OpenAPI、带游标分页和乐观锁的 Blueprint CRUD、组织隔离、绑定邮箱的 72 小时邀请、成员及角色管理、显式所有者、所有权转移、可逆停用、租约式可靠后台任务、由独立 Worker 投递的幂等 TLS-only 邮件通知、租户隔离且原子存储的文件资产，以及按租户隔离、跨实例一致的数据库 JSON 缓存，并配套与 Go/Java 共用且通过构建的 Vue/Element Plus 管理端；同时执行 Ruff、严格 mypy、Bandit、pytest 90% 覆盖率和双数据库真库门禁。其余 Python 平台能力与 Nuxt 商城仍在建设，Schema 在 1.0 前保持实验状态。

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
```

贡献前请阅读 [CONTRIBUTING.md](CONTRIBUTING.md)、[Agent 调用接口](docs/agent-interface.zh-CN.md)、[基础架构决策](docs/adr/0001-foundation.md)和[开发路线图](docs/roadmap.md)。

## 许可证

Apache-2.0
