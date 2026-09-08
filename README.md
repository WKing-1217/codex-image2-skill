# Codex Image2 Skill

让 Codex 通过自定义 OpenAI 兼容 API，直接调用 `gpt-image-2` 生成或编辑图片。

仓库内置 Windows 和 macOS 原生程序。普通用户不需要安装 Python、Node.js、Go 或其他运行环境。

## 一行安装、配置和测试

Windows 用户只需把下面整行发给 Codex：

```text
请在当前任务的“完全访问”模式下，下载并运行根目录的 install-windows.ps1 来安装或更新并初始化这个 Skill，成功展示测试图后再结束：https://github.com/WKing-1217/codex-image2-skill
```

接下来：

1. 发送指令前，请在当前任务输入框下方选择“完全访问”。只在设置里启用该选项还不够，必须让当前任务实际处于完全访问模式；Skill 不能替用户切换。
2. Codex 会下载仓库并运行 `install-windows.ps1`。脚本校验四个程序、安装或覆盖更新 Skill，并把旧版本保存在 `.codex\skill-backups` 以便恢复。
3. 脚本会在同一轮直接弹出“Codex Image2 安全配置”窗口，不依赖新 Skill 到下一轮才被识别。即使已有旧配置或环境变量也会重新打开窗口。
4. API 地址每次都会保持空白，请填写 API 服务商提供的完整地址。
5. 在隐藏输入框填写 API Key，然后点击“保存并测试”。
6. 程序会真实请求生成一张低质量测试图片（可能产生服务商费用）。只有图片可解码、成功保存且由 Codex 展示后，才算初始化完成，不需要重启。
7. 如果返回 401、没有图片或测试失败，初始化仍未完成。Codex 会说明原因；需要修改地址或密钥时会重新打开本地窗口。取消窗口则停止本次初始化。

不要只让标准 `skill-installer` 安装文件：它会在目标目录已存在时停止，而且新 Skill 通常到下一轮才可用。根目录安装脚本专门解决这两个问题。不要仅运行 `status` / `--dry-run` 后就宣布初始化完成。

> 不要把 API Key 发到 Codex 聊天里。配置窗口会把 Key 保存到当前 Windows 用户的凭据管理器，聊天、命令行参数和 Skill 文件都不会包含明文 Key。

## 模型权限

API Key 所属账户或分组必须支持 `gpt-image-2`。具体权限请向所使用的 API 服务商确认；如果不支持，测试时会提示模型或订阅权限不可用。

## 功能

- Windows 本地安全配置窗口
- 一键安装或覆盖更新，旧版自动备份
- 检测 Windows 沙箱私有桌面，避免窗口在后台无限等待
- 真实测试生图成功才标记初始化完成，无需重启 Codex
- 文生图
- 单图或多图编辑
- 可选 PNG Mask 局部编辑
- JSONL 并发批量生图
- 支持 Base64 和 URL 图片响应
- 普通生图命令支持有限重试；初始化每次提交只请求一次，避免自动重复计费
- 输出文件覆盖保护
- API Key 脱敏，不写入仓库、聊天或日志
- Windows x64/ARM64 与 macOS Intel/Apple Silicon 原生程序

## 配置保存位置

Windows 第一版使用：

- API Key：Windows Credential Manager，目标名称 `codex-image2/CODEX_API_KEY`
- API 地址：`%APPDATA%\CodexImage2\config.json`

程序在每次调用时直接读取这两个位置，因此已经打开的 Codex 不需要重新启动。

本地安全配置的地址和密钥作为一整套优先使用，旧环境变量不会覆盖窗口里新填写的配置。仅在没有本地已保存地址时，才使用环境变量中的完整地址和密钥；不会把两种来源混用。保存中断或缺少密钥时，请重新运行 `setup`。

环境变量仍兼容，主要供自动化、旧版和 macOS 使用：

```powershell
[Environment]::SetEnvironmentVariable("CODEX_API_URL", "https://api.example.com/v1", "User")
[Environment]::SetEnvironmentVariable("CODEX_API_KEY", "你的API密钥", "User")
```

上面的 `api.example.com` 只是占位示例，必须替换成自己的 API 服务地址。

环境变量方式不代表已经通过生图验证。Windows 普通客户优先使用安全配置窗口；窗口配置成功后立即生效，无须清理旧环境变量或重启。

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

Windows 上如果尚未通过初始化验证（包括只有旧版配置或环境变量），Skill 会运行本地配置向导，不会要求用户在聊天中粘贴密钥。

## Windows 安装脚本

把仓库下载或克隆到电脑后，在 Windows PowerShell 中运行：

```powershell
git clone https://github.com/WKing-1217/codex-image2-skill.git
powershell.exe -NoProfile -ExecutionPolicy Bypass -File ".\codex-image2-skill\install-windows.ps1"
```

脚本会完成校验、安装或更新，并立刻启动配置向导。它不会把 API Key 写进命令行或文件。仅检查下载包而不安装时可运行：

```powershell
powershell.exe -NoProfile -ExecutionPolicy Bypass -File ".\codex-image2-skill\install-windows.ps1" -ValidateOnly
```

如果安装后 Skill 没有立即出现在列表中，下一轮对话通常会自动识别；仍未出现时再重启 Codex。安装脚本已经在当前轮直接完成配置和测试，所以不需要等 Skill 被识别才弹窗。

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

查看程序版本（排查客户仍在使用旧文件时很有用）：

```powershell
& "codex-image2/bin/codex-image2-windows-amd64.exe" version
```

状态字段：

- `configured`：是否存在完整的地址和密钥，仅检查配置，不请求 API。
- `initialized`：本地配置是否曾通过新版 `setup` 的真实测试生图；旧版配置默认是 `false`。
- `verified_at` / `test_image`：上次成功初始化的时间和测试图片路径。
- `network_checked`：`status` 始终是 `false`；成功的 `setup` 才是 `true`。

历史验证不能保证密钥永远有效。明确要求“初始化 / 重新配置”时，即使状态为 `true` 也要重新打开窗口并真实生图。取消时保留原配置，但本次初始化不算完成；提交新配置后测试失败则保留新配置为未验证，`initialized` 为 `false`。

重新配置并生成测试图片：

```powershell
& "codex-image2/bin/codex-image2-windows-amd64.exe" setup
```

`setup` 必须真实生成测试图片，不再支持 `--no-test`。任何失败都会返回非零退出码，不能当成初始化成功。

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

macOS 的环境变量配置不会写入 Windows 初始化标记。配置后应实际执行一次 `generate --quality low`（提供提示词和输出路径）并查看生成图片；不能只凭 `status` 或 `--dry-run` 声称初始化成功。

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
go build -buildvcs=false -trimpath -ldflags "-s -w" -o ..\bin\codex-image2-windows-amd64.exe .

$env:GOARCH = "arm64"
go build -buildvcs=false -trimpath -ldflags "-s -w" -o ..\bin\codex-image2-windows-arm64.exe .
Pop-Location
```

测试：

```powershell
Push-Location codex-image2\src
go test ./...
go vet ./...
Pop-Location
powershell.exe -NoProfile -ExecutionPolicy Bypass -File ".\install-windows.ps1" -ValidateOnly
```

## 常见问题

### 配置后还需要重启吗？

不需要。新版程序每次运行都会读取 Windows Credential Manager 和 `%APPDATA%\CodexImage2\config.json`。

只有 Codex 没有自动识别刚安装的 Skill 时，才可能需要重启来刷新 Skill 列表。

### 为什么没有直接在聊天里询问密钥？

聊天内容可能进入历史记录、截图或调试信息。安全配置窗口使用隐藏输入框，并将 Key 保存到 Windows Credential Manager。

### 安装时为什么没有看到配置窗口？

- 确认当前任务实际选择了“完全访问”，而不是只在设置里启用了该选项。
- 不要在 WSL、SSH、Codex cloud 或其他无桌面的环境运行 Windows 配置程序。
- 新版程序检测到 Windows 沙箱的私有桌面会立即报出明确错误，不会在用户看不到的窗口上无限等待。
- 如果提示 PowerShell、System.Windows.Forms 或安全策略错误，检查 Windows PowerShell、AppLocker、杀毒软件和单位设备策略。
- 如果重复安装，请运行根目录 `install-windows.ps1`；标准 Skill 安装器不会覆盖已经存在的目录。

### 接口返回 401、403 或 404

- 401：重新运行 `setup`，检查 API Key。
- 403：当前账号或订阅分组可能没有 `gpt-image-2` 权限。
- 404：检查 API 地址，或确认服务端提供 `gpt-image-2`。

初始化过程中需要纠正配置时，可让 Codex：“重新打开本地配置窗口，真实生成并展示测试图片，成功后再确认初始化完成。”更新版会使用窗口保存的整套地址和密钥，不受旧 `CODEX_API_KEY` 环境变量覆盖。

### 接口返回 524 或超时

尝试降低质量、使用 `1024x1024`、减少批量并发，或稍后重试。

初始化每次点击“保存并测试”只提交一次生图请求，不自动循环重试。超时不代表服务端一定没有执行，重试前请留意服务商的用量记录。

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
