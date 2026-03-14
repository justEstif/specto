---
# project-media-consumption-analysis-qd9i
title: Add OpenAI-compatible LLM provider support
status: completed
type: task
priority: normal
created_at: 2026-03-14T18:42:27Z
updated_at: 2026-03-14T19:14:15Z
parent: project-media-consumption-analysis-eo0f
---

Add compat_oai/openai plugin as an alternative LLM provider. Supports OpenAI direct and Ollama (OpenAI-compatible mode). New LLM_PROVIDER=openai value + LLM_BASE_URL env var for custom endpoint.

## Summary of Changes

Added OpenAI-compatible LLM provider support via Genkit's compat_oai plugin:

### Modified files:
- `internal/enrichment/genkit.go` — Added `openai` provider case using `compat_oai/openai.OpenAI` plugin with custom `BaseURL` support via `option.WithBaseURL()`. Model name override at execution time (`ai.WithModelName`).
- `internal/enrichment/genkit_test.go` — Updated unsupported provider test (openai → anthropic)
- `cmd/web/main.go` — Reads `LLM_BASE_URL` env var
- `internal/enrichment/genkit.go` Config struct — Added `BaseURL` field
- `docs/api-key-setup.md` — Added OpenCode Zen, OpenAI direct, and Ollama (OpenAI-compat) setup guides
- `docs/enrichment.md` — Updated LLM env var reference
- `mise.toml` / `mise.local.toml` — Updated LLM config comments
- `go.mod` / `go.sum` — Added openai-go dependency

### Usage (OpenCode Zen):
```toml
LLM_PROVIDER = "openai"
LLM_MODEL = "gemini-3-flash"
LLM_API_KEY = "your-zen-key"
LLM_BASE_URL = "https://opencode.ai/zen/v1/"
```
