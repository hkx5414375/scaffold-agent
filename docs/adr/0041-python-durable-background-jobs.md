# ADR 0041：Python 可靠后台任务

- 状态：已接受
- 日期：2026-09-01

## 背景

Python 生成项目已经具备身份、业务 CRUD 和组织生命周期，但邮件、文件处理、导入导出等后续能力仍需要可靠异步执行。让不同 AI 在每个业务项目里重复实现队列，会产生不一致的幂等、租约、重试、租户传播和敏感数据日志策略。

## 决策

1. Python 适配器支持 `background-jobs` `0.1.0`，可单独使用，也可与 `organization-tenancy` 组合。租户项目强制任务携带组织标识；非租户项目使用显式全局范围。
2. `JobService.enqueue` 校验任务类型、最大 1 MiB 的合法 JSON、`[-100, 100]` 优先级、最多 100 次尝试和最大 128 字符幂等键。`(scope_key, type, dedupe_key)` 唯一约束保证重复提交返回同一任务。
3. PostgreSQL 与 MySQL 共用 Alembic `000200_background_jobs` 迁移。`SQLAlchemyJobRepository` 使用 `FOR UPDATE SKIP LOCKED` 原子抢占可用、重试或租约过期任务，并在抢占时增加尝试次数。
4. 完成、失败和续租必须提供当前 Worker 且租约尚未过期，否则返回稳定的 `lease_lost` 错误。处理器执行期间每三分之一租期续租；续租失败会触发传给 Handler 的 `threading.Event` 取消信号。
5. 失败从 5 秒开始指数退避，最长 15 分钟；达到尝试上限后进入持久化 `dead` 状态。保存的错误只包含异常类型，不包含异常消息或任务载荷。
6. `python -m <package>.jobs.worker` 作为独立进程运行，HTTP 进程不会隐式启动 Worker。内置 `system.noop` 是无副作用扩展示例。
7. SQLite 测试实际执行完整 Alembic 迁移图；生成质量门禁覆盖格式、类型、安全扫描、至少 90% 测试覆盖率和架构边界。PostgreSQL/MySQL 真库门禁覆盖幂等、抢占、完成、失败重试、租约过期回收和旧 Worker 拒绝。

## 结果

- Go、Java、Python 使用同一 Blueprint 能力名和可靠性语义，AI 只需调用稳定服务，不再重新设计通用队列。
- Python 通知、文件、任务管理和其他平台能力可以在后续切片复用同一队列。
- 多进程并发压测、任务保留清理、指标和管理端继续由后续能力补齐，不改变 `0.1.0` 的公开服务契约。
