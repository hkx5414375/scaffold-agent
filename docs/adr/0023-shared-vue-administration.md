# ADR 0023：Go 与 Java 共用 Vue 管理端

- 状态：已接受
- 日期：2026-09-01

## 背景

Go 与 Java 后端已经共用蓝图、OpenAPI、认证、CRUD、分页和乐观锁语义。若分别维护两套 Vue/Element Plus 模板，不仅会重复代码，还会让不同 AI 生成的项目逐渐出现字段类型、错误结构和交互行为差异。

## 决策

1. 将管理端模板从 Go 适配器目录提升到独立的 `internal/generator/admin` 公共包，Go 与 Java 生成器只传入同构的项目和业务字段数据。
2. `vue-admin` 升级到 `0.2.0`，代表模板已成为跨后端能力，而不是 Go 私有输出。
3. 两种后端生成完全相同的锁定依赖、登录状态、CRUD 页面、十进制字符串 64 位字段、游标分页和乐观锁交互。
4. 公共 API 客户端同时识别 Go 的 `{error: {code, message}}` 与 Java 的 `{code, message}` 稳定错误体；这项兼容只位于传输适配层，不改变任一后端的公开契约。
5. Java 只接受 `admin_ui: element-plus`；未实现的商城仍明确失败，不生成半成品。
6. CI 分别从 Go 和 Java 蓝图生成管理端，并执行锁文件安装、ESLint、Vitest、TypeScript、Vite 生产构建和 Prettier 校验。

## 结果

- 修复或增强一次管理端模板即可同时服务 Go 与 Java，减少维护量和 AI 上下文消耗。
- Java 用户得到与 Go 相同的管理体验，M6 的 OpenAPI 和 Vue 管理端对等项完成。
- 未来 Python 适配器只需满足同一传输数据形状即可复用该前端，不再复制模板。
