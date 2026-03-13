---
# project-media-consumption-analysis-p5w8
title: Logout button returns 403 Forbidden (CSRF issue)
status: completed
type: bug
priority: high
created_at: 2026-03-13T17:54:38Z
updated_at: 2026-03-13T18:33:59Z
---

## Problem

POST /logout returns 403 Forbidden. The logout button in the navbar does not work.

## Console Output

```
htmx.min.js:1  POST http://localhost:3000/logout 403 (Forbidden)
Response Status Error Code 403 from /logout
```

(The chrome-extension error in the console is unrelated — it's from a browser extension.)

## Likely Cause

CSRF token is missing or invalid on the logout POST request. The HTMX request probably isn't including the CSRF token header/field.

## To Investigate

- [ ] Check how the logout button/form sends the POST (hx-post vs form)
- [ ] Verify CSRF token is included in the request (header or hidden field)
- [ ] Check CSRF middleware configuration for the /logout route
- [ ] Fix and verify logout works

## Summary of Changes

Added CSRF token to both logout forms (desktop dropdown and mobile drawer) in `components/navbar.templ`. Created a `csrfTokenFromCtx` helper that reads the gorilla/csrf token directly from the templ context, avoiding the need to thread the token through Layout → navbar. Both forms now include the hidden `gorilla.csrf.Token` field.
