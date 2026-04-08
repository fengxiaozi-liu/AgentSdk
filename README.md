# Agent SDK

`agent` is the new top-level SDK package for reusable agent runtime capabilities.

Package layout:

- `agent/*`: core agent orchestration
- `tools/*`: top-level tool implementations
- `prompt/*`: prompt construction helpers
- `llm/models`: model definitions
- `llm/provider`: provider integrations
- `config`: SDK configuration models and runtime wiring helpers
- `extensions/*`: optional extensions such as LSP
- `infra/*`: infrastructure support packages such as db, logging, file utilities, diff, and format

Naming follows the current `internal/*` module semantics on purpose so the extraction can happen incrementally without a large rename-first refactor.
