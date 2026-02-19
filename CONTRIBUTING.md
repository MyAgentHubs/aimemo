# Contributing to aimemo

## Prerequisites

- Go 1.21+
- No CGO required (`modernc.org/sqlite` is pure Go)

## Running Tests

```bash
go test ./...
```

## Building

```bash
go build -o aimemo ./cmd/aimemo
```

## Project Structure

```
cmd/aimemo/         # Binary entry point
internal/
  cli/              # Cobra commands
  config/           # TOML config loader
  db/               # SQLite layer (entities, observations, relations, journal, search)
  locate/           # .aimemo directory discovery
  mcp/              # MCP stdio server (JSON-RPC 2.0)
examples/           # CLAUDE.md templates for users
```

## Making Changes

1. Fork the repo and create a branch from `main`
2. Make your changes with tests where applicable
3. Run `go test ./...` — all tests must pass
4. Submit a pull request with a clear description of the change

## Reporting Issues

Open a GitHub Issue. For security vulnerabilities, please email the maintainers
directly rather than opening a public issue.
