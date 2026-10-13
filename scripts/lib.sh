#!/usr/bin/env bash
# Shared helpers for nuytek-harness. Sourced, never executed.

STAMP="$(date +%Y%m%d-%H%M%S)"

die() {
  echo "error: $*" >&2
  exit 1
}

# y/N prompt; returns 1 on "no" and on non-TTY.
ask() {
  [ -t 0 ] || return 1
  local answer
  read -r -p "$1 [y/N] " answer
  case "$answer" in
    [Yy]|[Yy][Ee][Ss]) ;;
    *) return 1 ;;
  esac
}

# Non-TTY always aborts so nothing is deleted unattended.
confirm() {
  [ -t 0 ] || die "confirmation required but stdin is not a TTY (use --yes)"
  ask "$1" || die "aborted"
}

SNAPSHOT_ROOT="$HOME/.claude-backups"
SNAPSHOT=""

# Copy a path into this run's snapshot, mirroring its path under $HOME. The snapshot is created
# on first use; the first copy of a path wins so remove+install keeps the original.
snapshot() {
  local target="$1" copy
  if [ ! -e "$target" ] && [ ! -L "$target" ]; then return 0; fi
  if [ -z "$SNAPSHOT" ]; then
    SNAPSHOT="$SNAPSHOT_ROOT/$STAMP"
    while [ -e "$SNAPSHOT" ]; do SNAPSHOT="${SNAPSHOT}-1"; done
    mkdir -p "$SNAPSHOT"
  fi
  copy="$SNAPSHOT/${target#"$HOME"/}"
  if [ -e "$copy" ] || [ -L "$copy" ]; then return 0; fi
  mkdir -p "$(dirname "$copy")"
  cp -a "$target" "$copy"
}

# usage: copy_tree <source> <target> [extra tar args...]
copy_tree() {
  local source="$1" target="$2"
  shift 2
  [ -d "$source" ] || die "missing source: $source"
  mkdir -p "$target"
  tar \
    --exclude='__pycache__' --exclude='*/__pycache__' \
    --exclude='*.pyc' \
    "$@" \
    -C "$source" -cf - . | tar -C "$target" -xf -
}

install_file() {
  local source="$1" target="$2"
  [ -f "$source" ] || die "missing source: $source"
  mkdir -p "$(dirname "$target")"
  snapshot "$target"
  cp "$source" "$target"
}

clean_pycache() {
  local dir="$1"
  [ -d "$dir" ] || return 0
  find "$dir" -type d -name __pycache__ -prune -exec rm -rf -- {} +
  find "$dir" -type f -name '*.pyc' -delete
}
