# Codex Image2 Skill

让 Codex 通过自定义 OpenAI 兼容 API，直接调用 `gpt-image-2` 生成或编辑图片。

仓库内置 Windows 和 macOS 原生程序。普通用户不需要安装 Python、Node.js、Go 或其他运行环境。

## 一行安装、配置和测试

Windows 用户只需把下面整行发给 Codex：

```text
请安装并初始化这个 Skill：https://github.com/WKing-1217/codex-image2-skill；如果权限不足，请提示我开启“完全访问”；安装后运行本地安全配置向导，不要让我在聊天中发送 API 密钥，配置成功后立即生成并展示一张测试图片。
```

接下来：

1. 如果 Codex 提示权限不足，请由用户手动开启“完全访问”后让它继续。Skill 不能替用户修改权限。
2. 安装完成后会弹出“Codex Image2 安全配置”窗口。
3. API 地址不会预填，请填写 API 服务商提供的完整地址。
4. 在隐藏输入框填写 API Key，然后点击“保存并测试”。
5. 程序会立即生成一张低质量测试图片，Codex 随后展示结果，不需要重启。

> 不要把 API Key 发到 Codex 聊天里。配置窗口会把 Key 保存到当前 Windows 用户的凭据管理器，聊天、命令行参数和 Skill 文件都不会包含明文 Key。

## 模型权限

API Key 所属账户或分组必须支持 `gpt-image-2`。具体权限请向所使用的 API 服务商确认；如果不支持，测试时会提示模型或订阅权限不可用。

## 功能

- Windows 本地安全配置窗口
- 配置完成后立即测试，无需重启 Codex
- 文生图
- 单图或多图编辑
- 可选 PNG Mask 局部编辑
- JSONL 并发批量生图
- 支持 Base64 和 URL 图片响应
- 自动重试网络超时、429、5xx 和 524 错误
- 输出文件覆盖保护
- API Key 脱敏，不写入仓库、聊天或日志
- Windows x64/ARM64 与 macOS Intel/Apple Silicon 原生程序

## 配置保存位置

Windows 第一版使用：

- API Key：Windows Credential Manager，目标名称 `codex-image2/CODEX_API_KEY`
- API 地址：`%APPDATA%\CodexImage2\config.json`

程序在每次调用时直接读取这两个位置，因此已经打开的 Codex 不需要重新启动。

环境变量仍然兼容，并且优先级高于本地安全配置：

```powershell
[Environment]::SetEnvironmentVariable("CODEX_API_URL", "https://api.example.com/v1", "User")
[Environment]::SetEnvironmentVariable("CODEX_API_KEY", "你的API密钥", "User")
```

上面的 `api.example.com` 只是占位示例，必须替换成自己的 API 服务地址。

环境变量方式主要用于自动化或旧版兼容。给普通客户时优先使用安全配置窗口。

## 使用 Skill

生成图片：

```text
使用 $codex-image2 生成一张图片：
一只戴着宇航员头盔的橘猫站在月球表面，远处可以看到地球，电影感灯光。
```

修改图片：

```text
使用 $codex-image2 修改这张图片：
只把背景替换成雪山，人物、服装、姿势和构图保持不变。
```

如果尚未配置，Skill 会运行本地配置向导，不会要求用户在聊天中粘贴密钥。

## 手动安装

Windows PowerShell：

```powershell
git clone https://github.com/WKing-1217/codex-image2-skill.git
Copy-Item codex-image2-skill\codex-image2 "$HOME\.codex\skills\codex-image2" -Recurse
```

安装后手动运行配置向导：

```powershell
& "$HOME\.codex\skills\codex-image2\bin\codex-image2-windows-amd64.exe" setup
```

Codex 通常会自动发现新 Skill。如果没有出现在 Skill 列表中，再重新启动 Codex；安全配置和测试生图本身不需要重启。

## CLI 用法

选择与系统匹配的程序：

| 系统 | 可执行文件 | 安全配置向导 |
| --- | --- | --- |
| Windows x64 | `codex-image2/bin/codex-image2-windows-amd64.exe` | 支持 |
| Windows ARM64 | `codex-image2/bin/codex-image2-windows-arm64.exe` | 支持 |
| macOS Intel | `codex-image2/bin/codex-image2-darwin-amd64` | 暂不支持 |
| macOS Apple Silicon | `codex-image2/bin/codex-image2-darwin-arm64` | 暂不支持 |

查看配置状态，不会显示 Key：

```powershell
& "codex-image2/bin/codex-image2-windows-amd64.exe" status
```

重新配置并生成测试图片：

```powershell
& "codex-image2/bin/codex-image2-windows-amd64.exe" setup
```

清除安全配置：

```powershell
& "codex-image2/bin/codex-image2-windows-amd64.exe" reset --yes
```

生成图片：

```powershell
& "codex-image2/bin/codex-image2-windows-amd64.exe" generate `
  --prompt "A tiny blue nebula inside a glass bottle" `
  --quality auto `
  --out "output/imagegen/nebula.png"
```

编辑图片：

```powershell
& "codex-image2/bin/codex-image2-windows-amd64.exe" edit `
  --image "input.png" `
  --prompt "Replace only the background with a warm studio backdrop" `
  --out "output/imagegen/edited.png"
```

批量任务格式和完整工作流请查看 [`codex-image2/SKILL.md`](codex-image2/SKILL.md) 和 [`batch-format.md`](codex-image2/references/batch-format.md)。

## macOS

macOS 原生程序仍支持生图和改图，第一版安全配置窗口仅支持 Windows。macOS 用户需要在启动 Codex 的环境中配置：

```bash
export CODEX_API_URL="https://api.example.com/v1"
export CODEX_API_KEY="你的API密钥"
```

上面的 `api.example.com` 只是占位示例，必须替换成自己的 API 服务地址。

如果可执行权限没有保留：

```bash
chmod +x codex-image2/bin/codex-image2-darwin-*
```

## 从源码构建

源码只使用 Go 标准库：

```powershell
Push-Location codex-image2\src
$env:CGO_ENABLED = "0"

$env:GOOS = "windows"
$env:GOARCH = "amd64"
go build -trimpath -ldflags "-s -w" -o ..\bin\codex-image2-windows-amd64.exe .

$env:GOARCH = "arm64"
go build -trimpath -ldflags "-s -w" -o ..\bin\codex-image2-windows-arm64.exe .
Pop-Location
```

测试：

```powershell
Push-Location codex-image2\src
go test ./...
go vet ./...
Pop-Location
```

## 常见问题

### 配置后还需要重启吗？

不需要。新版程序每次运行都会读取 Windows Credential Manager 和 `%APPDATA%\CodexImage2\config.json`。

只有 Codex 没有自动识别刚安装的 Skill 时，才可能需要重启来刷新 Skill 列表。

### 为什么没有直接在聊天里询问密钥？

聊天内容可能进入历史记录、截图或调试信息。安全配置窗口使用隐藏输入框，并将 Key 保存到 Windows Credential Manager。

### 接口返回 401、403 或 404

- 401：重新运行 `setup`，检查 API Key。
- 403：当前账号或订阅分组可能没有 `gpt-image-2` 权限。
- 404：检查 API 地址，或确认服务端提供 `gpt-image-2`。

### 接口返回 524 或超时

尝试降低质量、使用 `1024x1024`、减少批量并发，或稍后重试。

### 是否支持所有中转服务？

服务需要兼容：

```text
POST /v1/images/generations
POST /v1/images/edits
```

API 地址必须使用 HTTPS；只有 `localhost` 和本机回环地址允许 HTTP。

## 安全说明

- 不要把真实 API Key 提交到 GitHub、Skill、提示词、截图或聊天消息。
- 不要通过命令行参数传递 Key。
- 建议为不同服务使用独立密钥并定期轮换。
- 卸载 Skill 前可以运行 `reset --yes` 删除本地 API 地址和 Windows 凭据。
- 发布包内的 `SHA256SUMS.txt` 可用于核对四个原生程序的 SHA-256。
- 从 GitHub 下载的未签名程序可能触发 Windows SmartScreen；面向大量客户分发时建议给发布二进制做代码签名。

## License

[MIT](LICENSE)

本项目基于 [fengfengzhidao/codex-image2-skill](https://github.com/fengfengzhidao/codex-image2-skill) 继续开发，并按照 MIT License 保留原作者许可与版权声明。
