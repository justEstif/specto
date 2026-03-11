# Development Workflow

## Overview

This document defines the preferred day-to-day development workflow for this repo.

It is intentionally practical: the goal is to reduce ambiguity while the system is still
being designed and built.

Related docs:
- [api.md](./api.md) — canonical client/server API surface
- [architecture.md](./architecture.md) — system boundaries and layering
- [plugin-guide.md](./plugin-guide.md) — plugin interface and sync behavior
- [schema.md](./schema.md) — persistence model

---

## Core Principle

When a behavior crosses a boundary, we should document and test it at that boundary.

Examples:
- **HTTP behavior** should be documented and exercised through HTTP requests
- **Plugin behavior** should be documented through plugin contracts and tests
- **DB behavior** should be documented through schema and query expectations

This keeps the design honest and helps prevent the docs from drifting away from actual usage.

---

## API Development Workflow

Since this project is building a documented internal HTTP API, we should use **httpyac** for
both:

1. **API documentation-in-use**
2. **manual and automated API testing**

Reference:
- https://httpyac.github.io/

### Why httpyac

We want one place where API work can be:
- readable by humans
- executable by developers
- easy to version in git
- close to the canonical route and payload definitions in `docs/api.md`

`httpyac` fits well because it lets us keep request examples as text files in the repo,
execute them locally, organize environments, and reuse them during development.

### Intended role of docs vs httpyac

- `docs/api.md` defines the **canonical API contract**
- `httpyac` request files define the **executable examples and test flows**

Put differently:
- if someone asks, “what is the route shape?”, the answer should live in `docs/api.md`
- if someone asks, “how do I actually call it and verify it works?”, the answer should live in `httpyac` files

---

## Proposed Repo Structure

When implementation starts, add an API requests directory like:

```text
http/
  environments/
    local.env.json
    test.env.json
  session.http
  plugins.http
  timeline.http
  insights.http
  share-profile.http
```

Suggested responsibilities:

- `http/session.http` — login/logout/session checks
- `http/plugins.http` — connect/import/disconnect/sync plugin flows
- `http/timeline.http` — timeline listing and filters
- `http/insights.http` — summary and aggregate queries
- `http/share-profile.http` — preview and share profile config flows
- `http/environments/*.env.json` — base URLs, auth/session helpers, local variables

If the API grows, split further by feature area.

---

## API Route Workflow

For every new API endpoint or route change:

1. **Design the route in `docs/api.md` first**
   - route path
   - method
   - request shape
   - response shape
   - error cases

2. **Add or update a `httpyac` request**
   - happy path example
   - at least one failure/validation example when relevant

3. **Implement the server behavior**
   - handler
   - validation
   - core call
   - serialization

4. **Run the `httpyac` requests against the local server**
   - confirm response shape matches docs
   - confirm status codes and error envelopes

5. **Update docs if implementation reveals a better API shape**
   - prefer improving the API over preserving a confusing route
   - keep `docs/api.md` and `http/` examples in sync

### Definition of done for an API route

A route is not fully done until all of the following are true:

- documented in `docs/api.md`
- exercised in `httpyac`
- implemented in server code
- aligned with auth/permissions expectations
- returns stable JSON envelopes and status codes

---

## httpyac Conventions

### File organization

Each `.http` file should group related operations and read top-to-bottom as a workflow.

Example for `plugins.http`:
- list plugins
- get one plugin
- start connect flow
- import file
- sync plugin
- disconnect plugin

### Naming

Use section comments that make intent obvious.

Example:

```http
### List plugins
GET {{baseUrl}}/api/v1/plugins

### Sync Spotify
POST {{baseUrl}}/api/v1/plugins/spotify/sync
```

### Environments

Use environment files for values like:
- `baseUrl`
- local test user identifiers if needed
- CSRF token helpers if needed
- session/cookie state when supported by the setup

Do not hardcode secrets into committed request files.

### Coverage expectations

For important routes, prefer covering:
- happy path
- unauthorized request
- validation error
- not found
- rate limited or conflict state when relevant

---

## Example Request Layout

Illustrative example only:

```http
@baseUrl = http://localhost:8080

### Session
GET {{baseUrl}}/api/v1/session

### List plugins
GET {{baseUrl}}/api/v1/plugins

### Get Spotify plugin state
GET {{baseUrl}}/api/v1/plugins/spotify

### Start Spotify connect flow
POST {{baseUrl}}/api/v1/plugins/spotify/connect
Content-Type: application/json

{}

### Sync Spotify
POST {{baseUrl}}/api/v1/plugins/spotify/sync
Content-Type: application/json

{}

### Timeline
GET {{baseUrl}}/api/v1/timeline?limit=20&offset=0

### Share profile preview
GET {{baseUrl}}/api/v1/share-profile/preview
```

---

## Workflow for Changing Existing API Routes

When changing an existing route:

1. update `docs/api.md`
2. update the matching `httpyac` request file
3. update implementation
4. verify old examples are removed so stale routes do not linger in docs

This matters because route drift creates high cognitive load: developers stop knowing
which doc is real, which example is current, and which behavior clients should trust.

---

## Non-API Workflow Guidance

### Architecture changes

When a change affects boundaries between client, server, core, plugins, or storage:
- update `docs/architecture.md`
- update `docs/api.md` if the client/server contract changes
- update `httpyac` files if the HTTP surface changes

### Plugin changes

When adding or changing a plugin:
- update `docs/plugin-guide.md` if the shared contract changes
- update the plugin-specific doc under `docs/plugins/`
- add or update `httpyac` flows for user-visible plugin operations

### Schema changes

When changing persisted fields that affect API responses:
- update `docs/schema.md`
- update `docs/api.md` response examples if needed
- update relevant `httpyac` examples

---

## Initial Action Items

Before active API implementation begins, we should add:

- an `http/` directory
- a small starter set of `.http` files for the canonical routes in `docs/api.md`
- one local environment file for running against the dev server
- a short README in `http/` explaining how to run the requests

---

## Practical Rule

If we add a route and cannot easily express it as a clean `httpyac` request,
that is a signal to re-check the API design.

Good routes should be easy to:
- describe in markdown
- call from a request file
- understand without reading server internals
