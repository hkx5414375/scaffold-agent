# ADR 0044：Python 跨实例应用缓存

- 状态：已接受
- 日期：2026-09-01

## 背景

AI 生成业务功能时常会临时加入进程内字典，或仅为少量派生值引入 Redis。前者在多副本间不一致，后者增加不必要的基础设施；两种做法都容易遗漏租户范围、过期边界、载荷大小和有界清理。

## 决策

1. Python `application-cache` `0.1.0` 使用项目已选择的 PostgreSQL 或 MySQL 保存 JSON，不生成 HTTP 接口或管理页面，也不要求 Redis。
2. `CacheService` 将键和租户范围限制为 191 个 UTF-8 字节，规范化 JSON 限制为 256 KiB，TTL 必须大于零且不超过 30 天。Python 非有限浮点数和不能编码为 JSON 的对象会被拒绝。
3. 租户项目必须提供组织标识，并把它作为 `scope_key`；非租户项目固定使用 `global`，忽略调用方传入的组织文本。不存在、过期和跨租户读取统一返回 `CacheErrorKind.MISS`。
4. PostgreSQL 使用 `ON CONFLICT`，MySQL 使用 `ON DUPLICATE KEY UPDATE`，SQLite 契约测试使用同义 Upsert。删除幂等，过期清理按最早过期时间排序且单次最多删除 10,000 行。
5. SQLAlchemy JSON 列在 PostgreSQL 使用 JSONB 变体。服务在持久化前使用稳定键序、紧凑分隔符和 UTF-8 计算真实大小；仓储与解码原始错误统一映射为不泄密的存储错误。
6. 生成应用把 `CacheService` 放入 `app.state.cache_service` 供业务组合使用。业务必须为键增加命名空间，在权威数据变更后主动失效，并且不得缓存凭据或暴露通用缓存 API。
7. SQLite 测试验证服务与迁移；CI 在 PostgreSQL/MySQL 真库验证原子覆盖、租户隔离、过期、幂等删除和有界清理。

## 结果

- Go、Java、Python 使用同一能力名和范围、大小、TTL、Miss、失效及清理语义，AI 不再重复设计缓存底层。
- 数据库缓存优先保证可移植性与副本一致性，不适合高吞吐热点；后续 Redis 仓储必须保持相同服务契约。
- 自动刷新、缓存标签、分布式锁和后台定时清理不属于 `0.1.0`。
