# aimemo

[![Go 1.22+](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org/) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat)](LICENSE) [![Release](https://img.shields.io/github/v/release/MyAgentHubs/aimemo?style=flat)](https://github.com/MyAgentHubs/aimemo/releases)

Zero-dependency MCP memory server for AI agents — persistent, searchable, local-first, single binary.

```
$ claude "let's keep working on the payment service"

  ╭─ memory_context ──────────────────────────────────────────────────╮
  │ [project: payment-service]                                        │
  │                                                                   │
  │ Last session (3 days ago):                                        │
  │  • Stripe webhook signature verification — DONE                   │
  │  • Idempotency key refactor — IN PROGRESS                         │
  │  • Blocked: race condition in concurrent refund handler           │
  │                                                                   │
  │ Related: Redis connection pool, pkg/payments/refund.go            │
  ╰───────────────────────────────────────────────────────────────────╯

  Picking up where we left off. The race condition in the refund
  handler looks like a missing mutex around the in-flight map.
  Let me check pkg/payments/refund.go ...

  [... Claude works through the fix ...]

  ╭─ memory_store (journal) ──────────────────────────────────────────╮
  │ Resolved refund race — added sync.Mutex around inFlightRefunds.   │
  │ Tests passing. Next: load test with k6 at 500 rps.               │
  ╰───────────────────────────────────────────────────────────────────╯
```

## 🧠 Why aimemo

- **No infra to babysit.** Single Go binary. No Docker, no Node.js runtime, no cloud account, no API keys. `brew install` in 30 seconds.
- **Memory stays with the project.** Stored in `.aimemo/` next to your code — commit it to git or add it to `.gitignore`. Switch branches; memory follows the directory.
- **Claude picks up exactly where it left off.** `memory_context` fires automatically on every session start. Claude sees what it was doing, what was blocked, what decisions were made. You stop repeating yourself.
- **Full-text search that ranks correctly.** FTS5 + BM25 scoring weighted by recency and access frequency. Relevant memories surface first; old noise fades naturally.
- **Concurrent sessions, no corruption.** SQLite WAL mode lets multiple Claude windows write simultaneously without locking each other out.
- **You stay in control.** Every tool Claude has, you have from the terminal. Inspect, edit, retract, export. Your memory is readable Markdown or JSON — never locked in a proprietary format.

## ⚡ Quick Start

```bash
# 1. Install
brew install MyAgentHubs/tap/aimemo

# 2. Initialize memory for your project (run from project root)
aimemo init

# 3. Register with Claude Code
claude mcp add-json aimemo '{"type":"stdio","command":"aimemo","args":["serve"]}'
```

Restart Claude Code. On the next session, Claude will automatically load project context.

## 🔧 How It Works

`aimemo serve` runs as a stdio MCP server; Claude Code manages the process lifecycle, so there is nothing to keep alive yourself. When Claude starts a session it calls `memory_context` to load relevant prior context; as it works it calls `memory_store` and `memory_link` to record decisions and relationships. You can call `aimemo search`, `aimemo list`, or `aimemo get` at any time to read or edit the same data from your terminal. Everything lives in a SQLite database inside `.aimemo/`, discovered by walking up from the current directory — the same way Git finds `.git/`.

## 🛠 MCP Tools

| Tool | Description | When Claude calls it |
|------|-------------|----------------------|
| `memory_context` | Returns ranked, recent observations for the current project | Session start — automatic |
| `memory_store` | Saves an observation (fact, decision, journal entry, TODO) | After completing a task or making a decision |
| `memory_search` | Full-text search across all observations, BM25-ranked | When it needs to recall something specific |
| `memory_forget` | Soft-deletes an observation by ID | When instructed to discard something |
| `memory_link` | Creates a named relationship between two observations | When it identifies a dependency or connection |

All tool schemas total under 2,000 tokens. Each call has a hard 5-second timeout — the server never stalls your session. Empty-state queries return in under 5 ms.

## 📋 CLI Reference

### Setup

| Command | Description |
|---------|-------------|
| `aimemo init` | Create `.aimemo/` in the current directory |
| `aimemo serve` | Start the MCP stdio server (called by Claude Code automatically) |
| `aimemo doctor` | Verify DB health, FTS5 support, WAL mode, and MCP registration |

### Memory

| Command | Description |
|---------|-------------|
| `aimemo add <text>` | Store an observation from the terminal |
| `aimemo observe <text>` | Alias for `add` |
| `aimemo retract <id>` | Surgically remove a single observation |
| `aimemo forget <pattern>` | Soft-delete all observations matching a pattern |
| `aimemo search <query>` | Full-text search with ranked results |
| `aimemo get <id>` | Show a single observation by ID |
| `aimemo link <id1> <id2> <label>` | Create a named link between two observations |
| `aimemo append <id> <text>` | Append text to an existing observation |

### Journal

| Command | Description |
|---------|-------------|
| `aimemo journal` | Open an interactive journal entry (respects `$EDITOR`) |
| `aimemo journal <text>` | Record a quick inline journal entry |

### Inspect & Export

| Command | Description |
|---------|-------------|
| `aimemo list` | List recent observations |
| `aimemo tags` | List all tags in use |
| `aimemo stats` | Show DB size, observation count, last-write time |
| `aimemo export --format md` | Export all memory to Markdown |
| `aimemo export --format json` | Export all memory to JSON |
| `aimemo import <file>` | Import from mcp-knowledge-graph JSONL or aimemo JSON |

All commands accept `--context <name>` to target a named context (a separate `.db` file inside `.aimemo/`).

## ⚙️ Configuration

`~/.aimemo/config.toml` — global defaults, all optional:

```toml
[defaults]
context = "main"          # default context name
max_results = 20          # observations returned by memory_context

[scoring]
recency_weight = 0.7      # 0–1, weight of recency vs. access frequency

[server]
timeout_ms = 5000         # hard timeout on every MCP call
log_level = "warn"        # "debug" | "info" | "warn" | "error"
```

Per-project overrides live in `.aimemo/config.toml` in the project root — same keys, project values win over global values.

## 🔀 Migration from mcp-knowledge-graph

```bash
# Export your existing graph
npx @modelcontextprotocol/inspector export > knowledge-graph.jsonl

# Import into aimemo
aimemo import knowledge-graph.jsonl
```

Entities become observations; relations become links; tags are preserved. Run `aimemo stats` to confirm the import count.

## 🤖 Claude Code Integration

Register the server once per machine:

```bash
claude mcp add-json aimemo '{"type":"stdio","command":"aimemo","args":["serve"]}'
```

Add the following to your project's `CLAUDE.md` so Claude knows memory is available and how to use it:

```markdown
## Memory

This project uses aimemo for persistent memory across sessions.

- Call `memory_context` at the start of every session to load prior context.
- Call `memory_store` with `type: journal` before ending a session to record
  what was completed, what is still in progress, and any blockers.
- Use `memory_link` to connect related observations (e.g. a bug to its fix,
  a decision to its rationale).
- Do not store secrets, credentials, or PII.
```

## 🤝 Contributing

Bug reports and feature requests go in [GitHub Issues](https://github.com/MyAgentHubs/aimemo/issues). Pull requests are welcome — please open an issue first for anything non-trivial so we can align on direction before you invest time writing code.
