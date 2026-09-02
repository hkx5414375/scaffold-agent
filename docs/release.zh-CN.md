# 发布流程

只有维护者可以发布版本，标签触发的 GitHub 工作流是唯一二进制发布入口。

1. 确认 `main` 工作区干净且全部 CI 通过。
2. 重跑 `go test ./...`、`go vet ./...`、`conformance` 和 `benchmark`。
3. 确认公共 Schema 快照没有变化，或变化符合升级策略并经过明确兼容审查。
4. 更新 `CHANGELOG.md`、安装、支持和升级文档。
5. 在已审查提交上创建 annotated tag 并推送：

   ```bash
   git tag -a v1.0.1 -m "Scaffold Agent v1.0.1"
   git push origin v1.0.1
   ```

6. 工作流会重新测试源码，生成 Linux/macOS/Windows 的 amd64/arm64 确定性归档、
   校验和、发布清单和 CycloneDX SBOM，并用短期 OIDC 身份签署来源与 SBOM 证明；
   打包后的 Linux 二进制通过验收后才创建 GitHub Release。
7. 使用独立目录下载一个正式资产，按公开安装文档重新验签。

不得上传维护者本机构建的二进制、绕过失败的证明、复用发布标签或把签名私钥放进
仓库 Secret。
