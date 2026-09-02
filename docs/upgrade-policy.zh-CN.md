# 升级与兼容策略

## Engine 二进制

`1.x` 遵循语义化版本：补丁版本修复缺陷和安全问题；次版本只能增加向后兼容的
能力；破坏 CLI、MCP、Blueprint、Plan、Result、Manifest 或 Capability Pack
既有语义的变更必须进入新的主版本。

升级时先验签新包，把旧二进制保留为 `.previous`，运行 `version`、`doctor`、
`conformance` 和 `benchmark` 后再替换日常入口。仅安装新 Engine 不会修改任何
已生成项目。

## 已生成项目

项目升级必须在项目根目录按以下顺序进行：

```text
scaffold_query(project)
scaffold_plan(action=upgrade)
scaffold_preview
scaffold_apply(apply_token)
scaffold_verify
```

能力锁不会被安装动作自动重算。只有 Blueprint 明确修改能力版本并生成新的
不可变 Plan 后，Engine 才会升级对应能力。用户拥有的文件和被用户修改的托管
文件不会被静默覆盖。

如果升级后项目验证失败，先保留现场和 Result；在没有后续人工修改的前提下，
可用同一 `plan_id` 执行 `scaffold-agent rollback`。已执行的业务数据库迁移必须按
生成项目自己的迁移策略处理，不能靠文件回滚倒退数据库。

## v1 公共协议

现有 `v1alpha1` 字样为兼容早期用户而保留；从首个正式稳定版本 Scaffold Agent
`v1.0.1` 起，它是稳定的 1.x 协议标识。1.x 可以增加可选字段、枚举新值或新能力版本，但不得删除
字段、改变已有字段含义、把可选字段改为必填或改变既有工具语义。破坏性变更必须
使用新的 `api_version` 和主版本。
