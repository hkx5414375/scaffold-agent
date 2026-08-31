# ADR 0027：Java 可靠后台任务

- 状态：已接受
- 日期：2026-09-01

## 背景

通知、文件处理、导入导出和长耗时业务不能各自重复实现异步队列。Java 生成项目需要与 Go 相同的幂等、租约、重试和租户传播规则，并且 Web 服务与 Worker 必须能够独立运行。

## 决策

1. Java 适配器支持 `background-jobs` `0.1.0`，可单独使用，也可与 `organization-tenancy` 组合。租户项目强制写入组织标识；非租户项目使用显式 `global` 范围。
2. 入队校验任务类型、最大 1 MiB 的合法 JSON、优先级、最多 100 次尝试和最大 128 字符幂等键。`(scope_key, type, dedupe_key)` 唯一约束保证重复请求返回同一任务。
3. PostgreSQL 与 MySQL 分别生成 `V000200__background_jobs.sql`。Worker 使用 `FOR UPDATE SKIP LOCKED` 原子抢占队列、重试或租约过期的任务，并在抢占时增加尝试次数。
4. 完成、失败和续租必须提供当前 Worker 且租约尚未过期，否则返回 `LEASE_LOST`。处理器执行期间每三分之一租期续租；续租失败会中断处理线程。
5. 失败使用从 5 秒开始、最多 15 分钟的指数退避。达到尝试上限后进入持久化 `dead` 状态，不再自动循环。
6. `WorkerRunner` 通过 `--spring.main.web-application-type=none --scaffold.worker.enabled=true` 作为独立进程运行；Web 进程默认不启动 Worker。内置 `system.noop` 是无副作用扩展示例。
7. 日志只记录任务、Worker 和异常类型，不记录任务载荷。处理器必须响应线程中断，后续能力通过实现 `JobWorker.Handler` 注册稳定任务类型。
8. 单元测试覆盖幂等、校验、退避、成功和失败分派；PostgreSQL/MySQL 真实数据库测试覆盖幂等、租约抢占、重试、完成、过期回收、租户外键和旧 Worker 拒绝。

## 结果

- Java 通知和任务管理能力可以复用统一队列，不再复制可靠性代码。
- 人工重试、载荷隔离的任务列表、保留清理和指标继续由后续能力实现。
