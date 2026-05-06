# Agent SDK

Reusable Go Agent SDK for agent orchestration, LLM provider integration, session/message/history services, tool execution, MCP tools, permissions, file hooks, and diff/patch core.

The SDK intentionally does not include Skill tools, CLI/TUI UI, terminal themes, or IDE/LSP extensions. Those belong in host applications.

## Packages

- `agent`: runtime orchestration and agent events
- `config`: SDK configuration and runtime defaults
- `data/db`, `data/repo`: data source and repository contracts
- `session`, `message`, `history`: domain services over repos
- `llm/models`, `llm/provider`: model metadata and provider clients
- `prompt`: JSON/YAML system prompt resolver
- `tools/core`: tool protocol, file hook events, and hook result merging
- `tools/base`: SDK-safe base tools
- `tools/mcp`: MCP tool discovery and execution
- `infra/diff`: diff/patch core only
- `utils/fileutil`, `logging`: shared support utilities

## Prompts

Default prompts live in `prompt/prompts.json`. A host can provide a JSON or YAML prompt file and set `config.PromptConfigPath`; prompts are then resolved by key, for example `coder`, `title`, `task`, or `summarizer`.

## Model Config

Agent config uses an explicit provider plus model string. The model does not need to appear in an SDK-maintained supported-model table; provider request construction resolves the minimal metadata needed at runtime.
