#!/usr/bin/env bash
set -euo pipefail

: "${HOME:?HOME must be set and non-empty}"
case "$HOME" in
  /*) ;;
  *) echo "HOME must be an absolute path" >&2; exit 1 ;;
esac

bin="$HOME/.local/bin/nuytek-harness"
state="$HOME/.local/state/nuytek-harness"

removed=0
for path in "$bin" "$state"; do
  # -L catches dangling links; no trailing slash so a symlinked dir loses only the link.
  if [ -e "$path" ] || [ -L "$path" ]; then
    rm -rf -- "$path"
    echo "removed: $path"
    removed=1
  fi
done

if [ "$removed" -eq 0 ]; then
  echo "nothing installed"
fi
