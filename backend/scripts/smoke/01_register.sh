#!/usr/bin/env bash
# Smoke: register a new user and send OTP.
set -euo pipefail
BASE_URL=${BASE_URL:-http://localhost:8080}

echo "=== Register ==="
curl -s -X POST "$BASE_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"smoketest@example.com","password":"Smoke1234!","full_name":"Smoke User"}' \
  | python3 -m json.tool
echo ""
echo "Check your mailer (mailtrap or mock) for OTP, then run 02_verify.sh"
