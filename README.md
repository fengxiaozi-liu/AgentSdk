# Agent SDK

Reusable Go Agent SDK for agent orchestration, LLM provider integration, session/message/history services, tool execution, MCP tools, permissions, file hooks, and diff/patch core.

The SDK intentionally does not include Skill tools, CLI/TUI UI, terminal themes, or IDE/LSP extensions. Those belong in host applications.

## Packages

- `agent`: runtime orchestration and agent events
- `config`: host-provided SDK configuration structs, defaults, validation, and injection
- `data/db`, `data/repo`: gorm-backed database connection/models and repository contracts
- `session`, `message`, `history`: domain services over repos
- `llm/models`, `llm/provider`: model metadata and provider clients
- `prompt`: JSON/YAML system prompt resolver
- `tools/core`: tool protocol, file hook events, and hook result merging
- `tools/base`: SDK-safe base tools
- `tools/mcp`: MCP tool discovery and execution
- `utils/diff`: diff/patch core only
- `utils/fileutil`, `logging`: shared support utilities

## Prompts

Default prompts live in `prompt/prompts.json`. A host can provide a JSON or YAML prompt file and set `config.PromptConfigPath`; prompts are then resolved by key, for example `coder`, `title`, `task`, or `summarizer`.

## Model Config

Model profiles use an explicit provider plus model string. Example metadata lives in `llm/models/models.json`; provider request construction can still use arbitrary model IDs.
