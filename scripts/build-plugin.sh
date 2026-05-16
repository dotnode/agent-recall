#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OS="${1:-$(go env GOOS)}"
ARCH="${2:-$(go env GOARCH)}"
VERSION="${VERSION:-$(python3 -c 'import json, sys; print(json.load(open(sys.argv[1]))["version"])' "$ROOT/.claude-plugin/plugin.json")}"
BUNDLE_NAME="agent-recall-plugin-${OS}-${ARCH}"
BUNDLE_DIR="$ROOT/dist/$BUNDLE_NAME"
PLUGIN_DIR="$BUNDLE_DIR/plugins/agent-recall"
BIN_NAME="agent-recall"

if [[ "$OS" == "windows" ]]; then
  BIN_NAME="agent-recall.exe"
fi

rm -rf "$BUNDLE_DIR"
mkdir -p "$BUNDLE_DIR/.claude-plugin" "$PLUGIN_DIR/.claude-plugin" "$PLUGIN_DIR/bin"

(
  cd "$ROOT"
  CGO_ENABLED=0 GOOS="$OS" GOARCH="$ARCH" go build -trimpath -ldflags "-s -w -X agent-recall/internal/version.Version=${VERSION}" -o "$PLUGIN_DIR/bin/$BIN_NAME" ./cmd/agent-recall
)

cp "$ROOT/.claude-plugin/plugin.json" "$PLUGIN_DIR/.claude-plugin/plugin.json"
cp -R "$ROOT/hooks" "$PLUGIN_DIR/hooks"
cp -R "$ROOT/commands" "$PLUGIN_DIR/commands"
cp -R "$ROOT/skills" "$PLUGIN_DIR/skills"
cp "$ROOT/.mcp.json" "$PLUGIN_DIR/.mcp.json"

python3 - "$BUNDLE_DIR/.claude-plugin/marketplace.json" "$VERSION" <<'PY'
import json
import sys

path, version = sys.argv[1], sys.argv[2]
marketplace = {
    "name": "dotnode",
    "owner": {"name": "dotnode"},
    "plugins": [
        {
            "name": "agent-recall",
            "version": version,
            "description": "External session memory evidence layer for coding agents.",
            "source": "./plugins/agent-recall",
        }
    ],
}
with open(path, "w", encoding="utf-8") as f:
    json.dump(marketplace, f, indent=2)
    f.write("\n")
PY

if [[ "$OS" != "windows" ]]; then
  chmod +x "$PLUGIN_DIR/bin/$BIN_NAME"
fi

(
  cd "$ROOT/dist"
  ARCHIVE="agent-recall-plugin-${OS}-${ARCH}.tar.gz"
  rm -f "$ARCHIVE"
  tar -czf "$ARCHIVE" "$BUNDLE_NAME"
  echo "Built $ROOT/dist/$ARCHIVE"
)
