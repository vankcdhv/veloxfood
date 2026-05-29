#!/usr/bin/env bash
# Smoke: logout (revoke access + refresh tokens).
set -euo pipefail
BASE_URL=${BASE_URL:-http://localhost:8080}
ACCESS_TOKEN=${ACCESS_TOKEN:?Need ACCESS_TOKEN env var. Run 03_login.sh first.}
REFRESH_TOKEN=${REFRESH_TOKEN:?Need REFRESH_TOKEN env var from login response.}

echo "=== Logout ==="
curl -s -X POST "$BASE_URL/api/v1/auth/logout" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"access_token\":\"$ACCESS_TOKEN\",\"refresh_token\":\"$REFRESH_TOKEN\"}" \
  | python3 -m json.tool
