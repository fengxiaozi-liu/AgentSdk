# 实施计划：Agent SDK 边界改造

**Feature Branch**: `refactor-sdk-boundary`  
**Created**: 2026-05-06  
**Status**: Ready  
**Spec**: `specs/refactor-sdk-boundary/spec.md`

---

## 摘要

本次改造把当前混有 CLI/TUI、LSP 工具增强、终端 diff 展示和底层数据访问细节的 Agent runtime，收敛为可被外部宿主复用的 Agent SDK。核心设计方向是：SDK 只保留 Agent 编排、会话/消息/history/权限、LLM provider、基础工具、MCP 工具和 hook/event 扩展边界；直接删除 CLI-only 模块和整个 `extensions` 模块；基础 diff/patch 能力保留但剥离终端展示；基础文件工具与 LSP 解耦；Data/Repo 同时完成 repo 抽象和底层数据源替换；允许破坏性 API 变更，优先获得清晰边界。

明确不做 `tools/skill`、Skill Tool、`utils/diff`。Hook/Event 本次最小覆盖 `view/edit/write/patch`。

## 项目适配输入

- 项目类型: Go SDK / Agent Runtime
- 结构约束: 单模块 Go 工程，模块名 `ferryman-agent`；当前核心目录包括 `agent.go`、`tools/*`、`infra/*`、`extensions/*`、`session`、`message`、`history`、`llm`、`prompt`、`config`，目标结构不再包含 `extensions`。
- 交付物类型: 代码 / 协议 / 配置 / 文档
- 特殊验证要求: 每阶段运行 `go test ./...`；涉及数据层替换时补充 repo/service 级测试；涉及工具行为时验证 `edit/write/patch/view` 基础能力和 diff metadata 不回退。

---

## Phase 0：调研与关键决策

### 实体与关系分析

| 实体 | 关键属性 | 与其他实体关系 | 来源 RQ |
|------|---------|---------------|---------|
| Agent SDK | Run/Cancel/Summarize/Update、toolset、provider、pubsub | 调用 session/message/history、provider 和 tools；对外暴露可复用运行时 | RQ-001 |
| Base Tool | Info、Run、权限、文件操作、ToolResponse metadata | 由 Agent 调用；依赖 permission、history、diff core；不得依赖 LSP diagnostics | RQ-004, RQ-005, RQ-006 |
| MCP Tool | MCP server 配置、远端工具 schema、权限请求 | 作为外部工具来源适配为 SDK Tool | RQ-001, RQ-003 |
| File Event / Hook | event type、path、old/new content、diff、session/tool call id | 由 `view/edit/write/patch` 发布；外部宿主注册 hook 消费 | RQ-009, RQ-015 |
| Diff/Patch Core | unified diff、增删行统计、patch 解析、patch 应用 | 被 `edit/write/patch` 使用；不得依赖 theme/lipgloss；不迁入 `utils/diff` | RQ-006, RQ-007, RQ-008 |
| Data Repo | SessionRepo、MessageRepo、HistoryRepo、transaction | service 通过 repo 访问新数据源；替代直接 sqlc 查询依赖 | RQ-010 |
| Prompt Config | YAML/JSON 配置、prompt key、system prompt 内容 | Agent/provider 使用；由 `prompt.go` 按 key 读取并返回 | RQ-012, RQ-016 |

### 状态流转（若适用）

| 实体 | 状态列表 | 流转规则 | 来源 RQ |
|------|---------|---------|---------|
| File Tool Execution | validate -> permission -> mutate/read -> history -> event -> response | 基础动作成功后发布 FileEvent；hook 失败不得破坏已完成的基础文件操作，除非后续设计明确需要强制失败策略 | RQ-005, RQ-009, RQ-015 |
| Data Access | direct sqlc -> repo boundary -> new datasource | 先建立 repo contract，再迁移 service 依赖，最后移除旧 `infra/db` 直接访问链路 | RQ-010 |
| Diff Capability | mixed renderer/core -> pure core + no terminal renderer | 保留生成与 patch 能力，删除彩色展示 API 和 theme/lipgloss 依赖 | RQ-006, RQ-007 |
| Tool Boundary | base + mcp + agent/lsp mixed -> core/base/mcp | 移除 LSP diagnostics tool 默认来源和整个 `extensions` 模块；子 Agent 工具若保留，需仅组合 SDK 基础工具和外部注入能力 | RQ-003, RQ-004 |

### 依赖调研

| 依赖对象 | 用途 | 已有能力 | 需要新增或修改 |
|----------|------|----------|----------------|
| `infra/format` | CLI 输出格式、spinner | SDK 仓库内存在 | 直接删除，并清理所有引用 |
| `infra/theme` | 终端主题、diff/markdown/syntax 颜色 | 被 `infra/diff` 展示逻辑引用 | 删除前先切断 diff core 对 theme/lipgloss 的依赖 |
| `extensions` | LSP client、protocol、watcher、输入补全 | SDK 仓库内存在 | 整体删除，并清理配置、工具和导出引用 |
| `infra/diff` | diff/patch core + side-by-side renderer | `GenerateDiff`、patch 解析/应用、彩色 renderer 混在一起 | 保留 core，删除 renderer、theme/lipgloss 引用，不迁 `utils/diff` |
| `tools/base` | SDK 基础工具 | `view/edit/write/patch/diagnostics` 依赖 LSP client | 删除 diagnostics 默认工具；文件工具改为 hook/event 后置扩展 |
| `tools/core` | 工具协议 | BaseTool、ToolCall、ToolResponse | 新增 FileEvent、HookResult、FileHook、registry/dispatcher 设计 |
| `infra/db` + sqlc/goose | SQLite 连接、migration、查询代码 | service 直接或间接使用 sqlc 查询 | 新建 `data/db`、`data/repo`，引入新数据源并迁移 service |
| `llm/models.SupportedModels` | 模型列表与能力元信息 | config/agent 多处依赖 | 缩小为最小模型描述或运行时配置验证 |
| `prompt` | 硬编码 agent prompt 文件 | prompt 与 provider/model/provider type 耦合，且提示词散落在 `coder.go`、`title.go` 等源码中 | 改为 YAML/JSON 配置驱动；`prompt.go` 按 key 读取系统提示词，删除硬编码 prompt 文件 |

### 调研结论

- **DR-001**: CLI/IDE-only 模块直接删除
  - 决策: 删除 `infra/format`、`infra/theme` 和整个 `extensions` 模块。
  - 理由: 规格已澄清选择直接删除，SDK 不承担 CLI/TUI 展示、输入交互或 LSP 能力。
  - 排除方案: 移到非默认目录或暂时保留，这会延长边界模糊状态。

- **DR-002**: diff 采用“保留 core，删除 renderer”
  - 决策: `GenerateDiff`、增删统计、patch 解析和 patch 应用保留；side-by-side 彩色渲染、theme/lipgloss 依赖删除。
  - 理由: `edit/write/patch` 的权限审批、metadata 和审计仍需要基础 diff。
  - 排除方案: 删除整个 diff 包，或迁入 `utils/diff`。

- **DR-003**: LSP 完全移出 SDK
  - 决策: 删除 `extensions/lsp` 及相关 LSP client/protocol/watcher 代码；`tools/base` 不直接 import LSP；diagnostics 工具从 SDK 默认工具中删除。
  - 理由: SDK 不再包含 `extensions` 部分，LSP 由未来 CLI/IDE 宿主自行实现和接入。
  - 排除方案: 保留当前 `lspClients` 注入方式，或只在 nil 时跳过 diagnostics。

- **DR-004**: Data/Repo 本次做完整替换
  - 决策: 新建 repo 抽象并完成底层数据源替换，之后 service 不再依赖旧查询层。
  - 理由: CQ 已澄清选择完整完成，避免只加抽象但不改变实际依赖。
  - 排除方案: 仅包一层 repo 继续使用旧 sqlc/goose。

- **DR-005**: API 可破坏，边界优先
  - 决策: 允许包路径、构造函数、配置结构和工具初始化签名发生破坏性变化。
  - 理由: 规格要求清晰 SDK 边界优先。
  - 排除方案: 保留兼容 shim 或 deprecated 路径。

---

## Phase 1：技术设计

### 模块设计

| 模块 | 职责 | 主要输入 | 主要输出 | 来源 RQ |
|------|------|---------|---------|---------|
| `tools/core` | 定义 Tool 协议、FileEvent、HookResult、FileHook、hook 注册与调度边界 | 工具执行上下文、文件事件、hook 列表 | ToolResponse、hook 增强结果 | RQ-001, RQ-009, RQ-015 |
| `tools/base` | 提供无 LSP 直接依赖的基础工具；负责基础文件操作和 diff metadata | ToolCall、permission、history、diff core | ToolResponse、FileEvent | RQ-004, RQ-005, RQ-006 |
| `tools/mcp` | 适配 MCP server 工具为 SDK Tool | MCP 配置、权限服务、远端 schema | BaseTool 列表、MCP 调用响应 | RQ-001, RQ-003 |
| `infra/diff` 或后续同等 core 包 | 提供纯 diff/patch 能力 | before/after content、patch text、file path | unified diff、additions/removals、patch commit | RQ-006, RQ-007, RQ-008 |
| `data/db` | 新数据源连接、migration、transaction | config data directory、migration | datasource handle、transaction context | RQ-010 |
| `data/repo` | SessionRepo、MessageRepo、HistoryRepo 抽象和实现 | domain entity、query params、transaction | 持久化实体、列表、更新结果 | RQ-010 |
| `session/message/history` | 通过 repo 完成业务实体服务 | repo、pubsub、业务参数 | Session/Message/File 业务结果 | RQ-010 |
| `llm/models` | 最小模型描述，不维护庞大支持列表展示职责 | model string、provider config | model identity/capability minimum | RQ-011 |
| `prompt` | 配置驱动 prompt resolver | prompt config path、prompt key、默认配置 | system prompt string | RQ-012, RQ-016 |
| `docs/architecture.md` | 记录落地后的事实架构 | 实际代码结构、模块职责 | 同步后的架构文档 | RQ-013 |

### 协议 / 接口设计（若适用）

| 服务或模块 | 方法或动作 | 请求关键字段 | 响应关键字段 | 来源 RQ |
|------------|-----------|-------------|-------------|---------|
| `tools/core` | RegisterFileHook / DispatchFileEvent | hook、FileEvent | HookResult 列表、error 列表或合并结果 | RQ-009 |
| `tools/core` | FileHook.OnFileEvent | context、event type、path、old/new content、diff、session id、tool call id | HookResult、error | RQ-009, RQ-015 |
| `tools/base/view` | Run | file path、range/offset 参数 | 文件内容、FileViewed event、hook metadata | RQ-005, RQ-015 |
| `tools/base/edit` | Run | file path、old/new string、content | diff metadata、history version、FileEdited/FileWritten event | RQ-005, RQ-006, RQ-015 |
| `tools/base/write` | Run | file path、content | diff metadata、history version、FileWritten event | RQ-005, RQ-006, RQ-015 |
| `tools/base/patch` | Run | patch text | per-file diff metadata、history version、FilePatched event | RQ-005, RQ-006, RQ-015 |
| `data/repo` | SessionRepo/MessageRepo/HistoryRepo methods | entity id、session id、filters、update params | domain entity、not found/validation errors | RQ-010 |
| `prompt` | ResolveSystemPromptByKey | config path、prompt key、optional fallback | system prompt string 或缺失错误 | RQ-012, RQ-016 |

### 数据或配置设计（若适用）

| 对象 | 字段或配置项 | 类型 | 约束 | 说明 |
|------|--------------|------|------|------|
| FileEvent | Type | string enum | `file.viewed`、`file.edited`、`file.written`、`file.patched`、`file.deleted` | Hook/Event 最小协议 |
| FileEvent | Path | string | 必填，工作目录内路径规则沿用工具层校验 | 事件目标文件 |
| FileEvent | OldContent / NewContent | string | 可空，view 可只填 NewContent 或留空 | 支持 diagnostics/format/index 等 hook |
| FileEvent | Diff | string | 文件变更事件应尽量提供 | 来自 diff core，不是彩色渲染 |
| HookResult | Content / Metadata | string / map | 可空 | 合并到 ToolResponse 或用于事件订阅 |
| DataConfig | Directory / DSN 等 | string | 默认值需兼容现有数据目录策略 | 新数据源连接参数，具体字段由实现阶段确定 |
| Prompt Config | prompts | map[string]string | key 必须唯一；缺失 key 返回清晰错误 | YAML/JSON 中的系统提示词集合 |
| Prompt Config | prompt_config_path | string | 可选；未配置时使用 SDK 默认配置文件路径或调用方传入路径 | 提示词配置文件位置 |
| Agent Config | prompt_key | string | 必填或有明确默认值 | Agent 按 key 获取 system prompt |
| Model Config | Provider / Model | string | model 字符串由调用方配置；provider 请求时验证 | 弱化内置 SupportedModels |

### 错误 / 枚举 / 文案（若适用）

| 类型 | 名称 | 说明 | 来源 RQ |
|------|------|------|---------|
| Enum | FileEventType | 文件事件类型，至少覆盖 viewed/edited/written/patched | RQ-009, RQ-015 |
| Error | ErrPermissionDenied | 文件工具权限拒绝沿用现有语义 | RQ-005 |
| Error | ErrHookFailed | hook 失败的可观测错误；是否阻断基础工具由调度策略决定，默认不破坏已完成基础动作 | RQ-009 |
| Error | ErrRepoNotFound | repo 层实体不存在错误，供 service 映射 | RQ-010 |
| Error | ErrInvalidModelConfig | 调用方配置的 provider/model 无法用于请求时返回 | RQ-011 |
| Error | ErrPromptKeyNotFound | prompt 配置中不存在请求的 key | RQ-012, RQ-016 |
| Error | ErrPromptConfigInvalid | YAML/JSON prompt 配置无法解析或结构不合法 | RQ-012, RQ-016 |

---

## 测试与验证

### 测试目标

| 目标 | 层次 | 类型 | 优先级 | 来源 RQ | 说明 |
|------|------|------|--------|---------|------|
| CLI-only 模块删除后无引用残留 | 构建 | Integration | P0 | RQ-002 | `go test ./...` 和 `rg` 检查 import |
| diff core 保留且无 theme/lipgloss 依赖 | 单元 | Unit | P0 | RQ-006, RQ-007 | 覆盖 GenerateDiff、增删统计、patch parse/apply |
| `edit/write/patch` 仍返回 diff metadata | 工具 | Unit/Integration | P0 | RQ-005, RQ-006 | 验证 Diff/Additions/Removals |
| SDK 不保留 `extensions` 模块 | 架构 | Static | P0 | RQ-004 | 删除目录并用 `rg` 静态检查引用 |
| `view/edit/write/patch` 发布 FileEvent | 工具 | Unit/Integration | P0 | RQ-009, RQ-015 | 使用测试 hook 断言事件内容 |
| Data/Repo 替换后 service 行为不回退 | 服务 | Unit/Integration | P0 | RQ-010 | session/message/history CRUD 与 cascade/版本行为 |
| provider/model 配置可使用任意 model 字符串 | 配置/LLM | Unit | P1 | RQ-011 | 验证不依赖庞大 SupportedModels 展示表 |
| prompt 可从 YAML/JSON 按 key 获取 | Agent/Prompt | Unit | P0 | RQ-012, RQ-016 | 验证 JSON/YAML 加载、key 查找、缺失 key 错误 |
| 架构文档同步 | 文档 | Manual | P1 | RQ-013 | 目录和模块职责与代码一致 |

### 验证要求

- `go test ./...` 通过。
- `rg -n "infra/format|infra/theme|extensions/" . -g "*.go"` 不应存在有效代码引用。
- 仓库目标结构不应包含 `extensions` 目录。
- `rg -n "lipgloss|infra/theme" infra/diff` 不应存在。
- `edit/write/patch` 的 diff metadata、权限审批 diff 和 history 记录行为需要有测试或人工验证记录。
- `view/edit/write/patch` 的 FileEvent 发布和 hook 合并行为需要测试覆盖。
- 数据层替换完成后，旧 sqlc/goose 依赖应不再处于 service 运行链路；如删除旧代码，需同步 `go.mod`。
- 更新 `docs/architecture.md`，只记录已经落地的事实。

---

## 风险与边界

| 风险项 | 影响 | 缓解措施 |
|--------|------|---------|
| 删除 `infra/theme` 时误删 diff core 所需能力 | `edit/write/patch` 失去 diff metadata 或 patch 能力 | 先拆 diff renderer/core，再删除 theme；增加 diff core 测试 |
| LSP 完全移出导致文件工具响应内容变化 | 调用方不再从 SDK 基础工具看到 diagnostics 文本 | 基础工具响应只保证基础动作结果；diagnostics 由未来宿主通过自有 Tool/Hook 承接 |
| Data/Repo + 数据源替换范围大 | 容易引入持久化行为回退或迁移失败 | repo contract 先行，service 测试覆盖 CRUD、版本、summary、message parts |
| 允许破坏性 API 变更 | 现有调用方需要同步升级 | 文档明确新入口；删除旧路径后用编译错误暴露迁移点 |
| Weakening SupportedModels 影响默认配置校验 | provider 初始化可能更晚失败 | 将模型验证移动到 provider 请求边界，错误信息清晰返回 |
| Prompt 配置化后 key 缺失或配置错误 | Agent 无法构造 system prompt | 提供默认配置、清晰错误和 prompt resolver 单元测试 |
| Hook 失败处理策略不清晰 | 文件已写入但增强失败时状态不一致 | 默认 hook 作为增强，不回滚基础文件操作；在 ToolResponse metadata 中暴露 hook error |
| 文档与实现不同步 | 后续任务依据错误架构推进 | 每个阶段完成后更新架构文档并运行静态检查 |
