---
# project-media-consumption-analysis-3yp6
title: Create development workflow doc
status: completed
type: task
priority: normal
created_at: 2026-03-11T17:03:46Z
updated_at: 2026-03-11T17:04:35Z
---

Document the preferred development workflow for this repo, including API development and testing practices.

Checklist:
- [x] Define the scope and audience of the workflow doc
- [x] Document API route workflow with httpyac
- [x] Add example structure for request collections and environments
- [x] Capture how workflow docs relate to the canonical API docs
- [x] Add summary of changes

## Summary of Changes

Created `docs/development-workflow.md` to define the preferred repo workflow for API and documentation development. The doc:
- establishes `docs/api.md` as the canonical API contract
- adopts `httpyac` for executable API documentation and testing
- proposes an `http/` directory structure with request collections and environments
- defines a step-by-step workflow for adding or changing API routes
- connects architecture, plugin, schema, and API documentation updates into one development loop
