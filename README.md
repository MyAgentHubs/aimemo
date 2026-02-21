# aimemo

[![Go 1.22+](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org/) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat)](LICENSE) [![Release](https://img.shields.io/github/v/release/MyAgentHubs/aimemo?style=flat)](https://github.com/MyAgentHubs/aimemo/releases)

[English](README.md) | [中文](README.zh-CN.md)

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
# Linux/macOS (one-line install):
curl -sSL https://raw.githubusercontent.com/MyAgentHubs/aimemo/main/install.sh | bash

# Or macOS via Homebrew:
brew install MyAgentHubs/tap/aimemo

# 2. Initialize memory for your project (run from project root)
aimemo init

# 3. Register with Claude Code
claude mcp add-json aimemo-memory '{"command":"aimemo","args":["serve"]}'
```

Restart Claude Code. On the next session, Claude will automatically load project context.

### Quick Start for OpenClaw

If you're using OpenClaw skills, see the [OpenClaw Integration](#-openclaw-integration) section below for per-skill memory isolation.

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
| `aimemo add <name> <type> [observations...] [--tag]` | Add an entity with one or more observations |
| `aimemo observe <entity-name> <observation>` | Add a new observation to an existing entity |
| `aimemo retract <entity-name> <observation>` | Remove a specific observation from an entity |
| `aimemo forget <entity-name> [--permanent]` | Soft-delete an entity (recoverable); use `--permanent` to hard-delete |
| `aimemo search <query>` | Full-text search with ranked results |
| `aimemo get <entity-name>` | Show an entity with all its observations and relations |
| `aimemo link <from> <relation> <to>` | Create a typed relation between two entities |
| `aimemo append <entity-name> <observation>` | Add an observation to an entity (alias for `observe`) |

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
| `aimemo import <file>` | Import from JSONL or JSON export file |

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

## 🤖 Claude Code Integration

Register the server once per machine:

```bash
claude mcp add-json aimemo-memory '{"command":"aimemo","args":["serve"]}'
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

## 🦞 OpenClaw Integration

aimemo solves OpenClaw's "remembers everything but understands none" problem with **per-skill memory isolation** and **zero infrastructure**.

### Why aimemo for OpenClaw?

**The Problem:**
- OpenClaw's native Markdown memory gets worse the more you use it
- Skills share memory, causing cross-contamination
- Context compression loses important context

**The Solution:**
- ✅ **Zero dependencies** — Single Go binary, no Docker/Node.js/databases
- ✅ **Per-skill isolation** — Each skill gets its own memory database
- ✅ **Actually works** — BM25 search + importance scoring finds what matters
- ✅ **Local-first** — All data stays on your machine

**vs Other Solutions:**

| | aimemo | Cognee | memsearch | Supermemory |
|---|--------|---------|-----------|-------------|
| **Dependencies** | Zero | Neo4j/Kuzu | Milvus | Cloud service |
| **Installation** | 30 sec | Complex | Complex | Sign up required |
| **Skill isolation** | Built-in | Manual | Manual | N/A |
| **Linux support** | ✅ Native | ✅ | ✅ | N/A |

### 5-Minute Setup

```bash
# 1. Install (Linux amd64/arm64)
curl -sSL https://raw.githubusercontent.com/MyAgentHubs/aimemo/main/install.sh | bash

# 2. Register MCP server with OpenClaw
claude mcp add-json aimemo-memory '{"command":"aimemo","args":["serve"]}'

# Or add to ~/.openclaw/openclaw.json:
# {
#   "mcpServers": {
#     "aimemo-memory": {
#       "command": "/usr/local/bin/aimemo",
#       "args": ["serve"]
#     }
#   }
# }

# 3. Initialize workspace memory
cd ~/.openclaw/workspace
aimemo init

# 4. Restart OpenClaw Gateway
# Linux: systemctl --user restart openclaw-gateway
# macOS: launchctl stop com.openclaw.gateway && launchctl start com.openclaw.gateway
```

### Per-Skill Memory Isolation

Each skill gets its own isolated memory by using the `context` parameter:

**In your SKILL.md:**
```markdown
---
name: my-skill
description: A skill with persistent memory
---

# My Skill

## Instructions

When doing work:

1. **Load memory FIRST**:
   ```
   memory_context({context: "my-skill"})
   ```

2. Do your task with loaded context

3. **Store learnings**:
   ```
   memory_store({
     context: "my-skill",
     entities: [{
       name: "preferences",
       entityType: "config",
       observations: ["User prefers snake_case"]
     }]
   })
   ```

**CRITICAL**: Always pass `context: "my-skill"` to prevent memory pollution.
```

**Result:**
```
~/.openclaw/workspace/.aimemo/
├── memory.db                    # Shared/default (no context)
├── memory-skill-a.db            # Skill A's isolated memory
├── memory-skill-b.db            # Skill B's isolated memory
└── memory-skill-c.db            # Skill C's isolated memory
```

### Complete Example

See [`examples/openclaw-github-pr-reviewer/`](examples/openclaw-github-pr-reviewer/) for a full working skill that:
- Reviews GitHub PRs
- Learns code style preferences
- Remembers patterns across sessions
- Stores feedback for improvement

### Documentation

- **[OpenClaw Integration Guide](docs/openclaw-integration.md)** — Step-by-step setup
- **[OpenClaw Workflow](docs/openclaw-workflow.md)** — Architecture deep-dive
- **[Example Skill](examples/openclaw-github-pr-reviewer/)** — Complete implementation

### Debugging

```bash
# List a skill's memory
aimemo list --context my-skill

# Search within a skill
aimemo search "keyword" --context my-skill

# Export for inspection
aimemo export --context my-skill --format json > memory.json

# Get database stats
aimemo stats --context my-skill
```

## 🖥 Client Support

aimemo works with any MCP-compatible AI coding client. The server command is always `aimemo serve`.

> **PATH note (macOS/Homebrew):** GUI apps may not inherit your shell PATH. If a client can't find `aimemo`, use the absolute path `/opt/homebrew/bin/aimemo` instead.

### Claude Code

```bash
claude mcp add-json aimemo-memory '{"command":"aimemo","args":["serve"]}'
```

Or commit `.mcp.json` to the project root (see the one in this repo as an example).

### Cursor

Project-local (`.cursor/mcp.json`) or global (`~/.cursor/mcp.json`):

```json
{
  "mcpServers": {
    "aimemo-memory": {
      "command": "aimemo",
      "args": ["serve"]
    }
  }
}
```

### Windsurf

Edit `~/.codeium/windsurf/mcp_config.json` (global only):

```json
{
  "mcpServers": {
    "aimemo-memory": {
      "command": "aimemo",
      "args": ["serve"]
    }
  }
}
```

### OpenAI Codex CLI

Project-local (`.codex/config.toml`) or global (`~/.codex/config.toml`):

```toml
[mcp_servers.aimemo-memory]
command = "aimemo"
args    = ["serve"]
```

### Cline (VS Code)

Edit `~/Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json`:

```json
{
  "mcpServers": {
    "aimemo-memory": {
      "command": "aimemo",
      "args": ["serve"],
      "disabled": false,
      "alwaysAllow": []
    }
  }
}
```

### Continue (VS Code / JetBrains)

Project-local (`.continue/mcpServers/aimemo-memory.yaml`):

```yaml
name: aimemo-memory
version: 0.0.1
schema: v1
mcpServers:
  - name: aimemo-memory
    command: aimemo
    args:
      - serve
```

Or add to global `~/.continue/config.yaml` under the `mcpServers:` key.

### Zed

In `~/.zed/settings.json` (global) or `.zed/settings.json` (project-local):

```json
{
  "context_servers": {
    "aimemo-memory": {
      "source": "custom",
      "command": "aimemo",
      "args": ["serve"],
      "env": {}
    }
  }
}
```

## 🤝 Contributing

Bug reports and feature requests go in [GitHub Issues](https://github.com/MyAgentHubs/aimemo/issues). Pull requests are welcome — please open an issue first for anything non-trivial so we can align on direction before you invest time writing code.
