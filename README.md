# Agent SDK

`agent` is the new top-level SDK package for reusable agent runtime capabilities.

Package layout:

- `agent/*`: core components
- `agent/config`: SDK configuration models and runtime wiring helpers
- `agent/extensions/*`: optional extensions such as LSP
- `agent/infra/*`: infrastructure support packages such as db, logging, file utilities, diff, and format

Naming follows the current `internal/*` module semantics on purpose so the extraction can happen incrementally without a large rename-first refactor.
