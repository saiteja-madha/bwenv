#!/usr/bin/env bash
set -uo pipefail

APP="e2e_$(date +%s)"
FAILED=0

fail() { printf '  FAIL  %s\n' "$*" >&2; FAILED=1; }
pass() { printf '  PASS  %s\n' "$*"; }

require() {
  command -v "$1" >/dev/null 2>&1 || {
    echo "FAIL: $1 is required" >&2
    exit 1
  }
}

require bws
require bwenv
require jq

: "${BWS_ACCESS_TOKEN:?set BWS_ACCESS_TOKEN}"
: "${BWS_PROJECT_ID:?set BWS_PROJECT_ID}"

bws_raw() { bws "$@" --output json --color no; }
bwenv_cmd() { bwenv --project-id "$BWS_PROJECT_ID" "$@"; }

# ── Cleanup by key pattern ─────────────────────────────────────────
# Deletes every secret whose key starts with the given prefix.
# Handles empty/error responses from bws without jq errors.
cleanup_prefix() {
  local prefix="$1"
  bws_raw secret list "$BWS_PROJECT_ID" 2>/dev/null | jq -r '
    if type == "array" then
      .[] | select(.key | startswith("'"$prefix"'")) | .id
    else
      empty
    end
  ' 2>/dev/null | while read -r id; do
    [[ -n "$id" ]] && bws_raw secret delete "$id" >/dev/null 2>&1 || true
  done
}

cleanup() {
  cleanup_prefix "${APP}__"
  cleanup_prefix shared__TZ
  cleanup_prefix shared__LOCALE
}
trap cleanup EXIT

# ── Startup: delete leftovers from aborted runs ────────────────────
echo "--- Startup: cleaning stale secrets ---"
cleanup
echo ""

strip_cr() { tr -d '\r'; }

echo "=== bwenv e2e ==="
echo "Project: $BWS_PROJECT_ID"
echo "App:     $APP"
echo ""

# ── Sanity ──────────────────────────────────────────────────────────
echo "--- Sanity (no bws project needed) ---"
bwenv version >/dev/null 2>&1 && pass "version" || fail "version"
bwenv completion bash >/dev/null 2>&1 && pass "completion bash" || fail "completion bash"
bwenv completion zsh >/dev/null 2>&1 && pass "completion zsh" || fail "completion zsh"
echo ""

# ── CRUD ────────────────────────────────────────────────────────────
echo "--- CRUD ---"

bws_raw secret create "${APP}__HOST" db.example.com "$BWS_PROJECT_ID" >/dev/null
bws_raw secret create "${APP}__PORT" 5432 "$BWS_PROJECT_ID" >/dev/null
bws_raw secret create "${APP}__SECRET" s3cret! "$BWS_PROJECT_ID" >/dev/null
pass "created 3 secrets"

output=$(bwenv_cmd get "$APP" HOST 2>&1 | strip_cr)
if echo "$output" | jq -e '.key == "HOST" and .value == "db.example.com"' >/dev/null 2>&1; then
  pass "get HOST"
else
  fail "get HOST: $output"
fi

output=$(bwenv_cmd list "$APP" 2>&1 | strip_cr)
COUNT=$(echo "$output" | jq length)
if [[ "$COUNT" -eq 3 ]]; then
  pass "list: $COUNT secrets"
else
  fail "list: expected 3, got $COUNT: $output"
fi

bwenv_cmd edit "$APP" PORT --value=9090 >/dev/null 2>&1 && pass "edit PORT value" || fail "edit PORT value"

output=$(bwenv_cmd get "$APP" PORT 2>&1 | strip_cr)
if echo "$output" | jq -e '.value == "9090"' >/dev/null 2>&1; then
  pass "verify PORT=9090"
else
  fail "verify PORT: $output"
fi

bwenv_cmd edit "$APP" PORT --key=SERVICE_PORT >/dev/null 2>&1 && pass "edit PORT -> SERVICE_PORT" || fail "edit PORT -> SERVICE_PORT"

output=$(bwenv_cmd get "$APP" SERVICE_PORT 2>&1 | strip_cr)
if echo "$output" | jq -e '.key == "SERVICE_PORT" and .value == "9090"' >/dev/null 2>&1; then
  pass "verify SERVICE_PORT=9090"
else
  fail "verify SERVICE_PORT: $output"
fi

bwenv_cmd delete "$APP" SERVICE_PORT HOST >/dev/null 2>&1 && pass "delete SERVICE_PORT HOST" || fail "delete SERVICE_PORT HOST"

output=$(bwenv_cmd list "$APP" 2>&1 | strip_cr)
COUNT=$(echo "$output" | jq length)
if [[ "$COUNT" -eq 1 ]]; then
  pass "list: 1 remaining (SECRET)"
else
  fail "list: expected 1, got $COUNT: $output"
fi

echo ""

# ── Import ──────────────────────────────────────────────────────────
echo "--- Import ---"

output=$(printf "DB_USER=admin\nDB_PASS=hunter2" | bwenv_cmd import "$APP" - 2>&1 | strip_cr)
if echo "$output" | jq -e '.created | length == 2' >/dev/null 2>&1; then
  pass "import created 2 secrets"
else
  fail "import created: $output"
fi

output=$(printf "DB_USER=root\nLOG_LEVEL=debug" | bwenv_cmd import "$APP" - 2>&1 | strip_cr)
CREATED=$(echo "$output" | jq -r '.created | length')
UPDATED=$(echo "$output" | jq -r '.updated | length')
if [[ "$UPDATED" -eq 1 && "$CREATED" -eq 1 ]]; then
  pass "import upserted 1 (DB_USER updated), created 1 (LOG_LEVEL)"
else
  fail "import upsert: created=$CREATED updated=$UPDATED: $output"
fi

echo ""

# ── Shared ──────────────────────────────────────────────────────────
echo "--- Shared ---"

bws_raw secret create shared__TZ UTC "$BWS_PROJECT_ID" >/dev/null
bws_raw secret create shared__LOCALE en_US "$BWS_PROJECT_ID" >/dev/null
bws_raw secret create "${APP}__TZ" America/New_York "$BWS_PROJECT_ID" >/dev/null

output=$(bwenv_cmd list "$APP" 2>&1 | strip_cr)
APP_TZ=$(echo "$output" | jq -r '.[] | select(.key == "TZ") | .value')
if [[ "$APP_TZ" == "America/New_York" ]]; then
  pass "list includes app TZ (no --include-shared)"
else
  fail "list app TZ: $APP_TZ"
fi

output=$(bwenv_cmd list "$APP" --include-shared 2>&1 | strip_cr)
TZ_VAL=$(echo "$output" | jq -r '.[] | select(.key == "TZ") | .value')
LOCALE_VAL=$(echo "$output" | jq -r '.[] | select(.key == "LOCALE") | .value')
if [[ "$TZ_VAL" == "America/New_York" && "$LOCALE_VAL" == "en_US" ]]; then
  pass "list --include-shared: app TZ overrides shared, LOCALE inherited"
else
  fail "list --include-shared: TZ=$TZ_VAL LOCALE=$LOCALE_VAL"
fi

output=$(bwenv_cmd get "$APP" TZ --include-shared 2>&1 | strip_cr)
if echo "$output" | jq -e '.value == "America/New_York"' >/dev/null 2>&1; then
  pass "get TZ --include-shared returns app value"
else
  fail "get TZ --include-shared: $output"
fi

echo ""

# ── Run ─────────────────────────────────────────────────────────────
echo "--- Run ---"

VAL=$(bwenv_cmd run "$APP" -- printenv SECRET 2>&1 | strip_cr | head -1)
if [[ "$VAL" == "s3cret!" ]]; then
  pass "run: SECRET=s3cret!"
else
  fail "run: SECRET expected s3cret!, got '$VAL'"
fi

LEAKED=$(bwenv_cmd run "$APP" -- printenv BWS_ACCESS_TOKEN 2>&1 | strip_cr | head -1)
if [[ -z "$LEAKED" ]]; then
  pass "run: BWS_ACCESS_TOKEN stripped"
else
  fail "run: BWS_ACCESS_TOKEN leaked: '$LEAKED'"
fi

PREFIXED=$(bwenv_cmd run "$APP" -- printenv "${APP}__SECRET" 2>&1 | strip_cr | head -1)
if [[ -z "$PREFIXED" ]]; then
  pass "run: app__ prefix stripped"
else
  fail "run: app__ prefix leaked: '$PREFIXED'"
fi

TZ_VAL=$(bwenv_cmd run "$APP" --include-shared -- printenv TZ 2>&1 | strip_cr | head -1)
if [[ "$TZ_VAL" == "America/New_York" ]]; then
  pass "run --include-shared: TZ=America/New_York"
else
  fail "run --include-shared: TZ expected America/New_York, got '$TZ_VAL'"
fi

INHERIT_COUNT=$(bwenv_cmd run "$APP" --no-inherit-env -- env 2>&1 | strip_cr | wc -l | tr -d ' ')
if [[ "$INHERIT_COUNT" -ge 4 ]]; then
  pass "run --no-inherit-env: $INHERIT_COUNT vars"
else
  fail "run --no-inherit-env: only $INHERIT_COUNT vars"
fi

UUID_VAR=$(bwenv_cmd run "$APP" --uuids-as-keynames -- env 2>&1 | strip_cr | grep -E '^_[0-9a-f_]+=' | head -1 || true)
if [[ -n "$UUID_VAR" ]]; then
  pass "run --uuids-as-keynames: $UUID_VAR"
else
  fail "run --uuids-as-keynames: no uuid vars found"
fi

echo ""

# ── Output Formats ──────────────────────────────────────────────────
echo "--- Output formats ---"

for fmt in json yaml env table tsv; do
  output=$(bwenv_cmd list "$APP" --output "$fmt" 2>&1 | strip_cr)
  if [[ -n "$output" ]]; then
    pass "list --output $fmt"
  else
    fail "list --output $fmt produced empty output"
  fi
done

echo ""

# ── Edge Cases ──────────────────────────────────────────────────────
echo "--- Edge cases ---"

# Values starting with '--' cannot be passed through bwenv directly
# because Cobra interprets them as unknown flags. bwenv handles them
# internally via the '--' separator when calling bws, but the CLI
# itself can't accept them as positional args.
# Test via raw bws with '--' before the key to protect '--flag-value':
bws --output json --color no secret create -- "${APP}__DASHED" '--flag-value' "$BWS_PROJECT_ID" >/dev/null
output=$(bwenv_cmd get "$APP" DASHED 2>&1 | strip_cr)
if echo "$output" | jq -e '.value == "--flag-value"' >/dev/null 2>&1; then
  pass "get DASHED: --flag-value preserved"
else
  fail "get DASHED: $output"
fi

output=$(bwenv_cmd create "$APP" SECRET "dup" 2>&1 | strip_cr)
if echo "$output" | grep -qi "already exists"; then
  pass "create duplicate: rejected"
else
  fail "create duplicate: not rejected: $output"
fi

output=$(bwenv_cmd get "$APP" NONEXISTENT 2>&1 | strip_cr)
if echo "$output" | grep -qi "not found"; then
  pass "get nonexistent: not found error"
else
  fail "get nonexistent: $output"
fi

output=$(bwenv_cmd delete "$APP" DASHED DASHED 2>&1 | strip_cr)
if echo "$output" | grep -qi "more than once"; then
  pass "delete duplicate key: rejected"
else
  fail "delete duplicate key: $output"
fi

bwenv_cmd create "$APP" DRY_RUN_TEST "should-not-exist" --dry-run >/dev/null 2>&1 && pass "create --dry-run" || fail "create --dry-run"
output=$(bwenv_cmd list "$APP" 2>&1 | strip_cr)
if echo "$output" | jq -e '.[] | select(.key == "DRY_RUN_TEST")' >/dev/null 2>&1; then
  fail "create --dry-run: key was actually created"
else
  pass "create --dry-run: key not persisted"
fi

echo ""

# ── Summary ─────────────────────────────────────────────────────────
if [[ "$FAILED" -eq 1 ]]; then
  echo "=== FAILED ===" >&2
  exit 1
fi
echo "=== ALL PASSED ==="
