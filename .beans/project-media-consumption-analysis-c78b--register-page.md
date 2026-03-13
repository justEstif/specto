---
# project-media-consumption-analysis-c78b
title: Register page
status: completed
type: feature
priority: high
created_at: 2026-03-13T15:23:02Z
updated_at: 2026-03-13T15:34:01Z
parent: project-media-consumption-analysis-nn0q
blocked_by:
    - project-media-consumption-analysis-d41y
---

Register page at /register. Same centered card layout as login. Fields: display name, email, password, confirm password. OAuth buttons on top. hx-post for form with inline validation. 'Already have an account? Sign in' link. Refs: docs/ui-design.md §Register, docs/auth.md.

## Tasks
- [x] Register page templ template (components/register.templ)
- [ ] OAuth buttons (Continue with Google, Continue with GitHub)
- [ ] Registration form: display name, email, password, confirm password
- [ ] hx-post="/register", hx-swap="outerHTML" on card for validation errors
- [ ] Client-side password match hint (optional)
- [ ] 'Already have an account? Sign in' link
- [ ] Handler + route wiring
- [ ] Responsive: same as login page

## Summary of Changes

Built the register page with centered card layout, OAuth buttons, display name/email/password/confirm password fields, HTMX form submission with inline validation errors, server-side validation, and CSRF protection.

### Files created
- components/register.templ - RegisterPage + RegisterCard components
- internal/handlers/pages.go - Shared page handlers for login, register, logout
