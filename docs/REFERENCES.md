# 参考与上游边界

本项目是围绕上游组件与架构思路编写的原创胶水/应用代码。

## Defuddle

仓库：`kepano/defuddle`

角色：

- 直接的 npm 依赖
- 已渲染 DOM 抽取
- 元数据/Schema.org 抽取
- 通过完整 bundle 做 Markdown 转换
- 站点专用 extractor variables

这里没有 vendor 任何 Defuddle 源码。

## Obsidian Web Clipper

仓库：`obsidianmd/obsidian-clipper`

角色：实现参考，用于：

- 浏览器扩展捕获生命周期
- Defuddle `parseAsync()` 集成
- 选区/高亮概念
- variables/templates
- 跨浏览器产品考量

本仓库不是 Obsidian Web Clipper 的 fork。

## Atomic

仓库：`kenforthewin/atomic`

角色：架构参考，用于：

- 本地优先捕获
- 排队投递
- 把核心业务逻辑与传输/UI 分开
- 未来的语义搜索 / MCP 方向

Atomic 代码没有复制进本仓库。

## allan-deng/web_clipper

角色：小型参考，用于确认「扩展 → localhost 服务」这一实用模式。

这里不使用它的 Readability/Turndown 抽取栈，因为抽取依赖是 Defuddle。

## 依赖策略

优先通过公开的 package/API 边界使用上游项目。只有当所需改动无法合理向上游贡献，也无法通过组合实现时，才考虑 fork。
