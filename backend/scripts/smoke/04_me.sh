#!/usr/bin/env bash
# Smoke: GET /me with Bearer token.
set -euo pipefail
BASE_URL=${BASE_URL:-http://localhost:8080}
ACCESS_TOKEN=${ACCESS_TOKEN:?Need ACCESS_TOKEN env var. Run 03_login.sh first.}

echo "=== GET /me ==="
curl -s -X GET "$BASE_URL/api/v1/me" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  | python3 -m json.tool
