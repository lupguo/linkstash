#!/bin/bash
# LinkStash PopClip Extension - Save selected URL to LinkStash

LINKSTASH_SERVER="${LINKSTASH_SERVER:-}"
LINKSTASH_TOKEN="${LINKSTASH_TOKEN:-}"

if [ -z "$LINKSTASH_SERVER" ] || [ -z "$LINKSTASH_TOKEN" ]; then
    echo "Error: LINKSTASH_SERVER and LINKSTASH_TOKEN must be set" >&2
    exit 1
fi

curl -s -X POST "$LINKSTASH_SERVER/api/urls" \
    -H "Authorization: Bearer $LINKSTASH_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"link\":\"$POPCLIP_TEXT\"}"

exit 0
