/* Sổ cái kép — mỗi luồng tiền sinh HAI bút toán, tổng luôn = 0.
   Điểm phải nói: SYSTEM được phép âm, vì số âm chính là nghĩa vụ phải trả. */
VF.scene('double-entry-ledger', {
  title: 'Sổ cái kép (double-entry)',
  controls: [
    {
      kind: 'select', id: 'flow', label: 'Luồng tiền', value: 'capture',
      options: [
        ['capture', 'Khách trả tiền đơn 42.000đ'],
        ['refund', 'Hoàn tiền đơn 42.000đ'],
        ['payout', 'Chi trả cho quán']
      ]
    },
    { kind: 'toggle', id: 'single', label: '⚡ Ghi một chiều (chỉ trừ ví khách)', value: false }
  ],

  build: function (o) {
    var steps = [];
    var amt = 42000;
    var flows = {
      capture: [['Ví khách', -amt], ['SYSTEM (clearing)', +amt]],
      refund: [['SYSTEM (clearing)', -amt], ['Ví khách', +amt]],
      payout: [['SYSTEM (clearing)', -amt], ['Ví quán', +amt]]
    };
    var pair = flows[o.flow];

    steps.push({
      view: ledgerTable([], 500000, 0),
      note: 'Trạng thái đầu: ví khách <code>500.000đ</code>, tài khoản <code>SYSTEM</code> = 0. <code>SYSTEM</code> là <b>tài khoản trung gian</b> (clearing) — tiền đi qua nó, không nằm lại.'
    });

    if (o.single) {
      steps.push({
        view: ledgerTable([pair[0]], 500000 + pair[0][1], 0, true),
        note: '<span class="hit">Ghi một chiều: tiền bốc hơi.</span> Ví khách <code>-42.000</code>, nhưng <b>không tài khoản nào nhận</b>. Tổng hệ thống <b>không còn cân</b>. Đến cuối tháng đối soát, bạn <b>không thể trả lời</b> được câu “42.000 này đi đâu?”. Sổ một chiều <b>không có khả năng tự kiểm tra</b>.'
      });
      return steps;
    }

    steps.push({
      view: ledgerTable([pair[0]], 500000 + (pair[0][0] === 'Ví khách' ? pair[0][1] : 0), pair[0][0] === 'SYSTEM (clearing)' ? pair[0][1] : 0),
      note: 'Bút toán thứ nhất: <code>' + pair[0][0] + ' ' + fmtSigned(pair[0][1]) + '</code>. <b>Chưa xong</b> — một mình nó thì sổ chưa cân.'
    });

    var wallet = 500000, sys = 0;
    pair.forEach(function (p) {
      if (p[0] === 'Ví khách') wallet += p[1];
      if (p[0] === 'SYSTEM (clearing)') sys += p[1];
    });

    steps.push({
      view: ledgerTable(pair, wallet, sys, false, true),
      note: '<b>Hai bút toán, tổng = 0.</b> Cả hai được ghi trong <b>một transaction</b>, khoá <code>FOR UPDATE</code> <b>cả hai ví</b> ⇒ không ai chen ngang được. Câu để nói: <i>“Mỗi luồng tiền sinh <b>hai</b> bút toán cân bằng. Nếu tổng khác 0 thì có bug — và ta <b>phát hiện được ngay</b>, thay vì phát hiện lúc đối soát cuối tháng.”</i>' +
        (sys < 0 ? ' <b>Chú ý <code>SYSTEM</code> đang ÂM</b> — và điều đó <b>đúng</b>: số âm ở tài khoản clearing chính là <b>nghĩa vụ hệ thống phải trả</b>. Chỉ <code>SYSTEM</code> được phép âm; ví người dùng thì không.' : '')
    });
    return steps;
  }
});

function fmtSigned(n) {
  return (n > 0 ? '+' : '−') + Math.abs(n).toLocaleString('vi-VN');
}

function ledgerTable(entries, wallet, sys, broken, done) {
  var sum = entries.reduce(function (a, e) { return a + e[1]; }, 0);
  var h = '<table class="grid"><thead><tr><th>Bút toán</th><th>Tài khoản</th><th style="text-align:right">Số tiền</th></tr></thead><tbody>';
  if (!entries.length) {
    h += '<tr><td colspan="3" style="color:var(--ink-faint)">— chưa có bút toán —</td></tr>';
  } else {
    entries.forEach(function (e, i) {
      h += '<tr' + (i === entries.length - 1 ? ' class="flash"' : '') + '><td class="num">' + (i + 1) +
           '</td><td class="mono">' + VF.esc(e[0]) + '</td><td class="num" style="color:' +
           (e[1] < 0 ? 'var(--stamp)' : 'var(--ok)') + ';font-weight:600">' + fmtSigned(e[1]) + ' đ</td></tr>';
    });
    var okSum = sum === 0;
    h += '<tr><td></td><td class="mono" style="font-weight:700">TỔNG</td>' +
         '<td class="num" style="font-weight:700;color:' + (okSum ? 'var(--ok)' : 'var(--stamp)') + '">' +
         (okSum ? '0 ✔ cân' : fmtSigned(sum) + ' ✖ LỆCH') + '</td></tr>';
  }
  h += '</tbody></table>';

  h += '<div style="display:grid;grid-template-columns:1fr 1fr;gap:24px;margin-top:16px">';
  h += '<table class="grid"><thead><tr><th>Số dư ví khách</th></tr></thead><tbody><tr><td class="num" style="font-size:18px">' +
       wallet.toLocaleString('vi-VN') + ' đ</td></tr></tbody></table>';
  h += '<table class="grid"><thead><tr><th>Số dư SYSTEM (clearing)</th></tr></thead><tbody><tr><td class="num" style="font-size:18px;color:' +
       (sys < 0 ? 'var(--wait)' : 'inherit') + '">' + fmtSigned(sys).replace('+', '') + ' đ' +
       (sys < 0 ? ' <span style="font-size:11px">← âm là ĐÚNG</span>' : '') +
       '</td></tr></tbody></table></div>';

  if (broken) {
    h += '<p style="font-family:var(--font-mono);font-size:12px;color:var(--stamp);margin-top:12px">' +
         '42.000đ rời khỏi ví khách nhưng không vào tài khoản nào. Sổ sách không thể đối soát.</p>';
  }
  return h;
}
