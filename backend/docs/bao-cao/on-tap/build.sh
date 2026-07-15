#!/usr/bin/env bash
# Gộp toàn bộ nguồn thành MỘT file tự chứa.
#
# Vì sao cần: Artifact trên claude.ai chặn mọi request ra file ngoài (CSP), nên
# link chia sẻ bắt buộc phải là một file duy nhất. Bản gộp cũng mở được offline
# bằng cách double-click, tiện lúc mang đi bảo vệ mà không có mạng.
#
# Sửa nội dung thì sửa trong on-tap/, rồi chạy lại script này.
set -euo pipefail
cd "$(dirname "$0")"

OUT="../on-tap-bao-ve.html"

SCENES=(
  saga-compensation two-pc-vs-saga sub-transaction-barrier transactional-outbox
  circuit-breaker kafka-retry-dlq trace-waterfall fail-closed-auth
  idempotent-receiver cqrs-read-model single-flight-refresh double-entry-ledger
)
CONTENT=(tong-quan van duc nam hoi-cheo)

{
  echo '<title>VeloxFood — Ôn tập bảo vệ đồ án hệ phân tán</title>'
  echo '<style>'
  cat css/tokens.css css/app.css
  echo '</style>'

  # Lấy phần thân trang từ index.html: từ <header> tới </main>
  sed -n '/<header class="masthead">/,/<\/main>/p' index.html

  echo '<script>'
  cat js/scene-engine.js
  for s in "${SCENES[@]}";  do cat "js/scenes/$s.js";   done
  for c in "${CONTENT[@]}"; do cat "js/content/$c.js";  done
  cat js/app.js
  echo '</script>'
} > "$OUT"

echo "→ $(cd .. && pwd)/on-tap-bao-ve.html  ($(wc -c < "$OUT" | tr -d ' ') bytes)"
