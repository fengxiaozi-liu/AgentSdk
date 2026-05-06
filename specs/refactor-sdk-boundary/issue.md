# 问题修复文档：Agent SDK 边界改造

**Feature Branch**: `master`
**Created**: 2026-05-06
**Input**: 用户问题：`"1. db应该使用gorm而不是自定义source；2. memory下的Repositories 不需要；3. repo下应该建立session.go、message.go、history.go来定义实现"`

---

### 用户问题

> 1. db应该使用gorm而不是自定义source
> 2. memory下的Repositories 不需要
> 3. repo下应该建立session.go、message.go、history.go来定义实现
> 4. infra下的diff文件夹应该迁移到utils中，不需要infra文件夹了
> 5. tools/base/internal 下的支持文件应放入 utils，file_state 能够与 utils 下的 file 合并
> 6. 引入 gorm 定义数据库，需要 database config 结构体并支持 sqlite/mysql 等 type
> 7. 扫描项目中是否还含有 .opencode/opencode/OpenCode/OPENCODE 字样，统一替换为 .ferryer/ferryer/Ferryer/FERRYER
> 8. llm/models 下模型信息应放入 json 文件加载，models.go 只保留 Model 结构体，且 Model 不需要 Provider 字段
> 9. GetAgentPrompt 不再需要，应删除；prompt 通过 key 获取，不再保留 AgentName 概念
> 10. 作为 Agent SDK，config 应由上游传入；config 包只保留配置 struct 和必要注入/校验，最好收敛为一个 config.go。defaultContextPaths 应移除，ShellConfig 不应作为全局 config 核心字段
> 11. message/attachment.go 内容可以移动到 content.go 下，没有必要游离在外

---

## 全局问题列表

| ISSUE | 问题简述 | 状态 |
|------|----------|------|
| ISSUE-001 | db 层仍使用自定义 Source，未使用 gorm | [✅️] |
| ISSUE-002 | memory.go 中仍保留 Repositories 聚合入口 | [✅️] |
| ISSUE-003 | repo 实现未按 session/message/history 拆分文件 | [✅️] |
| ISSUE-004 | infra/diff 仍未迁移到 utils，infra 目录仍存在 | [✅️] |
| ISSUE-005 | tools/base/internal/support 工具类未收敛到 utils/fileutil | [✅️] |
| ISSUE-006 | 缺少 gorm 数据库配置结构与 sqlite/mysql 类型支持 | [✅️] |
| ISSUE-007 | 项目中仍残留 opencode 品牌与目录命名 | [✅️] |
| ISSUE-008 | llm/models 仍通过 Go 代码硬编码模型清单且 Model 含 Provider 字段 | [✅️] |
| ISSUE-009 | Prompt 获取仍绑定 AgentName 与 GetAgentPrompt 封装 | [✅️] |
| ISSUE-010 | config 包仍承担上游应用配置加载与 CLI 初始化职责 | [✅️] |
| ISSUE-011 | message.Attachment 类型游离在 attachment.go 中 | [✅️] |

---

## ISSUE详情列表

### ISSUE-001: db 层仍使用自定义 Source，未使用 gorm

**问题状态**: `[✅️]`

**问题说明**:
当前数据层实现仍通过自定义 `data/db.Source` 作为底层数据源，与本次数据层改造应替换底层数据源的目标不一致。用户明确要求 db 应该使用 gorm，而不是继续维护自定义 source。

**当前现象**:
`data/db/source.go` 中定义了 `Source`、`Session`、`Message`、`File` 以及 `NewSource()`，并使用 map 与锁模拟数据存储。`data/repo/memory.go` 依赖 `datadb.Source` 完成 session/message/history 的读写。

**期望行为**:
db 层应基于 gorm 定义数据库连接、模型或迁移边界；repo 实现应通过 gorm 数据访问能力完成 session/message/history 的持久化操作，不再依赖自定义 `Source`。

**证据或复现**:
- 查看 `data/db/source.go`，存在 `type Source struct`、`NewSource()` 和 map 存储字段。
- 查看 `data/repo/memory.go`，`NewRepositories(source *datadb.Source)` 和各 repo 结构体均持有 `*datadb.Source`。

**澄清记录**:
- 澄清状态：无需澄清
- 当前阶段：Implement
- 澄清目的：用户已明确要求 gorm 替换自定义 source。
- 澄清结论：使用 gorm 数据源和模型替代 `data/db.Source`。

**根因分析**:
历史实现先以仓库事实中的内存 map source 作为轻量替换方案，偏差来源是早期数据层替换深度不足。

**影响范围**:
- `data/db`, `data/repo`, `session`, `message`, `history` 测试

**修复边界**:
- 允许：引入 gorm、定义数据库模型和连接入口、迁移 repo 实现。
- 不做：不引入业务 schema 之外的新实体。

**决策分析**:
基于用户问题和 CQ-002，使用 gorm 完成 sqlite/mysql 可替换数据源边界。

**修复任务**:
- [✅️] T001 [ISSUE-001] 用 gorm 模型和 `Open` 入口替换自定义 source `data/db/db.go`
- [✅️] T002 [ISSUE-001] repo 测试改用 gorm sqlite 内存库 `data/repo/*_test.go`

### ISSUE-002: memory.go 中仍保留 Repositories 聚合入口

**问题状态**: `[✅️]`

**问题说明**:
用户明确指出 memory 下的 `Repositories` 不需要。当前 `data/repo/memory.go` 中仍定义 `Repositories` 聚合结构和 `NewRepositories` 工厂函数，导致 repo 边界仍带有 memory 实现形态。

**当前现象**:
`data/repo/memory.go` 中定义 `type Repositories struct`，包含 `Sessions`、`Messages`、`History` 三个字段，并通过 `NewRepositories` 返回基于内存 source 的聚合 repo。

**期望行为**:
移除 memory 实现中的 `Repositories` 聚合入口；repo 包应以明确的 session/message/history repo 实现作为主要边界，并由新的 gorm 数据层或更合适的组装位置负责初始化。

**证据或复现**:
- 查看 `data/repo/memory.go`，文件顶部存在 `type Repositories struct` 和 `func NewRepositories(source *datadb.Source) Repositories`。
- 查看 `data/repo/memory_test.go`，测试仍通过 `NewRepositories(datadb.NewSource())` 使用 memory 聚合入口。

**澄清记录**:
- 澄清状态：无需澄清
- 当前阶段：Implement
- 澄清目的：用户明确指出 memory 下的 `Repositories` 不需要。
- 澄清结论：删除 memory 聚合和 `NewRepositories`。

**根因分析**:
历史实现为了快速组装三类 repo 增加聚合结构，偏离了用户期望的直接 repo 边界。

**影响范围**:
- `data/repo`, repo/service 测试构造

**修复边界**:
- 允许：删除 memory 聚合入口并更新测试构造。
- 不做：不新增新的全局 repo 聚合类型替代它。

**决策分析**:
直接暴露 `NewSessionRepo`、`NewMessageRepo`、`NewHistoryRepo` 更符合明确 repo 边界。

**修复任务**:
- [✅️] T003 [ISSUE-002] 删除 `data/repo/memory.go`
- [✅️] T004 [ISSUE-002] 更新测试直接构造具体 repo

### ISSUE-003: repo 实现未按 session/message/history 拆分文件

**问题状态**: `[✅️]`

**问题说明**:
用户要求 repo 下应该建立 `session.go`、`message.go`、`history.go` 来定义实现。当前三个 repo 的实现集中在 `data/repo/memory.go`，文件职责过宽，不利于按领域边界维护。

**当前现象**:
`sessionRepo`、`messageRepo`、`historyRepo` 三类实现以及转换函数都集中在 `data/repo/memory.go` 中，`data/repo` 目录下没有对应的 `session.go`、`message.go`、`history.go` 实现文件。

**期望行为**:
在 `data/repo` 下建立 `session.go`、`message.go`、`history.go`，分别承载 session、message、history 的 repo 实现；共享错误、接口和参数类型可继续保留在合适的公共文件中。

**证据或复现**:
- `rg --files data/repo data/db` 显示当前 repo 实现文件为 `data/repo/repo.go`、`data/repo/memory.go`、`data/repo/memory_test.go`。
- 查看 `data/repo/memory.go`，三类 repo 实现集中在同一个文件。

**澄清记录**:
- 澄清状态：无需澄清
- 当前阶段：Implement
- 澄清目的：用户已指定目标文件名。
- 澄清结论：按 session/message/history 拆分 repo 实现。

**根因分析**:
早期实现把三类 repo 合并进一个内存文件，属于仓库事实驱动的临时集中实现。

**影响范围**:
- `data/repo/session.go`, `data/repo/message.go`, `data/repo/history.go`

**修复边界**:
- 允许：按领域拆分 repo 实现文件。
- 不做：不改变 repo interface 的业务语义。

**决策分析**:
分文件实现能降低 repo 职责耦合，并匹配用户指定结构。

**修复任务**:
- [✅️] T005 [ISSUE-003] 新增 `session.go` 实现 session repo
- [✅️] T006 [ISSUE-003] 新增 `message.go` 实现 message repo
- [✅️] T007 [ISSUE-003] 新增 `history.go` 实现 history repo

### ISSUE-004: infra/diff 仍未迁移到 utils，infra 目录仍存在

**问题状态**: `[✅️]`

**问题说明**:
用户明确指出 `infra` 下的 `diff` 文件夹应该迁移到 `utils` 中，并且不需要继续保留 `infra` 文件夹。当前仓库仍以 `infra/diff` 承载 diff/patch core，和新的目录边界预期不一致。

**当前现象**:
`infra/diff` 目录仍存在，包含 `diff.go`、`patch.go`、`diff_test.go`。基础工具中的 `edit`、`write`、`patch` 仍直接导入 `ferryman-agent/infra/diff`。同时，当前 `spec.md`、`plan.md`、`tasks.md` 中仍有“不迁入 utils/diff”或继续保留 `infra/diff` 的描述，与用户新的调整意见存在冲突。

**期望行为**:
将 diff/patch core 从 `infra/diff` 迁移到 `utils` 下合适的位置，更新所有调用方 import，并移除不再需要的 `infra` 目录。相关 spec、plan、tasks、README、architecture 文档也应同步调整，避免继续要求保留 `infra/diff` 或禁止迁入 utils。

**证据或复现**:
- `rg --files infra tools/base/internal utils` 显示存在 `infra/diff/diff.go`、`infra/diff/patch.go`、`infra/diff/diff_test.go`。
- `rg "infra/diff" -n` 显示 `tools/base/edit.go`、`tools/base/write.go`、`tools/base/patch.go` 仍导入 `ferryman-agent/infra/diff`。
- `specs/refactor-sdk-boundary/spec.md` 的 RQ-008、`plan.md` 和 `tasks.md` 仍包含不迁 `utils/diff` 或保留 `infra/diff` 的约束。

**澄清记录**:
- 澄清状态：已澄清
- 当前阶段：Implement
- 澄清目的：用户新意见覆盖早期 spec 中“不迁入 utils/diff”的约束。
- 澄清结论：迁移到 `utils/diff` 并同步 spec/plan/tasks/docs。

**根因分析**:
偏差来自早期 CQ/plan 对 diff 位置的约束，后续用户明确调整了目录边界。

**影响范围**:
- `utils/diff`, `tools/base/edit.go`, `tools/base/write.go`, `tools/base/patch.go`, docs/spec/plan/tasks

**修复边界**:
- 允许：移动 diff 包并更新 import 与文档。
- 不做：不改变 diff/patch API 行为。

**决策分析**:
当前用户要求不保留 `infra`，因此以 `utils/diff` 作为 core 能力落点。

**修复任务**:
- [✅️] T008 [ISSUE-004] 将 `infra/diff` 移动到 `utils/diff`
- [✅️] T009 [ISSUE-004] 更新基础工具 import 和文档约束

### ISSUE-005: tools/base/internal/support 工具类未收敛到 utils/fileutil

**问题状态**: `[✅️]`

**问题说明**:
用户认为 `tools/base/internal` 下的支持文件本质上也属于工具类，应放入 `utils` 下；其中 `file_state` 能够与 `utils` 下的 file 能力合并。当前支持逻辑仍位于 base tool 的 internal 包内，未形成共享 utilities 边界。

**当前现象**:
`tools/base/internal/support` 下存在 `file_state.go` 和 `shell.go`。其中 `file_state.go` 负责记录文件读写时间，`shell.go` 负责持久 shell 执行支持。`utils/fileutil/fileutil.go` 已存在文件工具类能力，但尚未合并 file state 相关逻辑。

**期望行为**:
将 `tools/base/internal/support` 中可复用的工具类能力迁移或收敛到 `utils` 下合适包中；`file_state` 应与 `utils/fileutil` 或新的文件工具包进行合并，基础工具改为引用新的 utils 边界，减少 `tools/base/internal` 的工具类沉淀。

**证据或复现**:
- `rg --files tools/base/internal utils` 显示 `tools/base/internal/support/file_state.go`、`tools/base/internal/support/shell.go` 与 `utils/fileutil/fileutil.go` 并存。
- `rg "tools/base/internal/support" -n` 显示 `tools/base/bash.go`、`tools/base/edit.go`、`tools/base/view.go`、`tools/base/write.go`、`tools/base/patch.go` 仍依赖 internal support。
- 查看 `tools/base/internal/support/file_state.go`，其中包含 `RecordFileRead`、`GetLastReadTime`、`RecordFileWrite` 等文件状态工具函数。

**澄清记录**:
- 澄清状态：无需澄清
- 当前阶段：Implement
- 澄清目的：用户明确 file_state 可与 utils file 合并。
- 澄清结论：file state 放入 `utils/fileutil`，shell 支持放入 `utils/shell`。

**根因分析**:
工具内部 support 包沉淀了可复用工具函数，来源是早期 base tool 局部实现。

**影响范围**:
- `utils/fileutil`, `utils/shell`, `tools/base/*`

**修复边界**:
- 允许：迁移 support 工具函数并更新引用。
- 不做：不重写 bash tool 执行协议。

**决策分析**:
按职责拆分后，文件状态属于 fileutil，持久 shell 属于 shell utility。

**修复任务**:
- [✅️] T010 [ISSUE-005] 合并 file state 到 `utils/fileutil`
- [✅️] T011 [ISSUE-005] 移动 persistent shell 到 `utils/shell`

### ISSUE-006: 缺少 gorm 数据库配置结构与 sqlite/mysql 类型支持

**问题状态**: `[✅️]`

**问题说明**:
引入 gorm 后需要明确数据库配置边界。当前 `config.Data` 只有 `Directory` 字段，无法表达数据库类型、连接串、sqlite 路径、mysql 连接参数、连接池或迁移策略。用户明确要求需要一个 database config 结构体并定义 type，因为后续可能使用 sqlite，也可能使用 mysql。

**当前现象**:
`config/config_impl.go` 中 `type Data struct` 仅包含 `Directory string`。数据层当前仍是 `data/db/source.go` 的自定义内存 source，尚未形成基于 gorm 的 `DatabaseConfig`、数据库类型枚举或连接初始化入口。

**期望行为**:
在配置层新增数据库配置结构，例如 `DatabaseConfig` 与 `DatabaseType`，并挂载到 `Data` 下。配置应至少支持 `sqlite` 与 `mysql`，并考虑 `dsn`、sqlite `path`、mysql host/port/username/password/database/charset/parseTime/loc，以及 `autoMigrate`、连接池和 gorm 日志等级等可选字段。sqlite 默认路径可基于 `Data.Directory` 推导。

**证据或复现**:
- 查看 `config/config_impl.go`，`type Data struct { Directory string }` 只能表示本地数据根目录。
- 查看 `data/db/source.go`，当前数据源仍为自定义 `Source`，没有 gorm DB 初始化配置。
- 当前 `go.mod` 尚未体现 gorm sqlite/mysql 驱动接入边界。

**澄清记录**:
- 澄清状态：无需澄清
- 当前阶段：Implement
- 澄清目的：用户已明确 sqlite/mysql type 和 database config 需求。
- 澄清结论：新增 `DatabaseConfig` 与 `DatabaseType`。

**根因分析**:
旧配置只表达数据目录，未覆盖数据源连接配置，属于数据层迁移设计缺口。

**影响范围**:
- `config/config.go`, `data/db/db.go`, `go.mod`, `go.sum`

**修复边界**:
- 允许：增加数据库配置字段和 gorm driver。
- 不做：不实现 mysql 集成测试环境。

**决策分析**:
配置层只定义结构并由上游传入，db 层根据结构打开 sqlite/mysql。

**修复任务**:
- [✅️] T012 [ISSUE-006] 定义 `DatabaseConfig` 和 `DatabaseType`
- [✅️] T013 [ISSUE-006] 接入 gorm sqlite/mysql 连接入口

### ISSUE-007: 项目中仍残留 opencode 品牌与目录命名

**问题状态**: `[✅️]`

**问题说明**:
用户要求扫描项目中是否还含有 `.opencode` 字样，并全部替换为 `.ferryer`。实际扫描发现不仅有 `.opencode`，还存在 `opencode`、`OpenCode`、`OPENCODE` 等品牌、配置文件、环境变量、User-Agent、日志文件名和生成文本残留，需要统一替换为 ferryer 命名体系。

**当前现象**:
`config` 中默认数据目录、appName、配置文件名、环境变量前缀和 context path 仍使用 opencode/OpenCode。测试、logging、provider header、fetch/sourcegraph User-Agent、bash 生成签名、fileutil 忽略目录、shell 临时文件名等位置也仍包含 opencode 字样。

**期望行为**:
统一将 `.opencode` 替换为 `.ferryer`，将 `opencode`/`OpenCode`/`OPENCODE` 按语义替换为 `ferryer`/`Ferryer`/`FERRYER`。配置文件名、默认数据目录、环境变量前缀、日志名、临时文件名前缀、User-Agent、测试夹具和文档应保持一致，避免旧品牌残留。

**证据或复现**:
- `rg "\\.opencode|opencode|OpenCode|OPENCODE" -n` 显示命中 `config/config_impl.go`、`config/config_test.go`、`agent_test.go`、`logging/logger.go`、`llm/provider/*`、`tools/base/*`、`utils/fileutil/fileutil.go` 等文件。
- `config/config_impl.go` 中存在 `defaultDataDirectory = ".opencode"`、`appName = "opencode"`、`OPENCODE_DEV_DEBUG`、`OpenCode.md` 等旧命名。

**澄清记录**:
- 澄清状态：无需澄清
- 当前阶段：Implement
- 澄清目的：替换目标命名已明确。
- 澄清结论：代码、测试和文档中的旧品牌残留替换为 ferryer。

**根因分析**:
仓库源自 opencode 代码，品牌字符串残留在配置、日志、User-Agent 和测试中。

**影响范围**:
- `config`, `logging`, `llm/provider`, `tools/base`, `utils`, tests

**修复边界**:
- 允许：替换代码、测试、文档中的旧品牌字符串。
- 不做：不修改第三方协议或外部服务名称。

**决策分析**:
统一品牌能消除 SDK 对旧应用名称的耦合。

**修复任务**:
- [✅️] T014 [ISSUE-007] 替换 `.opencode/opencode/OpenCode/OPENCODE` 残留
- [✅️] T015 [ISSUE-007] 静态扫描确认无旧品牌残留

### ISSUE-008: llm/models 仍通过 Go 代码硬编码模型清单且 Model 含 Provider 字段

**问题状态**: `[✅️]`

**问题说明**:
用户指出 `llm/models` 下的模型信息本质上只是模型配置样例和模型上下文相关信息，应放入 json 文件并从 json 加载。`llm/models/models.go` 只需要保留 `Model` 结构体，且 `Model` 不需要 `Provider` 字段，因为通常是 provider 下包含有哪些模型，而不是模型自身携带 provider。

**当前现象**:
`llm/models/models.go` 定义了 `ModelID`、`ModelProvider`、`Model`、provider 常量、knownModels map、注册函数、`ProviderForModel`、`ResolveModel`。同时 `llm/models/openai.go`、`anthropic.go`、`gemini.go`、`groq.go`、`azure.go`、`openrouter.go`、`xai.go`、`vertexai.go`、`copilot.go` 等文件通过 Go map 硬编码各 provider 的模型信息。

**期望行为**:
将 provider 及其模型清单迁移为 json 配置文件加载。`llm/models/models.go` 中仅保留通用 `Model` 结构体和必要的加载/解析边界；`Model` 结构体移除 `Provider` 字段，provider 与模型的归属关系由 json 的层级结构表达，例如 provider 下包含 models 列表。模型上下文、默认 token、价格、reasoning、附件支持等仍作为模型配置字段保留。

**证据或复现**:
- 查看 `llm/models/models.go`，`Model` 当前包含 `Provider ModelProvider` 字段，并维护 `knownModels` 与注册逻辑。
- `rg --files llm/models` 显示多个 provider 专属 Go 文件，当前模型清单通过 Go 源码维护。
- `rg "ProviderForModel|ResolveModel|knownModels|Provider ModelProvider" llm/models -n` 可定位当前 provider 与 model 绑定逻辑。

**澄清记录**:
- 澄清状态：无需澄清
- 当前阶段：Implement
- 澄清目的：用户已明确 JSON 加载和移除 Model.Provider。
- 澄清结论：保留 `Model` 结构体，模型清单由 `models.json` provider 层级加载。

**根因分析**:
早期实现把 provider 与模型元数据固化在 Go map 中，导致模型配置扩展需要改代码。

**影响范围**:
- `llm/models`, `agent.go`, `config.go`, provider 构造逻辑

**修复边界**:
- 允许：移除 Go map 模型清单并改为 JSON catalog。
- 不做：不维护完整第三方最新模型列表。

**决策分析**:
JSON catalog 表达 provider 下包含哪些模型，`Model` 自身不再携带 provider。

**修复任务**:
- [✅️] T016 [ISSUE-008] 删除 provider 专属 Go 模型 map
- [✅️] T017 [ISSUE-008] 新增 `models.json` 和 catalog loader
- [✅️] T018 [ISSUE-008] 移除 `Model.Provider` 依赖

### ISSUE-009: Prompt 获取仍绑定 AgentName 与 GetAgentPrompt 封装

**问题状态**: `[✅️]`

**问题说明**:
用户指出 `GetAgentPrompt` 应该不再需要，可以删除。prompt 已经支持通过 key 获取，后续不应再保留 `AgentName` 概念来决定 prompt。当前实现仍通过 `GetAgentPrompt(agentName, provider)` 把 prompt key 与 `config.AgentName` 绑定，和配置驱动、key 驱动的 prompt 方向不一致。

**当前现象**:
`prompt/prompt.go` 已提供 `ResolveSystemPromptByKey(key string)`，但同时仍保留 `GetAgentPrompt(agentName config.AgentName, _ models.ModelProvider)`。该函数内部将 `AgentName` 转成 string 获取 prompt，并对 `AgentCoder`、`AgentTask` 做额外上下文拼接。`agent.go` 中创建 provider 时仍调用 `prompt.GetAgentPrompt(internalconfig.AgentName(agentName), model.Provider)`。

**期望行为**:
删除 `GetAgentPrompt`，调用方统一通过 prompt key 获取系统提示词，例如直接调用 `ResolveSystemPromptByKey(key)` 或更明确的 key-based resolver。prompt 配置不再依赖 `config.AgentName`；如仍需项目上下文拼接，应通过配置项、prompt key 元数据或调用方策略表达，而不是硬编码 `AgentCoder`/`AgentTask`。

**证据或复现**:
- 查看 `prompt/prompt.go`，存在 `ResolveSystemPromptByKey` 与 `GetAgentPrompt` 两套入口。
- `rg "GetAgentPrompt|AgentName|AgentCoder|AgentTask" -n` 显示 prompt、config、agent 等路径仍使用 `AgentName` 概念。
- `agent.go` 中 provider 初始化仍通过 `GetAgentPrompt` 获取 system message。

**澄清记录**:
- 澄清状态：无需澄清
- 当前阶段：Implement
- 澄清目的：用户明确 prompt 通过 key 获取。
- 澄清结论：删除 `GetAgentPrompt` 和 `AgentName` 绑定。

**根因分析**:
旧 prompt 入口为了兼容 coder/task/title/summarizer 角色，把 prompt key 固定为 AgentName。

**影响范围**:
- `prompt/prompt.go`, `agent.go`, `config.go`

**修复边界**:
- 允许：删除 AgentName prompt 封装，改用 key resolver。
- 不做：不自动拼接项目上下文。

**决策分析**:
key-based prompt resolver 更符合上游可配置 SDK 边界。

**修复任务**:
- [✅️] T019 [ISSUE-009] 删除 `GetAgentPrompt`
- [✅️] T020 [ISSUE-009] Agent provider 初始化按 prompt key 获取 system prompt

### ISSUE-010: config 包仍承担上游应用配置加载与 CLI 初始化职责

**问题状态**: `[✅️]`

**问题说明**:
用户明确指出项目作为 Agent SDK，config 应该由上游宿主传递过来，SDK 只需要提供配置 struct。当前 `config` 包分散在 `config.go`、`config_impl.go`、`init_impl.go`，并包含 viper 文件加载、环境变量读取、默认模型选择、日志初始化、配置文件回写、项目 init 标记、默认上下文文件扫描和全局 shell 配置等上游应用职责，不符合 SDK 边界。

**当前现象**:
`config/config_impl.go` 负责读取 `$HOME`、`$XDG_CONFIG_HOME`、工作目录下的 opencode 配置文件，使用 viper 设置默认值和环境变量，自动探测 provider API key 并设置默认 agent 模型。`config/init_impl.go` 负责 `ShouldShowInitDialog` 与 `MarkProjectInitialized`。`defaultContextPaths` 被设置为默认 `ContextPaths`，再由 `prompt.GetAgentPrompt` 自动读取项目上下文。`ShellConfig` 作为全局 config 字段，当前只被 bash tool 的持久 shell 使用。

**期望行为**:
将 `config` 包收敛为 SDK 配置类型定义与最小必要的注入/校验，优先保留单个 `config.go`。上游宿主负责读取 JSON/YAML/env/远程配置并传入 SDK；SDK 不再主动使用 viper 扫描配置文件、不再回写配置文件、不再执行项目初始化标记逻辑、不再根据环境变量自动选择 provider/model、不再初始化应用日志。移除 `defaultContextPaths` 和自动项目上下文拼接；如需要项目上下文，由上游显式传入 prompt/context。`ShellConfig` 不应作为全局 config 核心字段，应迁移为 bash tool 配置或工具初始化参数。

**证据或复现**:
- `rg "viper|config\\.Load|ShouldShowInitDialog|MarkProjectInitialized" config -n` 显示 `config` 包仍包含应用级配置加载、文件回写和 init 状态逻辑。
- `rg "defaultContextPaths|ContextPaths" config prompt -n` 显示默认上下文路径仍由 config 注入，并由 prompt 自动读取。
- `rg "ShellConfig|cfg\\.Shell|GetPersistentShell" config tools -n` 显示 `ShellConfig` 是全局 config 字段，但实际只服务于 bash tool shell 启动。
- `config/config.go` 与 `config/config_impl.go` 同时存在，配置 struct 与实现逻辑分散；`config_impl.go` 包含大量不属于 SDK 核心的上游应用行为。

**澄清记录**:
- 澄清状态：无需澄清
- 当前阶段：Implement
- 澄清目的：用户明确 SDK config 由上游传入。
- 澄清结论：config 收敛为单文件 struct、defaults、validate 和 `Use/Get`。

**根因分析**:
旧 config 包继承了应用层配置管理器职责，包含 viper、env、日志和 init 流程。

**影响范围**:
- `config/config.go`, `config/config_impl.go`, `config/init_impl.go`, tests, prompt, tools

**修复边界**:
- 允许：删除应用加载逻辑并调整调用方使用注入配置。
- 不做：不提供 CLI 配置文件读取器。

**决策分析**:
SDK 只定义配置结构和注入入口，上游宿主负责配置来源。

**修复任务**:
- [✅️] T021 [ISSUE-010] 删除 `config_impl.go` 与 `init_impl.go`
- [✅️] T022 [ISSUE-010] 移除 defaultContextPaths、ShellConfig、viper Load 逻辑
- [✅️] T023 [ISSUE-010] 更新测试使用 `config.Use`

### ISSUE-011: message.Attachment 类型游离在 attachment.go 中

**问题状态**: `[✅️]`

**问题说明**:
用户指出 `message/attachment.go` 的内容可以移动到 `content.go` 下，没有必要让 attachment 类型单独游离在外。当前 `Attachment` 仅是 message 输入附件的数据结构，和 `BinaryContent`、`ImageURLContent` 等 content 定义关系更近。

**当前现象**:
`message/attachment.go` 单独定义 `type Attachment struct`，包含 `FilePath`、`FileName`、`MimeType`、`Content` 四个字段。`message/content.go` 已集中定义 message role、finish reason、content part、text/image/binary/tool 等内容结构。`agent.go` 中 `Run` 接收 `message.Attachment` 并转换为 `message.BinaryContent`。

**期望行为**:
将 `Attachment` 类型移动到 `message/content.go` 中，与其他 content/part 类型放在一起；删除不再需要的 `message/attachment.go`，保持 message 包结构更集中。

**证据或复现**:
- `rg --files message` 显示同时存在 `message/content.go` 与 `message/attachment.go`。
- 查看 `message/attachment.go`，文件仅包含 `Attachment` 一个简单结构体。
- `rg "message.Attachment|Attachment" -n` 显示主要调用点在 `agent.go` 的 `Run` 入参和附件转 `BinaryContent` 逻辑。

**澄清记录**:
- 澄清状态：无需澄清
- 当前阶段：Implement
- 澄清目的：用户明确移动到 content.go。
- 澄清结论：`Attachment` 与 content 类型合并。

**根因分析**:
Attachment 是轻量输入内容类型，单独成文件造成 message 包结构松散。

**影响范围**:
- `message/content.go`, `message/attachment.go`

**修复边界**:
- 允许：移动类型定义并删除空文件。
- 不做：不改变 Agent Run 附件 API。

**决策分析**:
合并到 content.go 与 BinaryContent 等消息内容定义更一致。

**修复任务**:
- [✅️] T024 [ISSUE-011] 移动 `Attachment` 到 `message/content.go`
- [✅️] T025 [ISSUE-011] 删除 `message/attachment.go`
