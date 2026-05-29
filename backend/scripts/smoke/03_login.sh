#!/usr/bin/env bash
# Smoke: login and print access + refresh tokens.
set -euo pipefail
BASE_URL=${BASE_URL:-http://localhost:8080}
EMAIL=${EMAIL:-smoketest@example.com}
PASSWORD=${PASSWORD:-Smoke1234!}

echo "=== Login ==="
RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{\"identifier\":\"$EMAIL\",\"password\":\"$PASSWORD\"}")
echo "$RESPONSE" | python3 -m json.tool
ACCESS_TOKEN=$(echo "$RESPONSE" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('data',d).get('access_token',''))" 2>/dev/null || echo "")
echo ""
echo "ACCESS_TOKEN=$ACCESS_TOKEN"
echo ""
echo "Export and run 04_me.sh:"
echo "  export ACCESS_TOKEN='$ACCESS_TOKEN'"
