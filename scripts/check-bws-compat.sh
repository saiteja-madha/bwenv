#!/usr/bin/env bash
set -euo pipefail

bws_bin="${1:-bws}"

command -v "$bws_bin" >/dev/null 2>&1 || {
  printf 'bws compatibility: executable not found: %s\n' "$bws_bin" >&2
  exit 1
}

require_text() {
  local output="$1"
  local expected="$2"
  local context="$3"
  if ! grep -Fq -- "$expected" <<<"$output"; then
    printf 'bws compatibility: %s is missing %s\n' "$context" "$expected" >&2
    exit 1
  fi
}

root_help="$("$bws_bin" --help 2>&1)"
for option in --output --color --access-token --config-file --profile --server-url; do
  require_text "$root_help" "$option" "global help"
done
require_text "$root_help" "secret" "global help"
require_text "$root_help" "run" "global help"

secret_help="$("$bws_bin" secret --help 2>&1)"
for command in create delete edit get list; do
  require_text "$secret_help" "$command" "secret help"
done

run_help="$("$bws_bin" run --help 2>&1)"
for option in --shell --no-inherit-env --project-id --uuids-as-keynames; do
  require_text "$run_help" "$option" "run help"
done

version="$("$bws_bin" --version 2>&1)"
printf 'bws compatibility: %s\n' "$version"
