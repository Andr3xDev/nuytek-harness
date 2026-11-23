#!/usr/bin/env bash
# Checks resolve_specter_root, render_listed and unrender_listed against the fixtures shared with the Go tests.
set -uo pipefail

REPO="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
FIXTURES="$REPO/tui/internal/render/testdata"
WORK="$(mktemp -d)"
trap 'rm -rf -- "$WORK"' EXIT
HOME="$WORK/home"
export HOME
mkdir -p "$HOME"

# render_listed reads the list from $ROOT/targets; a private ROOT keeps this check independent of the real list.
ROOT="$WORK/harness"
mkdir -p "$ROOT/targets"
printf '%s\n' .claude/agents/listed.md .claude/CLAUDE.md > "$ROOT/targets/rendered-files.txt"

# shellcheck source=../lib.sh
source "$REPO/scripts/lib.sh"

STATE_FILE="$HOME/.local/state/nuytek-harness/state.toml"
failures=0

fail() { echo "FAIL: $*" >&2; failures=$((failures + 1)); }

set_state() {
  mkdir -p "$(dirname "$STATE_FILE")"
  printf '%s' "$1" > "$STATE_FILE"
}

clear_state() { rm -f -- "$STATE_FILE"; }

expect_root() {
  local name="$1" want="$2" got
  got="$(resolve_specter_root 2>/dev/null)" || got="<command failed>"
  [ "$got" = "$want" ] || fail "resolve: $name: got '$got', want '$want'"
}

DEFAULT_ROOT="$HOME/dev/specter"
CUSTOM_ROOT="$HOME/work/specs"

clear_state
expect_root "state file absent" "$DEFAULT_ROOT"
set_state $'other = "x"\n'
expect_root "no specter_root key" "$DEFAULT_ROOT"
set_state "specter_root = '$CUSTOM_ROOT'"$'\n'
expect_root "single-quoted saved root" "$CUSTOM_ROOT"
set_state "specter_root = \"$CUSTOM_ROOT\""$'\n'
expect_root "double-quoted saved root" "$CUSTOM_ROOT"
set_state "specter_root = \"$CUSTOM_ROOT\""$'\n[ui]\ntheme = "x"\n'
expect_root "top-level key before a table" "$CUSTOM_ROOT"
set_state $'[ui]\n'"specter_root = \"$CUSTOM_ROOT\""$'\n'
expect_root "key inside a table is ignored" "$DEFAULT_ROOT"
set_state "specter_root = \"$HOME\""$'\n'
expect_root "root equal to HOME" "$DEFAULT_ROOT"
set_state $'specter_root = "/opt/specter"\n'
expect_root "root outside HOME" "$DEFAULT_ROOT"
set_state $'specter_root = "work/specs"\n'
expect_root "relative root" "$DEFAULT_ROOT"
set_state "specter_root = \"$HOME/work/../../etc\""$'\n'
expect_root "dotdot leaving HOME" "$DEFAULT_ROOT"
set_state "specter_root = \"$HOME/a/../work\""$'\n'
expect_root "dotdot staying under HOME" "$DEFAULT_ROOT"
set_state "specter_root = \"$HOME/work/..\""$'\n'
expect_root "trailing dotdot" "$DEFAULT_ROOT"

while IFS=$'\t' read -r case_name state_line expected; do
  case "$case_name" in '' | '#'*) continue ;; esac
  set_state "${state_line//\$HOME/$HOME}"$'\n'
  expect_root "shared case: $case_name" "$HOME${expected#\~}"
done < "$FIXTURES/root-cases.txt"

# Copies fixture $input to $dest, runs "$func $component" and compares $dest with fixture $want.
expect_transform() {
  local name="$1" input="$2" dest="$3" func="$4" want="$5" component="$6"
  mkdir -p "$(dirname "$dest")"
  cp "$FIXTURES/$input" "$dest"
  "$func" "$component" 2>/dev/null || fail "$name: $func failed"
  cmp -s "$dest" "$FIXTURES/$want" || fail "$name: $dest differs from $want"
}

AGENTS="$HOME/.claude/agents"
clear_state
expect_transform "render, default root" source.md "$AGENTS/listed.md" render_listed rendered-default.md "$AGENTS"
set_state "specter_root = '$CUSTOM_ROOT'"$'\n'
expect_transform "render, custom root" source.md "$AGENTS/listed.md" render_listed rendered-custom.md "$AGENTS"
expect_transform "render, single-file component" source.md "$HOME/.claude/CLAUDE.md" render_listed rendered-custom.md "$HOME/.claude/CLAUDE.md"
expect_transform "unrender, custom root" rendered-custom.md "$AGENTS/listed.md" unrender_listed source.md "$AGENTS"
clear_state
expect_transform "unrender, name-byte boundary" boundary-rendered.md "$AGENTS/listed.md" unrender_listed boundary-source.md "$AGENTS"

cp "$FIXTURES/source.md" "$AGENTS/unlisted.md"
render_listed "$AGENTS" 2>/dev/null || fail "render: render_listed failed on an unlisted file"
cmp -s "$AGENTS/unlisted.md" "$FIXTURES/source.md" || fail "render: unlisted file was modified"

# A sed replacement would expand '&' to the matched text.
set_state $'specter_root = "'"$HOME"$'/a&b"\n'
printf 'root {{SPECTER_ROOT}}\n' > "$AGENTS/listed.md"
render_listed "$AGENTS" 2>/dev/null || fail "render: render_listed failed for a root with '&'"
[ "$(cat "$AGENTS/listed.md")" = "root ~/a&b" ] || fail "render: root with '&' was mangled: $(cat "$AGENTS/listed.md")"
printf 'root ~/a&b\n' > "$AGENTS/listed.md"
unrender_listed "$AGENTS" 2>/dev/null || fail "unrender: unrender_listed failed for a root with '&'"
[ "$(cat "$AGENTS/listed.md")" = "root {{SPECTER_ROOT}}" ] || fail "unrender: root with '&' was mangled: $(cat "$AGENTS/listed.md")"

# pull_all must not copy a listed file rendered for another root: unrender could not restore its placeholders.
pull_output="$(
  HOME="$WORK/pull-home" ROOT="$WORK/pull-harness"
  mkdir -p "$HOME/.local/state/nuytek-harness" "$ROOT/targets" "$ROOT/agents" "$HOME/.claude/agents"
  printf 'specter_root = "%s"\n' "$HOME/work/specs" > "$HOME/.local/state/nuytek-harness/state.toml"
  printf '%s\n' .claude/agents/stale.md .claude/agents/fresh.md > "$ROOT/targets/rendered-files.txt"
  printf 'root {{SPECTER_ROOT}}\n' > "$ROOT/agents/stale.md"
  printf 'root {{SPECTER_ROOT}}\n' > "$ROOT/agents/fresh.md"
  printf 'plain\n' > "$ROOT/agents/plain.md"
  printf 'root ~/old/place\nedited\n' > "$HOME/.claude/agents/stale.md"
  printf 'root ~/work/specs\nedited\n' > "$HOME/.claude/agents/fresh.md"
  printf 'plain edited\n' > "$HOME/.claude/agents/plain.md"
  # shellcheck source=../lib.sh
  source "$REPO/scripts/lib.sh"
  # shellcheck source=../targets/claude.sh
  source "$REPO/scripts/targets/claude.sh"
  pulled_components() { echo agents; }
  pull_all 2>&1
  [ "$(cat "$ROOT/agents/stale.md")" = 'root {{SPECTER_ROOT}}' ] || echo "FAIL: stale listed file was pulled"
  [ "$(cat "$ROOT/agents/fresh.md")" = $'root {{SPECTER_ROOT}}\nedited' ] || echo "FAIL: fresh listed file was not pulled unrendered"
  [ "$(cat "$ROOT/agents/plain.md")" = 'plain edited' ] || echo "FAIL: unlisted file was not pulled"
)"
case "$pull_output" in *"reinstall first"*) ;; *) fail "pull: no 'reinstall first' note: $pull_output" ;; esac
case "$pull_output" in *FAIL:*) fail "pull: $pull_output" ;; esac

if [ "$failures" -ne 0 ]; then
  echo "$failures check(s) failed" >&2
  exit 1
fi
echo "render-check: all checks passed"
