/* Transactional Outbox — vì sao không được "ghi DB xong rồi publish Kafka".
   Bật/tắt outbox rồi crash ở đúng khe hở để thấy sự kiện bốc hơi. */
VF.scene('transactional-outbox', {
  title: 'Transactional Outbox',
  controls: [
    { kind: 'toggle', id: 'naive', label: '⚡ Bỏ outbox (ghi DB rồi publish thẳng)', value: false },
    { kind: 'toggle', id: 'crash', label: '⚡ Service chết giữa chừng', value: true }
  ],

  build: function (o) {
    var L = [
      { name: 'Order', sub: 'service' },
      { name: 'order_db', sub: 'Postgres' },
      { name: 'Dispatcher', sub: 'worker 2s' },
      { name: 'Kafka', sub: 'order.events' }
    ];
    var S = 0, DB = 1, W = 2, K = 3;
    var steps = [], rows = [];
    function push(note, extra) {
      steps.push({ view: VF.lanes(L, rows.slice(), rows.length - 1) + (extra || ''), note: note });
    }

    if (o.naive) {
      rows.push({ from: S, to: DB, label: 'INSERT order · COMMIT', cls: 'ok' });
      push('Cách ngây thơ: ghi đơn, commit. Đơn <b>đã có thật</b> trong DB.',
        outboxState([['ĐƠN', 'đã commit', 'ok']], [], false));

      if (o.crash) {
        rows.push({ from: S, to: S, label: '💥 process chết', cls: 'fail' });
        steps.push({
          view: VF.lanes(L, rows, rows.length - 1) + outboxState([['ĐƠN', 'đã commit', 'ok']], [], false),
          note: '<span class="hit">Khe hở chết người.</span> Đơn <b>đã nằm trong DB</b> nhưng sự kiện <code>order.placed</code> <b>chưa bao giờ được phát</b>. Notification không báo khách, Reporting không thấy doanh thu, Delivery không biết có đơn cần giao. <b>Hệ thống lệch nhau vĩnh viễn</b> — và không ai biết, vì API đã trả <code>201</code> cho khách rồi.'
        });
      } else {
        rows.push({ from: S, to: K, label: 'publish order.placed', cls: 'ok' });
        push('Không chết thì vẫn chạy đúng. <b>Vấn đề là bạn không điều khiển được lúc nào process chết.</b> Bật nút “service chết” để thấy.',
          outboxState([['ĐƠN', 'đã commit', 'ok']], [], true));
      }
      return steps;
    }

    rows.push({ from: S, to: DB, label: 'INSERT order + INSERT outbox — MỘT tx', cls: 'ok' });
    push('Cách đúng: đơn và <b>bản ghi sự kiện</b> được ghi trong <b>cùng một transaction</b>. Không có Kafka nào ở đây cả — chỉ là hai lệnh INSERT vào <b>cùng một database</b>.',
      outboxState([['ĐƠN', 'đã commit', 'ok']], [['order.placed', 'PENDING']], false));

    if (o.crash) {
      rows.push({ from: S, to: S, label: '💥 process chết', cls: 'fail' });
      push('Process chết đúng chỗ hiểm. Nhưng lần này <b>sự kiện đã nằm an toàn trong DB</b>, cùng chỗ với đơn hàng.',
        outboxState([['ĐƠN', 'đã commit', 'ok']], [['order.placed', 'PENDING']], false));
      rows.push({ from: W, to: DB, label: 'quét bảng outbox (mỗi 2s)', cls: 'wait' });
      push('Service sống lại. Dispatcher quét bảng <code>outbox</code>, thấy dòng <code>PENDING</code> còn đó.',
        outboxState([['ĐƠN', 'đã commit', 'ok']], [['order.placed', 'PENDING']], false));
    } else {
      rows.push({ from: W, to: DB, label: 'quét bảng outbox (mỗi 2s)', cls: 'wait' });
      push('Dispatcher chạy nền, quét bảng <code>outbox</code>.',
        outboxState([['ĐƠN', 'đã commit', 'ok']], [['order.placed', 'PENDING']], false));
    }

    rows.push({ from: W, to: K, label: 'publish order.placed', cls: 'ok' });
    rows.push({ from: W, to: DB, label: 'đánh dấu SENT', cls: 'ok' });
    steps.push({
      view: VF.lanes(L, rows, rows.length - 1) + outboxState([['ĐƠN', 'đã commit', 'ok']], [['order.placed', 'SENT']], true),
      note: '<b>Sự kiện vẫn được phát.</b> Câu để nói: <i>“Sự kiện được ghi <b>cùng transaction</b> với business write, nên không bao giờ có chuyện đơn tạo rồi mà sự kiện mất. Đổi lại, dispatcher có thể phát <b>trùng</b> (crash sau publish, trước khi đánh dấu SENT) — nên phía nhận <b>bắt buộc</b> phải idempotent.”</i> Đó chính là <b>at-least-once</b>, và là lý do phần của Nam tồn tại.'
    });
    return steps;
  }
});

function outboxState(orders, events, published) {
  var h = '<div style="display:grid;grid-template-columns:1fr 1fr;gap:24px;margin-top:24px">';
  h += '<table class="grid"><thead><tr><th>order_db — bảng orders</th><th>trạng thái</th></tr></thead><tbody>';
  orders.forEach(function (r) { h += '<tr><td class="mono">' + r[0] + '</td><td><span class="chip ok">' + r[1] + '</span></td></tr>'; });
  h += '</tbody></table>';
  h += '<table class="grid"><thead><tr><th>order_db — bảng outbox</th><th>trạng thái</th></tr></thead><tbody>';
  if (!events.length) {
    h += '<tr><td colspan="2" style="color:var(--stamp)">— không tồn tại —</td></tr>';
  } else {
    events.forEach(function (r) {
      h += '<tr><td class="mono">' + r[0] + '</td><td><span class="chip ' +
           (r[1] === 'SENT' ? 'ok' : 'wait') + '">' + r[1] + '</span></td></tr>';
    });
  }
  h += '</tbody></table></div>';
  h += '<table class="grid" style="margin-top:12px"><thead><tr><th>Kafka — topic order.events</th></tr></thead><tbody><tr><td>' +
       (published
         ? '<span class="chip ok">order.placed</span> — consumer nhận được'
         : '<span class="chip dead">— trống —</span>') +
       '</td></tr></tbody></table>';
  return h;
}
