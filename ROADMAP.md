# 路线图

产品目标是：**把用户浏览的页面自动镜像到大尺寸本地桌面应用中**。

## P0 — 传输基础

**状态：已实现，并经 CI 验证。**

- [x] MV3 扩展
- [x] Defuddle 抽取
- [x] ContentPacket v1
- [x] localhost HTTP 传输
- [x] 重试队列
- [x] 先落原始数据的文件系统持久化
- [x] 可选的 OpenAI-compatible 分析
- [x] 测试与 CI

## P1 — 自动浏览器捕获

**状态：已实现，并经 CI 验证。**

- [x] 自动捕获普通页面加载
- [x] 活动标签跟踪
- [x] SPA/history 导航检测
- [x] URL 变化回退
- [x] 捕获延迟与 DOM 稳定性协调器
- [ ] 真正的活动停留时间策略
- [x] 规范 URL 归一化
- [x] 内容指纹与去重
- [x] 抑制 DOM 噪声导致的重复捕获
- [x] 发送活动页面事件，供 Follow Browser 使用
- [x] 暂停/恢复 Auto Capture
- [x] 域名允许列表 / 拒绝列表
- [x] 忽略不支持的浏览器 URL
- [x] 手动 `Capture now` 仅作为回退/调试
- [x] 持久重试投递

已达成的验收：

```text
打开页面 A → 本地自动收到 A
SPA 导航到 B → 本地自动收到 B
滚动 / 广告刷新 / DOM 噪声 → 不会捕获洪水
切换活动标签 → 本地 Follow Browser 状态变化
```

## P2 — 本地桌面应用

**状态：首个可用阅读器已实现。** 技术栈：**Wails + Go + Svelte**。

- [x] `apps/desktop` 脚手架
- [x] 面向现有 agent 的仅本地 Go 客户端
- [x] 基于文件系统的 History API
- [x] 单条捕获的 Reader API
- [x] History 面板
- [x] 大尺寸 Reader 面板
- [x] AI / Notes 面板
- [x] Follow Browser 模式
- [x] 实时当前页面状态
- [x] 搜索框
- [x] 视频页面的 transcript 视图
- [x] 前端 typecheck/build 已纳入 CI
- [ ] 在桌面进程内嵌入/复用 agent 运行时
- [ ] 桌面生命周期拥有本地捕获端点
- [ ] 桌面 UI 中的 Archive All 控制
- [ ] 详细的捕获/队列状态
- [ ] 从桌面暂停 Auto Capture
- [ ] 渲染后的 Markdown/清洗后的 HTML 阅读模式
- [ ] 捕获/数据/AI 设置 UI
- [ ] 原生 Wails 打包 CI

当前验收：

```text
在 Chrome/Edge 中浏览
→ 桌面 History 自动更新
→ Reader 跟随活动浏览器页面
→ 手动选择 History 会退出 Follow Browser
→ 正常阅读不发生在浏览器弹窗里
```

## P3 — 持久页面归档

- [ ] `source.html`
- [ ] 可选的 `raw.html`
- [ ] 图片/资源发现
- [ ] 本地下载资源
- [ ] 确定性资源命名
- [ ] 把 Markdown/HTML 引用改写为本地资源
- [ ] 捕获来源 / 抽取诊断
- [ ] SQLite catalog
- [ ] migrations
- [ ] 全文搜索
- [ ] 重复/冲突策略
- [ ] 从文件系统产物重建数据库

目标捕获结构：

```text
capture/
├── packet.json
├── source.md
├── source.html
├── raw.html          # 可选
├── analysis.json     # 派生
├── note.md           # 派生
└── assets/
```

## P4 — AI 理解

Auto Capture 与 Auto AI 保持独立。

- [ ] 处理任务状态
- [ ] prompt 注册表/版本管理
- [ ] provider/model 注册表
- [ ] 重试/退避
- [ ] 语义 Markdown 分块器
- [ ] 重新处理已有的 `packet.json`
- [ ] processor 注册表：article、documentation、repository、video、paper、discussion、generic
- [ ] 稳定的结构化 AIResult
- [ ] 模型/provider/prompt 来源信息
- [ ] 按页面类型、停留时间、体积和用户策略制定 Auto AI 规则

## P5 — 知识层

- [ ] 标签/概念索引
- [ ] 相关捕获
- [ ] embedding provider 接口
- [ ] 语义搜索
- [ ] 相关笔记边
- [ ] 聚类/主题视图
- [ ] 可重建的 embeddings

## P6 — Agent / 自动化表面

- [ ] MCP server
- [ ] `capture`、`search`、`read`、`reprocess`
- [ ] CLI
- [ ] WebSocket/SSE 事件
- [ ] 外部自动化 API

## P7 — 捕获生态

- [ ] Firefox
- [ ] Safari 调研
- [ ] Native Messaging
- [ ] 可选的 SingleFile 适配器
- [ ] 可选的 yt-dlp 回退
- [ ] RSS/URL 摄入
- [ ] 批量导入

## 明确的非目标

本项目不是 Obsidian 专用应用、浏览器弹窗阅读器、云端 SaaS、通用爬虫，也不是以向量数据库为先的产品。

核心产品：

```text
浏览器活动
→ 自动本地副本
→ 大尺寸桌面阅读器/历史
→ 可选 AI
→ 持久的个人网页归档
```
