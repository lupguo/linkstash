#!/bin/bash
# LinkStash PopClip Extension - Save selected URL to LinkStash

LINKSTASH_SERVER="${POPCLIP_OPTION_SERVER:-}"
LINKSTASH_TOKEN="${POPCLIP_OPTION_TOKEN:-}"

if [ -z "$LINKSTASH_SERVER" ] || [ -z "$LINKSTASH_TOKEN" ]; then
    echo "Error: Please configure Server URL and API Token in PopClip extension settings" >&2
    exit 1
fi

curl -s -X POST "$LINKSTASH_SERVER/api/urls" \
    -H "Authorization: Bearer $LINKSTASH_TOKEN" \
    -H "Content-Type: application/json" \
    -d "{\"link\":\"$POPCLIP_TEXT\"}"

exit 0
