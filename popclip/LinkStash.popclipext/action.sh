#!/bin/bash
# LinkStash PopClip Extension - Save selected URL to LinkStash
# Automatically exchanges secret_key for JWT, then saves the URL.

LINKSTASH_SERVER="${POPCLIP_OPTION_SERVER:-}"
LINKSTASH_SECRET_KEY="${POPCLIP_OPTION_SECRET_KEY:-}"

if [ -z "$LINKSTASH_SERVER" ] || [ -z "$LINKSTASH_SECRET_KEY" ]; then
    echo "✗ Please configure Server URL and Secret Key"
    exit 1
fi

# Step 1: Exchange secret_key for JWT token
AUTH_RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$LINKSTASH_SERVER/api/auth/token" \
    -H "Content-Type: application/json" \
    -d "{\"secret_key\":\"$LINKSTASH_SECRET_KEY\"}")

AUTH_CODE=$(echo "$AUTH_RESPONSE" | tail -1)
AUTH_BODY=$(echo "$AUTH_RESPONSE" | sed '$d')

if [ "$AUTH_CODE" -ne 200 ]; then
    echo "✗ Auth failed ($AUTH_CODE)"
    exit 1
fi

JWT_TOKEN=$(echo "$AUTH_BODY" | grep -o '"token":"[^"]*"' | head -1 | cut -d'"' -f4)

if [ -z "$JWT_TOKEN" ]; then
    echo "✗ Failed to parse token"
    exit 1
fi

# Step 2: Save URL with JWT token
RESPONSE=$(curl -s -w "\n%{http_code}" -X POST "$LINKSTASH_SERVER/api/urls" \
    -H "Authorization: Bearer $JWT_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"link\":\"$POPCLIP_TEXT\"}")

HTTP_CODE=$(echo "$RESPONSE" | tail -1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" -ge 200 ] && [ "$HTTP_CODE" -lt 300 ]; then
    echo "✓ Saved"
    exit 0
else
    # Extract error message from JSON response: {"error":{"message":"xxx"}}
    MSG=$(echo "$BODY" | grep -o '"message":"[^"]*"' | head -1 | cut -d'"' -f4)
    if [ -z "$MSG" ]; then
        MSG="HTTP $HTTP_CODE"
    fi
    echo "✗ $MSG"
    exit 1
fi
