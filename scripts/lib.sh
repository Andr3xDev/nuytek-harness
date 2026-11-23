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

SPECTER_PLACEHOLDER='{{SPECTER_ROOT}}'

# Prints the Specter root the TUI saved (top-level specter_root of its state file), else $HOME/dev/specter.
# Keeps the same guard as the hook and the TUI: absolute, strictly under HOME, no ".." segment.
resolve_specter_root() {
  local state="$HOME/.local/state/nuytek-harness/state.toml" root=""
  if [ -f "$state" ]; then
    root="$(awk '
      /^[[:space:]]*\[/ { exit }
      /^[[:space:]]*specter_root[[:space:]]*=/ {
        sub(/^[^=]*=[[:space:]]*/, "")
        if (match($0, /^"[^"]*"/) || match($0, /^'"'"'[^'"'"']*'"'"'/)) {
          if (substr($0, RLENGTH + 1) ~ /^[[:space:]]*(#.*)?$/) print substr($0, 2, RLENGTH - 2)
        }
        exit
      }' "$state")"
  fi
  while [[ "$root" == *//* ]]; do root="${root//\/\///}"; done
  while [[ "$root" == */./* ]]; do root="${root//\/.\///}"; done
  root="${root%/.}"
  case "$root" in
    */../* | */..) printf '%s\n' "$HOME/dev/specter" ;;
    "$HOME"/?*) printf '%s\n' "${root%/}" ;;
    *) printf '%s\n' "$HOME/dev/specter" ;;
  esac
}

# The resolved root as written into deployed files: always under HOME, so "~/<rel>".
rendered_specter_root() {
  local root
  root="$(resolve_specter_root)"
  printf '~/%s\n' "${root#"$HOME"/}"
}

# Files under <base> (default: <component-dest>) whose HOME-relative deployed path is in the rendered list.
_listed_files() {
  local rel="${1#"$HOME"/}" base="${2:-$1}" line file
  [ -f "$ROOT/targets/rendered-files.txt" ] || return 0
  while IFS= read -r line; do
    if [ "$line" = "$rel" ]; then file="$base"
    elif [[ "$line" == "$rel"/* ]]; then file="$base/${line#"$rel"/}"
    else continue
    fi
    if [ -f "$file" ]; then echo "$file"; fi
  done < "$ROOT/targets/rendered-files.txt"
}

# usage: _rewrite_file <file> <content>; atomic replace (temp file + mv, mode kept) so a failed write never truncates the deployed file.
_rewrite_file() {
  local file="$1" tmp
  tmp="$(mktemp "$file.XXXXXX")" || return 1
  cp -p "$file" "$tmp" && printf '%s' "$2" > "$tmp" && mv "$tmp" "$file" || { rm -f -- "$tmp"; return 1; }
}

# usage: render_listed <component-dest>; replaces the placeholder in the listed files under it.
render_listed() {
  local file content value
  value="$(rendered_specter_root)"
  while IFS= read -r file; do
    IFS= read -r -d '' content < "$file" || true
    _rewrite_file "$file" "${content//"$SPECTER_PLACEHOLDER"/"$value"}" || die "could not render $file"
  done < <(_listed_files "$1")
}

# usage: unrender_listed <component-dest> [<base>]; restores the placeholder, but not where the root is
# the prefix of a longer name (~/dev/specter-old). <base> is where the files are when not at <component-dest>.
unrender_listed() {
  local LC_COLLATE=C file content out rest after value
  value="$(rendered_specter_root)"
  while IFS= read -r file; do
    IFS= read -r -d '' content < "$file" || true
    out="" rest="$content"
    while [[ "$rest" == *"$value"* ]]; do
      out+="${rest%%"$value"*}"
      after="${rest#*"$value"}"
      if [[ "${after:0:1}" == [-_0-9a-zA-Z] ]]; then out+="$value"; else out+="$SPECTER_PLACEHOLDER"; fi
      rest="$after"
    done
    _rewrite_file "$file" "$out$rest" || die "could not unrender $file"
  done < <(_listed_files "$1" "${2:-}")
}

# usage: _names_root <text> <root>; true when <root> occurs not as the prefix of a longer name.
_names_root() {
  local LC_COLLATE=C rest="$1" after
  while [[ "$rest" == *"$2"* ]]; do
    after="${rest#*"$2"}"
    [[ "${after:0:1}" == [-_0-9a-zA-Z] ]] || return 0
    rest="$after"
  done
  return 1
}

# usage: stale_listed_files <component-dest> <component-source>; prints the repo files whose deployed copy
# cannot be unrendered: the source has the placeholder, but the copy still holds it or lacks the current root.
stale_listed_files() {
  local file source content repo value
  value="$(rendered_specter_root)"
  while IFS= read -r file; do
    if [ "$file" = "$1" ]; then source="$2"; else source="$2/${file#"$1"/}"; fi
    [ -f "$source" ] || continue
    IFS= read -r -d '' repo < "$source" || true
    [[ "$repo" == *"$SPECTER_PLACEHOLDER"* ]] || continue
    IFS= read -r -d '' content < "$file" || true
    if [[ "$content" == *"$SPECTER_PLACEHOLDER"* ]] || ! _names_root "$content" "$value"; then echo "$source"; fi
  done < <(_listed_files "$1")
}

clean_pycache() {
  local dir="$1"
  [ -d "$dir" ] || return 0
  find "$dir" -type d -name __pycache__ -prune -exec rm -rf -- {} +
  find "$dir" -type f -name '*.pyc' -delete
}
