# 架构

## 产品边界

浏览器扩展是传感器。本地桌面应用才是产品。

```text
┌──────────────────────── 浏览器 ────────────────────────┐
│                                                        │
│ 已渲染/已认证的 DOM                                     │
│        ↓                                               │
│ 导航观察器                                              │
│        ↓                                               │
│ debounce + DOM 稳定性                                   │
│        ↓                                               │
│ Defuddle                                               │
│        ↓                                               │
│ ContentPacket v1                                       │
│        ↓                                               │
│ background 投递 + 重试队列                               │
└────────────────────────┬───────────────────────────────┘
                         │ localhost HTTP
                         ▼
┌──────────────────── 本地 Go 核心 ──────────────────────┐
│                                                       │
│ 捕获端点                                               │
│      ↓                                                │
│ CaptureService                                        │
│      ├─ 先落原始数据的 Store                            │
│      ├─ 当前页面 / Follow Browser 状态                  │
│      ├─ 归档/索引服务                                   │
│      └─ 可选 Analyzer                                  │
└────────────────────────┬──────────────────────────────┘
                         │
                         ▼
┌──────────────────── 桌面应用 ──────────────────────────┐
│                                                       │
│ History        Reader                 AI / Notes       │
│                                                       │
│ Follow Browser | Archive All | Search | Settings      │
└───────────────────────────────────────────────────────┘
```

## 浏览器捕获模型

正常使用不得要求点击弹窗。

扩展会观察：

- 普通页面加载；
- 活动标签变化；
- SPA 导航（`pushState`、`replaceState`、`popstate`）；
- 不会引起整页重载的 URL 变化。

捕获协调器用于抑制噪声：

```text
导航信号
    ↓
debounce
    ↓
等待可用/稳定的 DOM
    ↓
Defuddle.parseAsync()
    ↓
规范 URL + 内容指纹
    ↓
是新的有意义页面？
  ├─ 否  → 忽略
  └─ 是  → 提交 ContentPacket
```

滚动、广告、懒加载小组件和不相关的 DOM 变动，不得产生捕获洪水。

手动捕获仍是回退/调试动作，不是主要交互。

## 为什么浏览器要保持很薄

扩展擅长：

- 看到已渲染页面，包括已认证内容；
- 检测导航和活动标签状态；
- 对实时 DOM 运行 Defuddle；
- 转发选区/transcript/元数据；
- 缓冲失败的投递。

扩展不应拥有：

- 主阅读 UI；
- 模型凭证；
- 本地文件系统编排；
- 索引/搜索数据库；
- 大型持久队列；
- 长时间处理；
- 知识关系。

## 桌面应用

首选目标技术栈：

```text
Wails
├── Go 核心
│   ├── 捕获服务器
│   ├── 归档
│   ├── 处理
│   ├── 搜索
│   └── AI providers
└── Svelte UI
    ├── History
    ├── Reader
    ├── AI / Notes
    ├── Follow Browser
    └── Settings
```

现有的 `apps/agent` 实现不会丢弃。其 services 应被复用或迁入共享 Go 核心，这样桌面应用就不会再造第二套实现。

## Follow Browser 与 Archive All

这是两个分开的概念。

### Follow Browser

浏览器发送活动页面/导航状态。本地 Reader 跟随用户当前正在查看的页面。

```text
活动浏览器标签
       ⇅
本地 Reader
```

### Archive All

每一个符合条件、已去重的页面都会存入 History。

```text
History
├── GitHub repository
├── article
├── Bilibili video + transcript
├── documentation page
└── ...
```

Follow Browser 可以在 Archive All 关闭时启用，反之亦然。

## Defuddle 边界

Defuddle 保持为上游依赖，并拥有抽取相关职责：

- 主内容检测；
- 一致的清理与 Markdown 转换；
- 元数据与 Schema.org 抽取；
- 站点专用 extractors；
- 在支持时，异步 extractor variables（例如 transcripts）。

本仓库拥有捕获策略、投递、归档、UI、AI 与知识行为。

## 协议

`ContentPacket` 是浏览器抽取与本地处理之间的持久边界。

稳定字段包括：

```text
protocolVersion
captureId
capturedAt
source
content
selection
highlights
metadata
media
```

未来的浏览器镜像字段应在协议 `1.x` 内做加法，例如导航/会话元数据或活动页面状态。

## 持久化模型

主数据：

```text
packet.json
source.md
source.html    # 计划中
raw.html       # 可选/计划中
assets/        # 计划中
```

派生数据：

```text
analysis.json
note.md
SQLite/FTS indexes
embeddings
relations
```

只要可能，派生数据就必须能从主归档重建。

Obsidian 不是系统依赖。普通文件系统目录才是默认存储。把该目录指向一个 Obsidian vault，只是一种可选用法。

## AI 策略

自动捕获与自动 AI 刻意保持独立。

```text
Auto Capture: ON
Auto AI:      OFF / rules
```

可能的规则：

- paper → 自动分析；
- video → 存在 transcript 时分析；
- 长文章 → 达到停留/长度阈值后再分析；
- 搜索/结果页 → 不分析。

AI 失败绝不会使页面归档失效或被删除。

## 失败模型

| 失败 | 行为 |
|---|---|
| Defuddle 失败 | 不提交无效 packet；诊断状态会对外暴露 |
| 本地应用宕机 | 扩展将捕获入队 |
| 认证被拒绝 | 队列项被保留 |
| 磁盘失败 | 投递失败，该项仍可重试 |
| AI 关闭 | 归档仍完全可用 |
| AI 失败 | 主捕获得以保留；错误可检查 |
| 重复导航 | 指纹/去重防止归档洪水 |
| 保存后响应丢失 | 同一 captureId 是幂等的 |

## 存储扩展点

文件系统是默认的持久接收端。未来的存储/目录集成必须留在核心 services 之后。

```go
type Store interface { ... }
```

SQLite 为文件系统产物建立目录，而不是取代它们。

## 传输扩展点

核心捕获/知识服务不得依赖传输。输入以后可能来自：

- 浏览器 HTTP；
- Native Messaging；
- CLI；
- MCP；
- 批量导入；
- URL/RSS 摄入。

## 参考经验

### Obsidian Web Clipper

可借鉴：

- 已渲染页面的抽取模式；
- Defuddle 集成；
- 选区/高亮概念。

不要继承：

- 仅面向 Obsidian 的目标假设；
- 把浏览器弹窗当作主阅读面；
- 把驻留在浏览器中的 AI 当作核心架构。

### Atomic

可借鉴：

- 本地优先处理；
- 核心与传输分离；
- 持久归档/知识思路；
- 未来的语义/MCP 方向。

在自动捕获和桌面 Reader 足够出色之前，不要继承数据库/embedding 的复杂度。
