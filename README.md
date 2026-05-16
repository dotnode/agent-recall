# agent-recall

`agent-recall` is an external session memory evidence layer for coding agents. It records Claude Code session transcript evidence via hooks and exposes targeted recall through MCP tools, without automatically injecting long summaries into the active context.

## What it provides

- Claude Code `Stop` and `PreCompact` hooks for transcript ingestion.
- Local append-only JSONL evidence store.
- Secret/token redaction before persistence.
- MCP tools: `recall`, `search`, `timeline`, `decisions`.
- Optional third-party model synthesis via `search_answer` using an OpenAI-compatible Chat Completions API.
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

This repository can be added as a Claude Code plugin marketplace:

```text
/plugin marketplace add dotnode/agent-recall
/plugin install agent-recall@dotnode
/reload-plugins
```

The marketplace source installs this repository directly. It includes lightweight `bin/agent-recall` launcher scripts that run the Go CLI from source, so source marketplace installs require Go on `PATH`. Packaged release artifacts remain self-contained and include a platform-specific compiled `bin/agent-recall` binary.

To update an existing marketplace install after a new release:

```text
/plugin marketplace update dotnode
/plugin update agent-recall
/reload-plugins
```

Verify the install with:

```text
/memory-status
/recall-session compact 前我们说到哪了
```

## Optional third-party model synthesis

By default, all MCP tools remain evidence-only and never call a model. To enable synthesis, configure an independent third-party model that implements the OpenAI-compatible Chat Completions API. When configuration is complete, the MCP server exposes an additional `search_answer` tool that searches local historical evidence first, then asks the configured model to answer from that evidence with citations.

Configure the MCP server environment with:

```json
{
  "mcpServers": {
    "agent-recall": {
      "command": "agent-recall",
      "args": ["mcp"],
      "env": {
        "AGENT_RECALL_MODEL_PROVIDER": "openai-compatible",
        "AGENT_RECALL_MODEL_BASE_URL": "https://api.example.com/v1",
        "AGENT_RECALL_MODEL_API_KEY": "your-api-key",
        "AGENT_RECALL_MODEL_NAME": "your-model-name",
        "AGENT_RECALL_MODEL_TIMEOUT": "20s"
      }
    }
  }
}
```

Do not commit real API keys. Existing tools (`recall`, `search`, `timeline`, `decisions`) still return historical evidence only; `search_answer` returns model synthesis over that evidence and should not be treated as current repository truth without verification.

## Manual verification

```bash
tmpdir=$(mktemp -d)
go run ./cmd/agent-recall hook-sync --store-dir "$tmpdir" --strict < testdata/hooks/stop.json
go run ./cmd/agent-recall status --store-dir "$tmpdir" --json
go run ./cmd/agent-recall recall --store-dir "$tmpdir" --json auth
printf '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}\n{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}\n' | go run ./cmd/agent-recall mcp --store-dir "$tmpdir"
```
