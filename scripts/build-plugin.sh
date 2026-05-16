#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OS="${1:-$(go env GOOS)}"
ARCH="${2:-$(go env GOARCH)}"
VERSION="${VERSION:-0.1.0}"
OUT_DIR="$ROOT/dist/plugin-${OS}-${ARCH}"
BIN_NAME="agent-recall"

if [[ "$OS" == "windows" ]]; then
  BIN_NAME="agent-recall.exe"
fi

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR/bin"

(
  cd "$ROOT"
  CGO_ENABLED=0 GOOS="$OS" GOARCH="$ARCH" go build -trimpath -ldflags "-s -w" -o "$OUT_DIR/bin/$BIN_NAME" ./cmd/agent-recall
)

cp -R "$ROOT/.claude-plugin" "$OUT_DIR/.claude-plugin"
cp -R "$ROOT/hooks" "$OUT_DIR/hooks"
cp -R "$ROOT/commands" "$OUT_DIR/commands"
cp -R "$ROOT/skills" "$OUT_DIR/skills"
cp "$ROOT/.mcp.json" "$OUT_DIR/.mcp.json"

if [[ "$OS" != "windows" ]]; then
  chmod +x "$OUT_DIR/bin/$BIN_NAME"
fi

(
  cd "$ROOT/dist"
  ARCHIVE="agent-recall-plugin-${OS}-${ARCH}.tar.gz"
  rm -f "$ARCHIVE"
  tar -czf "$ARCHIVE" "plugin-${OS}-${ARCH}"
  echo "Built $ROOT/dist/$ARCHIVE"
)
