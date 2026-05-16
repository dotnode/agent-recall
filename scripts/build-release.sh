#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
TARGETS=(
  "linux amd64"
  "linux arm64"
  "darwin amd64"
  "darwin arm64"
  "windows amd64"
)

rm -rf "$ROOT/dist"
mkdir -p "$ROOT/dist"

for target in "${TARGETS[@]}"; do
  read -r os arch <<<"$target"
  "$ROOT/scripts/build-plugin.sh" "$os" "$arch"
done

echo "Release artifacts written to $ROOT/dist"
