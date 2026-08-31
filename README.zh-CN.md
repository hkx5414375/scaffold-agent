# Scaffold Agent

[English](README.md)

Scaffold Agent 是一个供 AI 编码助手调用的、模型无关的本地工具型 Agent。它把版本化项目蓝图和可复用能力包，转换成确定、可测试、可升级的全栈应用。

本项目不是大模型、聊天界面或模型网关。Codex、Claude Code、Kimi Code 以及其他支持 MCP 的编码 Agent 负责理解和推理；Scaffold Agent 负责提供稳定的工程事实、安全的文件变更和可重复验证结果。

## 当前状态

项目正在从底层开始建设。目前已完成协议内核、确定性文件事务、稳定 JSON CLI、六工具 MCP 服务和第一版 Go/PostgreSQL 生成器。生成的 Go 服务已经包含数据库迁移、密码哈希、浏览器 Session、API Token 和安全审计事件；RBAC、业务 CRUD 和前端仍在建设，Schema 在 1.0 前保持实验状态。

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

第一个端到端里程碑是 Go 模块化单体 + PostgreSQL，包含 Session 与 Token 双认证、RBAC、审计日志和一个状态型 CRUD 模块。

## 本地开发

前置环境：

- Go 1.27 系列
- Git

运行基础检查：

```bash
go test ./...
go vet ./...
go run ./cmd/scaffold-agent doctor --json
go run ./cmd/scaffold-agent query --topic support
go run ./cmd/scaffold-agent validate --project-root ./examples/minimal --blueprint scaffold.yaml
```

贡献前请阅读 [CONTRIBUTING.md](CONTRIBUTING.md)、[Agent 调用接口](docs/agent-interface.zh-CN.md)、[基础架构决策](docs/adr/0001-foundation.md)和[开发路线图](docs/roadmap.md)。

## 许可证

Apache-2.0
