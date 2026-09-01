# ADR 0038: Python organization tenancy

- Status: Accepted
- Date: 2026-09-01

## Context

Python 已有身份、Blueprint CRUD 和共用管理端，但业务数据仍是全局范围。让 AI 在每个 FastAPI 项目中重新设计组织请求头、成员角色、SQL 条件、唯一约束和前端组织选择，会浪费 Token，并容易产生跨租户读取或写入缺陷。

## Decision

1. Python 适配器支持 `organization-tenancy` `0.1.0`，与 Go 和 Java 共用 `X-Organization-ID`、组织创建、组织查询和成员角色授权契约。
2. 创建组织时，在一个数据库事务内写入组织、创建者的管理员成员关系和成功审计；查询组织只返回当前身份实际加入的组织。
3. 身份认证仍产生全局 Principal。只有明确标记为租户范围的业务权限依赖才读取组织请求头、重新查询成员角色、检查权限，并返回带已验证组织标识的 Principal。
4. 业务记录、查询、变更、游标分页、乐观锁和唯一约束都包含组织标识。跨组织读取和变更沿用原有不存在错误，不泄露其他组织的数据存在性。
5. Alembic 使用独立 `000050_tenancy` 分支和 `000051_tenant_business` 合并迁移，使身份、已有业务迁移和租户扩展保持显式依赖。自动接入租户能力时，已有全局业务表必须为空；Engine 不猜测旧数据属于哪个组织。
6. PostgreSQL 与 MySQL 生成项目执行相同的冻结依赖、Ruff、严格 mypy、Bandit、pytest 和真实数据库门禁；SQLite 测试通过 Alembic batch migration 验证可移植迁移行为。
7. Vue 管理端复用共用模板，恢复合法组织选择、自动发送组织请求头，并在切换组织时重新挂载业务视图。
8. `0.1.0` 不接受配置；成员邀请、成员管理和组织生命周期保持后续独立版本，避免生成半套协作功能。

## Consequences

- Python、Go、Java 的租户 0.1 HTTP 和隔离语义对齐，AI 可以跨语言复用同一操作步骤。
- 全局身份操作（例如创建 API Token）不会因为启用租户能力而错误要求组织上下文。
- 给已有全局业务数据启用租户能力需要业务明确的数据归属迁移，脚手架会安全失败而不是静默分配。
