# 任务清单：Agent SDK 边界改造

**Spec**: `specs/refactor-sdk-boundary/spec.md`  
**Plan**: `specs/refactor-sdk-boundary/plan.md`  
**Created**: 2026-05-06  
**项目识别结果**: Go SDK / Agent Runtime  
**技术链依赖**: Phase 1 使用 Go 静态检查与测试基线；Phase 2 使用 Go 包依赖清理和 diff core 单元测试；Phase 3 使用 tools/core 协议设计和 tools/base 行为测试；Phase 4 使用数据访问层 repo 设计与新数据源迁移；Phase 5 使用 LLM/provider/config 与 YAML/JSON prompt resolver 边界收敛；Phase 6 使用架构文档与全量验证。

---

## 格式说明

- `[TaskID]`：如 `T001`，全局递增唯一编号
- `[技术域]`：该任务所属模块或架构层，如 `[Tools]` `[Data]` `[Config]` `[Docs]`
- `[RQ-xxx]`：对应需求编号，无法对应时使用临时功能点编号
- 每条任务必须包含精确文件路径或明确产物

---

## Phase 1：基线确认与边界入口清理

- [✅] T001 [Validation] [RQ-014] 记录当前全量测试基线和主要失败点，如有失败需归档到实现备注中 `go test ./...`
- [✅] T002 [Architecture] [RQ-001,RQ-003] 梳理当前 SDK 默认工具入口和构造链，确认 `tools/core`、`tools/base`、`tools/mcp` 的保留边界 `tools/tools.go`, `tools/core/core.go`, `tools/base/*`, `tools/mcp/tools.go`
- [✅] T003 [Cleanup] [RQ-002,RQ-004] 删除 CLI/IDE-only 目录并清理引用：`infra/format`, `infra/theme`, `extensions`
- [✅] T004 [Config] [RQ-002,RQ-004] 清理配置结构中与 theme、TUI、completion、LSP/extensions 直接相关的字段、默认值和初始化逻辑 `config/config_impl.go`, `config/config.go`, `config/init_impl.go`
- [✅] T005 [Build] [RQ-002,RQ-004,RQ-014] 修复删除 CLI/IDE-only 模块后的编译错误，并确保不存在有效 Go 引用 `rg -n "infra/format|infra/theme|extensions/" . -g "*.go"`

## Phase 2：diff core 保留与终端 renderer 移除

- [✅] T006 [Diff] [RQ-006,RQ-007,RQ-008] 拆除 `infra/diff` 中 side-by-side 彩色渲染、theme/lipgloss 相关类型和函数，并将纯 diff/patch core 迁入 `utils/diff/diff.go`, `utils/diff/patch.go`
- [✅] T007 [Diff] [RQ-006,RQ-007,RQ-008] 确认 `GenerateDiff`、增删行统计、patch 解析、patch 应用 API 仍满足 `edit/write/patch` 调用 `utils/diff/*`
- [✅] T008 [Tools] [RQ-006] 修复 `tools/base/edit.go`, `tools/base/write.go`, `tools/base/patch.go` 中因 diff 拆分产生的导入或 API 调整
- [✅] T009 [Test] [RQ-006,RQ-007,RQ-014] 新增或更新 diff core 测试，覆盖 diff 生成、增删统计、patch parse/apply 且断言不依赖 theme/lipgloss `utils/diff/*_test.go`
- [✅] T010 [Validation] [RQ-007,RQ-014] 验证 diff 包中不再出现终端展示依赖 `rg -n "lipgloss|infra/theme|theme\\." utils/diff`

## Phase 3：基础工具与 LSP/extensions 完全解耦

- [✅] T011 [Tools] [RQ-004,RQ-005] 移除 `tools/base` 文件工具中的 `lspClients` 字段、构造参数和 diagnostics 后处理逻辑 `tools/base/view.go`, `tools/base/edit.go`, `tools/base/write.go`, `tools/base/patch.go`
- [✅] T012 [Tools] [RQ-004] 删除 SDK 默认 diagnostics 工具 `tools/base/diagnostics.go`
- [✅] T013 [Tools] [RQ-003,RQ-004] 更新工具集构造，移除 LSP client option、diagnostics 注册和对子 Agent 工具的 LSP 传递 `tools/tools.go`, `tools/agent/tool.go`
- [✅] T014 [Prompt] [RQ-004,RQ-012,RQ-016] 移除 prompt 中与 LSP 信息直接耦合的内容，并准备将硬编码 prompt 文件迁出为 YAML/JSON 配置 `prompt/coder.go`, `prompt/prompt.go`
- [✅] T015 [Test] [RQ-004,RQ-005,RQ-014] 增加或更新基础工具测试，验证无 LSP/extensions 时 `view/edit/write/patch` 可运行 `tools/base/*_test.go`
- [✅] T016 [Validation] [RQ-004,RQ-014] 验证仓库不包含 `extensions` 目录且 Go 代码无 `extensions/` 引用 `rg -n "extensions/" . -g "*.go"`

## Phase 4：Tool Hook/Event 协议与文件工具接入

- [✅] T017 [ToolsCore] [RQ-009,RQ-015] 在 `tools/core` 定义 `FileEventType`、`FileEvent`、`HookResult`、`FileHook` 和 hook dispatcher/registry `tools/core/core.go` 或 `tools/core/hooks.go`
- [✅] T018 [ToolsCore] [RQ-009] 设计 ToolResponse 与 HookResult 合并策略，支持 hook content/metadata/error 合并且不回滚已完成基础文件操作 `tools/core/*`
- [✅] T019 [Tools] [RQ-009,RQ-015] 在 `view` 完成基础读取后发布 `file.viewed` 事件并合并 hook 结果 `tools/base/view.go`
- [✅] T020 [Tools] [RQ-005,RQ-006,RQ-009,RQ-015] 在 `edit` 完成文件写入、history 和 diff metadata 后发布 `file.edited` 或 `file.written` 事件 `tools/base/edit.go`
- [✅] T021 [Tools] [RQ-005,RQ-006,RQ-009,RQ-015] 在 `write` 完成文件写入、history 和 diff metadata 后发布 `file.written` 事件 `tools/base/write.go`
- [✅] T022 [Tools] [RQ-005,RQ-006,RQ-009,RQ-015] 在 `patch` 完成 patch 应用、history 和 per-file diff metadata 后发布 `file.patched` 事件 `tools/base/patch.go`
- [✅] T023 [Tools] [RQ-009,RQ-015] 更新工具构造入口，允许 SDK 调用方注册 hooks 并传入 base tools `tools/tools.go`
- [✅] T024 [Test] [RQ-009,RQ-015,RQ-014] 使用测试 hook 覆盖 `view/edit/write/patch` 四类事件、事件字段和 HookResult 合并行为 `tools/core/*_test.go`, `tools/base/*_test.go`

## Phase 5：Data/Repo 抽象与底层数据源替换

- [✅] T025 [Data] [RQ-010] 建立新数据源入口、迁移入口和事务边界 `data/db`
- [✅] T026 [Data] [RQ-010] 定义 repo contract：`SessionRepo`、`MessageRepo`、`HistoryRepo` 和通用错误语义 `data/repo`
- [✅] T027 [Data] [RQ-010] 实现 session repo，覆盖创建、读取、列表、更新、删除、成本/token、summary message 等能力 `data/repo/session*.go`
- [✅] T028 [Data] [RQ-010] 实现 message repo，覆盖 content parts 序列化、列表、创建、更新、删除和 session message count 行为 `data/repo/message*.go`
- [✅] T029 [Data] [RQ-010] 实现 history repo，覆盖文件快照、版本创建、按 session/path 查询和最新版本查询 `data/repo/history*.go`
- [✅] T030 [Service] [RQ-010] 迁移 `session.Service` 到 repo 依赖，移除对旧 `infra/db` 查询层的直接依赖 `session/session.go`
- [✅] T031 [Service] [RQ-010] 迁移 `message.Service` 到 repo 依赖，确保 message parts 行为不变 `message/message.go`, `message/content.go`
- [✅] T032 [Service] [RQ-010] 迁移 `history.Service` 到 repo 依赖，确保文件版本行为不变 `history/history.go`
- [✅] T033 [Integration] [RQ-001,RQ-010] 更新 Agent 和工具初始化所需的数据服务构造链 `agent.go`, `tools/tools.go`
- [✅] T034 [Cleanup] [RQ-010] 移除旧 sqlc/goose 运行链路及不再需要的依赖，必要时删除 `infra/db` 并同步 `go.mod`, `go.sum`
- [✅] T035 [Test] [RQ-010,RQ-014] 增加 repo/service 集成测试，覆盖 session/message/history CRUD、版本、summary、删除一致性 `data/repo/*_test.go`, `session/*_test.go`, `message/*_test.go`, `history/*_test.go`

## Phase 6：LLM model/provider 与配置化 prompt

- [✅] T036 [LLM] [RQ-011] 缩小 `llm/models` 的职责，移除庞大内置 SupportedModels 展示职责，仅保留 provider/model 请求所需最小描述 `llm/models/*`
- [✅] T037 [Config] [RQ-011] 更新配置加载与校验，使调用方可配置任意 model 字符串，并将模型有效性验证推迟到 provider 请求边界 `config/config_impl.go`
- [✅] T038 [Agent] [RQ-011,RQ-012] 更新 Agent provider 初始化逻辑，避免强依赖 `models.SupportedModels` 查表 `agent.go`
- [✅] T039 [Prompt] [RQ-012,RQ-016] 新增 YAML/JSON prompt 配置文件结构，沉淀 coder、title、task、summarizer 等 system prompt key `prompt/*.yaml`, `prompt/*.json` 或项目约定配置路径
- [✅] T040 [Prompt] [RQ-012,RQ-016] 重写 `prompt.go`，只负责加载 YAML/JSON 配置、按 key 获取 system prompt、返回缺失 key 或配置错误 `prompt/prompt.go`
- [✅] T041 [Prompt] [RQ-012,RQ-016] 删除硬编码 prompt 源文件并迁移引用：`prompt/coder.go`, `prompt/title.go`, `prompt/task.go`, `prompt/summarizer.go`
- [✅] T042 [Agent] [RQ-012,RQ-016] 更新 Agent/provider 初始化逻辑，按配置中的 prompt key 获取 system prompt 后传递给 provider `agent.go`, `config/*`
- [✅] T043 [Test] [RQ-011,RQ-012,RQ-014,RQ-016] 增加配置、Agent 初始化和 prompt resolver 测试，覆盖任意 model 字符串、JSON/YAML 加载、key 查找和缺失 key 错误 `config/*_test.go`, `prompt/*_test.go`, `agent*_test.go`

## Phase 7：目录收敛、依赖清理与文档同步

- [✅] T044 [FileUtil] [RQ-001,RQ-013] 将 `infra/fileutil` 收敛到 `utils/fileutil`，并将 diff core 收敛到 `utils/diff`，更新所有引用 `infra/fileutil`, `utils/fileutil`, `utils/diff`
- [✅] T045 [Logging] [RQ-001,RQ-013] 将 `infra/logging` 收敛到顶层 `logging`，更新所有引用 `infra/logging`, `logging`
- [✅] T046 [Cleanup] [RQ-002,RQ-004,RQ-007,RQ-010,RQ-016] 清理 `go.mod`/`go.sum` 中因删除 theme、format、extensions、旧 DB 链路和硬编码 prompt 文件而不再需要的依赖 `go.mod`, `go.sum`
- [✅] T047 [Docs] [RQ-013,RQ-016] 更新架构文档，只记录落地事实：无 `extensions`、无 CLI/TUI 模块、diff core 保留、tools/core/base/mcp 边界、data/repo 新结构、prompt 配置化 `docs/architecture.md`
- [✅] T048 [Docs] [RQ-013,RQ-016] 同步 README 或项目入口说明，说明 SDK 职责、宿主职责、不包含 Skill/extensions/CLI UI 的边界，以及 YAML/JSON prompt 配置方式 `README.md`
- [✅] T049 [Validation] [RQ-002,RQ-004,RQ-007,RQ-008,RQ-014,RQ-016] 执行静态验收命令并记录结果：`rg -n "infra/format|infra/theme|extensions/" . -g "*.go"`、`rg -n "lipgloss|infra/theme" utils/diff`、确认不存在 `infra/diff`、确认不存在硬编码 prompt 源文件
- [✅] T050 [Validation] [RQ-014] 运行全量测试并修复失败 `go test ./...`
- [✅] T051 [Traceability] [RQ-001,RQ-016] 做需求覆盖复核，确认 RQ-001 至 RQ-016 均有实现、测试或文档闭环 `specs/refactor-sdk-boundary/spec.md`, `specs/refactor-sdk-boundary/plan.md`, `specs/refactor-sdk-boundary/tasks.md`



