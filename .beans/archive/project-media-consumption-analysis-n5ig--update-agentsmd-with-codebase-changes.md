---
# project-media-consumption-analysis-n5ig
title: Update AGENTS.md with codebase changes
status: completed
type: task
priority: normal
created_at: 2026-03-14T21:16:12Z
updated_at: 2026-03-14T21:17:26Z
---

Update AGENTS.md to reflect current codebase: missing packages, env vars, docs, CI, logging infra

## Summary of Changes\n\nUpdated AGENTS.md to reflect current codebase state. Key additions:\n- Added internal/enrichment/, internal/llm/, internal/logger/, internal/plugins/ to structure diagram\n- Updated core/ description from 'pure domain logic' to 'domain types, interfaces, and service orchestration'\n- Added mise run pre-commit task\n- Added BASE_URL to required env vars, documented all optional env vars (OAuth, enrichment, LLM, API keys)\n- Added CI reference (.github/workflows/ci.yml)\n- Documented wide event logging and enrichment worker in architecture patterns\n- Added HTMX partials and OAuth routes to HTTP conventions\n- Noted http/fixtures/ exception for test fixture files\n- Added DaisyUI 5 + Tailwind CSS 4 specifics and Copilot instructions reference\n- Added 4 missing docs: api-key-setup.md, self-hosting.md, plugins/, research/\n- Updated graceful shutdown mention in main.go description
