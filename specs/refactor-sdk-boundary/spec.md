# 特性规格说明：Agent SDK 边界改造

**Feature Branch**: `refactor-sdk-boundary`  
**Created**: 2026-05-06  
**Status**: Ready  
**Input**: 用户要求阅读 `AGENTSDK改造计划.md`，明确本次改造需求；补充边界：本次不做 Skill，基础 diff 能力需要保留，且 diff 不迁入 `utils`。

---

## 需求描述

### 背景与动机

当前项目更接近一个从应用中抽取出来的 Agent runtime，Agent 编排、基础工具、CLI/TUI 展示、LSP 增强、数据访问和提示词策略之间存在交叉。改造目标是把项目整理为可被外部宿主复用的 Agent SDK，使 SDK 聚焦能力运行和编排，把终端交互、展示渲染、LSP 能力和可选增强能力从 SDK 中分离出去。

本次需求明确排除 Skill 机制建设。基础 diff 能力属于文件工具运行与审计所需能力，需要保留；应移除的是终端彩色 diff 展示和 theme/lipgloss 绑定，而不是 `edit/write/patch` 所需的纯 diff/patch 能力。

### 目标用户

- **SDK 调用方**：在 CLI、API Server、IDE 插件或其他宿主中复用 Agent SDK 的开发者。
- **SDK 维护者**：需要维护 Agent 编排、工具系统、会话持久化、LLM provider 和扩展边界的工程人员。
- **未来 CLI/IDE 宿主维护者**：需要在 SDK 之外自行接入主题、补全、spinner、LSP diagnostics 等交互或增强能力的工程人员。

### 核心需求

- **RQ-001**: SDK 必须只承担 Agent 运行、工具调用编排、会话/消息/历史/权限管理、LLM Provider 抽象、MCP Tool 接入和 hook/event 扩展机制等核心职责。
- **RQ-002**: SDK 必须移除或迁出 CLI-only 能力，包括 spinner、终端主题、CLI 输出格式、输入补全 provider 和终端彩色 diff 渲染。
- **RQ-003**: SDK 默认工具体系必须保持在 `tools/core`、`tools/base`、`tools/mcp` 三类边界内，本次不得新增或实现 `tools/skill` 或 Skill Tool 机制。
- **RQ-004**: SDK 不得保留 `extensions` 模块；LSP 不得作为 SDK 默认内置工具来源或底层扩展能力，基础工具不得直接依赖 LSP diagnostics。
- **RQ-005**: `edit/write/patch/view` 等基础文件工具必须只负责基础文件操作、参数校验、路径处理、权限检查、history 记录、ToolResponse 返回和必要的变更摘要。
- **RQ-006**: 基础 diff/patch 能力必须保留，保证 `edit/write/patch` 能继续生成变更摘要、权限审批 diff、返回 diff metadata，并支持 patch 解析与应用。
- **RQ-007**: SDK 不得保留面向终端展示的彩色 diff renderer；diff 核心能力不得依赖 theme/lipgloss 等 UI 样式能力。
- **RQ-008**: diff/patch 核心能力应迁入 `utils/diff`，不再保留 `infra/diff` 或 `infra` 目录。
- **RQ-009**: 工具系统必须提供可扩展 hook/event 边界，使外部宿主可以在文件查看、编辑、写入、patch 等事件后自行接入 LSP diagnostics、format、lint、test trigger、索引刷新等可选增强能力。
- **RQ-010**: 数据访问层需要完成 repo 抽象和底层数据源替换，使 session/message/history 能通过稳定的数据访问边界工作，并为后续替换数据源提供空间。
- **RQ-011**: LLM model/provider 边界需要简化，SDK 不应承担维护庞大内置 SupportedModels 展示表的职责，调用方应能通过配置选择模型。
- **RQ-012**: prompt 能力必须改为配置驱动，SDK 从 YAML 或 JSON 配置文件读取系统提示词，并通过 key 获取对应 system prompt 后传递给模型。
- **RQ-016**: SDK 不得继续使用 `coder.go`、`title.go`、`task.go`、`summarizer.go` 等硬编码 prompt 源文件；`prompt.go` 只负责加载配置、按 key 解析和返回系统提示词。
- **RQ-013**: 改造完成后，架构文档必须与实际目录、模块职责和边界保持一致。
- **RQ-014**: 每个阶段完成后必须保证现有核心能力可运行，且 `go test ./...` 通过。
- **RQ-015**: Hook/Event 机制必须至少覆盖 `view/edit/write/patch` 四类文件事件。

### 关键实体

- **Agent SDK**: 可被外部宿主复用的 Agent runtime，负责运行、编排、工具调用、会话状态和扩展事件。
- **SDK Caller / Host**: 调用 SDK 的外部项目，例如 CLI、API Server、IDE 插件或自动化服务。
- **Tool**: 模型可调用的能力单元，按统一协议暴露信息并执行请求。
- **Base Tool**: SDK 内置基础工具，如文件查看、编辑、写入、patch、bash、fetch、grep、glob、ls 等。
- **MCP Tool**: 由配置中的 MCP server 暴露并适配为 SDK Tool 的外部工具。
- **File Event / Hook**: 文件工具完成基础动作后产生的事件及外部扩展处理点。
- **Diff/Patch Core**: 文件工具需要的纯变更摘要、增删行统计、patch 解析和 patch 应用能力。
- **Session / Message / History**: 表达会话上下文、消息内容和文件版本历史的核心数据实体。
- **Data Repo**: 面向 service 的数据访问抽象，用于隔离业务服务与底层数据源。
- **Prompt Config**: YAML 或 JSON 格式的系统提示词配置，按 key 保存 coder、title、task、summarizer 等提示词内容。
- **Prompt Resolver**: `prompt.go` 中的提示词读取入口，负责加载配置并按 key 返回 system prompt。

### 约束与边界

- 本次改造不建设 Skill manifest、Skill loader、Skill registry、Skill Tool adapter，也不把 Skill 作为默认工具来源。
- 基础 diff/patch 能力保留；不得因为删除终端 diff renderer 或迁移目录而破坏 `edit/write/patch` 的变更摘要、权限审批或响应 metadata。
- `utils/diff` 作为 diff/patch core 的目标目录；迁移后不得保留 `infra/diff`。
- CLI/TUI 交互能力不属于 SDK 核心交付物；未来 CLI 项目可独立实现并注册所需 Tool 或 Hook。
- `infra/format`、`infra/theme` 和整个 `extensions` 模块必须直接从 SDK 仓库删除，由未来 CLI/IDE 宿主项目自行承接相关能力。
- 本次允许破坏性 API 变更，以清晰 SDK 边界为优先。
- LSP diagnostics 不属于 SDK 能力，不得阻塞基础文件工具可用性。
- Hook/Event 结果可以增强 ToolResponse，但 SDK 核心不得依赖具体 hook 来源或具体宿主能力。
- 系统提示词必须来自 YAML/JSON 配置，不得再通过 Go 源码常量或按文件硬编码维护。
- 架构文档只记录已落地事实；规划性内容应在实现后再同步为事实描述。
- 本规格只定义需求，不规定具体代码迁移顺序、包名细节、数据库 schema 或 API 设计。

---

## 待澄清问题

### CQ-001: CLI-only 模块的处理方式

> **类别**: 功能范围与行为

**Q (提问)**:
`infra/format`、`infra/theme`、`extensions` 这类 CLI/IDE-only 模块在 SDK 仓库内应如何处理？该问题会影响代码删除范围、兼容性和后续 CLI/IDE 项目的迁移成本。  
*参考选项*:
- A. 直接从 SDK 仓库删除，由未来 CLI 项目重新承接。
- B. 移到独立目录并明确标记为非 SDK 默认能力，待 CLI 项目迁出后删除。
- C. 暂时保留代码，但从 SDK 默认构建路径和公共 API 中移除。

**A (澄清结论)**:
选择 A。`infra/format`、`infra/theme` 和整个 `extensions` 模块直接从 SDK 仓库删除，由未来 CLI/IDE 宿主项目自行承接。

### CQ-002: 数据层改造的目标深度

> **类别**: 领域与数据模型

**Q (提问)**:
Data/Repo 改造是否必须在本次改造中完成到底层数据源替换，还是只需要先建立 repo 抽象边界？该问题会影响工作量、迁移风险和验收标准。  
*参考选项*:
- A. 本次完成 repo 抽象，并继续沿用现有底层数据源。
- B. 本次同时完成 repo 抽象和底层数据源替换。
- C. 本次只明确需求与边界，数据层实现后续单独推进。

**A (澄清结论)**:
选择 B。本次同时完成 repo 抽象和底层数据源替换。

### CQ-003: 公共 API 兼容性要求

> **类别**: 边界与失败处理

**Q (提问)**:
改造期间是否需要保持现有外部可见 Go API 的兼容性？该问题会影响包迁移方式、废弃策略和调用方升级成本。  
*参考选项*:
- A. 尽量保持兼容，必要时提供过渡封装或 deprecated 路径。
- B. 允许破坏性变更，以清晰 SDK 边界为优先。
- C. 只保证核心 Agent 运行入口兼容，工具和 infra 包允许调整。

**A (澄清结论)**:
选择 B。允许破坏性变更，以清晰 SDK 边界为优先。

### CQ-004: Hook/Event 的最小验收范围

> **类别**: 完成信号

**Q (提问)**:
Hook/Event 机制本次至少需要覆盖哪些文件工具事件才算完成？该问题会影响 Phase 4 的任务拆解和验收口径。  
*参考选项*:
- A. 覆盖 `view/edit/write/patch` 四类文件事件。
- B. 只覆盖会产生文件变更的 `edit/write/patch`。
- C. 先定义协议和注册机制，具体工具接入后续推进。

**A (澄清结论)**:
选择 A。Hook/Event 机制本次至少覆盖 `view/edit/write/patch` 四类文件事件。
