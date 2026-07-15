/* Consumer retry → DLQ, và chi tiết tinh tế nhất: khi DLQ cũng chết thì
   CỐ Ý không commit offset, để giữ thứ tự partition. */
VF.scene('kafka-retry-dlq', {
  title: 'Retry → DLQ và bảo toàn thứ tự partition',
  controls: [
    { kind: 'toggle', id: 'dlqDown', label: '⚡ DLQ cũng không ghi được', value: false }
  ],

  build: function (o) {
    var L = [
      { name: 'Kafka', sub: 'order.events · P0' },
      { name: 'Consumer', sub: 'notification' },
      { name: 'DLQ', sub: 'order.events.dlq' }
    ];
    var K = 0, C = 1, D = 2;
    var steps = [], rows = [];
    // offset 5 là message độc; 6 và 7 xếp sau nó trong CÙNG partition
    function push(note, off, done) {
      steps.push({ view: VF.lanes(L, rows.slice(), rows.length - 1) + partitionLog(off, done), note: note });
    }

    rows.push({ from: K, to: C, label: 'msg offset=5 (JSON hỏng)', cls: 'wait' });
    push('Consumer nhận message ở <b>offset 5</b>. Payload hỏng — xử lý ném lỗi.', 5, 4);

    rows.push({ from: C, to: C, label: 'retry #1 · chờ 0,5s', cls: 'wait' });
    push('Retry lần 1 sau <b>0,5 giây</b>. Backoff luỹ tiến — vì phần lớn lỗi là <b>tạm thời</b> (DB nghẽn, mạng chớp).', 5, 4);

    rows.push({ from: C, to: C, label: 'retry #2 · chờ 1s', cls: 'wait' });
    push('Retry lần 2 sau <b>1 giây</b>.', 5, 4);

    rows.push({ from: C, to: C, label: 'retry #3 · chờ 2s', cls: 'fail' });
    push('Retry lần 3 sau <b>2 giây</b> — vẫn hỏng. Payload hỏng thì retry bao nhiêu lần cũng vô ích. <b>Hết quota retry.</b>', 5, 4);

    if (o.dlqDown) {
      rows.push({ from: C, to: D, label: 'park vào DLQ ✖', cls: 'fail' });
      push('<span class="hit">DLQ cũng không ghi được.</span> Giờ consumer đứng trước hai lựa chọn, và <b>lựa chọn ở đây chính là chỗ đáng điểm</b>.', 5, 4);

      steps.push({
        view: VF.lanes(L, rows, rows.length - 1) + partitionLog(5, 4, true),
        note: '<b>Consumer CỐ Ý không commit offset và dừng lại ở message này.</b> Nó <b>không</b> bỏ qua để chạy tiếp. Vì sao? Nếu bỏ qua, <code>offset 6</code> (<code>order.cancelled</code>) sẽ được xử lý <b>trước</b> <code>offset 5</code> — mà cả hai là sự kiện của <b>cùng một đơn</b> trong <b>cùng một partition</b>. Kafka đảm bảo thứ tự <b>trong partition</b>, và ta <b>không được phép phá</b> đảm bảo đó. Thà tắc nghẽn còn hơn xử lý sai thứ tự. Đây là đánh đổi <b>availability vs correctness</b>, và ta chọn correctness.'
      });
    } else {
      rows.push({ from: C, to: D, label: 'park vào DLQ ✔', cls: 'comp' });
      push('Message được đẩy sang <code>order.events.dlq</code> kèm header truy vết: ' +
           '<code>x-original-topic</code> · <code>x-original-partition</code> · <code>x-original-offset</code> · <code>x-error</code> · <code>x-failed-at</code>. ' +
           '<b>Không mất dữ liệu</b> — sau này mở DLQ ra là biết chính xác message nào hỏng, hỏng vì gì, ở đâu.', 5, 4);

      rows.push({ from: C, to: K, label: 'commit offset 5', cls: 'ok' });
      push('Giờ mới <b>commit</b>. Message độc đã được cách ly an toàn ⇒ tiến lên được.', 5, 5);

      rows.push({ from: K, to: C, label: 'msg offset=6 ✔', cls: 'ok' });
      steps.push({
        view: VF.lanes(L, rows, rows.length - 1) + partitionLog(6, 6),
        note: '<b>Luồng chạy tiếp.</b> Một message hỏng <b>không</b> làm tắc cả partition. Đây là lý do phải có DLQ chứ không chỉ retry. Bật nút “DLQ cũng chết” để thấy hệ thống <b>cố ý</b> chọn tắc nghẽn.'
      });
    }
    return steps;
  }
});

function partitionLog(cur, committed, stuck) {
  var msgs = [
    [3, 'order.placed', 'đã xử lý'],
    [4, 'order.paid', 'đã xử lý'],
    [5, 'order.status_changed (JSON hỏng)', 'ĐANG KẸT'],
    [6, 'order.cancelled', 'chờ'],
    [7, 'order.delivered', 'chờ']
  ];
  var h = '<table class="grid" style="margin-top:24px"><thead><tr>' +
          '<th>offset</th><th>message trong partition 0</th><th>trạng thái</th></tr></thead><tbody>';
  msgs.forEach(function (m) {
    var isCur = m[0] === cur;
    var done = m[0] <= committed;
    var cls = done ? 'ok' : isCur ? (stuck ? 'fail' : 'wait') : 'dead';
    var label = done ? 'đã commit' : isCur ? (stuck ? '⛔ KẸT — không commit' : 'đang xử lý') : 'chờ tới lượt';
    h += '<tr' + (isCur ? ' class="flash"' : '') + '><td class="num">' + m[0] + '</td>' +
         '<td class="mono">' + VF.esc(m[1]) + '</td>' +
         '<td><span class="chip ' + cls + '">' + label + '</span></td></tr>';
  });
  h += '</tbody></table>';
  if (stuck) {
    h += '<p style="font-family:var(--font-mono);font-size:12px;color:var(--stamp);margin-top:12px">' +
         'Con trỏ commit đứng im ở offset 4. Offset 6, 7 <b>không được xử lý vượt mặt</b> — thứ tự partition được giữ.</p>';
  }
  return h;
}
