# aimemo Memory Instructions

## Session Start

At the beginning of every session, call `memory_context` to load project
context before doing any work. This gives you:
- Recent observations about entities in this project
- Top entities by importance score
- Recent journal entries from past sessions
- Current storage stats

If `memory_context` returns nothing useful, proceed normally — the memory
starts empty and builds up over time.

## During the Session

Call `memory_store` (entities mode) when you discover or confirm:
- Architecture decisions ("uses Redis for session store, not DB")
- Root causes of bugs you fixed ("race condition in auth middleware caused by X")
- Non-obvious constraints ("file uploads must be under 10MB due to Lambda limits")
- Key entities and their relationships (services, libraries, people, concepts)

```
memory_store({
  entities: [{
    name: "AuthService",
    entityType: "service",
    observations: [
      "Handles JWT issuance and validation",
      "Tokens expire after 15 minutes; refresh tokens last 30 days"
    ],
    tags: ["auth", "backend"]
  }]
})
```

Call `memory_store` (journal mode) when you:
- Complete a meaningful unit of work
- Encounter and resolve a non-obvious problem
- Make a significant decision the next session should know about

```
memory_store({
  journal: "Refactored auth middleware to use context.Context for timeout propagation. Root cause was missing cancel() call leaking goroutines."
})
```

## Session End

Before ending the session, write a brief journal entry:

```
memory_store({
  journal: "Implemented user profile page. Added Avatar, ProfileForm components. API endpoint is POST /api/profile. Still TODO: email validation on the backend."
})
```

This is the most important habit — it's what lets the next session pick up
exactly where you left off.

## Search Before Asking

Before asking the user to re-explain something, call `memory_search` first:

```
memory_search({ query: "database connection pooling" })
memory_search({ name: "PostgreSQL" })
memory_search({ journal: true, since: "7d" })
```

## What to Store

**Good candidates:**
- Decisions with non-obvious rationale ("chose X over Y because...")
- Bug root causes that took effort to find
- System constraints and limits
- Key people, services, files, and how they relate

**Skip:**
- Secrets, credentials, or PII
- Transient debug output or log snippets
- Things that are obvious from the code itself
- Speculative ideas that weren't acted on
