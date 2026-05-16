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

## Recommended Claude Code install: packaged release artifact

Use the packaged release artifact for normal Claude Code installs. It includes a precompiled platform-specific `bin/agent-recall` binary and does not require Go at runtime. Packaged hooks call that bundled binary through the plugin root path instead of relying on `PATH`, which avoids MCP or hook startup failures caused by source launchers or cold `go run` build caches.

Download the asset matching your platform from the GitHub Release:

| Platform | Asset |
| --- | --- |
| Linux x86_64 | `agent-recall-plugin-linux-amd64.tar.gz` |
| Linux arm64 | `agent-recall-plugin-linux-arm64.tar.gz` |
| macOS Intel | `agent-recall-plugin-darwin-amd64.tar.gz` |
| macOS Apple Silicon | `agent-recall-plugin-darwin-arm64.tar.gz` |
| Windows x86_64 | `agent-recall-plugin-windows-amd64.tar.gz` |

Each archive expands to a local Claude Code marketplace bundle:

```text
agent-recall-plugin-linux-amd64/
  .claude-plugin/marketplace.json
  plugins/agent-recall/
    .claude-plugin/plugin.json
    .mcp.json
    hooks/hooks.json
    commands/recall-session.md
    commands/memory-status.md
    skills/agent-recall/SKILL.md
    bin/agent-recall
```

Install from a stable absolute path, not a temporary directory:

```text
/plugin marketplace add /absolute/path/to/agent-recall-plugin-linux-amd64
/plugin install agent-recall@dotnode
/reload-plugins
```

If you previously installed the legacy source marketplace, migrate by removing it first:

```text
/plugin uninstall agent-recall@dotnode
/plugin marketplace remove dotnode
/plugin marketplace add /absolute/path/to/agent-recall-plugin-linux-amd64
/plugin install agent-recall@dotnode
/reload-plugins
```

Verify the install with:

```text
/memory-status
/recall-session compact 前我们说到哪了
```

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

## Claude Code installation options

| Install path | Status | Best for | Requires Go on `PATH` | Runtime command | Update path |
| --- | --- | --- | --- | --- | --- |
| Packaged release artifact | Recommended | Normal Claude Code use | No | Precompiled bundled binary | Download/install the newer release artifact, then `/plugin marketplace update dotnode`, `/plugin update agent-recall`, `/reload-plugins` |
| Local development installer | Development | Working from a source checkout | Yes, for building the local CLI first | The binary you build with `go build -o agent-recall ./cmd/agent-recall` | Rebuild and rerun `./agent-recall install claude-code` |
| Marketplace source install | Legacy/development | Testing marketplace source behavior | Yes | Tracked launcher scripts in `bin/` that run `go run ./cmd/agent-recall` | `/plugin marketplace update dotnode`, `/plugin update agent-recall`, `/reload-plugins` |

### Local development install

```bash
go build -o agent-recall ./cmd/agent-recall
./agent-recall install claude-code
```

This merges local Claude Code settings and creates the MCP config, slash commands, and skill files for the current project.

### Marketplace source install

This repository can still be added directly as a Claude Code plugin marketplace for development and legacy source-install testing:

```text
/plugin marketplace add dotnode/agent-recall
/plugin install agent-recall@dotnode
/reload-plugins
```

The source marketplace installs this repository directly from GitHub and uses the tracked `bin/agent-recall` launcher scripts. Those launchers execute `go run ./cmd/agent-recall`, so this path requires Go on `PATH` and can be slow on first MCP startup with a cold Go build cache. Normal users should install the packaged release artifact instead.

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
