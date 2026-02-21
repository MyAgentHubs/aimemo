# GitHub PR Reviewer with aimemo Memory

A complete example OpenClaw skill that demonstrates aimemo integration for persistent memory across sessions.

## What This Skill Does

Reviews GitHub Pull Requests and **learns your code style preferences over time**:

- 🧠 **Remembers** your naming conventions, formatting rules, and patterns
- 📝 **Stores** new discoveries from each review session
- 🔄 **Improves** with every PR reviewed
- 🎯 **Applies** learned preferences consistently

## Prerequisites

1. **aimemo installed** (v0.4.0+):
   ```bash
   # Linux/macOS
   curl -sSL https://raw.githubusercontent.com/MyAgentHubs/aimemo/main/install.sh | bash

   # Or macOS Homebrew
   brew install MyAgentHubs/tap/aimemo
   ```

2. **aimemo MCP server registered** with OpenClaw:
   ```bash
   claude mcp add-json aimemo-memory '{"command":"aimemo","args":["serve"]}'
   ```

3. **Workspace memory initialized**:
   ```bash
   cd ~/.openclaw/workspace
   aimemo init
   ```

4. **GitHub token** (for API access):
   ```bash
   export GITHUB_TOKEN="ghp_your_token_here"
   ```

## Installation

### Option 1: Copy to OpenClaw Skills Directory

```bash
# Copy this skill to your OpenClaw workspace
cp -r examples/openclaw-github-pr-reviewer \
  ~/.openclaw/workspace/skills/
```

### Option 2: Symlink (for development)

```bash
# Symlink for easier updates
ln -s $(pwd)/examples/openclaw-github-pr-reviewer \
  ~/.openclaw/workspace/skills/github-pr-reviewer-aimemo
```

### Option 3: Install from ClawHub (Coming Soon)

```bash
claude skill install github-pr-reviewer-aimemo
```

## Usage

### First Review (Learning Phase)

**You**: "Review PR #123 from my project"

**Skill**:
1. Asks about your basic preferences:
   - "Do you prefer snake_case or camelCase?"
   - "How do you handle errors?"
   - "What's your indentation style?"
2. Reviews the PR
3. Stores your answers for future reviews

### Subsequent Reviews (Using Learned Preferences)

**You**: "Review PR #456"

**Skill**:
1. Automatically loads learned preferences
2. Reviews PR using your established rules
3. Flags issues based on your standards
4. Stores any new patterns discovered

### Correcting Mistakes

**You**: "Actually, I'm fine with trailing commas in arrays"

**Skill**:
1. Updates preference in memory
2. Acknowledges: "Got it, I'll remember that"
3. Applies corrected rule in future reviews

## Example Session

```
You: Review https://github.com/myorg/myproject/pull/789

Skill: [Loads memory: "User prefers snake_case, explicit error types, 4-space indent"]

Skill: I've reviewed PR #789. Here's my feedback:

## Summary
Request changes - found 3 style issues

## Key Issues
1. Function `getUserData` should be `get_user_data` (snake_case preference)
2. Error on line 45 should have explicit type (AppError expected)
3. Indentation inconsistent (mix of 2 and 4 spaces)

## Suggestions
- Consider adding tests for the new validation logic
- The error message on line 67 could be more descriptive

## Positive Notes
- Good use of early returns
- Clear variable names
- Well-documented function

[Stores: "User wants consistent 4-space indent", "Validation logic needs tests"]

You: Great! Also, for future reviews, I want you to always check if the changelog was updated.

Skill: Understood! I've added "Verify CHANGELOG.md updated" to my review checklist. I'll check this in all future PR reviews.

[Stores: "Must verify CHANGELOG.md updated" in review-checklist]
```

## What Gets Stored

### Code Style Preferences
- Naming conventions (snake_case, camelCase, PascalCase)
- Indentation (spaces/tabs, 2/4 spaces)
- Formatting (trailing commas, semicolons, line breaks)

### Patterns & Best Practices
- Error handling approaches
- File organization patterns
- Testing standards
- Documentation requirements

### Anti-Patterns to Flag
- Specific issues user wants avoided
- Project-specific no-nos
- Security concerns

### User Corrections
- When you correct the skill's feedback
- Clarifications on preferences
- Exception cases

### Project Context
- Tech stack info
- Team conventions
- Domain-specific rules

## Memory Structure

All data is stored in:
```
~/.openclaw/workspace/.aimemo/memory-github-pr-reviewer-aimemo.db
```

**Entities**:
- `code-style-preferences`: Naming, formatting rules
- `error-handling-patterns`: How to handle errors
- `common-mistakes`: Things to always flag
- `user-corrections`: Feedback from user
- `project-context`: Project-specific info
- `review-checklist`: Things to always check

**Journal entries**: Session summaries showing learning progress

## Inspecting Memory

### List all stored preferences

```bash
aimemo list --context github-pr-reviewer-aimemo
```

### Search for specific topic

```bash
aimemo search "error handling" --context github-pr-reviewer-aimemo
```

### Get specific entity

```bash
aimemo get code-style-preferences --context github-pr-reviewer-aimemo
```

### Export everything

```bash
aimemo export --context github-pr-reviewer-aimemo --format json > memory.json
```

### Read journal entries

```bash
aimemo search --journal --since 7d --context github-pr-reviewer-aimemo
```

## Customization

### Modify SKILL.md

Edit the instructions to:
- Add language-specific rules (Python, Go, TypeScript, etc.)
- Include framework-specific checks (React, Django, etc.)
- Add CI/CD integration checks
- Customize review format

### Add Custom Entity Types

```markdown
In SKILL.md instructions:

Store security-specific rules:
memory_store({
  context: "github-pr-reviewer-aimemo",
  entities: [{
    name: "security-rules",
    entityType: "security",  // Custom type
    observations: ["Never commit .env files"],
    tags: ["security", "critical"]
  }]
})
```

## Troubleshooting

### Memory not loading

**Check if database exists**:
```bash
ls -lh ~/.openclaw/workspace/.aimemo/memory-github-pr-reviewer-aimemo.db
```

**Verify context parameter** in SKILL.md (must be exact):
```markdown
{context: "github-pr-reviewer-aimemo"}
```

### Skill forgets preferences

**Ensure `memory_store` is called**:
- Check OpenClaw logs for successful store operations
- Verify database is writable: `touch ~/.openclaw/workspace/.aimemo/test && rm ~/.openclaw/workspace/.aimemo/test`

### Wrong preferences loaded

**Update incorrect preference**:
```bash
# Via CLI
aimemo observe code-style-preferences "CORRECTION: Use camelCase not snake_case" \
  --context github-pr-reviewer-aimemo

# Or let skill overwrite in next session
```

### Database corruption

**Recover** (rare):
```bash
cd ~/.openclaw/workspace/.aimemo
sqlite3 memory-github-pr-reviewer-aimemo.db ".recover" | \
  sqlite3 memory-recovered.db
mv memory-github-pr-reviewer-aimemo.db memory-backup.db
mv memory-recovered.db memory-github-pr-reviewer-aimemo.db
```

**Start fresh**:
```bash
rm ~/.openclaw/workspace/.aimemo/memory-github-pr-reviewer-aimemo.db
# Skill will auto-create on next use
```

## Advanced Usage

### Multiple Projects

Use different skill instances per project:

```bash
# Install multiple times with different names
cp SKILL.md ~/.openclaw/workspace/skills/pr-reviewer-project-a/SKILL.md
cp SKILL.md ~/.openclaw/workspace/skills/pr-reviewer-project-b/SKILL.md

# Each gets isolated memory:
# memory-pr-reviewer-project-a.db
# memory-pr-reviewer-project-b.db
```

### Team Shared Preferences

Export and share baseline preferences:

```bash
# Export your preferences
aimemo export --context github-pr-reviewer-aimemo --format json > team-standards.json

# Team member imports
aimemo import team-standards.json --context github-pr-reviewer-aimemo
```

### Periodic Review

Check what skill has learned:

```bash
# Weekly review
aimemo list --context github-pr-reviewer-aimemo | less

# See recent changes
aimemo search --journal --since 7d --context github-pr-reviewer-aimemo
```

## Learning Progress Example

### After 1 Review
```
code-style-preferences:
  - Prefer snake_case for variables
  - 4-space indentation
```

### After 5 Reviews
```
code-style-preferences:
  - Prefer snake_case for variables
  - 4-space indentation
  - No trailing commas in objects
  - PascalCase for classes

error-handling-patterns:
  - Always use explicit error types (AppError)
  - Wrap third-party errors with context
```

### After 20 Reviews
```
code-style-preferences: 12 observations
error-handling-patterns: 8 observations
common-mistakes: 15 observations
review-checklist: 7 items
project-context: 10 observations

Journal entries: 20 sessions documented
```

The skill becomes increasingly valuable as its knowledge base grows.

## Contributing

Found an improvement? Submit a PR to:
https://github.com/MyAgentHubs/aimemo

Ideas for enhancements:
- Language-specific templates
- Integration with linters (ESLint, Pylint, etc.)
- Auto-formatting suggestions
- Security scanning integration
- Performance analysis

## License

MIT - See [LICENSE](../../LICENSE)

## See Also

- [aimemo Documentation](../../README.md)
- [OpenClaw Integration Guide](../../docs/openclaw-integration.md)
- [OpenClaw Workflow Details](../../docs/openclaw-workflow.md)
