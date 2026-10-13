#!/usr/bin/env bash
# Claude Code target for nuytek-harness. Sourced after ROOT and lib.sh are set.

SPECTER_ROOT="${NUYTEK_SPECTER_ROOT:-$HOME/dev/specter}"
MCP_SOURCE="$ROOT/claude/claude-mcp.json"
MCP_TARGET="$HOME/.claude.json"
SETTINGS_SOURCE="$ROOT/claude/claude-settings.json"
REFRESH_BACKUPS="$ROOT/.refresh-backups"

# name kind source(repo-relative) target(HOME-relative) pull(1|-)
# kind: dir | dirx (chmod +x *.sh) | file | filex (chmod +x)
COMPONENT_TABLE=(
  "agents dir agents .claude/agents 1"
  "commands dir claude/commands .claude/commands 1"
  "skills dir skills .claude/skills 1"
  "hooks dirx claude/hooks .claude/hooks 1"
  "sounds dir claude/sounds .claude/sounds 1"
  "claude-md file claude/CLAUDE.md .claude/CLAUDE.md 1"
  "settings file claude/claude-settings.json .claude/settings.json 1"
  "common-sdd file shared/common-sdd.md .claude/common-sdd.md 1"
  "engram-protocol file shared/engram-protocol.md .claude/engram-protocol.md 1"
  "context7 file shared/rules/context7.md .claude/rules/context7.md 1"
  "codegraph-health filex scripts/codegraph-health.sh .local/bin/codegraph-health -"
  "claude-out-context filex claude/scripts/claude-out-context.sh .local/bin/claude-out-context -"
)

declare -gA KIND SRC DEST PULL
COMPONENTS=()
for row in "${COMPONENT_TABLE[@]}"; do
  read -r name kind src dest pull <<<"$row"
  KIND[$name]=$kind SRC[$name]="$ROOT/$src" DEST[$name]="$HOME/$dest" PULL[$name]=$pull
  COMPONENTS+=("$name")
done
# plugins runs last so a failure there leaves everything else deployed.
COMPONENTS+=(mcp specter plugins)
SPECIAL_COMPONENTS=(specter plugins)

# --- generic file/dir components ---------------------------------------

install_generic() {
  local name="$1" kind="${KIND[$1]}" src="${SRC[$1]}" dest="${DEST[$1]}"
  case "$kind" in
    dir|dirx)
      clean_pycache "$src"
      snapshot "$dest"
      rm -rf -- "$dest"
      copy_tree "$src" "$dest"
      clean_pycache "$dest"
      if [ "$kind" = dirx ]; then chmod +x "$dest"/*.sh 2>/dev/null || true; fi
      ;;
    file|filex)
      install_file "$src" "$dest"
      if [ "$kind" = filex ]; then chmod +x "$dest"; fi
      ;;
  esac
}

# Files that exist in both the repo source and the deployed target.
removal_generic() {
  local name="$1" src="${SRC[$1]}" dest="${DEST[$1]}" rel
  case "${KIND[$name]}" in
    dir|dirx)
      [ -d "$src" ] || return 0
      while IFS= read -r rel; do
        if [ -e "$dest/$rel" ]; then echo "$dest/$rel"; fi
      done < <(cd "$src" && find . -type f ! -path '*/__pycache__/*' ! -name '*.pyc' | sed 's#^\./##' | sort)
      ;;
    file|filex)
      if [ -e "$dest" ]; then echo "$dest"; fi
      ;;
  esac
}

remove_generic() {
  local name="$1" src="${SRC[$1]}" dest="${DEST[$1]}" path rel
  snapshot "$dest"
  while IFS= read -r path; do rm -f -- "$path"; done < <(removal_generic "$name")
  case "${KIND[$name]}" in
    dir|dirx)
      while IFS= read -r rel; do rmdir -- "$dest/$rel" 2>/dev/null || true; done \
        < <(cd "$src" && find . -type d | sed 's#^\./##' | sort -r)
      rmdir -- "$dest" 2>/dev/null || true
      ;;
  esac
}

# --- mcp ----------------------------------------------------------------

install_mcp() {
  snapshot "$MCP_TARGET"
  node - "$MCP_SOURCE" "$MCP_TARGET" <<'NODE'
const fs = require('fs');
const [source, target] = process.argv.slice(2);
const mcpServers = JSON.parse(fs.readFileSync(source, 'utf8'));
const existing = fs.existsSync(target) ? fs.readFileSync(target, 'utf8') : null;
const config = existing === null ? {} : JSON.parse(existing);
config.mcpServers = { ...(config.mcpServers || {}), ...mcpServers };
fs.writeFileSync(`${target}.tmp`, JSON.stringify(config, null, 2) + '\n');
fs.renameSync(`${target}.tmp`, target);
NODE
}

mcp_edit() {
  node - "$MCP_SOURCE" "$MCP_TARGET" "$1" <<'NODE'
const fs = require('fs');
const [source, target, mode] = process.argv.slice(2);
if (!fs.existsSync(target)) process.exit(0);
const keys = Object.keys(JSON.parse(fs.readFileSync(source, 'utf8')));
const config = JSON.parse(fs.readFileSync(target, 'utf8'));
const present = keys.filter((key) => config.mcpServers && key in config.mcpServers);
if (mode === 'list') {
  for (const key of present) console.log(`${target}#mcpServers.${key}`);
} else {
  for (const key of present) delete config.mcpServers[key];
  fs.writeFileSync(target, JSON.stringify(config, null, 2) + '\n');
}
NODE
}

removal_mcp() { mcp_edit list; }

remove_mcp() {
  snapshot "$MCP_TARGET"
  mcp_edit remove
}

# --- specter ------------------------------------------------------------

install_specter() {
  local stores status

  command -v openspec >/dev/null 2>&1 || die "openspec is required to register store 'specter'"
  [ -d "$SPECTER_ROOT" ] || die "OpenSpec store root does not exist: $SPECTER_ROOT"
  stores="$(openspec store list --json)" || die "could not inspect registered OpenSpec stores"

  if printf '%s' "$stores" | node -e '
    const stores = JSON.parse(require("fs").readFileSync(0, "utf8")).stores || [];
    const store = stores.find(({ id }) => id === "specter");
    process.exit(store ? (store.root === process.argv[1] ? 0 : 2) : 1);
  ' "$SPECTER_ROOT"; then
    return
  else
    status=$?
  fi
  case "$status" in
    1) openspec store register "$SPECTER_ROOT" --id specter --yes ;;
    2) die "OpenSpec store 'specter' is registered to a different root" ;;
    *) die "could not parse registered OpenSpec stores" ;;
  esac
}

# --- plugins ------------------------------------------------------------

remove_cavecrew_assets() {
  local asset
  local -a claude_assets

  shopt -s nullglob
  claude_assets=(
    "$HOME/.claude/plugins/cache/caveman/caveman/"*/agents/cavecrew-*.md
    "$HOME/.claude/plugins/cache/caveman/caveman/"*/plugins/caveman/agents/cavecrew-*.md
    "$HOME/.claude/plugins/marketplaces/caveman/"agents/cavecrew-*.md
    "$HOME/.claude/plugins/marketplaces/caveman/"plugins/caveman/agents/cavecrew-*.md
  )
  shopt -u nullglob
  for asset in "${claude_assets[@]}"; do
    rm -f -- "$asset"
  done
}

# Tab-separated "kind name arg present(0|1)" rows: settings.json entries vs ~/.claude/plugins state.
plugins_plan() {
  node - "$SETTINGS_SOURCE" "$HOME/.claude/plugins" <<'NODE'
const fs = require('fs');
const [settingsFile, dir] = process.argv.slice(2);
const read = (file) => (fs.existsSync(file) ? JSON.parse(fs.readFileSync(file, 'utf8')) : {});
const settings = read(settingsFile);
const known = read(`${dir}/known_marketplaces.json`);
const installed = read(`${dir}/installed_plugins.json`).plugins || {};
for (const [name, { source }] of Object.entries(settings.extraKnownMarketplaces || {})) {
  console.log(['marketplace', name, source.repo || source.url, name in known ? 1 : 0].join('\t'));
}
for (const id of Object.keys(settings.enabledPlugins || {})) {
  console.log(['plugin', id, id, id in installed ? 1 : 0].join('\t'));
}
NODE
}

# Enable/disable state stays with settings.json; this only installs or updates.
install_plugins() {
  local kind id arg present yes_flag=""
  if [ "${NUYTEK_SKIP_PLUGINS:-}" = 1 ]; then
    echo "skipping plugins (NUYTEK_SKIP_PLUGINS=1)"
    return
  fi
  if ! command -v claude >/dev/null 2>&1; then
    echo "warning: claude not on PATH; skipping plugins" >&2
    return
  fi
  { [ -t 0 ] && [ -t 1 ]; } || yes_flag="-y"
  while IFS=$'\t' read -r kind id arg present; do
    if [ "$kind" = marketplace ]; then
      if [ "$present" = 1 ]; then
        claude plugin marketplace update "$id" || echo "warning: marketplace update failed: $id" >&2
      else
        claude plugin marketplace add "$arg" || echo "warning: marketplace add failed: $arg" >&2
      fi
    elif [ "$present" = 1 ]; then
      claude plugin update "$id" $yes_flag || echo "warning: plugin update failed: $id" >&2
    else
      claude plugin install "$id" || echo "warning: plugin install failed: $id" >&2
    fi
  done < <(plugins_plan)
  remove_cavecrew_assets
}

# --- dispatch -----------------------------------------------------------

install_component() {
  case "$1" in
    mcp|specter|plugins) "install_$1" ;;
    *) install_generic "$1" ;;
  esac
}

removal_list() {
  case "$1" in
    mcp) removal_mcp ;;
    specter|plugins) ;;
    *) removal_generic "$1" ;;
  esac
}

remove_component() {
  case "$1" in
    mcp) remove_mcp ;;
    specter|plugins) ;;
    *) remove_generic "$1" ;;
  esac
}

# --- backups and pull ---------------------------------------------------

clean_backups() {
  local dir found=false
  for dir in "$SNAPSHOT_ROOT" "$REFRESH_BACKUPS"; do
    if [ -d "$dir" ] && [ ! -L "$dir" ]; then
      echo "removing $dir"
      rm -rf -- "$dir"
      found=true
    fi
  done
  "$found" || echo "no installer backups found"
}

PULL_EXCLUDES=(
  --exclude='.git' --exclude='*/.git'
  --exclude='node_modules' --exclude='*/node_modules'
  --exclude='auth.json' --exclude='*/auth.json'
  --exclude='credentials.json' --exclude='*/credentials.json'
  --exclude='.env' --exclude='.env.*' --exclude='*/.env' --exclude='*/.env.*'
  --exclude='*.pem' --exclude='*.key'
)

pulled_components() {
  local name
  for name in "${COMPONENTS[@]}"; do
    if [ "${PULL[$name]:-}" = 1 ]; then echo "$name"; fi
  done
}

# usage: replace_path <from> <to>; replaces <to> with a copy of <from>.
replace_path() {
  local from="$1" to="$2"
  if [ -d "$from" ]; then
    rm -rf -- "$to"
    copy_tree "$from" "$to" "${PULL_EXCLUDES[@]}"
  elif [ -f "$from" ]; then
    rm -f -- "$to"
    mkdir -p "$(dirname "$to")"
    cp "$from" "$to"
  else
    die "missing managed source: $from"
  fi
}

pull_all() {
  local stamp name rel
  stamp="$(date -u +%Y%m%dT%H%M%SZ)"
  while [ -e "$REFRESH_BACKUPS/$stamp" ]; do stamp="${stamp}-1"; done

  while IFS= read -r name; do
    [ -e "${DEST[$name]}" ] || die "deployed path missing, nothing pulled: ${DEST[$name]}"
  done < <(pulled_components)

  (
    umask 077
    while IFS= read -r name; do
      rel="${SRC[$name]#"$ROOT"/}"
      if [ -e "${SRC[$name]}" ]; then replace_path "${SRC[$name]}" "$REFRESH_BACKUPS/$stamp/$rel"; fi
    done < <(pulled_components)
  )

  while IFS= read -r name; do
    replace_path "${DEST[$name]}" "${SRC[$name]}"
  done < <(pulled_components)
  clean_pycache "$ROOT/claude/hooks"
  find "$ROOT" -name .DS_Store -not -path "$ROOT/.git/*" -delete
  echo "refreshed managed config; backup: .refresh-backups/$stamp"
}

pull_restore() {
  local stamp="$1" name rel source="$REFRESH_BACKUPS/$1"
  [ -d "$source" ] || die "backup not found: $stamp"
  while IFS= read -r name; do
    rel="${SRC[$name]#"$ROOT"/}"
    rm -rf -- "${SRC[$name]}"
    if [ -e "$source/$rel" ]; then replace_path "$source/$rel" "${SRC[$name]}"; fi
  done < <(pulled_components)
  echo "restored managed config from .refresh-backups/$stamp"
}
