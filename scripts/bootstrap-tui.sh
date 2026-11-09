#!/usr/bin/env bash
set -euo pipefail

: "${HOME:?HOME must be set and non-empty}"
case "$HOME" in
  /*) ;;
  *) echo "HOME must be an absolute path" >&2; exit 1 ;;
esac

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

if ! command -v go >/dev/null 2>&1; then
  echo "error: go is required to build the TUI but was not found in PATH" >&2
  exit 1
fi

TMP="$(mktemp -d)"
trap 'rm -rf -- "$TMP"' EXIT

(cd "$ROOT/tui" && go build -o "$TMP/nuytek-harness" ./cmd/nuytek-harness)
install -Dm0755 "$TMP/nuytek-harness" "$HOME/.local/bin/nuytek-harness"

echo "installed: $HOME/.local/bin/nuytek-harness"
echo "run: nuytek-harness --repo $ROOT"
