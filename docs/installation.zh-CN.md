# 安装与验签

## 选择发布包

Scaffold Agent v1 提供六个平台包：

| 系统 | x86-64 | ARM64 |
| --- | --- | --- |
| Linux | `linux_amd64.tar.gz` | `linux_arm64.tar.gz` |
| macOS | `darwin_amd64.tar.gz` | `darwin_arm64.tar.gz` |
| Windows | `windows_amd64.zip` | `windows_arm64.zip` |

文件名前缀为 `scaffold-agent_<版本>_`。只从
`https://github.com/hkx5414375/scaffold-agent/releases` 下载发布资产。

## 下载后先验证

每次发布包含 `SHA256SUMS`、CycloneDX SBOM、发布清单，以及 GitHub
Artifact Attestations 生成的 Sigstore 来源证明。安装前先执行：

```bash
gh attestation verify scaffold-agent_1.0.1_linux_amd64.tar.gz \
  --repo hkx5414375/scaffold-agent
sha256sum -c SHA256SUMS --ignore-missing
```

Windows PowerShell 可检查单个文件：

```powershell
$asset = 'scaffold-agent_1.0.1_windows_amd64.zip'
$expected = (Select-String -LiteralPath SHA256SUMS -Pattern "  $asset$").Line.Split(' ')[0]
$actual = (Get-FileHash -LiteralPath $asset -Algorithm SHA256).Hash.ToLowerInvariant()
if ($actual -ne $expected) { throw 'SHA-256 校验失败' }
gh attestation verify $asset --repo hkx5414375/scaffold-agent
```

验签失败、仓库身份不符或哈希不符时不要运行二进制。

## 解压与加入 PATH

Linux/macOS：

```bash
tar -xzf scaffold-agent_1.0.1_linux_amd64.tar.gz
install -m 0755 scaffold-agent "$HOME/.local/bin/scaffold-agent"
```

Windows：解压 ZIP，把 `scaffold-agent.exe` 移到一个用户可写的固定目录，再把该
目录加入用户 `PATH`。不要从不受信任项目目录直接运行同名程序。

## 安装后验收

```bash
scaffold-agent version --json
scaffold-agent doctor --json
scaffold-agent conformance --json
scaffold-agent benchmark --json
```

然后按 [`integrations/`](../integrations/README.md) 中对应宿主的说明注册
`scaffold-agent mcp`。不要把模型 API Key 放进 Scaffold Agent 的 MCP 配置。
