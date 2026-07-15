/* Vì sao KHÔNG dùng 2PC. Câu này thầy gần như chắc chắn hỏi.
   Điểm chốt: 2PC giữ KHOÁ xuyên service trong lúc chờ; saga không giữ khoá nào. */
VF.scene('two-pc-vs-saga', {
  title: '2PC và Saga khi mạng đứt',
  controls: [
    {
      kind: 'select', id: 'engine', label: 'Giao thức', value: '2pc',
      options: [['2pc', 'Two-Phase Commit'], ['saga', 'Saga (ta dùng cái này)']]
    },
    { kind: 'toggle', id: 'cut', label: '⚡ Cắt mạng tới Payment', value: true }
  ],

  build: function (o) {
    var L = [
      { name: 'Điều phối', sub: o.engine === '2pc' ? 'coordinator' : 'DTM' },
      { name: 'Promotion', sub: 'voucher' },
      { name: 'Payment', sub: 'ví tiền' }
    ];
    var C = 0, A = 1, P = 2;
    var steps = [], rows = [];
    function push(note, extra) {
      steps.push({ view: VF.lanes(L, rows.slice(), rows.length - 1) + (extra || ''), note: note });
    }

    if (o.engine === '2pc') {
      rows.push({ from: C, to: A, label: 'PREPARE', cls: 'wait' });
      push('Pha 1 — hỏi Promotion: “sẵn sàng commit chưa?”. Promotion trả lời <b>yes</b> và <b>KHOÁ dòng voucher lại</b>.',
        twoPcLockRow(['🔒 voucher khoá', '—']));

      rows.push({ from: C, to: P, label: 'PREPARE', cls: o.cut ? 'fail' : 'wait' });
      if (o.cut) {
        push('Pha 1 tới Payment — <span class="hit">mạng đứt</span>. Coordinator <b>không biết</b> Payment đã prepare hay chưa. Nó chỉ biết: không có trả lời.',
          twoPcLockRow(['🔒 voucher khoá', '❓ không rõ']));
        rows.push({ from: C, to: C, label: '… chờ …', cls: 'dead' });
        push('<b>Coordinator bị kẹt.</b> Không dám commit (lỡ Payment không prepare được), không dám abort (lỡ Payment đã prepare và đang giữ khoá).',
          twoPcLockRow(['🔒 VẪN khoá', '❓ có thể cũng đang khoá']));
        steps.push({
          view: VF.lanes(L, rows, rows.length - 1) + twoPcLockRow(['🔒 khoá vô thời hạn', '❓ vô thời hạn']),
          note: '<b>Đây chính là lý do loại 2PC.</b> Voucher bị khoá suốt thời gian mạng chưa lành — <b>mọi khách khác dùng voucher đó đều bị chặn</b>. 2PC là giao thức <b>blocking</b>: nó đánh đổi tính sẵn sàng để lấy tính nhất quán tức thời. Với hệ mạng thật (partition là chuyện thường), cái giá đó quá đắt.'
        });
      } else {
        push('Payment cũng trả lời <b>yes</b>, và cũng khoá ví lại.', twoPcLockRow(['🔒 voucher khoá', '🔒 ví khoá']));
        rows.push({ from: C, to: A, label: 'COMMIT', cls: 'ok' });
        rows.push({ from: C, to: P, label: 'COMMIT', cls: 'ok' });
        push('Pha 2 — commit cả hai, nhả khoá. <b>Chạy đúng thì 2PC cho nhất quán mạnh.</b> Vấn đề chỉ lộ ra khi mạng đứt — bật nút cắt mạng để thấy.',
          twoPcLockRow(['✔ xong', '✔ xong']));
      }
      return steps;
    }

    // Saga
    rows.push({ from: C, to: A, label: 'ApplyPromotion ✔', cls: 'ok' });
    push('Saga <b>không có pha prepare</b>. Nhánh 1 <b>commit ngay</b> và <b>nhả khoá ngay</b> — voucher chuyển <code>RESERVED</code>, giao dịch DB đóng lại trong vài mili-giây.',
      twoPcLockRow(['✔ đã commit, KHÔNG khoá', '—']));

    rows.push({ from: C, to: P, label: 'Capture' + (o.cut ? ' ✖ Unavailable' : ' ✔'), cls: o.cut ? 'fail' : 'ok' });
    if (o.cut) {
      push('<span class="hit">Mạng đứt</span>. gRPC trả <code>Unavailable</code> sau <b>deadline 5 giây</b> — <b>không</b> treo vô hạn. Saga biết ngay là hỏng.',
        twoPcLockRow(['✔ RESERVED, không khoá', '✖ không tới được']));
      rows.push({ from: C, to: A, label: '↩ ReleaseUsage ✔', cls: 'comp' });
      steps.push({
        view: VF.lanes(L, rows, rows.length - 1) + twoPcLockRow(['✔ đã nhả chỗ giữ', '— không đụng tới']),
        note: '<b>Bù trừ, rồi kết thúc.</b> Không ai bị khoá một giây nào. Đổi lại: có một khoảng ngắn (vài trăm ms) voucher ở trạng thái <code>RESERVED</code> dù đơn sẽ hỏng — đó là <b>nhất quán cuối</b>. Ta <b>chấp nhận</b> cái giá đó để đổi lấy tính sẵn sàng.'
      });
    } else {
      push('Nhánh 2 xong, cũng commit ngay và nhả khoá ngay.', twoPcLockRow(['✔ RESERVED', '✔ đã trừ tiền']));
      rows.push({ from: C, to: A, label: 'ConfirmUsage ✔', cls: 'ok' });
      push('Nhánh 3 xong. Saga <span class="chip ok">succeed</span>. Bật nút cắt mạng để thấy khác biệt thật sự.',
        twoPcLockRow(['✔ CONFIRMED', '✔ xong']));
    }
    return steps;
  }
});

function twoPcLockRow(cells) {
  var h = '<table class="grid" style="margin-top:24px"><thead><tr>' +
          '<th>Promotion — trạng thái khoá</th><th>Payment — trạng thái khoá</th>' +
          '</tr></thead><tbody><tr>';
  cells.forEach(function (c) { h += '<td class="mono">' + VF.esc(c) + '</td>'; });
  return h + '</tr></tbody></table>';
}
