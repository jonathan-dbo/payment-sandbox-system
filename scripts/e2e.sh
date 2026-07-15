#!/usr/bin/env bash
#
# End-to-end smoke test for the Payment Sandbox API.
#
# Drives the full flow described in E2E_TESTING_GUIDE.md against a live server:
# register -> login -> refresh -> invoice CRUD -> public pay link -> payment intent
# -> admin simulate -> refund request/approve/process -> top-up request/approve
# -> role checks -> dashboard stats -> negative/error cases.
#
# Every request/response (status, headers subset, full JSON body) is logged to a
# timestamped report file under reports/, plus a live colorized summary on stdout.
#
# Usage:
#   ./scripts/e2e.sh                     # run against http://localhost:8080
#   BASE_URL=http://localhost:9090 ./scripts/e2e.sh
#   ./scripts/e2e.sh --keep-going         # don't exit on first failed assertion
#
# Exit code: 0 if all assertions passed, 1 otherwise.

set -uo pipefail

# ---------------------------------------------------------------------------
# Config
# ---------------------------------------------------------------------------
BASE_URL="${BASE_URL:-http://localhost:8080}"
KEEP_GOING=0
for arg in "$@"; do
  case "$arg" in
    --keep-going) KEEP_GOING=1 ;;
  esac
done

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
REPORT_DIR="$ROOT_DIR/reports"
mkdir -p "$REPORT_DIR"

TS="$(date +%Y%m%d-%H%M%S)"
REPORT_FILE="$REPORT_DIR/e2e-$TS.log"
JSON_SUMMARY_FILE="$REPORT_DIR/e2e-$TS.summary.json"
LATEST_LINK="$REPORT_DIR/latest.log"

# Unique-ish suffix so re-runs don't collide on unique email constraints.
RUN_ID="$(date +%s)-$$"

PASS_COUNT=0
FAIL_COUNT=0
STEP_NUM=0
declare -a SUMMARY_ROWS=()

# ---------------------------------------------------------------------------
# Colors (stdout only; report file stays plain text)
# ---------------------------------------------------------------------------
if [ -t 1 ]; then
  C_RESET=$'\033[0m'; C_GREEN=$'\033[32m'; C_RED=$'\033[31m'
  C_YELLOW=$'\033[33m'; C_BLUE=$'\033[34m'; C_BOLD=$'\033[1m'; C_DIM=$'\033[2m'
else
  C_RESET=""; C_GREEN=""; C_RED=""; C_YELLOW=""; C_BLUE=""; C_BOLD=""; C_DIM=""
fi

log() {
  # Plain text to report file, no color codes.
  printf '%s\n' "$1" >> "$REPORT_FILE"
}

section() {
  local title="$1"
  printf '\n%s\n' "${C_BOLD}${C_BLUE}==> ${title}${C_RESET}"
  log ""
  log "==================================================================="
  log "$title"
  log "==================================================================="
}

# ---------------------------------------------------------------------------
# HTTP helper
#
# request METHOD PATH [BODY] [TOKEN]
# Sets globals: HTTP_STATUS, HTTP_BODY (raw body text)
# ---------------------------------------------------------------------------
request() {
  local method="$1" path="$2" body="${3:-}" token="${4:-}"
  local url="${BASE_URL}${path}"
  local -a curl_args=(-sS -o /tmp/e2e_body.$$ -w "%{http_code}" -X "$method" "$url" -H "Content-Type: application/json")
  if [ -n "$token" ]; then
    curl_args+=(-H "Authorization: Bearer $token")
  fi
  if [ -n "$body" ]; then
    curl_args+=(-d "$body")
  fi

  STEP_NUM=$((STEP_NUM + 1))

  HTTP_STATUS="$(curl "${curl_args[@]}")"
  HTTP_BODY="$(cat /tmp/e2e_body.$$ 2>/dev/null)"
  rm -f /tmp/e2e_body.$$

  local pretty_body
  pretty_body="$(printf '%s' "$HTTP_BODY" | jq . 2>/dev/null || printf '%s' "$HTTP_BODY")"

  log ""
  log "[$STEP_NUM] $method $path"
  if [ -n "$token" ]; then
    log "  Authorization: Bearer ${token:0:16}...(truncated)"
  fi
  if [ -n "$body" ]; then
    log "  Request body:"
    printf '%s\n' "$(printf '%s' "$body" | jq . 2>/dev/null || printf '%s' "$body")" | sed 's/^/    /' >> "$REPORT_FILE"
  fi
  log "  Response status: $HTTP_STATUS"
  log "  Response body:"
  printf '%s\n' "$pretty_body" | sed 's/^/    /' >> "$REPORT_FILE"
}

# ---------------------------------------------------------------------------
# Assertions
# ---------------------------------------------------------------------------
assert_status() {
  local expected="$1" desc="$2"
  if [ "$HTTP_STATUS" = "$expected" ]; then
    PASS_COUNT=$((PASS_COUNT + 1))
    printf '  %s✓%s [%3s] %s\n' "$C_GREEN" "$C_RESET" "$HTTP_STATUS" "$desc"
    log "  RESULT: PASS (expected $expected, got $HTTP_STATUS) -- $desc"
    SUMMARY_ROWS+=("PASS|$HTTP_STATUS|$expected|$desc")
  else
    FAIL_COUNT=$((FAIL_COUNT + 1))
    printf '  %s✗%s [%3s] %s (expected %s)\n' "$C_RED" "$C_RESET" "$HTTP_STATUS" "$desc" "$expected"
    printf '      %sbody:%s %s\n' "$C_DIM" "$C_RESET" "$(printf '%s' "$HTTP_BODY" | head -c 300)"
    log "  RESULT: FAIL (expected $expected, got $HTTP_STATUS) -- $desc"
    SUMMARY_ROWS+=("FAIL|$HTTP_STATUS|$expected|$desc")
    if [ "$KEEP_GOING" -ne 1 ]; then
      finalize
      exit 1
    fi
  fi
}

assert_field_equals() {
  local field="$1" expected="$2" desc="$3"
  local actual
  actual="$(printf '%s' "$HTTP_BODY" | jq -r "$field" 2>/dev/null)"
  if [ "$actual" = "$expected" ]; then
    PASS_COUNT=$((PASS_COUNT + 1))
    printf '  %s✓%s %s (%s = %s)\n' "$C_GREEN" "$C_RESET" "$desc" "$field" "$actual"
    log "  RESULT: PASS -- $desc ($field = $actual)"
    SUMMARY_ROWS+=("PASS|$actual|$expected|$desc")
  else
    FAIL_COUNT=$((FAIL_COUNT + 1))
    printf '  %s✗%s %s (%s = %s, expected %s)\n' "$C_RED" "$C_RESET" "$desc" "$field" "$actual" "$expected"
    log "  RESULT: FAIL -- $desc ($field = $actual, expected $expected)"
    SUMMARY_ROWS+=("FAIL|$actual|$expected|$desc")
    if [ "$KEEP_GOING" -ne 1 ]; then
      finalize
      exit 1
    fi
  fi
}

extract() {
  # extract JQ_FILTER -- pulls a value out of $HTTP_BODY
  printf '%s' "$HTTP_BODY" | jq -r "$1" 2>/dev/null
}

finalize() {
  section "SUMMARY"
  printf '%s\n' "Total: $((PASS_COUNT + FAIL_COUNT))  ${C_GREEN}Passed: $PASS_COUNT${C_RESET}  ${C_RED}Failed: $FAIL_COUNT${C_RESET}"
  log "Total: $((PASS_COUNT + FAIL_COUNT))  Passed: $PASS_COUNT  Failed: $FAIL_COUNT"

  {
    printf '['
    local first=1
    for row in "${SUMMARY_ROWS[@]}"; do
      IFS='|' read -r result actual expected desc <<< "$row"
      [ "$first" -eq 1 ] && first=0 || printf ','
      jq -n --arg result "$result" --arg actual "$actual" --arg expected "$expected" --arg desc "$desc" \
        '{result:$result, actual:$actual, expected:$expected, description:$desc}'
    done
    printf ']'
  } | jq -s 'add // []' > "$JSON_SUMMARY_FILE" 2>/dev/null || true

  ln -sf "$(basename "$REPORT_FILE")" "$LATEST_LINK" 2>/dev/null || cp "$REPORT_FILE" "$LATEST_LINK"

  printf '\n%sFull log:%s    %s\n' "$C_BOLD" "$C_RESET" "$REPORT_FILE"
  printf '%sJSON summary:%s %s\n' "$C_BOLD" "$C_RESET" "$JSON_SUMMARY_FILE"
  printf '%sLatest alias:%s %s\n' "$C_BOLD" "$C_RESET" "$LATEST_LINK"
}

# ---------------------------------------------------------------------------
# Preflight
# ---------------------------------------------------------------------------
command -v jq >/dev/null 2>&1 || { echo "jq is required but not installed (brew install jq)"; exit 1; }
command -v curl >/dev/null 2>&1 || { echo "curl is required but not installed"; exit 1; }

: > "$REPORT_FILE"
log "Payment Sandbox E2E run @ $(date -u +%Y-%m-%dT%H:%M:%SZ)"
log "BASE_URL=$BASE_URL"
log "RUN_ID=$RUN_ID"

printf '%s\n' "${C_BOLD}Payment Sandbox E2E${C_RESET} against ${C_BLUE}${BASE_URL}${C_RESET}"
printf 'Report: %s\n' "$REPORT_FILE"

# ---------------------------------------------------------------------------
# 0. Health check
# ---------------------------------------------------------------------------
section "0. Health check"
request GET "/health"
assert_status 200 "GET /health"

# ---------------------------------------------------------------------------
# 1. Auth flow
# ---------------------------------------------------------------------------
section "1. Auth: register merchant + admin, login, refresh"

MERCHANT_EMAIL="merchant-${RUN_ID}@example.com"
ADMIN_EMAIL="admin-${RUN_ID}@example.com"

request POST "/auth/register" "$(jq -n --arg email "$MERCHANT_EMAIL" '{name:"Demo Merchant", email:$email, password:"secret123", role:"MERCHANT"}')"
assert_status 201 "register merchant"
MERCHANT_TOKEN="$(extract '.token')"
MERCHANT_ID="$(extract '.userId')"

request POST "/auth/register" "$(jq -n --arg email "$ADMIN_EMAIL" '{name:"Admin", email:$email, password:"secret123", role:"ADMIN"}')"
assert_status 201 "register admin"
ADMIN_TOKEN="$(extract '.token')"

# duplicate email -> 409
request POST "/auth/register" "$(jq -n --arg email "$MERCHANT_EMAIL" '{name:"Demo Merchant", email:$email, password:"secret123", role:"MERCHANT"}')"
assert_status 409 "duplicate register email -> 409"

# missing password -> 401 (usecase raises AuthError, not a validation error; see user_usecase.go)
request POST "/auth/register" "$(jq -n --arg email "nobody-${RUN_ID}@example.com" '{name:"No Pass", email:$email, role:"MERCHANT"}')"
assert_status 401 "register missing password -> 401"

# bogus role -> 401
request POST "/auth/register" "$(jq -n --arg email "bogus-${RUN_ID}@example.com" '{name:"Bogus", email:$email, password:"secret123", role:"BOGUS"}')"
assert_status 401 "register bogus role -> 401"

# login
request POST "/auth/login" "$(jq -n --arg email "$MERCHANT_EMAIL" '{email:$email, password:"secret123"}')"
assert_status 200 "login merchant"

# login wrong password -> 401
request POST "/auth/login" "$(jq -n --arg email "$MERCHANT_EMAIL" '{email:$email, password:"wrongpass"}')"
assert_status 401 "login wrong password -> 401"

# refresh
request POST "/auth/refresh" "" "$MERCHANT_TOKEN"
assert_status 200 "refresh merchant token"
MERCHANT_TOKEN="$(extract '.token')"

# refresh with no token -> 401
request POST "/auth/refresh" ""
assert_status 401 "refresh without token -> 401"

# ---------------------------------------------------------------------------
# 2. Invoice flow
# ---------------------------------------------------------------------------
section "2. Invoice CRUD"

request POST "/invoices" "$(jq -n --arg mid "$MERCHANT_ID" '{merchantId:$mid, amount:10000, currency:"USD"}')" "$MERCHANT_TOKEN"
assert_status 201 "create invoice"
INVOICE_ID="$(extract '.id')"
PAYMENT_TOKEN="$(extract '.paymentToken')"
assert_field_equals '.status' "PENDING" "invoice starts PENDING"

# amount 0 -> 400
request POST "/invoices" "$(jq -n --arg mid "$MERCHANT_ID" '{merchantId:$mid, amount:0, currency:"USD"}')" "$MERCHANT_TOKEN"
assert_status 400 "create invoice amount=0 -> 400"

# missing merchantId -> 201 (CreateInvoiceGin defaults merchantId to the JWT subject
# when the caller's role is MERCHANT and the field is omitted; see user_handler.go CreateInvoiceGin)
request POST "/invoices" '{"amount":10000,"currency":"USD"}' "$MERCHANT_TOKEN"
assert_status 201 "create invoice omitting merchantId -> defaults to JWT subject -> 201"
assert_field_equals '.merchantId' "$MERCHANT_ID" "defaulted merchantId matches JWT subject"

request GET "/invoices?page=1&pageSize=10" "" "$MERCHANT_TOKEN"
assert_status 200 "list invoices"

request GET "/invoices/$INVOICE_ID" "" "$MERCHANT_TOKEN"
assert_status 200 "get invoice detail"

request GET "/invoices/00000000-0000-0000-0000-000000000000" "" "$MERCHANT_TOKEN"
assert_status 404 "get unknown invoice -> 404"

request PUT "/invoices/$INVOICE_ID" '{"amount":12000,"currency":"USD"}' "$MERCHANT_TOKEN"
assert_status 200 "update invoice"
# NOTE: UpdateInvoiceGin serializes the raw domain Invoice struct (no json tags),
# so field names are PascalCase here (.Amount, .Status) -- unlike POST /invoices,
# which returns a camelCase map (.amount, .status). This asymmetry is a real API
# inconsistency worth flagging, not a test bug.
assert_field_equals '.Amount' "12000" "invoice amount updated (PascalCase field, see note above)"

# throwaway invoice for delete flow
request POST "/invoices" "$(jq -n --arg mid "$MERCHANT_ID" '{merchantId:$mid, amount:500, currency:"USD"}')" "$MERCHANT_TOKEN"
assert_status 201 "create throwaway invoice"
THROWAWAY_ID="$(extract '.id')"

request DELETE "/invoices/$THROWAWAY_ID" "" "$MERCHANT_TOKEN"
assert_status 204 "delete throwaway invoice"

# ---------------------------------------------------------------------------
# 3. Public pay link + payment intent
# ---------------------------------------------------------------------------
section "3. Public payment link + intent"

request GET "/pay/$PAYMENT_TOKEN"
assert_status 200 "resolve payment link (public)"

request GET "/pay/bogus-token-does-not-exist"
assert_status 404 "resolve unknown payment token -> 404"

request POST "/pay/$PAYMENT_TOKEN/intents" '{"method":"WALLET"}'
assert_status 201 "create payment intent"
INTENT_ID="$(extract '.ID')"
assert_field_equals '.Status' "PENDING" "intent starts PENDING"

request POST "/pay/$PAYMENT_TOKEN/intents" '{"method":"BOGUS"}'
assert_status 400 "create intent bogus method -> 400"

request POST "/pay/$PAYMENT_TOKEN/intents" ''
assert_status 400 "create intent empty body -> 400"

# ---------------------------------------------------------------------------
# 4. Admin simulate payment
# ---------------------------------------------------------------------------
section "4. Admin simulate payment intent"

request POST "/admin/payment-intents/$INTENT_ID/simulate" '{"outcome":"SUCCESS"}' "$ADMIN_TOKEN"
assert_status 200 "admin simulate SUCCESS"
assert_field_equals '.Status' "SUCCESS" "intent flips to SUCCESS"

request POST "/admin/payment-intents/$INTENT_ID/simulate" '{"outcome":"BOGUS"}' "$ADMIN_TOKEN"
assert_status 400 "simulate bogus outcome -> 400"

request POST "/admin/payment-intents/$INTENT_ID/simulate" '{"outcome":"SUCCESS"}' "$MERCHANT_TOKEN"
assert_status 403 "simulate with merchant token -> 403"

request GET "/invoices/$INVOICE_ID" "" "$MERCHANT_TOKEN"
assert_status 200 "re-fetch invoice after simulate"
# GetInvoiceGin also serializes the raw domain struct -> PascalCase (.Status).
assert_field_equals '.Status' "PAID" "invoice becomes PAID (PascalCase field)"

# ---------------------------------------------------------------------------
# 5. Refund flow
# ---------------------------------------------------------------------------
section "5. Refund request -> approve -> process"

request POST "/merchant/refunds" "$(jq -n --arg iid "$INVOICE_ID" --arg mid "$MERCHANT_ID" '{invoiceId:$iid, merchantId:$mid, amount:2000}')" "$MERCHANT_TOKEN"
assert_status 201 "request refund"
REFUND_ID="$(extract '.id')"
assert_field_equals '.status' "REQUESTED" "refund starts REQUESTED"

request GET "/merchant/refunds?merchantId=$MERCHANT_ID" "" "$MERCHANT_TOKEN"
assert_status 200 "list refund history"

request POST "/admin/refunds/$REFUND_ID/approve" "" "$ADMIN_TOKEN"
assert_status 200 "admin approve refund"
assert_field_equals '.status' "APPROVED" "refund becomes APPROVED"

request POST "/admin/refunds/$REFUND_ID/process" '{"success":true}' "$ADMIN_TOKEN"
assert_status 200 "admin process refund"
assert_field_equals '.status' "SUCCESS" "refund becomes SUCCESS"

# re-process an already-SUCCESS refund -> 200 (Refund.MarkSuccess treats SUCCESS->SUCCESS
# as an idempotent no-op by design, see internal/domain/refund/refund.go MarkSuccess).
# Contrast with REJECTED below, which IS a hard terminal state that errors on transition attempts.
request POST "/admin/refunds/$REFUND_ID/process" '{"success":true}' "$ADMIN_TOKEN"
assert_status 200 "re-process already-SUCCESS refund is idempotent -> 200"

# refund reject path on a second refund request
request POST "/merchant/refunds" "$(jq -n --arg iid "$INVOICE_ID" --arg mid "$MERCHANT_ID" '{invoiceId:$iid, merchantId:$mid, amount:1000}')" "$MERCHANT_TOKEN"
assert_status 201 "request second refund"
REFUND_ID_2="$(extract '.id')"

request POST "/admin/refunds/$REFUND_ID_2/reject" "" "$ADMIN_TOKEN"
assert_status 200 "admin reject second refund"
assert_field_equals '.status' "REJECTED" "second refund becomes REJECTED"

request POST "/admin/refunds/$REFUND_ID_2/process" '{"success":true}' "$ADMIN_TOKEN"
assert_status 400 "processing REJECTED refund -> 400"

# ---------------------------------------------------------------------------
# 6. Top-up flow
# ---------------------------------------------------------------------------
section "6. Top-up request -> admin approve"

request POST "/merchant/topups" "$(jq -n --arg mid "$MERCHANT_ID" --arg rk "topup-${RUN_ID}" '{merchantId:$mid, amount:5000, requestKey:$rk}')" "$MERCHANT_TOKEN"
assert_status 201 "request top-up"
TOPUP_ID="$(extract '.id')"
assert_field_equals '.status' "PENDING" "top-up starts PENDING"

# idempotency: same merchantId + requestKey returns same top-up
request POST "/merchant/topups" "$(jq -n --arg mid "$MERCHANT_ID" --arg rk "topup-${RUN_ID}" '{merchantId:$mid, amount:5000, requestKey:$rk}')" "$MERCHANT_TOKEN"
assert_status 201 "repeat top-up with same requestKey (idempotent)"
assert_field_equals '.id' "$TOPUP_ID" "idempotent top-up returns same id"

request POST "/merchant/topups" "$(jq -n --arg mid "$MERCHANT_ID" '{merchantId:$mid, amount:0}')" "$MERCHANT_TOKEN"
assert_status 400 "top-up amount=0 -> 400"

request GET "/merchant/topups?merchantId=$MERCHANT_ID" "" "$MERCHANT_TOKEN"
assert_status 200 "list top-up history"

request POST "/admin/topups/$TOPUP_ID/status" '{"success":true}' "$ADMIN_TOKEN"
assert_status 200 "admin approve top-up"
assert_field_equals '.status' "SUCCESS" "top-up becomes SUCCESS"

# ---------------------------------------------------------------------------
# 7. Role-check endpoints
# ---------------------------------------------------------------------------
section "7. Role-check endpoints"

request GET "/merchant/profile" "" "$MERCHANT_TOKEN"
assert_status 200 "merchant profile as merchant"

request GET "/merchant/profile" "" "$ADMIN_TOKEN"
assert_status 403 "merchant profile as admin -> 403"

request GET "/admin/dashboard" "" "$ADMIN_TOKEN"
assert_status 200 "admin dashboard as admin"

request GET "/admin/dashboard" "" "$MERCHANT_TOKEN"
assert_status 403 "admin dashboard as merchant -> 403"

# ---------------------------------------------------------------------------
# 8. Dashboard stats
# ---------------------------------------------------------------------------
section "8. Admin dashboard stats"

request GET "/admin/dashboard/stats?merchantId=$MERCHANT_ID" "" "$ADMIN_TOKEN"
assert_status 200 "dashboard stats for merchant"

request GET "/admin/dashboard/stats?startDate=2026-01-01T00:00:00Z&endDate=2026-12-31T23:59:59Z" "" "$ADMIN_TOKEN"
assert_status 200 "dashboard stats with date range"

request GET "/admin/dashboard/stats?startDate=not-a-date" "" "$ADMIN_TOKEN"
assert_status 400 "dashboard stats malformed date -> 400"

# ---------------------------------------------------------------------------
finalize
[ "$FAIL_COUNT" -eq 0 ]
exit $?
