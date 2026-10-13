#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STAMP="$(date +%Y%m%d-%H%M%S)"
BACKUP_TARGETS=(
  "$HOME/.claude/agents"
  "$HOME/.claude/commands"
  "$HOME/.claude/skills"
  "$HOME/.claude/hooks"
  "$HOME/.claude/sounds"
  "$HOME/.claude/CLAUDE.md"
  "$HOME/.claude/settings.json"
  "$HOME/.claude/common-sdd.md"
  "$HOME/.claude/engram-protocol.md"
  "$HOME/.claude/rules/context7.md"
  "$HOME/.local/bin/codegraph-health"
  "$HOME/.local/bin/claude-out-context"
  "$HOME/.claude.json"
  "$HOME/.config/opencode"
)

usage() {
  echo "usage: $0 [--clean-backups]"
}

clean_backups() {
  local target backup_dir found=false
  for target in "${BACKUP_TARGETS[@]}"; do
    backup_dir="${target}.backups"
    if [ -d "$backup_dir" ] && [ ! -L "$backup_dir" ]; then
      echo "removing $backup_dir"
      rm -rf "$backup_dir"
      found=true
    fi
  done
  "$found" || echo "no installer backups found"
}

case "$#" in
  0) ;;
  1)
    case "$1" in
      --clean-backups)
        clean_backups
        exit 0
        ;;
      --help|-h)
        usage
        exit 0
        ;;
      *)
        usage >&2
        exit 1
        ;;
    esac
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac

backup() {
  local target="$1"
  if [ -e "$target" ] || [ -L "$target" ]; then
    mkdir -p "${target}.backups"
    mv "$target" "${target}.backups/${STAMP}"
  fi
}

install_dir() {
  local source="$1"
  local target="$2"
  mkdir -p "$target"
  cp -a "$source/." "$target/"
}

install_file() {
  local source="$1"
  local target="$2"
  mkdir -p "$(dirname "$target")"
  backup "$target"
  cp "$source" "$target"
}

register_specter_store() {
  local stores status
  local store_root="/home/andrex/dev/specter"

  command -v openspec >/dev/null 2>&1 || {
    echo "error: openspec is required to register store 'specter'" >&2
    exit 1
  }
  [ -d "$store_root" ] || {
    echo "error: OpenSpec store root does not exist: $store_root" >&2
    exit 1
  }
  stores="$(openspec store list --json)" || {
    echo "error: could not inspect registered OpenSpec stores" >&2
    exit 1
  }

  if printf '%s' "$stores" | node -e '
    const stores = JSON.parse(require("fs").readFileSync(0, "utf8")).stores || [];
    const store = stores.find(({ id }) => id === "specter");
    process.exit(store ? (store.root === process.argv[1] ? 0 : 2) : 1);
  ' "$store_root"; then
    return
  else
    status=$?
  fi
  if [ "$status" -eq 1 ]; then
    openspec store register "$store_root" --id specter --yes
    return
  fi
  if [ "$status" -eq 2 ]; then
    echo "error: OpenSpec store 'specter' is registered to a different root" >&2
  else
    echo "error: could not parse registered OpenSpec stores" >&2
  fi
  exit 1
}

remove_cavecrew_assets() {
  local asset
  local -a claude_assets

  rm -f -- "$HOME/.config/opencode/agents/cavecrew-builder.md"
  rm -f -- "$HOME/.config/opencode/agents/cavecrew-investigator.md"
  rm -f -- "$HOME/.config/opencode/agents/cavecrew-reviewer.md"
  rm -rf -- "$HOME/.config/opencode/skills/cavecrew"
  node - "$HOME/.config/opencode/.caveman-opencode-ownership.json" <<'NODE'
const fs = require('fs');
const target = process.argv[2];
if (fs.existsSync(target)) {
  const ownership = JSON.parse(fs.readFileSync(target, 'utf8'));
  for (const entry of [
    'agents/cavecrew-builder.md',
    'agents/cavecrew-investigator.md',
    'agents/cavecrew-reviewer.md',
    'skills/cavecrew',
  ]) delete ownership.entries?.[entry];
  fs.writeFileSync(target, JSON.stringify(ownership, null, 2) + '\n');
}
NODE

  shopt -s nullglob
  claude_assets=(
    "$HOME/.claude/plugins/cache/caveman/caveman/"*/agents/cavecrew-*.md
    "$HOME/.claude/plugins/cache/caveman/caveman/"*/plugins/caveman/agents/cavecrew-*.md
    "$HOME/.claude/plugins/marketplaces/caveman/"agents/cavecrew-*.md
    "$HOME/.claude/plugins/marketplaces/caveman/"plugins/caveman/agents/cavecrew-*.md
  )
  for asset in "${claude_assets[@]}"; do
    rm -f -- "$asset"
  done
}

backup "$HOME/.claude/agents"
backup "$HOME/.claude/commands"
backup "$HOME/.claude/skills"
backup "$HOME/.claude/hooks"
backup "$HOME/.claude/sounds"
install_dir "$ROOT/agents" "$HOME/.claude/agents"
install_dir "$ROOT/commands" "$HOME/.claude/commands"
install_dir "$ROOT/skills" "$HOME/.claude/skills"
install_dir "$ROOT/hooks" "$HOME/.claude/hooks"
install_dir "$ROOT/opencode/sounds" "$HOME/.claude/sounds"
chmod +x "$HOME/.claude/hooks"/*.sh 2>/dev/null || true
install_file "$ROOT/configs/CLAUDE.md" "$HOME/.claude/CLAUDE.md"
install_file "$ROOT/configs/claude-settings.json" "$HOME/.claude/settings.json"
install_file "$ROOT/configs/common-sdd.md" "$HOME/.claude/common-sdd.md"
install_file "$ROOT/configs/engram-protocol.md" "$HOME/.claude/engram-protocol.md"
mkdir -p "$HOME/.claude/rules"
install_file "$ROOT/configs/context7.md" "$HOME/.claude/rules/context7.md"
mkdir -p "$HOME/.local/bin"
install_file "$ROOT/scripts/codegraph-health.sh" "$HOME/.local/bin/codegraph-health"
chmod +x "$HOME/.local/bin/codegraph-health"
install_file "$ROOT/scripts/claude-out-context.sh" "$HOME/.local/bin/claude-out-context"
chmod +x "$HOME/.local/bin/claude-out-context"

node - "$ROOT/configs/claude-mcp.json" "$HOME/.claude.json" "$STAMP" <<'NODE'
const fs = require('fs');
const [source, target, stamp] = process.argv.slice(2);
const mcpServers = JSON.parse(fs.readFileSync(source, 'utf8'));
const existing = fs.existsSync(target) ? fs.readFileSync(target, 'utf8') : null;
const config = existing === null ? {} : JSON.parse(existing);
config.mcpServers = { ...(config.mcpServers || {}), ...mcpServers };
if (existing !== null) {
  fs.mkdirSync(`${target}.backups`, { recursive: true });
  fs.renameSync(target, `${target}.backups/${stamp}`);
}
fs.writeFileSync(target, JSON.stringify(config, null, 2) + '\n');
NODE

backup "$HOME/.config/opencode"
mkdir -p "$HOME/.config/opencode"
tar \
  --exclude='./node_modules' \
  --exclude='./sessions' \
  --exclude='./cache' \
  --exclude='./logs' \
  -C "$ROOT/opencode" -cf - . | tar -C "$HOME/.config/opencode" -xf -

# node_modules is excluded from the tar deploy; refresh it so the V2 shims
# resolve @opencode/plugin and the notifier package from the config dir.
if command -v npm >/dev/null 2>&1; then
  (cd "$HOME/.config/opencode" && npm install --no-audit --no-fund)
fi

if [ -t 0 ]; then
  read -r -p "Register OpenSpec store 'specter' from /home/andrex/dev/specter? [y/N] " register_store
  case "$register_store" in
    [Yy]|[Yy][Ee][Ss]) register_specter_store ;;
  esac
fi

caveman_dir="$(mktemp -d)"
trap 'rm -rf -- "$caveman_dir"' EXIT
git clone --depth 1 --branch main https://github.com/JuliusBrussee/caveman.git "$caveman_dir"
(
  cd "$caveman_dir"
  # opencode side comes from the repo mirror (V2-ported plugin);
  # upstream caveman installer is V1-only and would clobber the port.
  node bin/install.js --only claude
)
remove_cavecrew_assets

# The long-running opencode service caches its plugin module graph; without a
# restart it fails to resolve freshly npm-installed packages for local plugins.
if [ -t 0 ]; then
  read -r -p "Restart the opencode service now? (disconnects active sessions) [y/N] " restart_service
  case "$restart_service" in
    [Yy]|[Yy][Ee][Ss]) opencode service restart ;;
  esac
fi

echo "installed code-agents config"
echo "next: set GITHUB_TOKEN, authenticate Linear/Claude connectors, ensure ~/.local/bin is in PATH, then run: claude doctor && opencode debug config"
