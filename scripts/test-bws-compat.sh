#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
fake="$tmp/bws"

cat >"$fake" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
case "${1:-}" in
  --version)
    printf 'bws 1.0.0-test\n'
    ;;
  secret)
    printf 'create delete edit get list\n'
    ;;
  run)
    printf '%s\n' '--shell --no-inherit-env --project-id --uuids-as-keynames'
    ;;
  --help)
    if [[ "${FAKE_BWS_BROKEN:-}" == "1" ]]; then
      printf '%s\n' '--output --color --access-token --config-file --profile secret run'
    else
      printf '%s\n' '--output --color --access-token --config-file --profile --server-url secret run'
    fi
    ;;
  *)
    exit 2
    ;;
esac
EOF
chmod +x "$fake"

"$root/scripts/check-bws-compat.sh" "$fake" >/dev/null

if FAKE_BWS_BROKEN=1 "$root/scripts/check-bws-compat.sh" "$fake" >/dev/null 2>&1; then
  printf 'compatibility check accepted a missing official flag\n' >&2
  exit 1
fi
