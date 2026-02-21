# aimemo

[![Go 1.22+](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org/) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat)](LICENSE) [![Release](https://img.shields.io/github/v/release/MyAgentHubs/aimemo?style=flat)](https://github.com/MyAgentHubs/aimemo/releases)

[English](README.md) | [中文](README.zh-CN.md)

零依赖的 AI Agent MCP 记忆服务器——持久化、可搜索、本地优先、单一二进制文件。

```
$ claude "继续搞支付服务"

  ╭─ memory_context ──────────────────────────────────────────────────╮
  │ [project: payment-service]                                        │
  │                                                                   │
  │ 上次会话（3 天前）：                                              │
  │  • Stripe webhook 签名验证 — 已完成                               │
  │  • 幂等键重构 — 进行中                                            │
  │  • 卡点：并发退款处理器的竞态条件                                 │
  │                                                                   │
  │ 相关：Redis 连接池, pkg/payments/refund.go                        │
  ╰───────────────────────────────────────────────────────────────────╯

  接着上次的继续。退款处理器的竞态条件看起来是 in-flight map 缺少
  mutex。先看一下 pkg/payments/refund.go ...

  [... Claude 定位并修复问题 ...]

  ╭─ memory_store (journal) ──────────────────────────────────────────╮
  │ 修复退款竞态——给 inFlightRefunds 加了 sync.Mutex。               │
  │ 测试全过。下一步：用 k6 在 500 rps 下做压测。                    │
  ╰───────────────────────────────────────────────────────────────────╯
```

## 🧠 为什么选 aimemo

- **不需要维护任何基础设施。** 单个 Go 二进制文件，不依赖 Docker、Node.js 运行时、云账号或 API Key。`brew install` 30 秒搞定。
- **记忆跟着项目走。** 存储在项目目录下的 `.aimemo/`，可以提交到 git 也可以加进 `.gitignore`，随你决定。切分支、记忆不丢。
- **Claude 能从上次断点继续。** 每次会话开始时自动调用 `memory_context`，Claude 能看到上次在做什么、卡在哪里、做了哪些决定。你不需要每次重新解释背景。
- **全文搜索真的有用。** FTS5 + BM25 排序，同时考虑时效性和访问频率。相关记忆排在前面，久远的噪音自然淡化。
- **多个 Claude 窗口同时写，不会数据损坏。** SQLite WAL 模式支持并发写入，多个会话同时操作互不干扰。
- **你始终掌控数据。** Claude 能做的操作，你在终端里也能做。查看、编辑、撤回、导出。记忆以 Markdown 或 JSON 格式存储，永远不会被锁死在专有格式里。

## ⚡ 快速开始

```bash
# 1. 安装
# Linux/macOS（一行安装）：
curl -sSL https://raw.githubusercontent.com/MyAgentHubs/aimemo/main/install.sh | bash

# 或者 macOS 通过 Homebrew：
brew install MyAgentHubs/tap/aimemo

# 2. 在项目根目录初始化记忆
aimemo init

# 3. 注册到 Claude Code
claude mcp add-json aimemo-memory '{"command":"aimemo","args":["serve"]}'
```

重启 Claude Code，下次打开会话时 Claude 会自动加载项目上下文。

### OpenClaw 快速开始

如果你在使用 OpenClaw skills，请查看下面的 [OpenClaw 集成](#-openclaw-集成)章节，了解如何实现 per-skill 记忆隔离。

## 🔧 工作原理

`aimemo serve` 以 stdio MCP 服务器的形式运行，进程生命周期由 Claude Code 托管，不需要你手动保活。每次会话开始时 Claude 调用 `memory_context` 加载之前的上下文；工作过程中调用 `memory_store` 和 `memory_link` 记录决策和关联关系。你随时可以在终端执行 `aimemo search`、`aimemo list` 或 `aimemo get` 来查看和编辑同一份数据。所有内容存在 `.aimemo/` 目录下的 SQLite 数据库里，查找方式和 Git 找 `.git/` 一样——从当前目录逐级向上搜索。

## 🛠 MCP 工具

| 工具 | 说明 | Claude 何时调用 |
|------|------|----------------|
| `memory_context` | 返回当前项目的排序后近期记忆 | 会话开始时自动调用 |
| `memory_store` | 保存一条记录（事实、决策、日志、TODO）| 完成一个任务或做出决策后 |
| `memory_search` | 对所有记录进行 BM25 全文搜索 | 需要回忆某个具体内容时 |
| `memory_forget` | 按 ID 软删除一条记录 | 被告知丢弃某条信息时 |
| `memory_link` | 在两条记录之间建立命名关联 | 识别出依赖关系或关联时 |

所有工具 schema 总计不超过 2,000 个 token。每次调用有 5 秒硬超时——服务器不会卡住你的会话。空数据库下查询耗时不超过 5 ms。

## 📋 CLI 参考

### 初始化与诊断

| 命令 | 说明 |
|------|------|
| `aimemo init` | 在当前目录创建 `.aimemo/` |
| `aimemo serve` | 启动 MCP stdio 服务器（由 Claude Code 自动调用）|
| `aimemo doctor` | 检查 DB 健康状态、FTS5 支持、WAL 模式和 MCP 注册情况 |

### 记忆管理

| 命令 | 说明 |
|------|------|
| `aimemo add <name> <type> [observations...] [--tag]` | 添加实体及其记录 |
| `aimemo observe <entity-name> <observation>` | 向已有实体追加一条记录 |
| `aimemo retract <entity-name> <observation>` | 从实体中删除指定记录 |
| `aimemo forget <entity-name> [--permanent]` | 软删除实体（可恢复）；`--permanent` 为硬删除 |
| `aimemo search <query>` | 全文搜索，按相关度排序 |
| `aimemo get <entity-name>` | 查看实体及其全部记录和关联 |
| `aimemo link <from> <relation> <to>` | 在两个实体间建立有类型的关联 |
| `aimemo append <entity-name> <observation>` | 向实体追加一条记录（`observe` 的别名）|

### 日志

| 命令 | 说明 |
|------|------|
| `aimemo journal` | 打开交互式日志编辑器（使用 `$EDITOR`）|
| `aimemo journal <text>` | 快速写入一条内联日志 |

### 查看与导出

| 命令 | 说明 |
|------|------|
| `aimemo list` | 列出近期记录 |
| `aimemo tags` | 列出所有使用中的标签 |
| `aimemo stats` | 显示 DB 大小、记录数、最后写入时间 |
| `aimemo export --format md` | 导出全部记忆为 Markdown |
| `aimemo export --format json` | 导出全部记忆为 JSON |
| `aimemo import <file>` | 从 JSONL 或 JSON 导出文件导入 |

所有命令都支持 `--context <name>`，用来切换命名上下文（`.aimemo/` 内独立的 `.db` 文件）。

## ⚙️ 配置

`~/.aimemo/config.toml` — 全局默认值，所有字段均可选：

```toml
[defaults]
context = "main"          # 默认上下文名称
max_results = 20          # memory_context 返回的记录数量

[scoring]
recency_weight = 0.7      # 0–1，时效性相对于访问频率的权重

[server]
timeout_ms = 5000         # MCP 调用的硬超时（毫秒）
log_level = "warn"        # "debug" | "info" | "warn" | "error"
```

项目级覆盖放在项目根目录的 `.aimemo/config.toml` 中，字段相同，项目值优先于全局值。

## 🤖 Claude Code 集成

在本机注册一次服务器：

```bash
claude mcp add-json aimemo-memory '{"command":"aimemo","args":["serve"]}'
```

在项目的 `CLAUDE.md` 中加入以下内容，让 Claude 知道记忆功能可用以及如何使用：

```markdown
## Memory

本项目使用 aimemo 在会话间保持持久记忆。

- 每次会话开始时调用 `memory_context` 加载之前的上下文。
- 会话结束前调用 `memory_store`（type: journal）记录本次完成了什么、
  还有什么在进行中、有哪些卡点。
- 用 `memory_link` 关联相关记录（例如 bug 和修复，决策和理由）。
- 不要存储密钥、凭证或个人信息。
```

## 🦞 OpenClaw 集成

aimemo 用**零基础设施**和 **per-skill 记忆隔离**解决了 OpenClaw "记住所有但理解无"的问题。

### 为什么在 OpenClaw 中使用 aimemo？

**问题：**
- OpenClaw 原生的 Markdown 记忆系统越用越差
- Skills 共享记忆，导致交叉污染
- 上下文压缩会丢失重要信息

**解决方案：**
- ✅ **零依赖** — 单个 Go 二进制文件，无需 Docker/Node.js/数据库
- ✅ **Per-skill 隔离** — 每个 skill 有独立的记忆数据库
- ✅ **真正有效** — BM25 搜索 + 重要性评分找到关键信息
- ✅ **本地优先** — 所有数据留在你的机器上

**vs 其他方案：**

| | aimemo | Cognee | memsearch | Supermemory |
|---|--------|---------|-----------|-------------|
| **依赖** | 零 | Neo4j/Kuzu | Milvus | 云服务 |
| **安装** | 30秒 | 复杂 | 复杂 | 需注册 |
| **Skill 隔离** | 内置 | 手动 | 手动 | N/A |
| **Linux 支持** | ✅ 原生 | ✅ | ✅ | N/A |

### 5分钟设置

```bash
# 1. 安装（Linux amd64/arm64）
curl -sSL https://raw.githubusercontent.com/MyAgentHubs/aimemo/main/install.sh | bash

# 2. 在 OpenClaw 注册 MCP 服务器
claude mcp add-json aimemo-memory '{"command":"aimemo","args":["serve"]}'

# 或添加到 ~/.openclaw/openclaw.json：
# {
#   "mcpServers": {
#     "aimemo-memory": {
#       "command": "/usr/local/bin/aimemo",
#       "args": ["serve"]
#     }
#   }
# }

# 3. 初始化 workspace 记忆
cd ~/.openclaw/workspace
aimemo init

# 4. 重启 OpenClaw Gateway
# Linux: systemctl --user restart openclaw-gateway
# macOS: launchctl stop com.openclaw.gateway && launchctl start com.openclaw.gateway
```

### Per-Skill 记忆隔离

每个 skill 通过使用 `context` 参数获得独立的记忆：

**在你的 SKILL.md 中：**
```markdown
---
name: my-skill
description: 一个有持久记忆的 skill
---

# My Skill

## 指令

执行工作时：

1. **首先加载记忆**：
   ```
   memory_context({context: "my-skill"})
   ```

2. 使用加载的上下文执行任务

3. **存储学到的东西**：
   ```
   memory_store({
     context: "my-skill",
     entities: [{
       name: "preferences",
       entityType: "config",
       observations: ["用户偏好 snake_case"]
     }]
   })
   ```

**关键**：始终传递 `context: "my-skill"` 以防止记忆污染。
```

**结果：**
```
~/.openclaw/workspace/.aimemo/
├── memory.db                    # 共享/默认（无 context）
├── memory-skill-a.db            # Skill A 的隔离记忆
├── memory-skill-b.db            # Skill B 的隔离记忆
└── memory-skill-c.db            # Skill C 的隔离记忆
```

### 完整示例

查看 [`examples/openclaw-github-pr-reviewer/`](examples/openclaw-github-pr-reviewer/) 获取一个完整的工作 skill：
- 审查 GitHub PR
- 学习代码风格偏好
- 跨会话记住模式
- 存储反馈以改进

### 文档

- **[OpenClaw 集成指南](docs/openclaw-integration.md)** — 分步设置
- **[OpenClaw 工作流程](docs/openclaw-workflow.md)** — 架构深入解析
- **[示例 Skill](examples/openclaw-github-pr-reviewer/)** — 完整实现

### 调试

```bash
# 列出 skill 的记忆
aimemo list --context my-skill

# 在 skill 内搜索
aimemo search "关键词" --context my-skill

# 导出检查
aimemo export --context my-skill --format json > memory.json

# 获取数据库统计
aimemo stats --context my-skill
```

## 🖥 客户端支持

aimemo 兼容所有支持 MCP 协议的 AI 编程客户端，服务器命令统一为 `aimemo serve`。

> **PATH 注意（macOS/Homebrew）：** GUI 应用可能不继承 Shell 的 PATH。如果客户端找不到 `aimemo`，请改用绝对路径 `/opt/homebrew/bin/aimemo`。

### Claude Code

```bash
claude mcp add-json aimemo-memory '{"command":"aimemo","args":["serve"]}'
```

或在项目根目录提交 `.mcp.json`（参考本仓库的示例文件）。

### Cursor

项目级（`.cursor/mcp.json`）或全局（`~/.cursor/mcp.json`）：

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

编辑 `~/.codeium/windsurf/mcp_config.json`（仅全局）：

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

项目级（`.codex/config.toml`）或全局（`~/.codex/config.toml`）：

```toml
[mcp_servers.aimemo-memory]
command = "aimemo"
args    = ["serve"]
```

### Cline（VS Code）

编辑 `~/Library/Application Support/Code/User/globalStorage/saoudrizwan.claude-dev/settings/cline_mcp_settings.json`：

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

### Continue（VS Code / JetBrains）

项目级（`.continue/mcpServers/aimemo-memory.yaml`）：

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

或在全局 `~/.continue/config.yaml` 的 `mcpServers:` 下添加同样内容。

### Zed

在 `~/.zed/settings.json`（全局）或 `.zed/settings.json`（项目级）中：

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

## 🤝 参与贡献

Bug 反馈和功能建议请提 [GitHub Issue](https://github.com/MyAgentHubs/aimemo/issues)。欢迎 PR——如果改动较大，建议先开 Issue 讨论方向，避免白费力气。
