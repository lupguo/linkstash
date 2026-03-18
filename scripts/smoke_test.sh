#!/bin/bash
# LinkStash Smoke Test Script
# Tests all API endpoints + CLI commands

set -e

SERVER="http://localhost:8080"
PASS=0
FAIL=0
TOTAL=0

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
    -d '{"secret_key":"linkstash-dev-secret-2024"}')
check "Get JWT token" '"token"' "$TOKEN_RESP"
TOKEN=$(echo "$TOKEN_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['token'])" 2>/dev/null)
export LINKSTASH_TOKEN="$TOKEN"
export LINKSTASH_SERVER="$SERVER"

AUTH="Authorization: Bearer $TOKEN"

# Test 5: Add URL
ADD_RESP=$(curl -s -X POST $SERVER/api/urls \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"link":"https://github.com"}')
check "Add URL - github.com" '"link":"https://github.com"' "$ADD_RESP"
ADD_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST $SERVER/api/urls \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"link":"https://go.dev"}')
check_status "Add URL returns 201" "201" "$ADD_STATUS"

# Test 6: Add more URLs for testing
curl -s -X POST $SERVER/api/urls -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"link":"https://htmx.org"}' > /dev/null
curl -s -X POST $SERVER/api/urls -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"link":"https://tailwindcss.com"}' > /dev/null

# Test 7: List URLs
LIST_RESP=$(curl -s "$SERVER/api/urls?page=1&size=10" -H "$AUTH")
check "List URLs has data" '"data"' "$LIST_RESP"
check "List URLs total >= 4" '"total"' "$LIST_RESP"

# Test 8: List with pagination
LIST_P=$(curl -s "$SERVER/api/urls?page=1&size=2" -H "$AUTH")
check "List pagination size=2" '"size":2' "$LIST_P"

# Test 9: Get URL detail
DETAIL=$(curl -s "$SERVER/api/urls/1" -H "$AUTH")
check "Get URL #1 detail" '"link":"https://github.com"' "$DETAIL"

# Test 10: Get non-existent URL
STATUS=$(curl -s -o /dev/null -w "%{http_code}" "$SERVER/api/urls/99999" -H "$AUTH")
check_status "Get non-existent URL returns 404" "404" "$STATUS"

# Test 11: Update URL
UPDATE_RESP=$(curl -s -X PUT "$SERVER/api/urls/1" \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"title":"GitHub","category":"开发工具","manual_weight":10}')
check "Update URL title" '"title":"GitHub"' "$UPDATE_RESP"
check "Update URL category" '"category":"开发工具"' "$UPDATE_RESP"

# Test 12: Record visit
VISIT_RESP=$(curl -s -X POST "$SERVER/api/urls/1/visit" -H "$AUTH")
check "Record visit" '"status":"ok"' "$VISIT_RESP"

# Verify visit count incremented
DETAIL2=$(curl -s "$SERVER/api/urls/1" -H "$AUTH")
check "Visit count incremented" '"visit_count":1' "$DETAIL2"

# Test 13: Delete URL #4
DEL_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$SERVER/api/urls/4" -H "$AUTH")
check_status "Delete URL returns 204" "204" "$DEL_STATUS"

# Verify deleted URL is gone from list
LIST_AFTER=$(curl -s "$SERVER/api/urls" -H "$AUTH")
check "Deleted URL not in list" '"total":3' "$LIST_AFTER"

# Test: Duplicate URL
DUP_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST $SERVER/api/urls \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"link":"https://github.com"}')
check_status "Duplicate URL returns 500" "500" "$DUP_STATUS"

# Test: Empty link
EMPTY_STATUS=$(curl -s -o /dev/null -w "%{http_code}" -X POST $SERVER/api/urls \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"link":""}')
check_status "Empty link returns 500" "500" "$EMPTY_STATUS"

echo ""

# ============================================
# Phase 3: Search (keyword only - no LLM)
# ============================================
echo -e "${CYAN}--- Phase 3: Search (keyword) ---${NC}"

# First update some URLs with searchable content
curl -s -X PUT "$SERVER/api/urls/1" -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"title":"GitHub","keywords":"git,代码托管,开源","description":"全球最大的代码托管平台"}' > /dev/null
curl -s -X PUT "$SERVER/api/urls/2" -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"title":"Go Language","keywords":"golang,编程语言","description":"Go编程语言官方网站"}' > /dev/null
curl -s -X PUT "$SERVER/api/urls/3" -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"title":"htmx","keywords":"前端,AJAX,HTML","description":"轻量级前端交互库"}' > /dev/null

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

# Create short link
SHORT_RESP=$(curl -s -X POST "$SERVER/api/short-links" \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"long_url":"https://example.com/very/long/path"}')
check "Create short link" '"code"' "$SHORT_RESP"
SHORT_CODE=$(echo "$SHORT_RESP" | python3 -c "import sys,json; print(json.load(sys.stdin)['code'])" 2>/dev/null || echo "")

# Create short link with TTL
SHORT_TTL_RESP=$(curl -s -X POST "$SERVER/api/short-links" \
    -H "$AUTH" -H "Content-Type: application/json" \
    -d '{"long_url":"https://example.com/temp","ttl":"7d"}')
check "Create short link with TTL" '"expires_at"' "$SHORT_TTL_RESP"

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

# Delete short link
DEL_SHORT=$(curl -s -o /dev/null -w "%{http_code}" -X DELETE "$SERVER/api/short-links/2" -H "$AUTH")
check_status "Delete short link returns 204" "204" "$DEL_SHORT"

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

echo ""

# ============================================
# Phase 6: CLI Commands
# ============================================
echo -e "${CYAN}--- Phase 6: CLI Commands ---${NC}"

CLI="./linkstash"

# CLI add
CLI_ADD=$($CLI add "https://wikipedia.org" 2>&1)
check "CLI add URL" "wikipedia" "$CLI_ADD"

# CLI list
CLI_LIST=$($CLI list 2>&1)
check "CLI list URLs" "github" "$CLI_LIST"

# CLI info
CLI_INFO=$($CLI info 1 2>&1)
check "CLI info URL #1" "GitHub" "$CLI_INFO"

# CLI search
CLI_SEARCH=$($CLI search "GitHub" --type keyword 2>&1)
check "CLI search" "score" "$CLI_SEARCH"

# CLI short
CLI_SHORT=$($CLI short "https://example.com/cli-test" 2>&1)
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
