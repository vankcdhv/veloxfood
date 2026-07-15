/* Sub-transaction barrier — cơ chế khó nhất trong cả báo cáo.
   Nó giải HAI bài toán khác nhau, nên có 2 kịch bản riêng. Tắt barrier để thấy
   chính xác cái gì hỏng. */
VF.scene('sub-transaction-barrier', {
  title: 'Sub-transaction barrier',
  controls: [
    {
      kind: 'select', id: 'case', label: 'Kịch bản', value: 'dup',
      options: [
        ['dup', 'Bài toán 1 — gọi trùng nhánh'],
        ['nul', 'Bài toán 2 — bù trừ đến trước']
      ]
    },
    { kind: 'toggle', id: 'off', label: '⚡ TẮT barrier', value: false }
  ],

  build: function (o) {
    var L = [
      { name: 'DTM', sub: 'saga engine' },
      { name: 'Payment', sub: 'nhánh 2' },
      { name: 'payment_db', sub: 'barrier + ví' }
    ];
    var D = 0, P = 1, B = 2;
    var steps = [], rows = [];
    function push(note, extra) {
      steps.push({ view: VF.lanes(L, rows.slice(), rows.length - 1) + (extra || ''), note: note });
    }

    if (o.case === 'dup') {
      rows.push({ from: D, to: P, label: 'Capture (lần 1)', cls: 'ok' });
      push('DTM gọi <code>Capture</code>. Bình thường.', barrierState([], 500000));

      rows.push({ from: P, to: B, label: 'ghi barrier + trừ ví — MỘT tx', cls: 'ok' });
      push(o.off
        ? 'Không có barrier ⇒ chỉ trừ ví. Ví <code>500.000 → 423.000</code>.'
        : 'Chỗ mấu chốt: dòng barrier <code>(gid, 02, action)</code> và việc trừ ví nằm trong <b>CÙNG một transaction</b>. Cả hai cùng sống hoặc cùng chết.',
        barrierState(o.off ? [] : [['02', 'action', 'đã ghi']], 423000));

      rows.push({ from: P, to: D, label: '✔ succeed — nhưng gói tin MẤT', cls: 'fail' });
      push('Payment trả lời succeed, nhưng <span class="hit">gói tin trả lời rơi giữa đường</span>. DTM không nhận được ⇒ nó tưởng nhánh này chưa chạy.',
        barrierState(o.off ? [] : [['02', 'action', 'đã ghi']], 423000));

      rows.push({ from: D, to: P, label: 'Capture (lần 2 — retry)', cls: 'wait' });
      push('DTM <b>retry</b>. Đây không phải bug — <b>mọi</b> hệ phân tán đều phải retry, vì không thể phân biệt “service chết” với “trả lời bị mất”.',
        barrierState(o.off ? [] : [['02', 'action', 'đã ghi']], 423000));

      if (o.off) {
        rows.push({ from: P, to: B, label: 'trừ ví LẦN NỮA', cls: 'fail' });
        steps.push({
          view: VF.lanes(L, rows, rows.length - 1) + barrierState([], 346000),
          note: '<b>Khách bị trừ tiền hai lần.</b> Ví <code>500.000 → 423.000 → 346.000</code> cho <b>một</b> đơn. Đây là <b>duplicate branch call</b> — và nó xảy ra ở mọi hệ thống retry mà không khử trùng lặp.'
        });
      } else {
        rows.push({ from: P, to: B, label: 'INSERT barrier → trùng khoá', cls: 'wait' });
        push('Payment thử ghi lại dòng barrier <code>(gid, 02, action)</code>. Khoá chính trùng ⇒ <b>INSERT thất bại</b>. Payment hiểu ngay: “việc này tôi làm rồi”.',
          barrierState([['02', 'action', 'đã có — CHẶN']], 423000));
        rows.push({ from: P, to: D, label: '✔ succeed (bỏ qua, không trừ lại)', cls: 'ok' });
        steps.push({
          view: VF.lanes(L, rows, rows.length - 1) + barrierState([['02', 'action', 'đã ghi']], 423000),
          note: '<b>Ví vẫn 423.000.</b> Barrier biến <code>Capture</code> thành <b>idempotent</b>: gọi bao nhiêu lần cũng chỉ trừ một lần. Và nó làm được thế <b>không cần</b> Payment tự nghĩ ra cơ chế khử trùng lặp riêng.'
        });
      }
      return steps;
    }

    // Bài toán 2 — null compensation / bù trừ vượt trước
    rows.push({ from: D, to: P, label: 'Capture', cls: 'wait' });
    push('DTM gọi <code>Capture</code>, nhưng <span class="hit">gói tin đi rất chậm</span> (nghẽn mạng). Payment chưa nhận được.',
      barrierState([], 500000));

    rows.push({ from: D, to: P, label: '↩ Refund (bù trừ)', cls: 'comp' });
    push('DTM hết kiên nhẫn (deadline), coi nhánh 2 là hỏng, và gọi <b>bù trừ</b>. <code>Refund</code> <b>đến trước</b> <code>Capture</code>.',
      barrierState([], 500000));

    if (o.off) {
      rows.push({ from: P, to: B, label: 'Refund: không có gì để hoàn → LỖI', cls: 'fail' });
      push('Không có barrier ⇒ Payment tra bảng, không thấy payment nào, và <b>báo lỗi</b>. DTM thấy bù trừ hỏng ⇒ <b>retry mãi mãi</b>.',
        barrierState([], 500000));
      rows.push({ from: D, to: P, label: 'Capture (gói tin chậm, giờ mới tới)', cls: 'fail' });
      steps.push({
        view: VF.lanes(L, rows, rows.length - 1) + barrierState([], 423000),
        note: '<b>Thảm hoạ.</b> <code>Capture</code> lết tới sau khi bù trừ đã chạy ⇒ <b>trừ tiền của một đơn đã bị huỷ</b>, và <b>không bao giờ có ai hoàn lại</b>. Tiền mất vĩnh viễn. Đây là <b>dangling action</b> — kèm theo <b>null compensation</b> (bù trừ cho việc chưa xảy ra) làm saga treo.'
      });
    } else {
      rows.push({ from: P, to: B, label: 'ghi barrier (gid, 02, compensate)', cls: 'comp' });
      push('Có barrier: <code>Refund</code> vẫn <b>ghi dòng chặn</b> rồi trả <b>succeed</b> mà <b>không hoàn đồng nào</b> — vì đúng là chưa trừ đồng nào. Đây chính là <b>null compensation</b>, và nó phải là <b>no-op</b>.',
        barrierState([['02', 'compensate', 'đã ghi']], 500000));
      rows.push({ from: D, to: P, label: 'Capture (gói tin chậm, giờ mới tới)', cls: 'wait' });
      push('Gói tin <code>Capture</code> lết tới. Payment ghi barrier <code>(gid, 02, action)</code> và <b>nhìn thấy dòng compensate đã có trước</b>.',
        barrierState([['02', 'compensate', 'đã ghi'], ['02', 'action', 'đến SAU compensate']], 500000));
      rows.push({ from: P, to: D, label: '✔ succeed (bỏ qua — KHÔNG trừ tiền)', cls: 'ok' });
      steps.push({
        view: VF.lanes(L, rows, rows.length - 1) + barrierState([['02', 'compensate', 'đã ghi'], ['02', 'action', 'BỊ CHẶN']], 500000),
        note: '<b>Ví vẫn nguyên 500.000.</b> Barrier chặn được cả action đến muộn. Câu để nói: <i>“Barrier ghi một dòng chặn trong cùng transaction với business write. Nó giải hai bài toán: <b>gọi trùng nhánh</b> và <b>null compensation</b> — và cả trường hợp action lết tới sau khi đã bù trừ.”</i>'
      });
    }
    return steps;
  }
});

function barrierState(bars, wallet) {
  // 500.000 = nguyên vẹn · 423.000 = trừ đúng một lần · 346.000 = trừ hai lần (bug)
  var tone = wallet === 346000 ? 'bad' : (wallet === 500000 ? 'ok' : 'mute');
  var hint = wallet === 346000 ? 'ĐÃ TRỪ HAI LẦN cho một đơn'
           : wallet === 423000 ? 'trừ đúng một lần'
           : 'chưa trừ đồng nào';

  var h = VF.stat([{ label: 'Ví khách', value: wallet.toLocaleString('vi-VN') + ' đ', tone: tone, hint: hint }]);

  h += '<table class="grid" style="margin-top:12px"><thead><tr>' +
       '<th>dtm_barrier — branch</th><th>op</th><th>ghi chú</th></tr></thead><tbody>';
  if (!bars.length) {
    h += '<tr><td colspan="3" style="color:var(--ink-faint)">— trống —</td></tr>';
  } else {
    bars.forEach(function (b) {
      var hot = /CHẶN/.test(b[2]);
      h += '<tr' + (hot ? ' class="flash"' : '') + '><td class="num">' + b[0] + '</td><td class="mono">' +
           b[1] + '</td><td class="mono">' + VF.esc(b[2]) + '</td></tr>';
    });
  }
  return h + '</tbody></table>';
}
