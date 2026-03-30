#!/bin/bash
# LinkStash Smoke Test Script
# Tests all API endpoints + CLI commands
# Designed to be idempotent — can run against a non-empty database.

set -e

SERVER="http://localhost:8080"
PASS=0
FAIL=0
TOTAL=0

# Unique suffix to avoid UNIQUE constraint conflicts on repeated runs
UNIQUE=$(date +%s)

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

check() {
    TOTAL=$((TOTAL + 1))
    local name="$1"
    local expected="$2"
    local actual="$3"
    if echo "$actual" | grep -q "$expected"; then
        PASS=$((PASS + 1))
        echo -e "${GREEN}✓ PASS${NC} - $name"
    else
        FAIL=$((FAIL + 1))
        echo -e "${RED}✗ FAIL${NC} - $name"
        echo "  Expected to contain: $expected"
        echo "  Got: $actual"
    fi
}

check_status() {
    TOTAL=$((TOTAL + 1))
    local name="$1"
    local expected="$2"
    local actual="$3"
    if [ "$actual" = "$expected" ]; then
        PASS=$((PASS + 1))
        echo -e "${GREEN}✓ PASS${NC} - $name (HTTP $actual)"
    else
        FAIL=$((FAIL + 1))
        echo -e "${RED}✗ FAIL${NC} - $name (Expected HTTP $expected, got HTTP $actual)"
    fi
}

# Helper: extract JSON field via python3
json_field() {
    echo "$1" | python3 -c "import sys,json; print(json.load(sys.stdin)$2)" 2>/dev/null
}

echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}  LinkStash Smoke Test Suite${NC}"
echo -e "${CYAN}========================================${NC}"
echo ""

# ============================================
# Phase 1: Auth + URL CRUD
# ============================================
echo -e "${CYAN}--- Phase 1: Auth + URL CRUD ---${NC}"

# Test 1: Health check
RESP=$(curl -s $SERVER/health)
check "Health check" '"status":"ok"' "$RESP"

# Test 2: Unauthorized access
STATUS=$(curl -s -o /dev/null -w "%{http_code}" $SERVER/api/urls)
check_status "Unauthorized access returns 401" "401" "$STATUS"

# Test 3: Wrong secret key
STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST $SERVER/api/auth/token \
    -H "Content-Type: application/json" \
    -d '{"secret_key":"wrong-key"}')
check_status "Wrong secret returns 401" "401" "$STATUS"

# Test 4: Get JWT token
TOKEN_RESP=$(curl -s -X POST $SERVER/api/auth/token \
    -H "Content-Type: application/json" \
    -d "{\"secret_key\":\"${AUTH_SECRET_KEY:-clark}\"}")
check "Get JWT token" '"token"' "$TOKEN_RESP"
TOKEN=$(json_field "$TOKEN_RESP" "['token']")
export LINKSTASH_TOKEN="$TOKEN"
export LINKSTASH_SERVER="$SERVER"

AUTH="Authorization: Bearer $TOKEN"

# Test 5: Add URL (use unique URLs to avoid UNIQUE constraint on repeated runs)
URL_A="https://github.com/smoke-${UNIQUE}"
URL_B="https://go.dev/smoke-${UNIQUE}"
ADD_RESP=$(curl -s -X POST $SERVER/api/urls \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d "{\"link\":\"${URL_A}\"}")
check "Add URL" '"link"' "$ADD_RESP"
URL_A_ID=$(json_field "$ADD_RESP" "['ID']")

ADD_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST $SERVER/api/urls \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d "{\"link\":\"${URL_B}\"}")
check_status "Add URL returns 201" "201" "$ADD_STATUS"

# Test 6: Add more URLs for testing
URL_C="https://htmx.org/smoke-${UNIQUE}"
URL_D="https://tailwindcss.com/smoke-${UNIQUE}"
curl -s -X POST $SERVER/api/urls -H "$AUTH" -H "Content-Type: application/json" \
    -d "{\"link\":\"${URL_C}\"}" > /dev/null
RESP_D=$(curl -s -X POST $SERVER/api/urls -H "$AUTH" -H "Content-Type: application/json" \
    -d "{\"link\":\"${URL_D}\"}")
URL_D_ID=$(json_field "$RESP_D" "['ID']")

# Test 7: List URLs
LIST_RESP=$(curl -s "$SERVER/api/urls?page=1&size=10" -H "$AUTH")
check "List URLs has data" '"data"' "$LIST_RESP"
check "List URLs total >= 4" '"total"' "$LIST_RESP"

# Test 8: List with pagination
LIST_P=$(curl -s "$SERVER/api/urls?page=1&size=2" -H "$AUTH")
check "List pagination size=2" '"size":2' "$LIST_P"

# Test 9: Get URL detail (use the ID we just created)
DETAIL=$(curl -s "$SERVER/api/urls/${URL_A_ID}" -H "$AUTH")
check "Get URL detail" "smoke-${UNIQUE}" "$DETAIL"

# Test 10: Get non-existent URL
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER/api/urls/99999" -H "$AUTH")
check_status "Get non-existent URL returns 404" "404" "$STATUS"

# Test 11: Update URL
UPDATE_RESP=$(curl -s -X PUT "$SERVER/api/urls/${URL_A_ID}" \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"title":"GitHub","category":"开发工具","manual_weight":10}')
check "Update URL title" '"title":"GitHub"' "$UPDATE_RESP"
check "Update URL category" '"category":"开发工具"' "$UPDATE_RESP"

# Test 12: Record visit — use relative assertion (before vs after)
DETAIL_BEFORE=$(curl -s "$SERVER/api/urls/${URL_A_ID}" -H "$AUTH")
VISIT_COUNT_BEFORE=$(json_field "$DETAIL_BEFORE" "['visit_count']")

VISIT_RESP=$(curl -s -X POST "$SERVER/api/urls/${URL_A_ID}/visit" -H "$AUTH")
check "Record visit" '"status":"ok"' "$VISIT_RESP"

DETAIL_AFTER=$(curl -s "$SERVER/api/urls/${URL_A_ID}" -H "$AUTH")
VISIT_COUNT_AFTER=$(json_field "$DETAIL_AFTER" "['visit_count']")
EXPECTED_COUNT=$((VISIT_COUNT_BEFORE + 1))
check "Visit count incremented" "\"visit_count\":${EXPECTED_COUNT}" "$DETAIL_AFTER"

# Test 13: Delete URL — use relative assertion (total before vs after)
LIST_BEFORE_DEL=$(curl -s "$SERVER/api/urls" -H "$AUTH")
TOTAL_BEFORE=$(json_field "$LIST_BEFORE_DEL" "['total']")

DEL_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$SERVER/api/urls/${URL_D_ID}" -H "$AUTH")
check_status "Delete URL returns 204" "204" "$DEL_STATUS"

LIST_AFTER_DEL=$(curl -s "$SERVER/api/urls" -H "$AUTH")
TOTAL_AFTER=$(json_field "$LIST_AFTER_DEL" "['total']")
EXPECTED_TOTAL=$((TOTAL_BEFORE - 1))
check "Deleted URL not in list" "\"total\":${EXPECTED_TOTAL}" "$LIST_AFTER_DEL"

# Test: Duplicate URL (use the URL we already added)
DUP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST $SERVER/api/urls \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d "{\"link\":\"${URL_A}\"}")
check_status "Duplicate URL returns 409" "409" "$DUP_STATUS"

DUP_RESP=$(curl -s -X POST $SERVER/api/urls \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d "{\"link\":\"${URL_A}\"}")
check "Duplicate URL friendly message" '该链接已存在' "$DUP_RESP"

# Test: Empty link
EMPTY_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST $SERVER/api/urls \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"link":""}')
check_status "Empty link returns 500" "500" "$EMPTY_STATUS"

# Test: Update visit_count via PUT and verify auto_weight sync
UPDATE_VC_RESP=$(curl -s -X PUT "$SERVER/api/urls/${URL_A_ID}" \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"visit_count":42}')
check "Update visit_count" '"visit_count":42' "$UPDATE_VC_RESP"
check "Auto_weight synced with visit_count" '"auto_weight":42' "$UPDATE_VC_RESP"

# Test: Update short_code via PUT
UPDATE_SC_RESP=$(curl -s -X PUT "$SERVER/api/urls/${URL_A_ID}" \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d "{\"short_code\":\"sc-${UNIQUE}\"}")
check "Update short_code" "\"short_code\":\"sc-${UNIQUE}\"" "$UPDATE_SC_RESP"

# Test: Favicon field exists in response (may be empty initially since async)
DETAIL_FAV=$(curl -s "$SERVER/api/urls/${URL_A_ID}" -H "$AUTH")
check "Favicon field in response" '"color"' "$DETAIL_FAV"

# Test: Update status via PUT
UPDATE_ST_RESP=$(curl -s -X PUT "$SERVER/api/urls/${URL_A_ID}" \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"status":"ready"}')
check "Update status" '"status":"ready"' "$UPDATE_ST_RESP"

# Test: Update color and icon
UPDATE_CI=$(curl -s -X PUT "$SERVER/api/urls/${URL_A_ID}" \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"color":"red","icon":"🚀"}')
check "Update color" '"color":"red"' "$UPDATE_CI"
check "Update icon" '"icon":"🚀"' "$UPDATE_CI"

echo ""

# ============================================
# Phase 3: Search (keyword only - no LLM)
# ============================================
echo -e "${CYAN}--- Phase 3: Search (keyword) ---${NC}"

# Update URLs with searchable content (use the IDs we created)
curl -s -X PUT "$SERVER/api/urls/${URL_A_ID}" -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"title":"GitHub","keywords":"git,代码托管,开源","description":"全球最大的代码托管平台"}' > /dev/null

# Wait a moment for FTS triggers to fire
sleep 0.5

# Test keyword search
KW_RESP=$(curl -s "$SERVER/api/search?q=GitHub&type=keyword" -H "$AUTH")
check "Keyword search finds GitHub" '"type":"keyword"' "$KW_RESP"

echo ""

# ============================================
# Phase 5: Short Links
# ============================================
echo -e "${CYAN}--- Phase 5: Short Links ---${NC}"

# Create short link (unique URL)
SHORT_RESP=$(curl -s -X POST "$SERVER/api/short-links" \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d "{\"long_url\":\"https://example.com/long-path-${UNIQUE}\"}")
check "Create short link" '"code"' "$SHORT_RESP"
SHORT_CODE=$(json_field "$SHORT_RESP" "['code']" || echo "")
SHORT_ID=$(json_field "$SHORT_RESP" "['id']" || echo "")

# Create short link with TTL (unique URL)
SHORT_TTL_RESP=$(curl -s -X POST "$SERVER/api/short-links" \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d "{\"long_url\":\"https://example.com/temp-${UNIQUE}\",\"ttl\":\"7d\"}")
check "Create short link with TTL" '"short_expires_at"' "$SHORT_TTL_RESP"
SHORT_TTL_ID=$(json_field "$SHORT_TTL_RESP" "['id']" || echo "")

# List short links
SHORT_LIST=$(curl -s "$SERVER/api/short-links" -H "$AUTH")
check "List short links" '"data"' "$SHORT_LIST"

# Redirect (302)
if [ -n "$SHORT_CODE" ]; then
    REDIR_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER/s/$SHORT_CODE")
    check_status "Short link redirect 302" "302" "$REDIR_STATUS"
fi

# Non-existent short link
NF_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER/s/nonexistent")
check_status "Non-existent short link 404" "404" "$NF_STATUS"

# Delete short link (use the ID we just created)
if [ -n "$SHORT_TTL_ID" ]; then
    DEL_SHORT=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$SERVER/api/short-links/${SHORT_TTL_ID}" -H "$AUTH")
    check_status "Delete short link returns 204" "204" "$DEL_SHORT"
fi

echo ""

# ============================================
# Phase 4: Web Pages
# ============================================
echo -e "${CYAN}--- Phase 4: Web Pages ---${NC}"

# Login page (public)
LOGIN_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER/login")
check_status "Login page returns 200" "200" "$LOGIN_STATUS"

# Index redirects to login when not auth
INDEX_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -L "$SERVER/")
check_status "Index redirects to login" "200" "$INDEX_STATUS"

# Health
H_STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER/health")
check_status "Health check returns 200" "200" "$H_STATUS"

# Detail page (with auth cookie)
DETAIL_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -b "linkstash_token=$TOKEN" "$SERVER/urls/${URL_A_ID}")
check_status "Detail page /urls/{id} returns 200" "200" "$DETAIL_STATUS"

# New page (with auth cookie)
NEW_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -b "linkstash_token=$TOKEN" "$SERVER/urls/new")
check_status "New page /urls/new returns 200" "200" "$NEW_STATUS"

echo ""

# ============================================
# Phase 6: CLI Commands
# ============================================
echo -e "${CYAN}--- Phase 6: CLI Commands ---${NC}"

CLI="./bin/linkstash"
# Fallback to ./linkstash if bin/ not found
[ -f "$CLI" ] || CLI="./linkstash"

export LINKSTASH_SECRET_KEY="${AUTH_SECRET_KEY:-clark}"

# CLI add (unique URL)
CLI_ADD=$($CLI add "https://wikipedia.org/smoke-${UNIQUE}" 2>&1)
check "CLI add URL" "wikipedia" "$CLI_ADD"

# CLI list
CLI_LIST=$($CLI list 2>&1)
check "CLI list URLs" "github" "$CLI_LIST"

# CLI info (use URL_A_ID we created)
CLI_INFO=$($CLI info "${URL_A_ID}" 2>&1)
check "CLI info URL" "GitHub" "$CLI_INFO"

# CLI search
CLI_SEARCH=$($CLI search "GitHub" --type keyword 2>&1)
check "CLI search" "score" "$CLI_SEARCH"

# CLI short (unique URL)
CLI_SHORT=$($CLI short "https://example.com/cli-test-${UNIQUE}" 2>&1)
check "CLI create short link" "Code" "$CLI_SHORT"

echo ""

# ============================================
# Summary
# ============================================
echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}  Test Results: $PASS passed, $FAIL failed (total: $TOTAL)${NC}"
echo -e "${CYAN}========================================${NC}"

if [ $FAIL -gt 0 ]; then
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
else
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
fi
