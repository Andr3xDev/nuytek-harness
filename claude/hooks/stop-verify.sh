#!/usr/bin/env bash
# Stop hook: warns when tests or lint fail. It never blocks.
# Blocking would break TDD RED and can loop forever; real verification lives in
# implementer and sdd-verify. Always exits 0.

INPUT=$(cat)

command -v jq >/dev/null 2>&1 || exit 0
[ "$(printf '%s' "$INPUT" | jq -r '.stop_hook_active // false')" = "true" ] && exit 0

# Clean tree: nothing changed, nothing to verify.
if git rev-parse --is-inside-work-tree >/dev/null 2>&1 && [ -z "$(git status --porcelain)" ]; then
  exit 0
fi

CHECK_TIMEOUT_SECONDS=25
failures=()

run_check() {
  local label=$1
  local extra_ok_code=$2
  shift 2
  timeout "$CHECK_TIMEOUT_SECONDS" "$@" >/dev/null 2>&1
  case $? in
    0 | "$extra_ok_code") ;;
    124) failures+=("$label timed out") ;;
    *) failures+=("$label failing") ;;
  esac
}

if [ -f pyproject.toml ] || [ -f setup.py ] || [ -f pytest.ini ]; then
  command -v pytest >/dev/null 2>&1 && run_check "pytest" 5 pytest --tb=no --no-header -q
  [ -f pyproject.toml ] && command -v ruff >/dev/null 2>&1 && run_check "ruff" 0 ruff check --quiet
elif [ -f package.json ]; then
  jq -e '.scripts.test' package.json >/dev/null 2>&1 && run_check "npm test" 0 npm test --silent
fi

if [ "${#failures[@]}" -gt 0 ]; then
  summary=$(IFS=', '; echo "${failures[*]}")
  jq -n --arg message "Stop check (warning, not blocking): $summary" '{systemMessage: $message}'
fi

exit 0
