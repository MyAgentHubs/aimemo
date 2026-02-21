# OpenClaw Integration Guide

Get aimemo working with your OpenClaw skills in 5 minutes.

## Table of Contents

- [Why aimemo for OpenClaw?](#why-aimemo-for-openclaw)
- [Quick Start](#quick-start)
- [Creating a Skill with Memory](#creating-a-skill-with-memory)
- [Best Practices](#best-practices)
- [Examples](#examples)
- [FAQ](#faq)

---

## Why aimemo for OpenClaw?

OpenClaw's native Markdown-based memory has known limitations (["remembers everything but understands none"](https://blog.dailydoseofds.com/p/openclaws-memory-is-broken-heres) - community feedback highlighting retrieval and ranking issues). aimemo solves this with:

**✅ Zero Infrastructure**
- Single Go binary, no Docker/Node.js/cloud accounts
- Install in 30 seconds: `curl | bash`
- No database servers to manage

**✅ Built for Skills**
- Per-skill memory isolation (no cross-contamination)
- Simple `context` parameter for complete separation
- Each skill gets its own SQLite database

**✅ Actually Works**
- BM25 full-text search + importance scoring
- Ranks by recency AND access frequency
- Context compression doesn't lose memory

**✅ Developer-Friendly**
- CLI tools for debugging (`aimemo list`, `aimemo search`)
- Export to Markdown/JSON
- Human-readable storage

### vs Other Solutions

| Solution | aimemo | Cognee | memsearch | Supermemory |
|----------|--------|---------|-----------|-------------|
| **Dependencies** | Zero | Neo4j/Kuzu | Milvus | Cloud service |
| **Installation** | 30 sec | Complex | Complex | Sign up |
| **Skill isolation** | Built-in | Manual | Manual | N/A |
| **Privacy** | Local-first | Local | Local | Cloud |
| **Maintenance** | None | High | Medium | N/A |

---

## Quick Start

### 1. Install aimemo

**Linux** (amd64/arm64):
```bash
curl -sSL https://raw.githubusercontent.com/MyAgentHubs/aimemo/main/install.sh | bash
```

**macOS** (Homebrew):
```bash
brew install MyAgentHubs/tap/aimemo
```

**Verify**:
```bash
aimemo --version
# Should show v0.4.0 or later
```

### 2. Register MCP Server

**Option A: Command line** (recommended):
```bash
claude mcp add-json aimemo-memory '{"command":"aimemo","args":["serve"]}'
```

**Option B: Manual config**:

Edit `~/.openclaw/openclaw.json`:
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

> **macOS Note**: If OpenClaw can't find `aimemo`, use absolute path:
> `"command": "/opt/homebrew/bin/aimemo"` (Homebrew Intel)
> `"command": "/usr/local/bin/aimemo"` (Linux/compiled)

### 3. Initialize Memory

```bash
cd ~/.openclaw/workspace
aimemo init
```

Output:
```
Created .aimemo directory
Initialized database: memory.db
✓ Ready to use
```

### 4. Restart OpenClaw

**macOS**:
```bash
# Restart the Gateway
launchctl stop com.openclaw.gateway && launchctl start com.openclaw.gateway
```

**Linux**:
```bash
systemctl --user restart openclaw-gateway
```

### 5. Verify MCP Server

Check OpenClaw logs:
```bash
# macOS
tail -f ~/Library/Logs/OpenClaw/gateway.log

# Linux
journalctl --user -u openclaw-gateway -f
```

Look for:
```
[MCP] Started server: aimemo-memory
[MCP] Discovered tools: memory_context, memory_store, memory_search, ...
```

---

## Creating a Skill with Memory

### Minimal Example

Create `~/.openclaw/workspace/skills/my-skill/SKILL.md`:

```markdown
---
name: my-skill
description: A skill with persistent memory
---

# My Skill with Memory

## Instructions

When the user asks you to remember something:

1. **Load existing memory** (ALWAYS do this first):
   ```
   Call memory_context with: {context: "my-skill"}
   ```

2. **Do your task** using the loaded context

3. **Store new learnings**:
   ```
   Call memory_store with:
   {
     context: "my-skill",
     entities: [{
       name: "entity-name",
       entityType: "knowledge",
       observations: ["New thing you learned"]
     }]
   }
   ```

## Critical Rules

- **ALWAYS pass `context: "my-skill"`** to all memory tools
- Load memory BEFORE doing work
- Store learnings AFTER completing work
```

### Key Points

1. **Context Parameter is Required**
   - Without it, skill uses shared database (risks collision)
   - With it, skill gets isolated `memory-my-skill.db`

2. **Load Memory First**
   - Call `memory_context` at the start of every session
   - Agent has zero memory without this call

3. **Store Progressively**
   - Store learnings as you discover them
   - Don't wait until session end

---

## Best Practices

### 1. Naming Convention

Use the skill name as context:

```markdown
---
name: github-pr-reviewer
---

# Instructions

Call memory_context with: {context: "github-pr-reviewer"}
Call memory_store with: {context: "github-pr-reviewer", ...}
```

### 2. Entity Organization

Use semantic, searchable names:

```javascript
// ✅ GOOD
{
  name: "user-code-style-preferences",
  entityType: "preferences",
  observations: [
    "Prefer snake_case for variables",
    "No trailing commas in arrays"
  ]
}

// ❌ BAD
{
  name: "data",
  entityType: "stuff",
  observations: ["various things"]
}
```

### 3. Journal Entries

Write session summaries:

```javascript
memory_store({
  context: "my-skill",
  journal: "Completed: analyzed 3 PRs. In progress: waiting for user feedback on naming conventions. Blocker: none."
})
```

### 4. Tags for Organization

```javascript
memory_store({
  context: "my-skill",
  entities: [{
    name: "error-handling-rule-1",
    entityType: "rule",
    observations: ["Always wrap errors with context"],
    tags: ["error-handling", "best-practices"]
  }]
})

// Later search by tag
memory_search({
  context: "my-skill",
  tags: ["error-handling"]
})
```

### 5. Importance Clues

Recent + frequently accessed = high importance (automatic).
But you can boost visibility by:
- Referencing entities in journal entries
- Linking related entities with `memory_link`
- Updating observations (refreshes timestamp)

---

## Examples

### Example 1: Code Review Skill

```markdown
---
name: github-pr-reviewer
description: Review PRs with learned style preferences
metadata: {"openclaw": {"requires": {"env": ["GITHUB_TOKEN"]}}}
---

# GitHub PR Reviewer

This skill learns your code style preferences over time.

## Instructions

When asked to review a PR:

1. **Load preferences**:
   ```
   memory_context({context: "github-pr-reviewer"})
   ```

2. **Fetch and review PR** using loaded preferences

3. **Store new patterns** you discover:
   ```
   memory_store({
     context: "github-pr-reviewer",
     entities: [{
       name: "code-style-preferences",
       entityType: "preferences",
       observations: ["User prefers early returns over nested if-else"]
     }]
   })
   ```

4. **End-of-session summary**:
   ```
   memory_store({
     context: "github-pr-reviewer",
     journal: "Reviewed PR #456. Updated: error handling preferences. Next: discuss naming conventions."
   })
   ```
```

### Example 2: Slack Notifier Skill

```markdown
---
name: slack-notifier
description: Send Slack notifications with learned preferences
metadata: {"openclaw": {"requires": {"env": ["SLACK_TOKEN"]}}}
---

# Slack Notifier

Learns when and how you want notifications.

## Instructions

Before sending notifications:

1. **Load notification preferences**:
   ```
   memory_context({context: "slack-notifier"})
   ```

2. **Check preferences** (time windows, channels, formats)

3. **Send notification** if appropriate

4. **Store sent message hash** (for dedup):
   ```
   memory_store({
     context: "slack-notifier",
     entities: [{
       name: "sent-messages",
       entityType: "dedup",
       observations: ["hash:abc123 - daily-digest sent 2026-02-20"]
     }]
   })
   ```

5. **Learn from user feedback**:
   ```
   memory_store({
     context: "slack-notifier",
     entities: [{
       name: "notification-preferences",
       entityType: "preferences",
       observations: ["User disabled notifications during 9am-5pm (work focus time)"]
     }]
   })
   ```
```

### Example 3: JIRA Automation Skill

```markdown
---
name: jira-automation
description: Automate JIRA tickets with learned templates
metadata: {"openclaw": {"requires": {"env": ["JIRA_API_TOKEN"]}}}
---

# JIRA Automation

Creates tickets using learned templates and priority rules.

## Instructions

When creating JIRA tickets:

1. **Load templates and rules**:
   ```
   memory_context({context: "jira-automation"})
   ```

2. **Apply learned templates** for ticket type

3. **Set priority** based on learned rules

4. **Create ticket**

5. **Store template refinements**:
   ```
   memory_store({
     context: "jira-automation",
     entities: [{
       name: "bug-ticket-template",
       entityType: "template",
       observations: [
         "Always include: repro steps, expected behavior, actual behavior",
         "Tag with: bug, needs-triage"
       ]
     }]
   })
   ```
```

---

## FAQ

### Q: Do I need to run `aimemo init` for each skill?

**A**: No. Run it once in `~/.openclaw/workspace`. Each skill automatically gets its own isolated database file.

### Q: What happens if I forget the `context` parameter?

**A**: The skill will use the shared `memory.db` database. This can cause:
- Memory pollution (skill A sees skill B's data)
- Search returning irrelevant results
- Accidental overwrites

**Always use `context: "skill-name"`**.

### Q: Can two skills share memory?

**A**: Yes, use the same context name:

```javascript
// Skill A
memory_store({context: "shared-knowledge", ...})

// Skill B
memory_context({context: "shared-knowledge"})
```

Or omit context entirely (uses `memory.db`).

### Q: How do I debug what's stored?

```bash
# List all memories for a skill
aimemo list --context my-skill

# Search within a skill
aimemo search "keyword" --context my-skill

# Get specific entity
aimemo get entity-name --context my-skill

# Export everything
aimemo export --context my-skill --format json > memory.json
```

### Q: Can I use aimemo outside of OpenClaw?

**A**: Yes! aimemo works with any MCP-compatible client:
- Claude Code
- Cursor
- Windsurf
- Cline
- Continue
- Zed

See [main README](../README.md#client-support) for setup instructions.

### Q: How do I backup memories?

```bash
# Backup all skill memories
tar -czf aimemo-backup-$(date +%Y%m%d).tar.gz \
  ~/.openclaw/workspace/.aimemo/

# Restore
tar -xzf aimemo-backup-20260220.tar.gz -C ~/.openclaw/workspace/
```

### Q: What if memory gets corrupted?

```bash
# Try SQLite recovery
cd ~/.openclaw/workspace/.aimemo
sqlite3 memory-skill-name.db ".recover" | sqlite3 recovered.db

# If unrecoverable, delete and reinitialize
rm memory-skill-name.db
# Skill will auto-create on next use
```

### Q: How big can memories get?

SQLite is tested with databases up to 281TB. For typical skill usage:
- 1000 entities ≈ 1-5 MB
- 10,000 entities ≈ 10-50 MB
- Performance stays fast (< 50ms queries)

No practical limits for skill memories.

### Q: Does aimemo require internet?

**A**: No. Fully offline. All data stays local in SQLite databases.

### Q: Can I edit memories manually?

**A**: Yes, but use CLI tools:

```bash
# Add observation
aimemo observe entity-name "New observation" --context my-skill

# Remove observation
aimemo retract entity-name "Old observation" --context my-skill

# Delete entity
aimemo forget entity-name --context my-skill
```

Avoid editing SQLite directly (risks corruption).

### Q: What if OpenClaw can't find aimemo?

GUI apps may not inherit your shell's PATH. Solutions:

1. **Use absolute path** in `openclaw.json`:
   ```json
   {
     "command": "/usr/local/bin/aimemo"  # Linux
     "command": "/opt/homebrew/bin/aimemo"  # macOS Homebrew
   }
   ```

2. **Check installation**:
   ```bash
   which aimemo
   /usr/local/bin/aimemo
   ```

3. **Verify permissions**:
   ```bash
   ls -l $(which aimemo)
   -rwxr-xr-x  ... aimemo  # Should be executable
   ```

---

## Next Steps

- **Read the workflow guide**: [openclaw-workflow.md](openclaw-workflow.md) for deep dive
- **See example skill**: [examples/openclaw-github-pr-reviewer/](../examples/openclaw-github-pr-reviewer/)
- **Try the CLI**: `aimemo --help` for all commands
- **Join the community**: [GitHub Discussions](https://github.com/MyAgentHubs/aimemo/discussions)

---

## Troubleshooting

If something isn't working:

1. **Check MCP registration**:
   ```bash
   cat ~/.openclaw/openclaw.json | grep -A 5 aimemo
   ```

2. **Test aimemo manually**:
   ```bash
   cd ~/.openclaw/workspace
   aimemo serve
   # Should wait for stdin, not exit
   ```

3. **View OpenClaw logs**:
   ```bash
   # macOS
   tail -f ~/Library/Logs/OpenClaw/gateway.log

   # Linux
   journalctl --user -u openclaw-gateway -f
   ```

4. **Verify database**:
   ```bash
   ls -lh ~/.openclaw/workspace/.aimemo/
   # Should see memory.db and memory-*.db files
   ```

5. **Still stuck?** Open an issue: https://github.com/MyAgentHubs/aimemo/issues
