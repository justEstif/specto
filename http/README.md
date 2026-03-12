# HTTP Request Files

Executable API documentation using [httpyac](https://httpyac.github.io/).

## Usage

```bash
# Run all requests against local server
mise run api-test

# Run a specific file
httpyac send http/health.http -e local

# Run a specific request by name
httpyac send http/health.http --name "Health check" -e local
```

## Environments

- `local` — `http://localhost:3000` (default dev server)

## File organization

Each `.http` file groups related API operations. See `docs/development-workflow.md` for conventions.
