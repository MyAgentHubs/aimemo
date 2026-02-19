# aimemo Memory Instructions

## Session Start
At the beginning of every session, call `memory_context` to load project context before doing any work.

## During the Session
Call `memory_store` (entities mode) when you:
- Learn something new about the codebase architecture or design decisions
- Fix a bug or identify a root cause
- Make a significant implementation choice

Call `memory_store` (journal mode) when you:
- Complete a meaningful unit of work
- Encounter and resolve a non-obvious problem

## Session End
Before the session ends, write a journal entry summarizing what was done:
```
memory_store({ journal: "..." })
```

## Search Before Asking
Before asking the user to re-explain something, call `memory_search` first.
