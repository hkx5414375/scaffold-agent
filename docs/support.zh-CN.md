# 支持策略

最新稳定的 1.x 次版本获得缺陷修复和安全更新；上一个次版本在后继版本发布后六
个月内获得安全修复。1.0 之前的开发构建没有支持周期。

生成应用不依赖 Scaffold Agent 运行。Go、Java、Python、Node.js、PostgreSQL、
MySQL 与第三方依赖的支持范围以生成该项目时锁定的版本为准。新 Engine 可以提供
升级 Plan，但不会静默修改旧项目。

可复现且不敏感的问题使用公开 Issue；安全问题使用 GitHub 私密漏洞报告。报告应
包含 Engine 版本、操作系统、移除秘密后的 Blueprint、命令或 MCP 工具、稳定诊断
码和最小复现。不要上传客户数据、凭据、私有源码、`.scaffold-agent` 运行产物或
生成备份。
