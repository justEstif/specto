---
# project-media-consumption-analysis-kpad
title: Implement user auth (signup/login/session)
status: completed
type: feature
priority: high
created_at: 2026-03-12T02:37:25Z
updated_at: 2026-03-12T02:52:45Z
parent: project-media-consumption-analysis-hj33
blocked_by:
    - project-media-consumption-analysis-gsai
---

Add user registration, login, logout, and session management. Start with email/password. OAuth (Google/GitHub) can come later. Secure password hashing (bcrypt). Session cookies. Auth middleware to protect routes.

## Summary of Changes

- Migration 009: added `password_hash` column to users table
- `internal/auth` package: Register (bcrypt), Login, session helpers, context user
- Auth handlers: register, login, session, logout — all JSON API endpoints
- RequireAuth middleware: JSON 401 for API routes, redirect for browser
- Cookie sessions via gorilla/sessions (7-day expiry, HttpOnly, SameSite=Lax)
- Input validation: required fields, min 8-char password
- Error responses match docs/api.md envelope format
- httpyac request files for all auth flows (happy + error paths)
- SESSION_SECRET env var in mise.toml
