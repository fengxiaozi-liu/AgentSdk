# AGENT SDK 改造计划

## 1. 改造定位

本次改造的目标是把当前项目从“带 CLI/TUI 能力的 Agent 应用代码”整理为“可被外部宿主复用的 Agent SDK”。

改造后的 SDK 只负责：

- Agent 运行与工具调用编排
- 会话、消息、历史与权限管理
- LLM Provider 抽象
- Tool 体系定义与内置工具
- MCP Tool 接入
- 可扩展 hook/event 机制

SDK 不负责：

- CLI spinner
- 终端主题
- CLI 输出格式
- 输入补全
- LSP 工具封装
- 终端 diff 彩色渲染

后续 CLI/IDE 宿主项目可以引入本 SDK，并在宿主层自行创建 LSP 相关工具、补全、主题、spinner 等交互能力。

## 2. 改造前架构

当前代码更接近一个从应用中抽出的 Agent runtime，核心能力、工具能力、CLI 能力和扩展能力存在交叉。

```text
agent.go
  Agent 主运行逻辑、模型调用、工具调用循环

session/
message/
history/
  会话、消息、文件历史服务

tools/
  core/
    Tool 抽象
  base/
    ls/glob/grep/view/edit/write/patch/bash/fetch/diagnostics
    部分工具直接依赖 LSP
  mcp/
    MCP server tool 适配
  agent/
    子 Agent 工具

extensions/
  lsp/
    LSP client、协议、诊断缓存
  completions/
    文件/目录输入补全

infra/
  db/
    SQLite/sqlc/goose
  diff/
    diff/patch 逻辑，同时混有终端展示能力
  fileutil/
    文件发现、rg/fzf、glob fallback
  format/
    CLI 输出格式、spinner
  theme/
    终端主题、diff/markdown/syntax 颜色
  logging/
    日志

llm/
  models/
    模型 ID、provider、支持模型列表
  provider/
    各 provider 适配

prompt/
permission/
pubsub/
config/
```

### 2.1 改造前主要问题

| 问题 | 当前表现 | 影响 |
|------|----------|------|
| SDK 和 CLI/IDE 边界不清 | `infra/format`、`infra/theme`、`extensions` 留在 SDK 内 | SDK 携带终端展示、交互和 IDE/LSP 假设 |
| 基础工具和 LSP 强绑定 | `view/edit/write/patch/diagnostics` 直接依赖 LSP client | 不启用 LSP 时工具层仍被扩展污染 |
| data 层抽象不足 | service 直接靠底层查询实现 | 后续切换数据源困难 |
| diff 混入展示逻辑 | diff 基础算法和终端颜色渲染耦合 | CLI 展示能力难以迁出 |
| model/provider 关系偏重 | SDK 维护大量模型支持信息 | SDK 需要频繁跟随模型变化 |
| prompt 偏厚 | 系统提示词与运行策略耦合较多 | 调用方自定义不够轻量 |

## 3. 改造后核心架构

改造后 SDK 的核心结构建议如下：

```text
agent/
  Agent runtime
  请求生命周期
  模型调用循环
  工具调用循环
  取消、摘要、标题生成

session/
  Session 实体
  Session service

message/
  Message 实体
  ContentPart
  ToolCall / ToolResult message
  Message service

history/
  文件历史
  文件快照
  版本记录

permission/
  工具权限
  授权请求
  会话级权限状态

prompt/
  薄 prompt 入口
  从 YAML/JSON 配置读取 system prompt
  按 key 获取对应 system prompt

llm/
  provider/
    Provider 接口
    OpenAI/Anthropic/Gemini/Azure 等实现
  models/
    保留最小模型描述结构
    不维护庞大的内置支持模型表

tools/
  core/
    Tool 接口
    ToolInfo
    ToolCall
    ToolResponse
    Tool registry
    Hook/Event 定义
  base/
    ls
    glob
    grep
    view
    edit
    write
    patch
    bash
    fetch
  mcp/
    MCP server tool -> SDK Tool

data/
  db/
    GORM 数据源
    migration
    transaction
  repo/
    SessionRepo
    MessageRepo
    HistoryRepo

utils/
  fileutil/
    文件发现
    路径过滤
    rg/fzf 命令构造
    glob fallback

logging/
  日志

config/
pubsub/
```

### 3.1 改造后模块关系

```text
SDK Caller
  -> agent.Service
    -> session.Service
    -> message.Service
    -> history.Service
    -> llm/provider.Provider
    -> tools/core.Tool
      -> tools/base
      -> tools/mcp
    -> permission.Service
    -> pubsub events

session/message/history service
  -> data/repo
    -> data/db

tools/base
  -> utils/fileutil
  -> utils/diff
  -> permission
  -> history
  -> tools/core hooks

LSP / completions
  -> 不进入 SDK
  -> 由 CLI/IDE 宿主项目自行实现并按需封装成 Tool/Hook
```

## 4. 关键设计决策

### 4.1 SDK 中不保留 LSP 与 extensions

改造后 `tools` 下只保留 SDK 原生工具来源：

```text
tools/core
tools/base
tools/mcp
```

LSP 不作为 SDK 内置工具来源，也不作为 SDK 底层 extensions 保留。原因是：

- LSP 是代码场景增强能力，不是 Agent SDK 必需能力。
- LSP server 启动、语言选择、项目监听、诊断展示更贴近 CLI 或 IDE 宿主。
- LSP client、协议、watcher、diagnostics 与输入补全都更适合作为宿主能力维护。

后续 CLI 可以创建：

```text
cli/tools/lsp/diagnostics
cli/hooks/lsp_diagnostics
```

并将它们注册进 SDK 的 toolset 或 hook registry。

### 4.2 `edit/write/patch` 只保留基础能力

基础文件修改工具只负责文件操作本身：

```text
edit
  精确替换已有内容

write
  创建或覆盖文件

patch
  应用 patch
```

它们应保留：

- 参数校验
- 路径处理
- 权限检查
- 文件读写
- history 记录
- diff 摘要
- ToolResponse 返回

它们不直接负责：

- LSP diagnostics
- format on save
- lint
- test trigger
- embedding index refresh
- git status refresh

这些能力通过 hook 扩展。

### 4.3 Hook/Event 作为扩展点

在 `tools/core` 中提供文件事件扩展机制：

```go
type FileEventType string

const (
    FileViewed  FileEventType = "file.viewed"
    FileEdited  FileEventType = "file.edited"
    FileWritten FileEventType = "file.written"
    FilePatched FileEventType = "file.patched"
    FileDeleted FileEventType = "file.deleted"
)

type FileEvent struct {
    Type       FileEventType
    Path       string
    OldContent string
    NewContent string
    Diff       string
    SessionID  string
    ToolCallID string
}

type FileHook interface {
    OnFileEvent(ctx context.Context, event FileEvent) (*HookResult, error)
}
```

基础工具执行流程：

```text
1. 执行基础工具逻辑
2. 记录 history
3. 发布 FileEvent
4. 收集 HookResult
5. 合并到 ToolResponse
```

这样 SDK 核心不依赖 LSP；CLI/IDE 宿主如需 diagnostics，可自行实现并通过 hook 注入。

### 4.4 MCP 作为外部工具来源

`MCP` 属于 `tools` 下的外部工具来源。

```text
tools/base
  内置 Go 工具

tools/mcp
  MCP server 暴露的远程工具
```

Agent 只认识 `tools/core.Tool`，不关心工具来自哪里。

## 5. 改造前后对比

| 维度 | 改造前 | 改造后 |
|------|--------|--------|
| SDK 定位 | 混有 CLI/TUI 能力的 Agent runtime | 可嵌入式 Agent SDK |
| CLI 能力 | 存在 spinner、theme、output format、completion | 迁出到独立 CLI 项目 |
| tools 结构 | `base/mcp/agent` | `core/base/mcp` |
| LSP | base tools 直接依赖 LSP，且仓库包含 `extensions/lsp` | SDK 不保留 LSP 与 `extensions`，相关能力由宿主创建 |
| 文件工具 | 文件操作后直接做 LSP diagnostics | 文件操作只做基础逻辑，扩展走 hook |
| diff | 算法和终端渲染混合 | SDK 只保留纯 diff/patch |
| fileutil | `infra/fileutil` | `utils/fileutil` |
| db | `infra/db`，偏底层查询实现 | `data/db + data/repo` |
| logging | `infra/logging` | 顶层 `logging` |
| model | 内置大量 SupportedModels | 用户配置 model，provider 请求时验证 |
| prompt | 较厚的系统提示词构造 | 配置化 prompt，`prompt.go` 按 key 读取 YAML/JSON 中的 system prompt |
| memory | 独立聚合层 | 由 session/message/history 关系表达上下文 |

## 6. 分阶段实施计划

### Phase 1: 删除或迁出 CLI-only 模块

目标：

- 删除 `infra/format`
- 删除或迁出 `infra/theme`
- 删除 `extensions`

注意：

- 删除 `theme` 前先确认 diff 核心能力不再依赖终端颜色。
- 补全能力由未来 CLI 项目实现。

验收：

- SDK 不包含 spinner。
- SDK 不包含 CLI 输出格式处理。
- SDK 不包含输入补全 provider。
- `go test ./...` 通过。

### Phase 2: 拆分 diff 展示与核心算法

目标：

- 保留 diff/patch 核心能力。
- 移除 diff 对 theme/lipgloss 的依赖。
- 保持 diff/patch 核心能力在现有基础包内，不迁入 `utils` 目录。

验收：

- `edit/write/patch` 仍能生成变更摘要。
- SDK 不包含彩色终端 diff renderer。

### Phase 3: 基础工具与 LSP / extensions 解耦

目标：

- `tools/base` 移除 `lspClients` 字段。
- `view/edit/write/patch` 不再 import `extensions/lsp`。
- 删除 `tools/base/diagnostics.go` 或迁出到 CLI 项目。
- 删除 `extensions/lsp`、`extensions/completions` 以及整个 `extensions` 模块。

验收：

- `tools/base` 与 LSP 无直接依赖。
- 无 LSP 时基础工具完整可用。
- SDK 目录中不再存在 `extensions` 模块。

### Phase 4: 引入 Tool Hook/Event

目标：

- 在 `tools/core` 中定义 hook/event。
- `edit/write/patch/view` 在完成基础逻辑后发布事件。
- Toolset 支持注册 hooks。

验收：

- 外部宿主可以注入自定义 hook。
- hook 结果可以合并进 ToolResponse。
- SDK 内部不需要知道 hook 的具体能力来源。

### Phase 5: Data/Repo 改造

目标：

- 新建 `data/db`。
- 新建 `data/repo`。
- service 依赖 repo，不直接依赖底层查询。
- 使用 GORM 作为数据源操作方式。
- 删除独立 memory 聚合层。

验收：

- session/message/history service 通过 repo 访问数据。
- 后续可替换数据源。
- 上下文仍由 session 下的 message 和 history 表达。

### Phase 6: LLM 与 Prompt 简化

目标：

- 弱化或移除内置 SupportedModels。
- provider 使用用户配置的 model 字符串。
- 移除 SDK 对“支持模型列表”的展示职责。
- prompt 从 YAML/JSON 配置文件读取 system prompt，并按 key 获取对应提示词。
- 删除 `prompt/coder.go`、`prompt/title.go`、`prompt/task.go`、`prompt/summarizer.go` 等硬编码 prompt 源文件。

验收：

- SDK 不需要跟随模型发布频繁更新。
- 调用方可以通过 YAML/JSON 配置和 prompt key 控制系统提示词。

### Phase 7: 目录收敛

目标迁移：

```text
infra/fileutil -> utils/fileutil
infra/db       -> data/db
infra/logging  -> logging
```

验收：

- 目录命名与职责一致。
- 架构文档同步更新。
- `go test ./...` 通过。

## 7. 最终交付状态

改造完成后，SDK 应达到：

- 核心运行时清晰。
- `tools/core/base/mcp` 边界稳定。
- LSP 与 `extensions` 不进入 SDK。
- CLI/TUI 能力全部迁出。
- 文件工具通过 hook 扩展后续能力。
- 数据层具备 repo 抽象。
- LLM 和 Prompt 更薄、更可配置。
- SDK 可被独立 CLI、API Server、IDE 插件或其他宿主复用。

最终边界一句话：

```text
Agent SDK 负责能力运行和编排；CLI 负责用户交互和展示；扩展负责可选增强能力。
```
