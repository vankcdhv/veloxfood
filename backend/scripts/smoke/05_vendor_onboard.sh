#!/usr/bin/env bash
# Smoke: vendor onboard (submit vendor registration request).
set -euo pipefail
BASE_URL=${BASE_URL:-http://localhost:8080}
ACCESS_TOKEN=${ACCESS_TOKEN:?Need ACCESS_TOKEN env var. Run 03_login.sh first.}

echo "=== Vendor Onboard ==="
curl -s -X POST "$BASE_URL/api/v1/vendors/onboard" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "vendor_name": "Smoke Canteen",
    "business_type": "canteen",
    "address": "123 Smoke St",
    "phone": "0901234567"
  }' \
  | python3 -m json.tool
