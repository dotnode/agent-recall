# CLAUDE.md

本文件为 Claude Code（claude.ai/code）在本仓库中工作时提供指导。

## 常用命令

- 格式化 Go 代码：
  ```bash
  gofmt -w cmd internal
  ```
- 运行全部测试：
  ```bash
  go test ./...
  ```
- 运行单个 package 测试：
  ```bash
  go test ./internal/transcript -run TestParseTextAndToolUse
  ```
- 本地构建 CLI：
  ```bash
  go build -o agent-recall ./cmd/agent-recall
  ```
- 构建当前平台的 Claude Code plugin artifact：
  ```bash
  scripts/build-plugin.sh
  ```
- 构建全部 release plugin artifacts：
  ```bash
  scripts/build-release.sh
  ```
- 使用 fixture 数据演练 hook ingestion：
  ```bash
  tmpdir=$(mktemp -d)
  go run ./cmd/agent-recall hook-sync --store-dir "$tmpdir" --strict < testdata/hooks/stop.json
  go run ./cmd/agent-recall recall --store-dir "$tmpdir" --json auth
  ```
- 不写入文件地检查 install 变更：
  ```bash
  go run ./cmd/agent-recall install claude-code --dry-run
  ```

## 开发流程要求

- 每个功能性变更在改变命令、架构、集成行为、发布行为或维护者工作流时，都必须同步更新本 `CLAUDE.md` 文件。
- Feature updates 应在测试通过后提交到 Git。推送 `vX.Y.Z` tag 会触发 release workflow，因此在打 tag 前要确保 feature commits 准确且 release-ready。
- 每次 Git commit 前，检查该 commit 是否包含代码或行为变更。如果包含，就在同一个 commit 中同步 bump `.claude-plugin/plugin.json` 和 `internal/version/version.go`，并将该变更视为 release-bound；只有在用户明确确认后，才 push `vX.Y.Z` tag。
- CI 会运行格式检查、版本一致性检查、`go test ./...`、无网络 CLI/MCP smoke test，以及本地 CLI build。

## 架构概览

`agent-recall` 是一个 Go 单二进制的外部 memory evidence layer，面向 coding agents。它不会自动把长摘要注入 Claude Code context；相反，hooks 会在上下文外记录 session transcript evidence，MCP tools 让 agent 仅在需要时召回有针对性的历史证据。

CLI 入口是 `cmd/agent-recall/main.go`，subcommand routing 位于 `internal/cli/cli.go`。支持的 subcommands 包括 `hook-sync`、`hook-flush`、`mcp`、`recall`、`status` 和 `install claude-code`。

Ingestion path 从 `internal/hooks/sync.go` 开始：Claude Code hooks 提供 `transcript_path`，transcript reader 通过 cursors 只消费新的 JSONL lines，transcript parsing 提取 message/tool text，redaction 在持久化前执行，records 会 append 到 store。`hook-flush` 还会从匹配到的历史文本中派生轻量级 `decision` evidence。

本地持久化层是 `internal/store` 中的 append-only JSONL。`events.jsonl` 存储 `EvidenceRecord` values；`cursor.json` 跟踪 transcript offsets；`store.lock` 防止并发 hook writes。Store path resolution 位于 `internal/config/paths.go`，按优先级支持 `--store-dir`、`AGENT_RECALL_HOME`，然后是 OS-specific defaults。`status` 会容忍 malformed JSONL evidence lines，但会以 `bad_lines` 报告数量，使 store corruption 可见。

Recall 在 `internal/search/search.go` 中实现。它扫描 stored evidence，应用简单 keyword scoring 和 filters，并返回 snippets；返回内容始终带有固定 notice，说明 recalled content 是 historical evidence，而不是 instructions。

Optional model synthesis 由 `internal/config/model.go` 配置，并由 `internal/model/client.go` 使用 OpenAI-compatible Chat Completions protocol 基于 `net/http` 实现；不使用任何 vendor SDK。除非设置了 `AGENT_RECALL_MODEL_PROVIDER=openai-compatible`、`AGENT_RECALL_MODEL_BASE_URL`、`AGENT_RECALL_MODEL_API_KEY` 和 `AGENT_RECALL_MODEL_NAME`，否则 model features 默认禁用。

MCP server 是 `internal/mcp/server.go` 中的最小 JSON-RPC stdio 实现。它支持 `initialize`、`tools/list`、`tools/call`，并始终暴露 `recall`、`search`、`timeline` 和 `decisions`。当 model configuration 完整时，它还会暴露 `search_answer`，该工具会先搜索本地 evidence，再使用配置的第三方 model synthesis answer。MCP stdout 必须只输出 JSON-RPC；diagnostics 应写入 stderr。

Claude Code integration 有三种形式：

1. Source/local installer：`internal/install/claude.go` 会为本地开发合并 `.claude/settings.local.json`、`.mcp.json`、commands 和 skill files。它通过解析 hook command 和 subcommand 来更新已有 agent-recall hooks，而不是使用宽松的 substring matching。
2. Marketplace source install：`.claude-plugin/marketplace.json` 允许 `/plugin marketplace add dotnode/agent-recall` 以 `agent-recall@dotnode` 暴露 plugin。Plugin source 使用 object 形式 `{"source": "github", "repo": "dotnode/agent-recall"}`，因为 Claude Code 的 relative-path string source 仅支持指向 marketplace 仓库下的子目录（如 `"./plugins/x"`），无法用 `"./"` 或 `"."` 指向 marketplace 仓库根，旧版本上更会直接报 "unsupported source type"。Source marketplace installs 使用 tracked `bin/agent-recall` launcher scripts，这要求 `PATH` 中有 Go。
3. Plugin packaging：root-level `.claude-plugin/plugin.json`、`hooks/hooks.json`、`.mcp.json`、`commands/` 和 `skills/` 会被 `scripts/build-plugin.sh` 复制到 platform-specific artifacts 中。

## Plugin packaging model

本仓库使用 per-platform plugin artifacts。每个 artifact 都包含一个 platform-specific binary，名称为 `bin/agent-recall`（Windows 上为 `bin/agent-recall.exe`），并包含相同的 plugin metadata 和 Claude Code resources。因此 hook 和 MCP configuration 可以直接调用 `agent-recall`，无需 platform-specific command names。

Tracked root `bin/agent-recall` 和 `bin/agent-recall.cmd` 文件是 marketplace installs 的 source-install launchers，不是 release binaries。Release builds 会在 `dist/` 中用 compiled platform binaries 替换它们。

`dist/` 是 generated output，并且有意被 Git 忽略。
