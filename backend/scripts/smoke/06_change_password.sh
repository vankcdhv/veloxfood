#!/usr/bin/env bash
# Smoke: change password for authenticated user.
set -euo pipefail
BASE_URL=${BASE_URL:-http://localhost:8080}
ACCESS_TOKEN=${ACCESS_TOKEN:?Need ACCESS_TOKEN env var. Run 03_login.sh first.}
OLD_PASSWORD=${OLD_PASSWORD:-Smoke1234!}
NEW_PASSWORD=${NEW_PASSWORD:-Smoke5678!}

echo "=== Change Password ==="
curl -s -X POST "$BASE_URL/api/v1/auth/change-password" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d "{\"old_password\":\"$OLD_PASSWORD\",\"new_password\":\"$NEW_PASSWORD\"}" \
  | python3 -m json.tool
