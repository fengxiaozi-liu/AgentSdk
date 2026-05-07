# Agent SDK Refactor TODO

> 状态约定：`- [ ]` 未完成，`- [x] ✅` 已完成。

## 配置与边界

- [x] ✅ 清理 `config.Config` 中旧的 `Data.Directory` 依赖。
- [x] ✅ 保留或新增 `WorkingDir string`，作为 workspace root 来源。
- [x] ✅ 清理旧字段与旧方法引用，例如 `Providers`、`ModelProfiles`、`ModelProfile()`。
- [x] ✅ 确认数据库路径只通过 `Config.Database.Path` 或 `Config.Database.DSN` 配置。

## Workspace

- [x] ✅ 新增 `tools/workspace/workspace.go`。
- [x] ✅ 定义 `Workspace` 结构体，至少包含 `Root string`。
- [x] ✅ 实现 `Workspace.Resolve(path string) (string, error)`。
- [x] ✅ 可选实现 `Workspace.Contains(path string) bool`。
- [x] ✅ 为 `Workspace.Resolve` 添加单元测试，覆盖相对路径、绝对路径和越界路径。

## 工具迁移

- [x] ✅ 将 `tools/base` 逐步迁移为 `tools/workspace`。
- [x] ✅ 修改 workspace 工具包名为 `workspace`。
- [x] ✅ 更新所有引用 `tools/base` 的 import。
- [x] ✅ 修改 `NewViewTool`，显式接收 `Workspace`。
- [x] ✅ 修改 `NewEditTool`，显式接收 `Workspace`。
- [x] ✅ 修改 `NewWriteTool`，显式接收 `Workspace`。
- [x] ✅ 修改 `NewPatchTool`，显式接收 `Workspace`。
- [x] ✅ 修改 `NewGrepTool`，显式接收 `Workspace`。
- [x] ✅ 修改 `NewGlobTool`，显式接收 `Workspace`。
- [x] ✅ 修改 `NewLsTool`，显式接收 `Workspace`。
- [x] ✅ 修改 `NewBashTool`，显式接收 `Workspace`。
- [x] ✅ 修改 `NewFetchTool`，在需要路径上下文时显式接收 `Workspace`。

## 移除全局工作目录依赖

- [x] ✅ 移除 `bash` 对 `config.WorkingDirectory()` 的直接调用。
- [x] ✅ 移除 `edit` 对 `config.WorkingDirectory()` 的直接调用。
- [x] ✅ 移除 `grep` 对 `config.WorkingDirectory()` 的直接调用。
- [x] ✅ 移除 `glob` 对 `config.WorkingDirectory()` 的直接调用。
- [x] ✅ 移除 `patch` 对 `config.WorkingDirectory()` 的直接调用。
- [x] ✅ 移除 `write` 对 `config.WorkingDirectory()` 的直接调用。
- [x] ✅ 移除 `fetch` 对 `config.WorkingDirectory()` 的直接调用。
- [x] ✅ 移除 `permission` 对 `config.WorkingDirectory()` 的直接调用。
- [x] ✅ 文件路径解析统一改为使用 `Workspace.Resolve(...)`。
- [x] ✅ `bash` 的执行目录改为使用 `Workspace.Root`。
- [x] ✅ 权限请求中的 `Path` 改为使用 resolve 后路径或 workspace root。

## Wire 与 Agent 装配

- [x] ✅ 调整 `wireContainer(config *config.Config)`，由 Wire 提供 `Workspace`。
- [x] ✅ Wire 装配 DB、repo、session/message/history service、permission service。
- [x] ✅ Wire 装配 workspace tools 集合。
- [x] ✅ `NewAgent` 改为接收 `Config`、session/message/history service、tools 集合来创建 Agent。
- [x] ✅ `NewAgent` 不直接调用 Wire。
- [x] ✅ 新增或调整 `MainAgent`，启动时通过 `wireContainer` 获取 services/tools。
- [x] ✅ `MainAgent` 调用 `NewAgent(...)` 创建全局主 Agent。

## 子 Agent 工具

- [x] ✅ 更新 `tools/agent`，创建子 Agent 时复用已有 session/message service。
- [x] ✅ 子 Agent 的只读工具通过 `Workspace` 构造。
- [x] ✅ 确认子 Agent 不绕过 workspace 边界。

## 测试与验证

- [x] ✅ 更新 `view` 工具测试。
- [x] ✅ 更新 `edit` 工具测试。
- [x] ✅ 更新 `write` 工具测试。
- [x] ✅ 更新 `patch` 工具测试。
- [x] ✅ 更新 `grep` 工具测试。
- [x] ✅ 更新 `glob` 工具测试。
- [x] ✅ 更新 `ls` 工具测试。
- [ ] 更新 `bash` 工具测试。
- [x] ✅ 更新 Wire 初始化测试。
- [x] ✅ 更新 `MainAgent` 初始化测试。
- [x] ✅ 执行 `go generate ./...`。
- [x] ✅ 执行 `go test ./...`。
