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
Go 本地 agent
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

P2 内仍计划：

- 将 Go agent 生命周期嵌入桌面进程
- 渲染后的 Markdown / 清洗后的 HTML 模式
- 桌面端对 Auto Capture 与队列状态的控制
- 归档与 AI 的桌面设置
- macOS / Linux 原生打包 CI

目前桌面应用连接独立运行的本地 agent。这样在嵌入式运行时重构期间，UI/数据边界可以保持稳定。

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

### 2. 启动本地 agent

要求：Go 1.22+。

打开一个 PowerShell 窗口：

```powershell
cd apps/agent

$env:DCA_DATA_DIR="$HOME\dca-data"
$env:DCA_TOKEN="replace-with-a-long-random-token"

go run ./cmd/clipper-agent
```

默认端点：

```text
http://127.0.0.1:27123
```

在另一个 PowerShell 中做健康检查：

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

在仓库中再开一个 PowerShell：

```powershell
cd apps/desktop

$env:DCA_TOKEN="replace-with-a-long-random-token"

go mod tidy
go run .
```

桌面窗口从同一个本地 agent 读取数据。

若要做 Wails 热重载开发，请安装已锁定的兼容 CLI：

```powershell
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
wails dev
```

仓库锁定 Wails v2.12.0，因为现有项目使用 Go 1.22；更新的 Wails v2.13.0 模块需要 Go 1.25。

## Windows 打包

桌面阅读器可用 Wails CLI 打成 Windows exe。打包 **不会** 把 Go agent 嵌进桌面进程。

本地（PowerShell）：

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

- `defuddle-browser-mirror.exe`（Wails 桌面阅读器）
- `clipper-agent.exe`（独立 Go agent）
- 若 runner 上 NSIS 可用，还会有安装包

完整使用仍需三件套，桌面 exe 只是阅读器：

1. 本机先跑 `clipper-agent.exe`（`DCA_DATA_DIR`、`DCA_TOKEN` 等环境变量与开发时相同；默认 `http://127.0.0.1:27123`）
2. 浏览器中另行加载扩展（`apps/extension/dist`）
3. 再打开 `defuddle-browser-mirror.exe`

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
```

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
