#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
status=0

while IFS=: read -r file _ target; do
  target="${target%%#*}"
  [[ -z "$target" || "$target" == http://* || "$target" == https://* || "$target" == mailto:* ]] && continue
  target="${target#<}"
  target="${target%>}"
  if [[ "$target" == /* ]]; then
    resolved="${root}${target}"
  else
    resolved="$(dirname "$file")/${target}"
  fi
  if [[ ! -e "$resolved" ]]; then
    printf 'Broken Markdown link: %s -> %s\n' "$file" "$target" >&2
    status=1
  fi
done < <(grep -RnoE '\[[^]]+\]\(([^)]+)\)' "$root/README.md" "$root/CONTRIBUTING.md" "$root/docs" 2>/dev/null | sed -E 's#^([^:]+):([0-9]+):.*\]\(([^)]+)\).*#\1:\2:\3#')

exit "$status"
