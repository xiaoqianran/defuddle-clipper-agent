# AGENTS.md

本仓库按职责刻意拆分。

## 产品不变量

浏览器扩展是 **传感器/桥接**。本地桌面应用才是 **主产品 UI**。

正常使用应变成：

```text
正常浏览
→ 页面被自动捕获
→ 本地桌面 History/Reader 更新
```

不要围绕浏览器弹窗来优化产品。

## 不可妥协的架构规则

1. `apps/extension` 只负责浏览器观察、Defuddle 抽取、捕获协调与投递。
2. 扩展应尽量无界面；弹窗/手动捕获只是回退/调试 UI。
3. 自动捕获必须处理普通导航和 SPA 导航，且不能产生重复洪水。
4. 阅读、历史、搜索、AI、文件以及未来的知识功能属于本地应用/核心。
5. `apps/agent` 包含当前的 Go 核心/localhost 服务。未来的 `apps/desktop` 必须复用或抽出该核心，而不是再实现一份。
6. `packages/protocol/content-packet.schema.json` 是浏览器与本地核心之间的契约。
7. 绝不要让扩展写入任意本地路径。
8. 绝不要把 AI 成功当作源内容持久化的前提。
9. Auto Capture 与 Auto AI 是分开的能力与设置。
10. 不要把 Defuddle 源码复制进本仓库。把它当作上游依赖来消费。
11. 传输处理器必须保持很薄。业务逻辑放在 services 里。
12. 文件系统产物是默认的持久源。Obsidian 可选，绝不能成为核心依赖。
13. 派生数据（`analysis.json`、`note.md`、索引、embeddings）只要可行，就必须能从主捕获产物重新生成。
14. 新的协议字段应在 `1.x` 内做加法；破坏性变更需要主版本协议升级。
15. 绝不要记录 API key、Bearer token、完整原始 HTML，或完整的 AI 请求载荷。

## 自动捕获规则

捕获协调器应判断有意义的页面转换，而不是任意 DOM 变化。

必需概念：

- 活动标签；
- 普通导航；
- SPA 导航（`pushState`、`replaceState`、`popstate`）；
- debounce / DOM 稳定性；
- 规范 URL；
- 抽取内容指纹；
- 在有用时设置最短停留时间；
- 暂停/恢复；
- 允许列表/拒绝列表；
- 重试队列。

滚动、广告、懒加载小组件或不相关的 DOM 变动，不得产生重复捕获。

## 桌面应用方向

首选技术栈：Wails + Go + Svelte。

桌面应用应提供：

- History；
- 大尺寸 Reader；
- Follow Browser 模式；
- Archive All 模式；
- AI / Notes 面板；
- 搜索；
- 捕获/AI 设置。

一旦桌面应用成为默认分发形态，它就应拥有本地服务器的生命周期。

## 提交约定

使用 Conventional Commits：

- `feat(extension): ...`
- `feat(desktop): ...`
- `feat(agent): ...`
- `feat(protocol): ...`
- `fix(extension): ...`
- `fix(agent): ...`
- `docs(product): ...`
- `docs(architecture): ...`
- `chore(ci): ...`

改动保持内聚。测试应与它们所验证的行为放在同一次提交中。

## Go

- 优先标准库，除非某依赖能实质提升产品。
- HTTP 适配器只关注传输问题。
- 在 protocol/core 边界校验外部输入。
- 对派生产物使用原子文件替换。
- 拒绝不安全的非回环默认值。
- 始终关闭 response body。
- 为出站 HTTP 添加超时。
- 让 services 既能被独立 agent 复用，也能被 Wails 桌面应用复用。

## TypeScript 扩展

- MV3。
- 扩展中不要用沉重的 UI 框架。
- Content script 负责观察/抽取；background service worker 负责传输/重试，并跟踪标签级状态。
- API key 不要放在扩展里。那里只能存放本地桥接 token。
- 把页面 DOM/内容当作敌意输入。
- 把失败的捕获排入 `chrome.storage.local` 队列。
- 在构造 `ContentPacket` 之前，不要静默截断源内容。
- 优先使用显式的导航/指纹状态机，而不是靠宽泛的 MutationObserver 驱动重新捕获。

## 测试

合并浏览器/核心变更之前：

```bash
cd apps/agent && go test ./...
npm run typecheck
npm run build
```

自动捕获相关工作只要可行，就必须为导航去重/状态转换补充测试。

桌面工作在被视为完成之前，必须在 CI 中加入自己的构建/测试任务。

若执行环境无法安装依赖，不要声称构建已通过；CI 才是事实来源。
