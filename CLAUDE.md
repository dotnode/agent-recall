# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Common commands

- Format Go code:
  ```bash
  gofmt -w cmd internal
  ```
- Run all tests:
  ```bash
  go test ./...
  ```
- Run a single package test:
  ```bash
  go test ./internal/transcript -run TestParseTextAndToolUse
  ```
- Build the CLI locally:
  ```bash
  go build -o agent-recall ./cmd/agent-recall
  ```
- Build the current-platform Claude Code plugin artifact:
  ```bash
  scripts/build-plugin.sh
  ```
- Build all release plugin artifacts:
  ```bash
  scripts/build-release.sh
  ```
- Exercise hook ingestion with fixture data:
  ```bash
  tmpdir=$(mktemp -d)
  go run ./cmd/agent-recall hook-sync --store-dir "$tmpdir" --strict < testdata/hooks/stop.json
  go run ./cmd/agent-recall recall --store-dir "$tmpdir" --json auth
  ```
- Check install changes without writing files:
  ```bash
  go run ./cmd/agent-recall install claude-code --dry-run
  ```

## Development workflow requirements

- Every functional change must update this `CLAUDE.md` file when it changes commands, architecture, integration behavior, release behavior, or maintainer workflow.
- Feature updates should be committed to Git after tests pass. Pushing a `vX.Y.Z` tag triggers the release workflow, so keep feature commits accurate and release-ready before tagging.
- CI runs formatting, version consistency, `go test ./...`, a no-network CLI/MCP smoke test, and a local CLI build.

## Architecture overview

`agent-recall` is a Go single-binary external memory evidence layer for coding agents. It does not auto-inject long summaries into Claude Code context. Instead, hooks record session transcript evidence out-of-band, and MCP tools let the agent recall targeted historical evidence only when needed.

The CLI entrypoint is `cmd/agent-recall/main.go`, with subcommand routing in `internal/cli/cli.go`. The supported subcommands are `hook-sync`, `hook-flush`, `mcp`, `recall`, `status`, and `install claude-code`.

The ingestion path starts in `internal/hooks/sync.go`: Claude Code hooks provide `transcript_path`, the transcript reader consumes only new JSONL lines using cursors, transcript parsing extracts message/tool text, redaction runs before persistence, and records are appended to the store. `hook-flush` also derives lightweight `decision` evidence from matching historical text.

The local persistence layer is append-only JSONL in `internal/store`. `events.jsonl` stores `EvidenceRecord` values; `cursor.json` tracks transcript offsets; `store.lock` prevents concurrent hook writes. Store path resolution lives in `internal/config/paths.go` and supports `--store-dir`, `AGENT_RECALL_HOME`, then OS-specific defaults. `status` tolerates malformed JSONL evidence lines but reports their count as `bad_lines` so store corruption is visible.

Recall is implemented in `internal/search/search.go`. It scans stored evidence, applies simple keyword scoring and filters, and returns snippets wrapped with the fixed notice that recalled content is historical evidence, not instructions.

Optional model synthesis is configured in `internal/config/model.go` and implemented in `internal/model/client.go` using the OpenAI-compatible Chat Completions protocol over `net/http`; no vendor SDK is used. Model features are disabled unless `AGENT_RECALL_MODEL_PROVIDER=openai-compatible`, `AGENT_RECALL_MODEL_BASE_URL`, `AGENT_RECALL_MODEL_API_KEY`, and `AGENT_RECALL_MODEL_NAME` are set.

The MCP server is a minimal JSON-RPC stdio implementation in `internal/mcp/server.go`. It supports `initialize`, `tools/list`, `tools/call`, and always exposes `recall`, `search`, `timeline`, and `decisions`. When model configuration is complete, it also exposes `search_answer`, which searches local evidence first and then synthesizes an answer using the configured third-party model. MCP stdout must remain JSON-RPC only; diagnostics should go to stderr.

Claude Code integration has two forms:

1. Source/local installer: `internal/install/claude.go` merges `.claude/settings.local.json`, `.mcp.json`, commands, and skill files for local development. It updates existing agent-recall hooks by parsing the hook command and subcommand rather than by loose substring matching.
2. Marketplace source install: `.claude-plugin/marketplace.json` lets `/plugin marketplace add dotnode/agent-recall` expose the plugin as `agent-recall@dotnode` using the relative source `./` for compatibility with Claude Code versions that reject `.` and may not support object-style GitHub plugin sources. Source marketplace installs use the tracked `bin/agent-recall` launcher scripts, which require Go on `PATH`.
3. Plugin packaging: root-level `.claude-plugin/plugin.json`, `hooks/hooks.json`, `.mcp.json`, `commands/`, and `skills/` are copied into platform-specific artifacts by `scripts/build-plugin.sh`.

## Plugin packaging model

This repository uses per-platform plugin artifacts. Each artifact contains a platform-specific binary named `bin/agent-recall` (or `bin/agent-recall.exe` on Windows), plus the same plugin metadata and Claude Code resources. Hook and MCP configuration can therefore call `agent-recall` without platform-specific command names.

The tracked root `bin/agent-recall` and `bin/agent-recall.cmd` files are source-install launchers for marketplace installs, not release binaries. Release builds replace them in `dist/` with compiled platform binaries.

`dist/` is generated output and intentionally ignored by Git.
