---
# project-media-consumption-analysis-9u6b
title: Login page
status: completed
type: feature
priority: high
created_at: 2026-03-13T15:22:55Z
updated_at: 2026-03-13T15:33:42Z
parent: project-media-consumption-analysis-nn0q
blocked_by:
    - project-media-consumption-analysis-d41y
---

Login page at /login. Centered card (max-w-sm), vertically centered. OAuth buttons (Google, GitHub) as full-width buttons, divider, email/password form. hx-post for email/password with inline validation errors. Redirect to / if already logged in. Refs: docs/ui-design.md §Login, docs/auth.md.

## Tasks
- [x] Login page templ template (components/login.templ)
- [x] OAuth buttons (Continue with Google, Continue with GitHub)
- [x] Email/password form with hx-post="/login", hx-swap="outerHTML" on card
- [x] Inline validation error display (wrong credentials, missing fields)
- [x] 'Don't have an account? Register' link
- [x] Redirect logged-in users to /
- [x] Handler + route wiring in cmd/web/main.go
- [x] Responsive: card full-width on mobile with px-4 padding

## Summary of Changes

Built the login page with:
- Centered card (max-w-sm) with OAuth buttons (Google/GitHub with SVG icons), divider, email/password form
- HTMX form submission: hx-post to /login, hx-swap outerHTML on the card for inline validation errors
- Separate LoginPage (full page) and LoginCard (swap target) templ components
- HX-Redirect header on success for clean HTMX navigation
- CSRF token passed through hidden form field
- Redirect to / if already logged in

### Files created/modified
- `components/login.templ` — LoginPage + LoginCard components
- `internal/handlers/pages.go` — LoginPage, LoginSubmit, RegisterPage, RegisterSubmit, LogoutSubmit handlers
- `cmd/web/main.go` — Routes for /login, /register, /logout
