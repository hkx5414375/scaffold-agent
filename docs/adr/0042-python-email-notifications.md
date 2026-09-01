# ADR 0042：Python 可靠邮件通知

- 状态：已接受
- 日期：2026-09-01

## 背景

邀请、审批、导出和交易事件需要异步通知。让不同 AI 在业务项目中直接连接 SMTP，会重复实现幂等和重试，容易产生邮件头注入、明文 SMTP、Web 进程启动失败范围扩大，以及正文或凭据进入日志的问题。

## 决策

1. Python `notifications` `0.1.0` 依赖 `background-jobs` `^0.1.0`。只选择通知时，Engine 自动解析并锁定可靠任务能力。
2. `NotificationService.enqueue_email` 要求稳定幂等键，规范化单个收件邮箱和主题，拒绝邮件头控制字符，将文本与 HTML 正文合计限制在 512 KiB，并提交 `notifications.email.deliver` 任务。
3. 通知服务没有 HTTP 路由。业务能力必须先完成自身授权，再调用内部服务；脚手架不生成可绕过业务权限的“任意发邮件”端点。
4. Web 进程只创建通知入队服务，不读取 SMTP 配置。独立 Worker 在轮询前一次性从 `SMTP_ADDRESS`、`SMTP_FROM`、`SMTP_TLS_MODE` 和可选的成对用户名密码构造 Handler；配置无效会让 Worker 启动失败。
5. SMTP 只允许隐式 TLS 或强制 STARTTLS，使用系统信任库、主机名校验、TLS 1.2 以上协议和 30 秒网络超时。Handler 与发送器在关键步骤检查 Worker 的取消事件。
6. 任务解码要求固定版本、固定字段和字符串类型。投递错误只返回通用分类，不包含邮件正文、SMTP 地址、用户名、密码或服务端原始错误。
7. SQLite 测试执行完整 Alembic 迁移；PostgreSQL/MySQL 真库门禁验证规范化载荷、租户范围与幂等持久化。SMTP 测试使用内存替身，不访问外网。

## 结果

- Go、Java、Python 使用同一能力名、任务类型和安全边界，AI 只需调用语言原生通知服务。
- 邮件正文会保留在 `background_jobs.payload_json` 中，部署方必须为数据库加密、访问、备份和保留策略负责。
- 模板、回执、SMS、Push、Webhook 和载荷清理策略留给后续版本，不扩大 `0.1.0` 的职责。
