# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.4.0] - 2026-02-20

### Added

- **OpenClaw Integration Documentation**
  - [OpenClaw Integration Guide](docs/openclaw-integration.md) with 5-minute setup
  - [OpenClaw Workflow](docs/openclaw-workflow.md) with complete architecture diagrams
  - Per-skill memory isolation using `context` parameter
  - Comparison with other memory solutions (Cognee, memsearch, Supermemory)

- **Linux Support**
  - One-line install script (`install.sh`) for Linux amd64/arm64
  - Automatic architecture detection
  - Post-install instructions and troubleshooting
  - Native static binaries (CGO_ENABLED=0)

- **Example OpenClaw Skill**
  - Complete GitHub PR reviewer skill ([examples/openclaw-github-pr-reviewer/](examples/openclaw-github-pr-reviewer/))
  - Demonstrates code style preference learning
  - Shows pattern recognition across sessions
  - Includes debugging and customization guide

- **AI Agent Documentation**
  - `llms.txt` file following [llmstxt.org](https://llmstxt.org/) specification
  - Machine-readable project documentation
  - Optimized for AI agent consumption
  - Includes all tools, commands, and integration guides

### Improved

- **README Updates (English & Chinese)**
  - Added OpenClaw Quick Start section
  - Added Linux one-line installation option
  - Added per-skill memory isolation explanation
  - Added file system layout diagrams
  - Added comparison table with alternative solutions

- **Documentation Structure**
  - Created `docs/` directory for comprehensive guides
  - Created `examples/` directory for working skill implementations
  - Improved navigation with clear section hierarchy

### Changed

- Quick Start section now shows both Homebrew and curl installation methods
- Client Support section reorganized for better clarity

### Technical

- Verified Linux amd64/arm64 build compatibility
- Tested context parameter creates isolated databases correctly
- Confirmed pure Go SQLite (modernc.org/sqlite) works on Linux
- Validated goreleaser configuration for multi-platform releases

## [0.3.0] - 2026-02-17

### Added

- Initial public release
- MCP stdio server implementation
- Five core memory tools: `memory_context`, `memory_store`, `memory_search`, `memory_forget`, `memory_link`
- SQLite + FTS5 full-text search with BM25 ranking
- Importance scoring (recency + access frequency)
- Context parameter for memory isolation
- CLI tools for human inspection and management
- Configuration system (global and per-project)
- WAL mode for concurrent access
- Journal entry support
- Entity and observation management
- Tag-based organization
- Relationship linking
- Export/import (Markdown and JSON)
- Client support for Claude Code, Cursor, Windsurf, Cline, Continue, Zed
- Homebrew formula for macOS installation
- Comprehensive README documentation (English and Chinese)
- MIT License

### Technical

- Go 1.22+ required
- Zero runtime dependencies
- Single binary distribution
- Pure Go SQLite via modernc.org/sqlite
- MCP protocol version 2024-11-05
- 5-second timeout per MCP call
- < 2000 token schema overhead

---

## Release Links

- [v0.4.0](https://github.com/MyAgentHubs/aimemo/releases/tag/v0.4.0) - OpenClaw Integration Release
- [v0.3.0](https://github.com/MyAgentHubs/aimemo/releases/tag/v0.3.0) - Initial Release
