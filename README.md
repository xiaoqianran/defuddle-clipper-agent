# Defuddle Clipper Agent

本地优先的 **浏览器镜像与网页收件箱**。

浏览器扩展刻意保持精简，且基本无界面。主界面是本地桌面上的大尺寸阅读器：跟随你正在浏览的页面，保存持久历史，并可展示可选的 AI 派生笔记。

```text
Chrome / Edge
    ↓ 普通页面加载 / SPA 导航 / 活动标签切换
MV3 浏览器传感器
    ↓
Defuddle
    ↓
ContentPacket v1
    ↓ localhost
Go 本地 agent（独立进程，或嵌入桌面）
    ├─ 持久归档
    ├─ 当前浏览器状态
    ├─ History / Reader API
    └─ 可选 AI
    ↓
Wails + Svelte 桌面应用
    ├─ History
    ├─ 大尺寸 Reader
    ├─ Follow Browser
    ├─ Transcript
    └─ AI / Notes
```

预期交互是：

```text
在浏览器中打开页面
→ 页面被自动复制
→ 出现在本地 History
→ 大尺寸 Reader 跟随当前浏览器页面
```

不依赖 Obsidian。`DCA_DATA_DIR` 可以是任意普通本地文件夹。

## 当前状态

### P0 — 传输与持久化 ✅

- Defuddle 抽取与 Markdown 转换
- ContentPacket v1
- localhost HTTP 传输
- 扩展侧持久重试队列
- 先落原始数据的文件系统持久化
- 可选的 OpenAI-compatible AI
- 长文档分块 + 综合
- 测试与 CI

### P1 — 自动浏览器镜像 ✅

- 普通页面加载会自动捕获
- 通过 `webNavigation` 观察 History API / SPA 导航
- 上报活动标签/窗口焦点，供 Follow Browser 使用
- 捕获延迟 + DOM 稳定性检查
- 规范 URL 归一化
- 内容指纹去重
- 抑制嘈杂页面更新造成的重复捕获
- 暂停/恢复 Auto Capture
- 域名允许列表 / 拒绝列表
- 忽略不支持的浏览器 URL
- 手动 **Capture now** 仅作为回退/调试手段保留

### P2 — 首版桌面 Reader ✅ / 部分完成

已实现：

- `apps/desktop` Wails + Svelte 应用
- 1500×950 大尺寸桌面窗口
- 基于文件系统的 History API
- 单条捕获的 Reader API
- History/搜索面板
- 大尺寸 Reader 面板
- AI / Notes 面板
- Follow Browser 模式
- 当 Defuddle 提供 transcript 数据时的 transcript 视图
- 实时活动浏览器状态
- 桌面前端已纳入根目录 npm typecheck/build CI
- Windows 原生 Wails 打包 CI（Actions artifact `windows-amd64`）
- 桌面进程嵌入/复用 Go agent 运行时：双击 exe 即启动本地捕获 HTTP 服务；若本机已有 clipper-agent 在跑则复用

P2 内仍计划：

- 渲染后的 Markdown / 清洗后的 HTML 模式
- 桌面端对 Auto Capture 与队列状态的控制
- 归档与 AI 的桌面设置
- macOS / Linux 原生打包 CI

桌面 UI 仍通过 HTTP 客户端访问 `http://127.0.0.1:27123`（可用 `DCA_AGENT_URL` 覆盖，仅回环）。捕获/AI 逻辑继续复用 `apps/agent`，不在 Svelte 里再实现一套。

## 仓库布局

```text
.
├── apps/
│   ├── extension/          # 自动浏览器传感器
│   ├── agent/              # Go localhost 服务 + 归档/AI
│   └── desktop/            # Wails + Svelte 大尺寸本地阅读器
├── packages/
│   └── protocol/           # ContentPacket v1 JSON schema
├── docs/
│   ├── ARCHITECTURE.md
│   ├── SECURITY.md
│   └── REFERENCES.md
├── AGENTS.md
└── ROADMAP.md
```

## 快速开始 — Windows / PowerShell

### 1. 构建扩展与桌面前端

要求：Node.js 20+。

```powershell
git clone https://github.com/xiaoqianran/defuddle-clipper-agent.git
cd defuddle-clipper-agent

npm install
npm run typecheck
npm run build
```

`npm run build` 会同时构建：

```text
apps/extension/dist
apps/desktop/frontend/dist
```

### 2. 日常只开一个东西：桌面 exe

默认端点：

```text
http://127.0.0.1:27123
```

**双击 `Defuddle.exe`（或 `defuddle-browser-mirror.exe`）就够了。** 不要再单独开 `clipper-agent`。桌面会在进程内启动同一套 Go agent（`DCA_ADDR` 默认 `127.0.0.1:27123`）。若该地址上已经有本 agent 在跑（`GET /health` 返回 `protocolVersion` `1.0`），桌面会复用它，而不会再绑一次端口。关闭桌面时，也只会停掉**本进程启动**的服务器。

浏览器扩展只需在 Chrome / Edge 里**加载一次**，之后正常上网即可。

无界面/脚本场景仍可单独跑 agent（Go 1.22+）：

```powershell
cd apps/agent

$env:DCA_DATA_DIR="$HOME\dca-data"
$env:DCA_TOKEN="replace-with-a-long-random-token"

go run ./cmd/clipper-agent
```

健康检查：

```powershell
Invoke-RestMethod http://127.0.0.1:27123/health
```

### 3. 加载扩展

打开：

```text
chrome://extensions
```

然后：

```text
Developer mode
→ Load unpacked
→ select apps/extension/dist
```

打开扩展设置并填写：

```text
Agent URL: http://127.0.0.1:27123
Token:     与 DCA_TOKEN 相同的值
Auto Capture: ON
Follow Browser: ON
```

之后正常浏览即可，不必再按 **Capture now**。

### 4. 启动大尺寸桌面 Reader

桌面启动即带本地 agent。环境变量与独立 agent 相同（`DCA_ADDR`、`DCA_DATA_DIR`、`DCA_TOKEN`、`DCA_AI_*`、`DCA_OPENAI_*`）。`DCA_AGENT_URL` 仍给桌面 HTTP 客户端用（仅回环，默认 `http://127.0.0.1:27123`）；若桌面自己启动了服务器，客户端会改连实际绑定地址。

本地跑桌面**不需要** Wails CLI，也**不需要** gcc / MinGW。Wails 在 Windows 上必须带 `production` 标签，否则只会弹出标题为 `Error` 的对话框。

```powershell
cd apps/desktop

$env:DCA_DATA_DIR="$HOME\dca-data"
$env:DCA_TOKEN="replace-with-a-long-random-token"
$env:CGO_ENABLED="0"

go run -tags desktop,production .
```

Windows GUI 子系统下，启动/绑定错误会写入 `DCA_DATA_DIR\desktop.log`（若未设置数据目录，则写到 `%USERPROFILE%\.defuddle-clipper-agent\desktop.log`）。

热重载（可选，才需要 Wails CLI）：

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
wails dev
```

仓库锁定 Wails v2.12.0，因为现有项目使用 Go 1.22；更新的 Wails v2.13.0 模块需要 Go 1.25。

## Windows 打包

日常本地运行用上一节的 `go run -tags desktop,production .` 即可。下面的 Wails CLI / gcc 只用于打带图标和安装包的发行版。

`defuddle-browser-mirror.exe` **已内嵌** Go agent 生命周期：双击即可监听 `127.0.0.1:27123`。Windows 打包产物仍可附带独立 `clipper-agent.exe`，供无界面或已有 agent 在跑时使用。

不装 Wails CLI 也可以出 exe：

```powershell
cd apps/desktop
$env:CGO_ENABLED="0"
go build -tags desktop,production -o defuddle-browser-mirror.exe .
```

若要带图标 / NSIS 安装包，才需要已锁定的 Wails CLI（以及它背后的 gcc）：

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
cd apps/desktop
wails build
```

`wails build` 会按 `wails.json` 运行 `frontend:install` / `frontend:build`。产物默认在：

```text
apps/desktop/build/bin/defuddle-browser-mirror.exe
```

GitHub Actions：推送到 `main`、推送 `v*` 标签、或手动 `workflow_dispatch` 会运行 `.github/workflows/windows-desktop.yml`。到仓库 **Actions** 页打开对应 run，下载 `windows-amd64` artifact。其中包含：

- `defuddle-browser-mirror.exe`（Wails 桌面阅读器，内嵌本地捕获 HTTP 服务）
- `clipper-agent.exe`（独立 Go agent，可选，无界面场景）
- 若 runner 上 NSIS 可用，还会有安装包

完整使用：

1. 打开 `defuddle-browser-mirror.exe`（若本机已有 `clipper-agent.exe` 在跑则复用）
2. 浏览器中另行加载扩展（`apps/extension/dist`），Agent URL 仍为 `http://127.0.0.1:27123`

Windows 10+ 通常已自带 WebView2 Runtime，构建机与最终用户一般不必再装自定义 bootstrapper。若个别机器缺少 WebView2，从 Microsoft 安装官方运行时即可。

## 桌面行为

### Follow Browser

启用后：

```text
活动浏览器标签变化
→ 扩展上报当前 URL/标题
→ 桌面轮询本地浏览器状态
→ 匹配的镜像捕获在 Reader 中打开
```

若页面仍在抽取中，桌面会等到它出现在 History 后再打开。

### 查看旧历史

点击较旧的 History 条目会有意关闭 Follow Browser，这样 Reader 不会立刻跳回当前浏览器标签。

想让 Reader 再次跟踪浏览器时，重新启用 **Follow Browser**。

### Transcript

当 Defuddle 通过 extractor variables 暴露视频 transcript 时，捕获会把它放在：

```text
ContentPacket.media.transcript
```

桌面 Reader 随后会提供 **Transcript** 标签页。

## 本地归档

```text
<DCA_DATA_DIR>/
└── captures/
    └── YYYY/MM/DD/<capture-id>/
        ├── packet.json
        ├── source.md
        ├── analysis.json       # 可选的 AI 派生物
        ├── analysis-error.txt  # AI 失败时
        └── note.md
```

`packet.json` 和 `source.md` 会在 AI 运行之前写入。因此 AI 失败不会丢失已捕获的页面。

计划中的 P3 增补：

```text
source.html
raw.html
assets/
SQLite catalog
full-text search
```

文件系统仍是主数据；未来的索引必须可重建。

## 本地 API

当前重要端点：

```text
GET  /health
POST /v1/captures
GET  /v1/captures?limit=100
GET  /v1/captures/{captureId}
POST /v1/captures/{captureId}/reprocess
POST /v1/browser/active
GET  /v1/browser/state
GET  /v1/policy
PUT  /v1/policy
GET  /v1/status?limit=100
POST /v1/sensor/heartbeat
GET  /v1/events
```

`/v1/policy` 是捕获控制面（Auto Capture、Archive All、延迟、域名列表），由桌面写入、扩展拉取。`/v1/events` 是 SSE，桌面用它即时刷新 History，而不是靠秒级轮询。

桌面通过其 Go 桥接使用这些 API；Svelte UI 不会直接拿到 Bearer token。

## 可选 AI

自动捕获与 AI 相互独立。AI 默认关闭。

任何 OpenAI-compatible 的 `/chat/completions` 端点都可以（OpenAI、NVIDIA NIM、vLLM、本地服务器）。

```powershell
$env:DCA_AI_ENABLED="true"
$env:DCA_OPENAI_BASE_URL="https://api.example.com/v1"
$env:DCA_OPENAI_API_KEY="..."
$env:DCA_OPENAI_MODEL="your-model-id"
```

NVIDIA NIM 示例。该模型是多模态 **text-out**：可以结合 Markdown 阅读封面图，但不会生成图片（不使用 `/v1/images/generations`）。

```powershell
$env:DCA_AI_ENABLED="true"
$env:DCA_OPENAI_BASE_URL="https://integrate.api.nvidia.com/v1"
$env:DCA_OPENAI_API_KEY=""
$env:DCA_OPENAI_MODEL="google/diffusiongemma-26b-a4b-it"
$env:DCA_AI_TIMEOUT_SECONDS="180"
```

当 `ContentPacket.metadata.image` 是 `http(s)` URL 时，agent 会把它作为 OpenAI 风格的 `image_url` 部分转发。Data URI 以及其他非 http scheme 会被跳过。`analysis.json` 会记录模型、provider host、prompt 版本、是否发送了图片，以及分析时间戳。API key 绝不会写入已存储的文件。

即使 provider 宕机，捕获的源内容仍会保存。

## 设计规则

1. **浏览器是传感器；桌面才是产品。**
2. **先捕获，后增强。**
3. **纯文件是默认的持久格式。**
4. **Obsidian 可选，绝不是依赖。**
5. **Auto Capture 与 Auto AI 是分开的策略。**
6. **Defuddle 保持上游；不要把抽取逻辑分叉进本仓库。**
7. **ContentPacket 是稳定的浏览器/本地边界。**
8. **派生的 AI/搜索产物必须可重建。**

其余 P2 工作以及 P3–P7 计划见 [`ROADMAP.md`](ROADMAP.md)。

## 许可证

MIT。见 [`LICENSE`](LICENSE)。
