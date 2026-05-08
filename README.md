# Agent SDK

Reusable Go Agent SDK for agent orchestration, LLM provider integration, session/message/history services, tool execution, MCP tools, permissions, file hooks, and diff/patch core.

The SDK intentionally does not include Skill tools, CLI/TUI UI, terminal themes, or IDE/LSP extensions. Those belong in host applications.

## Packages

- `internal/agent`: runtime orchestration and agent events
- `internal/config`: host-provided SDK configuration structs, defaults, validation, and injection
- `internal/data/db`, `internal/data/repo`: gorm-backed database connection/models and repository contracts
- `internal/memory/session`, `internal/memory/message`, `internal/memory/history`: domain services over repos
- `internal/data/llm/models`, `internal/data/llm/client`: model metadata and vendor SDK adapters
- `internal/provider`: application-level ProviderService routing from model IDs to provider clients
- `internal/prompt`: JSON/YAML system prompt resolver
- `internal/tools`: tool protocol, file hook events, and hook result merging
- `internal/capability/workspace`: SDK-safe workspace tools
- `internal/capability/mcp`: MCP tool discovery and execution
- `internal/utils/diff`: diff/patch core only
- `internal/utils/fileutil`, `internal/data/logging`: shared support utilities

## Prompts

Default prompts live in `internal/prompt/prompts.json`. A host can provide a JSON or YAML prompt file and set `internal/config.PromptConfigPath`; prompts are then resolved by key, for example `coder`, `title`, `task`, or `summarizer`.

## Model Config

Provider configuration is provider-scoped and each provider declares a `models` list. `ProviderConfig`, `ModelConfig`, and `DatabaseConfig` are defined in `internal/config/config.go`.

```json
{
  "providers": [
    {
      "provider": "openai",
      "apiKey": "...",
      "baseURL": "",
      "models": [
        {
          "model_id": "gpt-4.1",
          "api_model": "gpt-4.1",
          "maxTokens": 8192,
          "reasoning_effort": "medium",
          "weight": 1,
          "priority": 0
        }
      ]
    }
  ],
  "agent": {
    "model_id": "gpt-4.1",
    "provider": "openai"
  },
  "titleAgent": {
    "model_id": "gpt-4.1",
    "provider": "openai"
  },
  "summarizeAgent": {
    "model_id": "gpt-4.1",
    "provider": "openai"
  }
}
```

The old single `modelConfig` field is intentionally not compatible; migrate it to a one-item `models` list. Model metadata lives in `internal/data/llm/models/models.json`, and `max_tokens` is exposed at runtime as `models.Model.MaxTokens`.
