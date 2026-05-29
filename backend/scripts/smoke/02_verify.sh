#!/usr/bin/env bash
# Smoke: verify OTP for registration. Requires OTP_CODE env var.
set -euo pipefail
BASE_URL=${BASE_URL:-http://localhost:8080}
OTP_CODE=${OTP_CODE:-123456}
EMAIL=${EMAIL:-smoketest@example.com}

echo "=== Verify Register OTP ==="
curl -s -X POST "$BASE_URL/api/v1/auth/verify-register" \
  -H "Content-Type: application/json" \
  -d "{\"destination\":\"$EMAIL\",\"code\":\"$OTP_CODE\"}" \
  | python3 -m json.tool
