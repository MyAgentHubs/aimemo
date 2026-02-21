# OpenClaw Integration Workflow

This document explains how aimemo works with OpenClaw skills, including the complete architecture, data flow, and runtime behavior.

## Table of Contents

- [Architecture Overview](#architecture-overview)
- [Initialization Flow](#initialization-flow)
- [Runtime Call Chain](#runtime-call-chain)
- [File System Layout](#file-system-layout)
- [Memory Isolation](#memory-isolation)
- [Data Flow](#data-flow)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

---

## Architecture Overview

```
┌──────────────────────────────────────────────────────────────────┐
│                    OpenClaw Gateway Process                       │
│                                                                   │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │               Agent Runtime                              │    │
│  │                                                          │    │
│  │  User Request                                            │    │
│  │    ↓                                                     │    │
│  │  [Agent matches request to skill]                       │    │
│  │    ↓                                                     │    │
│  │  ┌─────────────────────────────────────────────┐       │    │
│  │  │  Skill: github-pr-reviewer                  │       │    │
│  │  │  (SKILL.md instructions)                    │       │    │
│  │  │                                              │       │    │
│  │  │  → Load memory:                             │       │    │
│  │  │    memory_context({                         │       │    │
│  │  │      context: "github-pr-reviewer"          │       │    │
│  │  │    })                                        │       │    │
│  │  └──────────────────┬──────────────────────────┘       │    │
│  │                     │                                   │    │
│  │                     │ JSON-RPC over stdio               │    │
│  │                     ▼                                   │    │
│  │  ┌──────────────────────────────────────────────────┐  │    │
│  │  │         MCP Client (Gateway-managed)             │  │    │
│  │  │                                                   │  │    │
│  │  │  Request:                                        │  │    │
│  │  │  {                                               │  │    │
│  │  │    "method": "tools/call",                       │  │    │
│  │  │    "params": {                                   │  │    │
│  │  │      "name": "memory_context",                   │  │    │
│  │  │      "arguments": {                              │  │    │
│  │  │        "context": "github-pr-reviewer"           │  │    │
│  │  │      }                                           │  │    │
│  │  │    }                                             │  │    │
│  │  │  }                                               │  │    │
│  │  └───────────────────┬──────────────────────────────┘  │    │
│  └────────────────────────┼─────────────────────────────────┘    │
│                           │                                      │
│                           │ stdio (stdin/stdout)                 │
│                           │                                      │
└───────────────────────────┼──────────────────────────────────────┘
                            │
                            ▼
┌───────────────────────────────────────────────────────────────────┐
│                  aimemo serve (MCP Server)                         │
│                  CWD: ~/.openclaw/workspace                        │
│                                                                    │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │  1. Receive MCP request                                  │    │
│  │     {context: "github-pr-reviewer"}                      │    │
│  │                                                           │    │
│  │  2. Call FindProjectDB("github-pr-reviewer")             │    │
│  │     → Walk up from CWD looking for .aimemo/              │    │
│  │     → Found: ~/.openclaw/workspace/.aimemo/              │    │
│  │     → Return: .../memory-github-pr-reviewer.db           │    │
│  │                                                           │    │
│  │  3. Open database                                         │    │
│  │     db.Open(".../memory-github-pr-reviewer.db")          │    │
│  │                                                           │    │
│  │  4. Execute query                                         │    │
│  │     - Calculate importance score (recency + access)      │    │
│  │     - BM25 full-text search                              │    │
│  │     - Return top 20 observations                         │    │
│  │                                                           │    │
│  │  5. Return result                                         │    │
│  │     {                                                     │    │
│  │       "content": [{                                       │    │
│  │         "type": "text",                                   │    │
│  │         "text": "Last session: ...\n                      │    │
│  │                  Preferences: snake_case, no trailing ," │    │
│  │       }]                                                  │    │
│  │     }                                                     │    │
│  └──────────────────────────────────────────────────────────┘    │
│                                                                    │
└────────────────────────────┬───────────────────────────────────────┘
                             │
                             │ JSON-RPC Response
                             ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Agent receives memory context                 │
│                                                                  │
│  → Use loaded preferences to perform task                       │
│  → Store new discoveries:                                       │
│                                                                  │
│    memory_store({                                               │
│      context: "github-pr-reviewer",                             │
│      entities: [{                                               │
│        name: "code-style-preferences",                          │
│        entityType: "preferences",                               │
│        observations: ["User prefers early returns"]             │
│      }]                                                         │
│    })                                                           │
└─────────────────────────────────────────────────────────────────┘
```

### Key Components

1. **OpenClaw Gateway**: Manages the agent lifecycle and MCP server connections
2. **Agent Runtime**: Executes skills based on user requests
3. **MCP Client**: Bridges skills to MCP servers via JSON-RPC
4. **aimemo serve**: MCP server providing memory tools
5. **SQLite Database**: Persistent storage for each skill's memory

---

## Initialization Flow

### One-Time Setup (Per Machine)

```
┌────────────────────────────────────────────────────────────┐
│ Step 1: Install aimemo                                     │
│                                                             │
│ Linux/macOS:                                               │
│   curl -sSL https://raw.githubusercontent.com/MyAgentHubs/aimemo/main/install.sh | bash           │
│                                                             │
│ Or via Homebrew (macOS):                                   │
│   brew install MyAgentHubs/tap/aimemo                      │
│                                                             │
│ Verify:                                                     │
│   aimemo --version                                         │
└────────────────────────────────────────────────────────────┘
         ↓
┌────────────────────────────────────────────────────────────┐
│ Step 2: Register MCP server with OpenClaw                  │
│                                                             │
│   claude mcp add-json aimemo-memory \                      │
│     '{"command":"aimemo","args":["serve"]}'                │
│                                                             │
│ Or add to ~/.openclaw/openclaw.json:                       │
│   {                                                         │
│     "mcpServers": {                                         │
│       "aimemo-memory": {                                    │
│         "command": "aimemo",                                │
│         "args": ["serve"]                                   │
│       }                                                     │
│     }                                                       │
│   }                                                         │
└────────────────────────────────────────────────────────────┘
         ↓
┌────────────────────────────────────────────────────────────┐
│ Step 3: Initialize workspace memory                        │
│                                                             │
│   cd ~/.openclaw/workspace                                 │
│   aimemo init                                              │
│                                                             │
│ Creates: ~/.openclaw/workspace/.aimemo/memory.db           │
└────────────────────────────────────────────────────────────┘
         ↓
┌────────────────────────────────────────────────────────────┐
│ Step 4: Restart OpenClaw                                   │
│                                                             │
│ The Gateway will now spawn aimemo serve on startup         │
└────────────────────────────────────────────────────────────┘
```

### Per-Skill Setup (Optional)

If a skill wants isolated memory, it should declare in SKILL.md:

```markdown
## Setup

This skill uses aimemo for persistent memory. The memory is automatically
isolated using the context parameter - no manual setup required.

On first use, aimemo will create:
  ~/.openclaw/workspace/.aimemo/memory-{skill-name}.db
```

---

## Runtime Call Chain

### Request Flow

```
User says: "Review this PR with my code style preferences"
    ↓
┌─────────────────────────────────────────────────────────┐
│ 1. OpenClaw Agent matches to github-pr-reviewer skill   │
└───────────────────────┬─────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│ 2. Skill instruction: Load memory first                 │
│    → Call memory_context({context: "github-pr-..."})   │
└───────────────────────┬─────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│ 3. MCP Client sends JSON-RPC request to aimemo serve    │
│    Protocol: stdio (stdin/stdout)                       │
└───────────────────────┬─────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│ 4. aimemo serve processes request:                      │
│    a. Parse context: "github-pr-reviewer"               │
│    b. Find DB: memory-github-pr-reviewer.db             │
│    c. Query: SELECT with importance scoring             │
│    d. Return: Top observations + journal entries        │
└───────────────────────┬─────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│ 5. Agent receives memory context                        │
│    Example: "Last session: prefer snake_case..."        │
└───────────────────────┬─────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│ 6. Agent performs task with loaded context              │
│    → Fetch PR code                                      │
│    → Review with learned preferences                    │
│    → Provide feedback to user                           │
└───────────────────────┬─────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│ 7. Agent stores new learnings                           │
│    → Call memory_store({context: "...", entities: ...}) │
└───────────────────────┬─────────────────────────────────┘
                        ↓
┌─────────────────────────────────────────────────────────┐
│ 8. aimemo persists to SQLite                            │
│    → Update observations table                          │
│    → Update access timestamps                           │
│    → Recalculate importance scores                      │
└─────────────────────────────────────────────────────────┘
```

### Timing (Typical)

- MCP call overhead: < 10ms
- Memory query (cold): < 50ms
- Memory query (cached): < 5ms
- Memory store: < 20ms

---

## File System Layout

```
~/.openclaw/
├── openclaw.json                  # OpenClaw config (MCP registration)
│   {
│     "mcpServers": {
│       "aimemo-memory": {
│         "command": "aimemo",
│         "args": ["serve"]
│       }
│     }
│   }
│
└── workspace/
    ├── .aimemo/                   # aimemo memory root
    │   ├── memory.db              # Default/shared memory
    │   ├── memory-github-pr-reviewer.db      # ← Skill A
    │   ├── memory-slack-notifier.db          # ← Skill B
    │   └── memory-jira-automation.db         # ← Skill C
    │
    └── skills/                    # OpenClaw skills
        ├── github-pr-reviewer/
        │   └── SKILL.md           # ← Declares aimemo usage
        │
        ├── slack-notifier/
        │   └── SKILL.md
        │
        └── jira-automation/
            └── SKILL.md
```

### Database File Naming Convention

- No context parameter: `memory.db`
- With context parameter: `memory-{context}.db`

Example:
```javascript
// Uses memory.db (default/shared)
memory_context({})

// Uses memory-github-pr-reviewer.db (isolated)
memory_context({context: "github-pr-reviewer"})
```

**Context Name Requirements:**
- Context names are sanitized to `[a-z0-9-]` (lowercase alphanumeric and hyphens)
- Invalid characters are replaced with `-`, leading/trailing `-` are trimmed
- Empty or invalid context names default to `memory.db`

Examples:
```
"My-Skill"      → memory-my-skill.db
"skill_name"    → memory-skill-name.db
"SKILL@123"     → memory-skill-123.db
"---"           → memory.db (sanitizes to empty, uses default)
```

---

## Memory Isolation

### Why Per-Skill Isolation Matters

```
┌────────────────────────────────────────────────────────┐
│  Skill A: github-pr-reviewer                           │
│  context: "github-pr-reviewer"                         │
│     ↓                                                  │
│  memory-github-pr-reviewer.db                          │
│  ├─ code-style-preferences                             │
│  ├─ review-patterns                                    │
│  └─ user-feedback-history                              │
│                                                         │
│  ✅ Only sees its own data                             │
│  ❌ Cannot see slack-notifier's data                   │
└────────────────────────────────────────────────────────┘

┌────────────────────────────────────────────────────────┐
│  Skill B: slack-notifier                               │
│  context: "slack-notifier"                             │
│     ↓                                                  │
│  memory-slack-notifier.db                              │
│  ├─ notification-preferences                           │
│  ├─ sent-message-hashes (dedup)                        │
│  └─ user-timezone-settings                             │
│                                                         │
│  ✅ Only sees its own data                             │
│  ❌ Cannot see github-pr-reviewer's data               │
└────────────────────────────────────────────────────────┘
```

### How Isolation Works

1. **Context Parameter**: Each skill passes its unique identifier
2. **Separate Databases**: Different `.db` files prevent cross-contamination
3. **FTS Isolation**: Full-text search only queries the skill's database
4. **No Shared State**: Skills cannot accidentally read/write each other's memory

### When to Share Memory

Use the default database (no context parameter) for:
- Cross-skill shared knowledge
- Project-wide facts
- User preferences that apply globally

Example:
```javascript
// Shared: all skills can access
memory_store({
  entities: [{
    name: "user-timezone",
    entityType: "preference",
    observations: ["UTC+8"]
  }]
})

// Isolated: only slack-notifier can access
memory_store({
  context: "slack-notifier",
  entities: [{
    name: "notification-times",
    entityType: "schedule",
    observations: ["Send daily digest at 9am"]
  }]
})
```

---

## Data Flow

### Memory Store Flow

```
Skill discovers new pattern
        ↓
memory_store({
  context: "github-pr-reviewer",
  entities: [{
    name: "code-style",
    entityType: "preferences",
    observations: ["Use early returns"]
  }]
})
        ↓
┌─────────────────────────────────────────┐
│ aimemo MCP server                       │
│                                          │
│ 1. Parse request                         │
│    context = "github-pr-reviewer"       │
│                                          │
│ 2. Open/create DB                       │
│    memory-github-pr-reviewer.db         │
│                                          │
│ 3. Check if entity exists                │
│    SELECT * WHERE name='code-style'     │
│                                          │
│ 4. Upsert entity                         │
│    INSERT OR UPDATE                     │
│                                          │
│ 5. Add observations                      │
│    - Deduplicate                        │
│    - Update timestamps                  │
│    - Recalculate importance             │
│                                          │
│ 6. Update FTS index                      │
│    INSERT INTO fts_observations         │
└─────────────────────────────────────────┘
        ↓
Database persisted
```

### Memory Context Flow

```
Skill starts work
        ↓
memory_context({
  context: "github-pr-reviewer"
})
        ↓
┌─────────────────────────────────────────┐
│ aimemo MCP server                       │
│                                          │
│ 1. Open DB                               │
│    memory-github-pr-reviewer.db         │
│                                          │
│ 2. Calculate importance scores           │
│    score = 0.6/LOG(hours_ago+2)         │
│          + 0.4*LOG(access_count+1)      │
│                                          │
│ 3. Query recent + important              │
│    - Last 24h by default                │
│    - Top 20 by importance               │
│    - Include journal entries            │
│                                          │
│ 4. Format response                       │
│    - Group by entity                    │
│    - Show relationships                 │
│    - Include metadata                   │
└─────────────────────────────────────────┘
        ↓
Skill receives context
```

### Memory Search Flow

```
Skill needs specific info
        ↓
memory_search({
  context: "github-pr-reviewer",
  query: "error handling"
})
        ↓
┌─────────────────────────────────────────┐
│ aimemo MCP server                       │
│                                          │
│ 1. Open DB                               │
│    memory-github-pr-reviewer.db         │
│                                          │
│ 2. FTS5 search                           │
│    SELECT * FROM fts_observations       │
│    WHERE fts_observations MATCH 'error' │
│    AND fts_observations MATCH 'handling'│
│                                          │
│ 3. BM25 ranking                          │
│    - Term frequency                     │
│    - Inverse document frequency         │
│    - Document length normalization      │
│                                          │
│ 4. Apply importance weights              │
│    final_score = bm25 * importance      │
│                                          │
│ 5. Return top results                    │
│    - Limit: 10 (default)                │
│    - Include context snippets           │
└─────────────────────────────────────────┘
        ↓
Skill receives search results
```

---

## Best Practices

### For Skill Authors

#### 1. Always Use Context Parameter

```markdown
✅ GOOD:
memory_context({context: "my-skill-name"})

❌ BAD:
memory_context({})  // Uses shared DB, risks collision
```

#### 2. Load Memory First

```markdown
## Instructions

When the user asks you to [do task]:

1. **FIRST**: Load memory
   memory_context({context: "my-skill-name"})

2. Then proceed with the task

3. Store learnings before finishing
```

#### 3. Store Progressively

```javascript
// ✅ Store as you learn
memory_store({
  context: "my-skill",
  entities: [{name: "pattern-1", ...}]
})

// Later...
memory_store({
  context: "my-skill",
  entities: [{name: "pattern-2", ...}]
})

// ❌ Don't try to store everything at once
// (risks losing data if interrupted)
```

#### 4. Write Journal Entries

```javascript
// At session end
memory_store({
  context: "my-skill",
  journal: "Completed: task X. In progress: task Y. Blocker: Z."
})
```

#### 5. Use Semantic Entity Names

```javascript
// ✅ GOOD: Clear, searchable
{name: "user-code-style-preferences", entityType: "preferences"}
{name: "common-bug-patterns", entityType: "knowledge"}

// ❌ BAD: Vague, hard to search
{name: "data", entityType: "stuff"}
{name: "temp", entityType: "thing"}
```

### For OpenClaw Administrators

#### 1. Backup Memory Databases

```bash
# Backup all skill memories
tar -czf aimemo-backup-$(date +%Y%m%d).tar.gz \
  ~/.openclaw/workspace/.aimemo/
```

#### 2. Monitor Database Size

```bash
# Check size of all memory databases
du -sh ~/.openclaw/workspace/.aimemo/*.db
```

#### 3. Inspect Memory (Debugging)

```bash
# List memories for a specific skill
aimemo list --context github-pr-reviewer

# Search within a skill's memory
aimemo search "error handling" --context github-pr-reviewer

# Export for inspection
aimemo export --context github-pr-reviewer --format json > memory.json
```

#### 4. Clean Up Old Memories

```bash
# Soft-delete an entity (recoverable)
aimemo forget entity-name --context github-pr-reviewer

# Hard-delete permanently
aimemo forget entity-name --context github-pr-reviewer --permanent
```

---

## Troubleshooting

### Problem: Skill can't find aimemo

**Symptoms**: Error like "tool 'memory_context' not found"

**Solutions**:
1. Verify MCP server registration:
   ```bash
   cat ~/.openclaw/openclaw.json | grep aimemo
   ```

2. Check if aimemo is in PATH:
   ```bash
   which aimemo
   ```

3. Try absolute path in config:
   ```json
   {
     "mcpServers": {
       "aimemo-memory": {
         "command": "/usr/local/bin/aimemo",
         "args": ["serve"]
       }
     }
   }
   ```

### Problem: Memory not persisting

**Symptoms**: Skill seems to forget learnings between sessions

**Solutions**:
1. Verify database exists:
   ```bash
   ls -lh ~/.openclaw/workspace/.aimemo/
   ```

2. Check database is writable:
   ```bash
   touch ~/.openclaw/workspace/.aimemo/test && rm ~/.openclaw/workspace/.aimemo/test
   ```

3. Verify skill is using correct context:
   - Check SKILL.md instructions
   - Ensure context parameter matches skill name

### Problem: Skills sharing memory unexpectedly

**Symptoms**: Skill A sees data from Skill B

**Solution**:
- Both skills are missing context parameter
- Add `context: "skill-name"` to all memory tool calls

### Problem: MCP server not starting

**Symptoms**: OpenClaw logs show "failed to start MCP server"

**Solutions**:
1. Test aimemo serve manually:
   ```bash
   cd ~/.openclaw/workspace
   aimemo serve
   # Should wait for stdin, not exit immediately
   ```

2. Check aimemo version:
   ```bash
   aimemo --version
   # Should be v0.4.0 or later for OpenClaw support
   ```

3. Restart OpenClaw Gateway:
   ```bash
   # macOS
   launchctl stop com.openclaw.gateway
   launchctl start com.openclaw.gateway

   # Linux
   systemctl --user restart openclaw-gateway
   ```

### Problem: Database corruption

**Symptoms**: Error like "database disk image is malformed"

**Solutions**:
1. Try SQLite recovery:
   ```bash
   cd ~/.openclaw/workspace/.aimemo
   sqlite3 memory-skill-name.db ".recover" | sqlite3 memory-skill-name-recovered.db
   mv memory-skill-name.db memory-skill-name.db.backup
   mv memory-skill-name-recovered.db memory-skill-name.db
   ```

2. If unrecoverable, restore from backup or reinitialize:
   ```bash
   rm ~/.openclaw/workspace/.aimemo/memory-skill-name.db
   # Skill will auto-create on next use
   ```

### Problem: Search returns irrelevant results

**Symptoms**: `memory_search` returns unrelated observations

**Solutions**:
1. Use more specific queries:
   ```javascript
   // ❌ Too broad
   memory_search({query: "code"})

   // ✅ Specific
   memory_search({query: "error handling patterns"})
   ```

2. Filter by entity type:
   ```javascript
   memory_search({
     query: "preferences",
     type: "preferences"  // Only search preferences
   })
   ```

3. Use tags for better organization:
   ```javascript
   memory_store({
     entities: [{
       name: "style-rule-1",
       observations: ["..."],
       tags: ["code-style", "naming"]
     }]
   })

   // Later search by tag
   memory_search({tags: ["code-style"]})
   ```

---

## Advanced Topics

### Custom Memory Contexts

You can create hierarchical memory by using dotted context names:

```javascript
// Project-level memory
memory_context({context: "myproject"})

// Feature-specific memory
memory_context({context: "myproject.auth"})
memory_context({context: "myproject.billing"})
```

Note: These are still independent databases - aimemo doesn't currently support hierarchical queries.

### Sharing Memory Between Skills

If two skills need to share specific knowledge:

1. **Option A**: Use a shared context
   ```javascript
   // Both skills use same context
   memory_store({
     context: "shared-knowledge",
     entities: [...]
   })
   ```

2. **Option B**: Use the default database
   ```javascript
   // No context = shared
   memory_store({
     entities: [...]
   })
   ```

### Performance Tuning

For skills with large memories (>1000 entities):

1. **Adjust limit in queries**:
   ```javascript
   memory_context({
     context: "my-skill",
     limit: 50  // Default is 20
   })
   ```

2. **Use targeted searches**:
   ```javascript
   // Instead of loading all context
   memory_search({
     context: "my-skill",
     query: "specific topic",
     limit: 5
   })
   ```

3. **Archive old data**:
   ```bash
   # Export old memories
   aimemo export --context my-skill > backup.json

   # Clear database
   rm ~/.openclaw/workspace/.aimemo/memory-my-skill.db

   # Re-import only recent data
   # (manual filtering required)
   ```

---

## See Also

- [OpenClaw Integration Guide](openclaw-integration.md) - Step-by-step setup
- [Main README](../README.md) - aimemo overview
- [CLI Reference](../README.md#cli-reference) - Command-line tools
- [Example Skill](../examples/openclaw-github-pr-reviewer/) - Complete working example
