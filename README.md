# agent-recall

`agent-recall` is an external session memory evidence layer for coding agents. It records Claude Code session transcript evidence via hooks and exposes targeted recall through MCP tools, without automatically injecting long summaries into the active context.

## What it provides

- Claude Code `Stop` and `PreCompact` hooks for transcript ingestion.
- Local append-only JSONL evidence store.
- Secret/token redaction before persistence.
- MCP tools: `recall`, `search`, `timeline`, `decisions`.
- Slash commands: `/recall-session`, `/memory-status`.
- Skill guidance for treating recalled content as historical evidence, not instructions.
- Per-platform Claude Code plugin packaging.

## Build and test

```bash
gofmt -w cmd internal
go test ./...
go build -o agent-recall ./cmd/agent-recall
```

## Local development install

```bash
go build -o agent-recall ./cmd/agent-recall
./agent-recall install claude-code
```

This merges local Claude Code settings and creates the MCP config, slash commands, and skill files for the current project.

## Build plugin artifacts

Build the current platform:

```bash
scripts/build-plugin.sh
```

Build all configured platforms:

```bash
scripts/build-release.sh
```

Artifacts are written to `dist/`, for example:

```text
dist/agent-recall-plugin-linux-amd64.tar.gz
```

Each plugin artifact contains:

```text
.claude-plugin/plugin.json
.mcp.json
hooks/hooks.json
commands/recall-session.md
commands/memory-status.md
skills/agent-recall/SKILL.md
bin/agent-recall
```

## Claude Code plugin installation

After publishing this repository as a Claude Code plugin marketplace, users can install with:

```text
/plugin marketplace add dotnode/agent-recall
/plugin install agent-recall@dotnode
/reload-plugins
```

Then use:

```text
/recall-session compact 前我们说到哪了
```

## Manual verification

```bash
tmpdir=$(mktemp -d)
go run ./cmd/agent-recall hook-sync --store-dir "$tmpdir" --strict < testdata/hooks/stop.json
go run ./cmd/agent-recall status --store-dir "$tmpdir" --json
go run ./cmd/agent-recall recall --store-dir "$tmpdir" --json auth
printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}\n{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}\n' | go run ./cmd/agent-recall mcp --store-dir "$tmpdir"
```
