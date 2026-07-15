/* Saga đặt hàng — 3 nhánh, bù trừ chạy NGƯỢC.
   Số liệu khớp Demo 1 đã chạy thật trên DTM UI. */
VF.scene('saga-compensation', {
  title: 'Saga đặt hàng — bù trừ ngược',
  controls: [{
    kind: 'select', id: 'failAt', label: 'Chèn lỗi ở', value: 'capture',
    options: [
      ['none', 'không — happy path'],
      ['apply', 'ApplyPromotion (nhánh 1)'],
      ['capture', 'Capture (nhánh 2)'],
      ['confirm', 'ConfirmUsage (nhánh 3)']
    ]
  }],

  build: function (o) {
    var L = [
      { name: 'Order', sub: 'điều phối' },
      { name: 'DTM', sub: 'saga engine' },
      { name: 'Promotion', sub: 'nhánh 1 & 3' },
      { name: 'Payment', sub: 'nhánh 2' }
    ];
    var A = 2, P = 3, D = 1, O = 0;
    var steps = [], rows = [];

    function push(note) {
      steps.push({ view: VF.lanes(L, rows.slice(), rows.length - 1), note: note });
    }

    rows.push({ from: O, to: D, label: 'SubmitSaga(gid)', cls: 'ok' });
    push('Order <b>không tự gọi</b> Promotion hay Payment. Nó nộp saga cho DTM rồi chờ. ' +
         'DTM mới là bên gọi từng nhánh — đây là <b>orchestration</b>.');

    var failApply = o.failAt === 'apply';
    rows.push({ from: D, to: A, label: 'ApplyPromotion' + (failApply ? ' ✖' : ' ✔'), cls: failApply ? 'fail' : 'ok' });
    push(failApply
      ? 'Nhánh 1 <span class="hit">hỏng ngay</span> (voucher hết lượt). Chưa nhánh nào chạy trước nó ⇒ chuỗi bù trừ rất ngắn.'
      : 'Nhánh 1 xong: voucher chuyển <code>RESERVED</code>. <b>Chưa trừ quota thật</b> — mới chỉ giữ chỗ.');

    if (!failApply) {
      var failCap = o.failAt === 'capture';
      rows.push({ from: D, to: P, label: 'Capture' + (failCap ? ' ✖' : ' ✔'), cls: failCap ? 'fail' : 'ok' });
      push(failCap
        ? 'Nhánh 2 <span class="hit">hỏng</span> — ví không đủ số dư. Ở đây <b>chưa đồng nào bị trừ</b>. Nhớ kỹ chi tiết này.'
        : 'Nhánh 2 xong: sổ cái ghi 2 bút toán, tiền rời ví khách.');

      if (!failCap) {
        var failCf = o.failAt === 'confirm';
        rows.push({ from: D, to: A, label: 'ConfirmUsage' + (failCf ? ' ✖' : ' ✔'), cls: failCf ? 'fail' : 'ok' });
        push(failCf
          ? 'Nhánh 3 <span class="hit">hỏng</span> ở bước cuối — <b>tiền đã bị trừ rồi</b>. Đây là ca bù trừ đắt nhất.'
          : 'Nhánh 3 xong: voucher <code>RESERVED → CONFIRMED</code>, <code>used_count</code> tăng thật.');
      }
    }

    if (o.failAt === 'none') {
      steps.push({
        view: VF.lanes(L, rows, rows.length - 1) + sagaBranchTable('none'),
        note: '<b>Cả 3 nhánh succeed.</b> Order ghi đơn + outbox trong <b>một transaction</b>, phát <code>order.placed</code>. Saga <span class="chip ok">succeed</span>.'
      });
      return steps;
    }

    // Bù trừ chạy NGƯỢC thứ tự các nhánh đã chạy
    if (o.failAt === 'confirm') {
      rows.push({ from: D, to: A, label: '↩ ReleaseUsage (nhánh 3)', cls: 'comp' });
      push('Bù trừ bắt đầu, và nó chạy <b>ngược</b>. Nhánh 3 tự bù trừ chính nó trước.');
    }
    if (o.failAt === 'confirm' || o.failAt === 'capture') {
      var isNull = o.failAt === 'capture';
      rows.push({ from: D, to: P, label: '↩ Refund (nhánh 2)' + (isNull ? ' — no-op' : ''), cls: 'comp' });
      push(isNull
        ? '<span class="hit">Đây là câu quan trọng nhất của cả demo.</span> <code>Refund</code> trả về <b>succeed</b> dù <b>chưa hề trừ đồng nào</b> — vì <code>Capture</code> đã hỏng. Bù trừ cho một việc chưa từng xảy ra gọi là <b>null compensation</b>, và nó <b>bắt buộc phải là no-op idempotent</b>, không được báo lỗi.'
        : 'Tiền đã trừ thật ⇒ <code>Refund</code> hoàn lại thật: hai bút toán ngược chiều, ví về nguyên trạng.');
    }
    rows.push({ from: D, to: A, label: '↩ ReleaseUsage (nhánh 1)', cls: 'comp' });
    push('Nhả chỗ giữ voucher. Bù trừ đã đi ngược hết chuỗi ⇒ hệ thống về đúng trạng thái trước khi đặt.');

    steps.push({
      view: VF.lanes(L, rows, rows.length - 1) + sagaBranchTable(o.failAt),
      note: '<b>Saga <span class="chip fail">failed</span> — nhưng không rò rỉ gì.</b> Ví không đổi · <code>used_count</code> không tăng · không đẻ ra đơn · sổ cái không có bút toán mới. Khách nhận <code>400</code>.'
    });
    return steps;
  }
});

/* Bảng nhánh, dựng giống hệt màn DTM UI thầy sẽ nhìn thấy lúc demo */
function sagaBranchTable(failAt) {
  var rows;
  if (failAt === 'none') {
    rows = [
      ['01', 'action', 'succeed', 'ApplyPromotion'],
      ['02', 'action', 'succeed', 'Capture'],
      ['03', 'action', 'succeed', 'ConfirmUsage']
    ];
  } else if (failAt === 'apply') {
    rows = [
      ['01', 'action', 'failed', 'ApplyPromotion'],
      ['01', 'compensate', 'succeed', 'ReleaseUsage'],
      ['02', 'action', 'prepared', 'Capture'],
      ['03', 'action', 'prepared', 'ConfirmUsage']
    ];
  } else if (failAt === 'capture') {
    rows = [
      ['01', 'action', 'succeed', 'ApplyPromotion'],
      ['01', 'compensate', 'succeed', 'ReleaseUsage'],
      ['02', 'action', 'failed', 'Capture'],
      ['02', 'compensate', 'succeed', 'Refund'],
      ['03', 'action', 'prepared', 'ConfirmUsage']
    ];
  } else {
    rows = [
      ['01', 'action', 'succeed', 'ApplyPromotion'],
      ['01', 'compensate', 'succeed', 'ReleaseUsage'],
      ['02', 'action', 'succeed', 'Capture'],
      ['02', 'compensate', 'succeed', 'Refund'],
      ['03', 'action', 'failed', 'ConfirmUsage'],
      ['03', 'compensate', 'succeed', 'ReleaseUsage']
    ];
  }
  var cls = { succeed: 'ok', failed: 'fail', prepared: 'wait' };
  var h = '<p style="font-family:var(--font-mono);font-size:10px;letter-spacing:.12em;' +
          'text-transform:uppercase;color:var(--ink-faint);margin:24px 0 8px">' +
          'Màn DTM UI sẽ hiện đúng thế này</p>';
  h += '<table class="grid"><thead><tr><th>branch</th><th>op</th><th>status</th><th>RPC</th></tr></thead><tbody>';
  rows.forEach(function (r) {
    h += '<tr><td class="num">' + r[0] + '</td><td>' + r[1] +
         '</td><td><span class="chip ' + cls[r[2]] + '">' + r[2] + '</span></td>' +
         '<td><code>' + r[3] + '</code></td></tr>';
  });
  return h + '</tbody></table>';
}
