# Agent SDK

Reusable Go Agent SDK for agent orchestration, LLM provider integration, session/message/history services, tool execution, MCP tools, permissions, file hooks, and diff/patch core.

The SDK intentionally does not include Skill tools, CLI/TUI UI, terminal themes, or IDE/LSP extensions. Those belong in host applications.

## Packages

- `internal/agent`: runtime orchestration and agent events
- `internal/config`: host-provided SDK configuration structs, defaults, validation, and injection
- `internal/data/db`, `internal/data/repo`: gorm-backed database connection/models and repository contracts
- `internal/memory/session`, `internal/memory/message`, `internal/memory/history`: domain services over repos
- `internal/data/llm/models`, `internal/data/llm/provider`: model metadata and provider clients
- `internal/prompt`: JSON/YAML system prompt resolver
- `internal/tools`: tool protocol, file hook events, and hook result merging
- `internal/capability/workspace`: SDK-safe workspace tools
- `internal/capability/mcp`: MCP tool discovery and execution
- `internal/utils/diff`: diff/patch core only
- `internal/utils/fileutil`, `internal/data/logging`: shared support utilities

## Prompts

Default prompts live in `internal/prompt/prompts.json`. A host can provide a JSON or YAML prompt file and set `internal/config.PromptConfigPath`; prompts are then resolved by key, for example `coder`, `title`, `task`, or `summarizer`.

## Model Config

Model profiles use an explicit provider plus model string. Example metadata lives in `internal/data/llm/models/models.json`; provider request construction can still use arbitrary model IDs.
