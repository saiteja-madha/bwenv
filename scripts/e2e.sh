#!/usr/bin/env bash
set -uo pipefail

RUN_ID="$(date +%s)_$$_${RANDOM}"
APP="e2e_${RUN_ID}"
SHARED_TZ_KEY="E2E_${RUN_ID}_TZ"
SHARED_LOCALE_KEY="E2E_${RUN_ID}_LOCALE"
FAILED=0

fail() { printf '  FAIL  %s\n' "$*" >&2; FAILED=1; }
pass() { printf '  PASS  %s\n' "$*"; }
check() {
  local label="$1"
  shift
  if "$@" >/dev/null 2>&1; then
    pass "$label"
  else
    fail "$label"
  fi
}

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

# Cleanup is restricted to this run's random app namespace and two exact,
# run-specific shared keys. It never deletes generic shared keys.
cleanup() {
  local original_status="$1"
  local cleanup_failed=0
  local ids

  trap - EXIT
  ids="$(
    bws_raw secret list "$BWS_PROJECT_ID" 2>/dev/null |
      jq -r \
        --arg app_prefix "${APP}__" \
        --arg shared_tz "shared__${SHARED_TZ_KEY}" \
        --arg shared_locale "shared__${SHARED_LOCALE_KEY}" \
        'if type == "array" then
           .[]
           | select(
               (.key | startswith($app_prefix))
               or .key == $shared_tz
               or .key == $shared_locale
             )
           | .id
         else
           empty
         end'
  )" || {
    printf 'WARN: unable to list test secrets during cleanup\n' >&2
    cleanup_failed=1
    ids=""
  }

  while IFS= read -r id; do
    [[ -z "$id" ]] && continue
    if ! bws_raw secret delete "$id" >/dev/null 2>&1; then
      printf 'WARN: unable to delete test secret %s\n' "$id" >&2
      cleanup_failed=1
    fi
  done <<<"$ids"

  if [[ "$cleanup_failed" -ne 0 && "$original_status" -eq 0 ]]; then
    original_status=1
  fi
  exit "$original_status"
}
trap 'cleanup "$?"' EXIT

strip_cr() { tr -d '\r'; }

echo "=== bwenv e2e ==="
echo "Project: $BWS_PROJECT_ID"
echo "App:     $APP"
echo "Scope:   only ${APP}__* and this run's unique shared keys"
echo ""

# ── Sanity ──────────────────────────────────────────────────────────
echo "--- Sanity (no bws project needed) ---"
check "version" bwenv version
check "completion bash" bwenv completion bash
check "completion zsh" bwenv completion zsh
echo ""

# ── CRUD ────────────────────────────────────────────────────────────
echo "--- CRUD ---"

check "create HOST" bwenv_cmd create "$APP" HOST db.example.com
check "create PORT" bwenv_cmd create "$APP" PORT 5432
check "create SECRET" bwenv_cmd create "$APP" SECRET 's3cret!'

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

check "edit PORT value" bwenv_cmd edit "$APP" PORT --value=9090

output=$(bwenv_cmd get "$APP" PORT 2>&1 | strip_cr)
if echo "$output" | jq -e '.value == "9090"' >/dev/null 2>&1; then
  pass "verify PORT=9090"
else
  fail "verify PORT: $output"
fi

check "edit PORT -> SERVICE_PORT" bwenv_cmd edit "$APP" PORT --key=SERVICE_PORT

output=$(bwenv_cmd get "$APP" SERVICE_PORT 2>&1 | strip_cr)
if echo "$output" | jq -e '.key == "SERVICE_PORT" and .value == "9090"' >/dev/null 2>&1; then
  pass "verify SERVICE_PORT=9090"
else
  fail "verify SERVICE_PORT: $output"
fi

check "delete SERVICE_PORT HOST" bwenv_cmd delete "$APP" SERVICE_PORT HOST

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

output=$(printf "DB_USER=root\nLOG_LEVEL=debug" | bwenv_cmd import "$APP" - 2>&1 | strip_cr)
UNCHANGED=$(echo "$output" | jq -r '.unchanged | length')
if [[ "$UNCHANGED" -eq 2 ]]; then
  pass "import reports 2 unchanged secrets"
else
  fail "import unchanged: expected 2, got $UNCHANGED: $output"
fi

echo ""

# ── Shared ──────────────────────────────────────────────────────────
echo "--- Shared ---"

check "create unique shared timezone" bwenv_cmd create shared "$SHARED_TZ_KEY" UTC
check "create unique shared locale" bwenv_cmd create shared "$SHARED_LOCALE_KEY" en_US
check "create app override for shared timezone" \
  bwenv_cmd create "$APP" "$SHARED_TZ_KEY" America/New_York

output=$(bwenv_cmd list "$APP" 2>&1 | strip_cr)
APP_TZ=$(echo "$output" | jq -r --arg key "$SHARED_TZ_KEY" '.[] | select(.key == $key) | .value')
if [[ "$APP_TZ" == "America/New_York" ]]; then
  pass "list includes app override (no --include-shared)"
else
  fail "list app override: $APP_TZ"
fi

output=$(bwenv_cmd list "$APP" --include-shared 2>&1 | strip_cr)
TZ_VAL=$(echo "$output" | jq -r --arg key "$SHARED_TZ_KEY" '.[] | select(.key == $key) | .value')
LOCALE_VAL=$(echo "$output" | jq -r --arg key "$SHARED_LOCALE_KEY" '.[] | select(.key == $key) | .value')
if [[ "$TZ_VAL" == "America/New_York" && "$LOCALE_VAL" == "en_US" ]]; then
  pass "list --include-shared: app value overrides shared, shared-only value inherited"
else
  fail "list --include-shared: override=$TZ_VAL inherited=$LOCALE_VAL"
fi

output=$(bwenv_cmd get "$APP" "$SHARED_TZ_KEY" --include-shared 2>&1 | strip_cr)
if echo "$output" | jq -e '.value == "America/New_York"' >/dev/null 2>&1; then
  pass "get --include-shared returns app value"
else
  fail "get --include-shared: $output"
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

TZ_VAL=$(bwenv_cmd run "$APP" --include-shared -- printenv "$SHARED_TZ_KEY" 2>&1 | strip_cr | head -1)
if [[ "$TZ_VAL" == "America/New_York" ]]; then
  pass "run --include-shared: app value overrides shared"
else
  fail "run --include-shared: expected America/New_York, got '$TZ_VAL'"
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

if bwenv_cmd create "$APP" DASHED -- '--flag-value' >/dev/null 2>&1; then
  pass "create preserves a value beginning with --"
else
  fail "create rejected a value beginning with --"
fi
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

check "create --dry-run" bwenv_cmd create "$APP" DRY_RUN_TEST "should-not-exist" --dry-run
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
