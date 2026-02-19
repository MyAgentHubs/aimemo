# aimemo

[![Go 1.22+](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org/) [![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg?style=flat)](LICENSE) [![Release](https://img.shields.io/github/v/release/MyAgentHubs/aimemo?style=flat)](https://github.com/MyAgentHubs/aimemo/releases)

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
brew install MyAgentHubs/tap/aimemo

# 2. 在项目根目录初始化记忆
aimemo init

# 3. 注册到 Claude Code
claude mcp add-json aimemo '{"type":"stdio","command":"aimemo","args":["serve"]}'
```

重启 Claude Code，下次打开会话时 Claude 会自动加载项目上下文。

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
| `aimemo add <text>` | 从终端存入一条记录 |
| `aimemo observe <text>` | `add` 的别名 |
| `aimemo retract <id>` | 精确删除单条记录 |
| `aimemo forget <pattern>` | 软删除所有匹配的记录 |
| `aimemo search <query>` | 全文搜索，按相关度排序 |
| `aimemo get <id>` | 按 ID 查看单条记录 |
| `aimemo link <id1> <id2> <label>` | 在两条记录间建立命名关联 |
| `aimemo append <id> <text>` | 追加内容到已有记录 |

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
| `aimemo import <file>` | 从 mcp-knowledge-graph JSONL 或 aimemo JSON 导入 |

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

## 🔀 从 mcp-knowledge-graph 迁移

```bash
# 导出现有知识图谱
npx @modelcontextprotocol/inspector export > knowledge-graph.jsonl

# 导入到 aimemo
aimemo import knowledge-graph.jsonl
```

实体转为记录，关系转为链接，标签保留。运行 `aimemo stats` 确认导入数量。

## 🤖 Claude Code 集成

在本机注册一次服务器：

```bash
claude mcp add-json aimemo '{"type":"stdio","command":"aimemo","args":["serve"]}'
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

## 🤝 参与贡献

Bug 反馈和功能建议请提 [GitHub Issue](https://github.com/MyAgentHubs/aimemo/issues)。欢迎 PR——如果改动较大，建议先开 Issue 讨论方向，避免白费力气。
