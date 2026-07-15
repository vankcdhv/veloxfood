/* Circuit breaker — máy trạng thái + số đo THẬT của Demo 2.
   Kịch bản cố định vì nó tái hiện đúng buổi chạy thật: dừng store-service,
   bắn liên tiếp, mạch mở ở request thứ 5, độ trễ 5.055ms → 25ms. */
VF.scene('circuit-breaker', {
  title: 'Circuit breaker — sony/gobreaker',
  controls: [],

  build: function () {
    // [nhãn, độ trễ ms, HTTP, trạng thái sau bước, ghi chú]
    var seq = [
      ['store-service còn sống', null, null, 'closed',
       'Mạch <b>đóng</b>. Mọi lời gọi <code>GetStoreForOrder</code> đi thẳng tới store-service. Bấm tiếp để dừng store-service.'],
      ['⚡ docker compose stop store-service', null, null, 'closed',
       '<span class="hit">store-service chết.</span> Mạch <b>vẫn đóng</b> — breaker chưa biết gì. Nó chỉ học từ <b>lỗi thật</b>.'],
      ['request #1', 5055, 400, 'closed',
       '<b>Mất đúng 5,055 giây.</b> Đây không phải TCP treo vô hạn — <b>deadline 5s</b> của <code>pkg/grpcx</code> đã cắt. Fail-fast, chứ không chờ đến chết.'],
      ['request #2', 3432, 400, 'closed',
       'Vẫn đi qua mạch. Breaker đang đếm: <b>2 request, 2 lỗi hạ tầng</b>.'],
      ['request #3', 361, 400, 'closed', 'Đếm tiếp: 3/3 lỗi.'],
      ['request #4', 358, 400, 'closed', 'Đếm tiếp: 4/4 lỗi. <b>Vẫn chưa mở</b> — ngưỡng là <b>≥5 request trong 30 giây</b>.'],
      ['request #5', 360, 400, 'open',
       '<span class="hit">Mạch MỞ.</span> Đủ hai điều kiện: <b>≥5 request/30s</b> <b>VÀ</b> <b>≥60% lỗi hạ tầng</b>. Gauge <code>circuit_breaker_state</code> nhảy lên <b>2</b>.'],
      ['request #6', 25, 400, 'open',
       '<b>25 mili-giây.</b> Request bị <b>từ chối ngay tại client</b>, không hề chạm tới store-service. Độ trễ giảm <b>200 lần</b> so với #1.'],
      ['request #7', 24, 400, 'open',
       'Đây mới là giá trị thật của breaker: nó <b>không cứu</b> được request này (vẫn hỏng), nhưng nó <b>ngừng đập vào một service đang chết</b> và <b>trả lời thất bại tức thì</b> thay vì bắt khách chờ 5 giây.'],
      ['▶ docker compose start store-service', 24, 400, 'open',
       'store-service sống lại — nhưng mạch <b>vẫn mở</b>. Breaker <b>không biết</b> điều đó, và <b>cố ý không thử</b> cho tới khi hết <code>Timeout 15s</code>.'],
      ['… qua mốc 15 giây …', null, null, 'half',
       'Mạch chuyển <b>half-open</b> (gauge = <b>1</b>). Nó cho <b>đúng 3 request thăm dò</b> đi qua. Nếu bất kỳ cái nào hỏng ⇒ <b>mở lại ngay</b>.'],
      ['probe #1', 210, 201, 'half', 'Thăm dò 1: <code>201</code> — đơn tạo <b>thật</b>.'],
      ['probe #2', 190, 201, 'half', 'Thăm dò 2: <code>201</code>.'],
      ['probe #3', 185, 201, 'closed',
       '<b>Mạch đóng lại</b> (gauge = <b>0</b>). Hệ thống <b>tự phục hồi</b>, không ai phải restart gì. Đó là toàn bộ vòng đời: <code>closed → open → half-open → closed</code>.']
    ];

    var rows = [];
    return seq.map(function (s, i) {
      if (s[1] !== null) rows.push([s[0], s[1], s[2], s[3]]);
      return {
        view: breakerMachine(s[3]) + breakerLog(rows, s[3]),
        note: s[4]
      };
    });
  }
});

/* Máy trạng thái vẽ bằng SVG: ba nút tròn + các cung chuyển trạng thái có nhãn
   điều kiện. Nhìn thấy CUNG mới hiểu vì sao nó chuyển, chứ ba cái hộp cạnh nhau
   thì chỉ là ba cái hộp. */
function breakerMachine(state) {
  // H phải chứa cả cung vòng dưới (đáy của nó ở cy + r + 52 = 200) — thiếu là bị cắt
  var W = 880, H = 216, cy = 108, r = 40;
  var nodes = [
    { id: 'closed', x: 130, label: 'ĐÓNG', sub: 'gauge 0', color: 'var(--ok)' },
    { id: 'open', x: 440, label: 'MỞ', sub: 'gauge 2', color: 'var(--stamp)' },
    { id: 'half', x: 750, label: 'HALF-OPEN', sub: 'gauge 1', color: 'var(--wait)' }
  ];
  var s = '<svg class="fsm" viewBox="0 0 ' + W + ' ' + H + '" preserveAspectRatio="xMidYMid meet" role="img">';

  s += fsmEdge(130 + r, 440 - r, cy, '≥5 req/30s VÀ ≥60% lỗi hạ tầng', 'up', 'var(--stamp)');
  s += fsmEdge(440 + r, 750 - r, cy, 'sau Timeout 15s', 'up', 'var(--wait)');
  s += fsmEdge(750 - r, 440 + r, cy, '1 probe hỏng → mở lại', 'down', 'var(--stamp)');
  // Cung dài half-open → closed, vòng dưới cả sơ đồ
  s += '<path class="fsm-edge" style="color:var(--ok)" fill="none" d="M750 ' + (cy + r) +
       ' q0 52 -70 52 H200 q-70 0 -70 -52"/>' +
       '<path class="fsm-tip" style="color:var(--ok)" d="M130 ' + (cy + r) + ' l5 9 l-10 0 Z"/>' +
       '<text class="fsm-edge-label" style="color:var(--ok)" x="440" y="' + (cy + 68) + '">' +
       '3 probe thành công → đóng lại</text>';

  nodes.forEach(function (n) {
    var on = n.id === state;
    s += '<g class="fsm-node' + (on ? ' on' : '') + '" style="color:' + n.color + '">' +
         '<circle cx="' + n.x + '" cy="' + cy + '" r="' + r + '"/>' +
         '<text class="fsm-node-label" x="' + n.x + '" y="' + (cy - 2) + '">' + n.label + '</text>' +
         '<text class="fsm-node-sub" x="' + n.x + '" y="' + (cy + 14) + '">' + n.sub + '</text>' +
         '</g>';
  });
  return s + '</svg>';
}

function fsmEdge(x1, x2, cy, label, bend, color) {
  var lift = bend === 'up' ? -46 : 46;
  var my = cy + lift;
  var dir = x2 > x1 ? 1 : -1;
  var tip = x2 - dir * 2;
  return '<path class="fsm-edge" style="color:' + color + '" fill="none" ' +
         'd="M' + x1 + ' ' + cy + ' Q' + ((x1 + x2) / 2) + ' ' + my + ' ' + tip + ' ' + cy + '"/>' +
         '<path class="fsm-tip" style="color:' + color + '" d="M' + x2 + ' ' + cy +
         ' l' + (-dir * 9) + ' ' + (lift > 0 ? 3 : -3) + ' l' + (-dir * 2) + ' 8 Z"/>' +
         '<text class="fsm-edge-label" style="color:' + color + '" x="' + ((x1 + x2) / 2) + '" y="' +
         (cy + lift * 0.75) + '">' + VF.esc(label) + '</text>';
}

function breakerLog(rows, state) {
  if (!rows.length) return '';
  var h = '<table class="grid" style="margin-top:24px"><thead><tr>' +
          '<th>Lời gọi</th><th style="text-align:right">Độ trễ</th><th>HTTP</th><th>Mạch</th>' +
          '</tr></thead><tbody>';
  rows.forEach(function (r, i) {
    var last = i === rows.length - 1;
    var slow = r[1] >= 3000, fast = r[1] <= 30;
    h += '<tr' + (last ? ' class="flash"' : '') + '><td class="mono">' + VF.esc(r[0]) + '</td>' +
         '<td class="num" style="' + (slow ? 'color:var(--stamp);font-weight:700' : fast ? 'color:var(--ok);font-weight:700' : '') + '">' +
         r[1].toLocaleString('vi-VN') + ' ms</td>' +
         '<td><span class="chip ' + (r[2] === 201 ? 'ok' : 'fail') + '">' + r[2] + '</span></td>' +
         '<td class="mono">' + r[3] + '</td></tr>';
  });
  h += '</tbody></table>';
  h += '<p style="font-family:var(--font-mono);font-size:12px;color:var(--ink-faint);margin-top:12px">' +
       'Chỉ lỗi <b style="color:var(--stamp)">hạ tầng</b> (Unavailable, DeadlineExceeded) mới làm mạch trip. ' +
       'Lỗi <b>nghiệp vụ</b> (NotFound, InvalidArgument) thì KHÔNG — service vẫn khoẻ, nó chỉ đang từ chối đúng.</p>';
  return h;
}
