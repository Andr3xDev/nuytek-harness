#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORK="$(mktemp -d)"
trap 'rm -rf -- "$WORK"' EXIT
HOME="$WORK/home"
export HOME
export NUYTEK_SKIP_PLUGINS=1
CLI="$ROOT/scripts/nuytek-harness"

fail() { echo "FAIL: $*" >&2; exit 1; }

mkdir -p "$HOME/.claude"/{agents,commands,skills,hooks,rules}
printf 'old\n' > "$HOME/.claude/agents/stale.md"
printf '{"other":{"keep":true},"mcpServers":{"existing":{"command":"keep"}}}\n' > "$HOME/.claude.json"

"$CLI" install claude </dev/null
SNAP="$(ls -d "$HOME"/.claude-backups/*)"
[ -f "$SNAP/.claude/agents/stale.md" ] || fail "install made no snapshot of replaced agents"
grep -q '"existing"' "$SNAP/.claude.json" || fail "install snapshot lost original .claude.json"
[ -z "$(find "$HOME" -name '*.backups')" ] || fail "per-component backups created"

mcp_check() {
  node - "$HOME" "$ROOT" "$1" <<'NODE'
const fs = require('fs');
const [home, root, mode] = process.argv.slice(2);
for (const file of [`${home}/.claude/settings.json`, `${home}/.claude.json`]) JSON.parse(fs.readFileSync(file, 'utf8'));
const config = JSON.parse(fs.readFileSync(`${home}/.claude.json`, 'utf8'));
const source = JSON.parse(fs.readFileSync(`${root}/claude/claude-mcp.json`, 'utf8'));
if (!config.other?.keep || !config.mcpServers.existing) process.exit(1);
if (mode === 'merged') {
  if (JSON.stringify(config.mcpServers.github) !== JSON.stringify(source.github)) process.exit(1);
} else if (Object.keys(source).some((key) => key in config.mcpServers)) process.exit(1);
NODE
}
mcp_check merged || fail "mcp merge"

[ -f "$HOME/.claude/hooks/stop-verify.sh" ] || fail "hook not deployed"
[ -f "$HOME/.claude/sounds/complete.oga" ] || fail "sound not deployed"
[ -f "$HOME/.claude/rules/context7.md" ] || fail "context7 not deployed"
[ -x "$HOME/.local/bin/claude-out-context" ] || fail "claude-out-context not deployed"
[ ! -e "$HOME/.claude/agents/stale.md" ] || fail "stale agent survived install"
[ -z "$(find "$HOME/.claude/hooks" -name '__pycache__' -o -name '*.pyc')" ] || fail "__pycache__ deployed"

printf 'new\n' > "$HOME/.claude/agents/stale.md"
"$CLI" update agents </dev/null
[ ! -e "$HOME/.claude/agents/stale.md" ] || fail "update agents did not replace dir"
"$CLI" update mcp </dev/null
mcp_check merged || fail "update mcp"
"$CLI" update all </dev/null

printf 'foreign\n' > "$HOME/.claude/hooks/foreign.sh"
"$CLI" remove hooks --yes </dev/null
[ ! -e "$HOME/.claude/hooks/stop-verify.sh" ] || fail "remove hooks left repo file"
[ -f "$HOME/.claude/hooks/foreign.sh" ] || fail "remove hooks deleted foreign file"
[ -f "$(ls -d "$HOME"/.claude-backups/* | tail -1)/.claude/hooks/stop-verify.sh" ] || fail "remove hooks made no snapshot"

"$CLI" remove mcp --yes </dev/null
mcp_check removed || fail "remove mcp"

if "$CLI" remove agents </dev/null; then fail "remove without --yes should abort"; fi
[ -n "$(ls "$HOME/.claude/agents")" ] || fail "aborted remove deleted files"

"$CLI" clean-backups </dev/null
[ ! -e "$HOME/.claude-backups" ] || fail "clean-backups left ~/.claude-backups"

"$CLI" install claude </dev/null
"$CLI" reinstall claude --yes </dev/null
[ -f "$HOME/.claude/hooks/stop-verify.sh" ] || fail "reinstall"
grep -lq '"existing"' "$HOME"/.claude-backups/*/.claude.json || fail "reinstall lost the original .claude.json backup"

if "$CLI" update nonexistent </dev/null; then fail "unknown component should exit non-zero"; fi
ln -s "$CLI" "$WORK/linked-cli"
"$WORK/linked-cli" --help </dev/null | grep -q "^usage:" || fail "symlink invocation"

if command -v claude >/dev/null 2>&1; then
  claude mcp list >/dev/null
else
  echo 'SKIP: claude unavailable; MCP list not checked'
fi

REPO="$WORK/repo"
mkdir -p "$REPO"
tar --exclude='./.git' --exclude='./legacy' -C "$ROOT" -cf - . | tar -C "$REPO" -xf -
printf 'stale\n' > "$REPO/agents/stale.md"
printf 'new\n' > "$HOME/.claude/agents/current.md"
printf 'local\n' > "$HOME/.claude/common-sdd.md"
printf 'secret\n' > "$HOME/.claude/agents/.env"
printf 'secret\n' > "$HOME/.claude/agents/private.key"
printf 'secret\n' > "$HOME/.claude/agents/auth.json"
mkdir -p "$HOME/.claude/hooks/__pycache__"
printf 'x\n' > "$HOME/.claude/hooks/__pycache__/x.pyc"
"$REPO/scripts/nuytek-harness" pull claude </dev/null
STAMP="$(basename "$(ls -d "$REPO/.refresh-backups"/*)")"
[ -f "$REPO/.refresh-backups/$STAMP/agents/README.md" ] || fail "pull made no snapshot"
[ -f "$REPO/agents/current.md" ] || fail "pull did not copy"
[ ! -e "$REPO/agents/stale.md" ] || fail "pull kept stale file"
[ "$(cat "$REPO/shared/common-sdd.md")" = local ] || fail "pull did not update shared md"
for path in .env private.key auth.json; do
  [ ! -e "$REPO/agents/$path" ] || fail "pull copied secret $path"
done
[ -z "$(find "$REPO/claude/hooks" -name '__pycache__' -o -name '*.pyc')" ] || fail "pull copied __pycache__"
"$REPO/scripts/nuytek-harness" pull claude --restore "$STAMP" </dev/null
[ -f "$REPO/agents/README.md" ] && [ ! -e "$REPO/agents/current.md" ] || fail "restore"
"$REPO/scripts/nuytek-harness" clean-backups claude </dev/null
[ ! -e "$REPO/.refresh-backups" ] || fail "clean-backups left .refresh-backups"
echo 'smoke passed'
