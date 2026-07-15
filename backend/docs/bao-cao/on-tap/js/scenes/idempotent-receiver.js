/* Idempotent receiver — Demo 3. Tua offset Kafka về 0, ép giao lại toàn bộ.
   Số thật: 8 event → 8 thông báo. Tắt processed_events → 16. */
VF.scene('idempotent-receiver', {
  title: 'At-least-once + idempotent receiver',
  controls: [
    { kind: 'toggle', id: 'off', label: '⚡ TẮT bảng processed_events', value: false }
  ],

  build: function (o) {
    var L = [
      { name: 'Kafka', sub: 'order.events' },
      { name: 'Notification', sub: 'consumer' },
      { name: 'notification_db', sub: 'processed + notifications' }
    ];
    var K = 0, C = 1, D = 2;
    var steps = [], rows = [];
    function push(note, processed, notifs, hi) {
      steps.push({
        view: VF.lanes(L, rows.slice(), rows.length - 1) + idemState(processed, notifs, hi),
        note: note
      });
    }

    rows.push({ from: K, to: C, label: '8 event (offset 0…7)', cls: 'ok' });
    push('Lần chạy bình thường: Kafka giao <b>8 sự kiện</b>.', 0, 0);

    rows.push({ from: C, to: D, label: 'INSERT processed_events + INSERT notification — MỘT tx', cls: 'ok' });
    push(o.off
      ? 'Chỉ ghi thông báo. Không có bảng khử trùng lặp.'
      : 'Mỗi sự kiện ghi <b>hai</b> thứ trong <b>cùng một transaction</b>: một dòng <code>processed_events</code> (khoá chính = <code>event_id</code>) và bản ghi thông báo. Cùng sống hoặc cùng chết.',
      o.off ? 0 : 8, 8);

    rows.push({ from: C, to: K, label: 'commit offset = 8', cls: 'ok' });
    push('Xử lý xong 8/8. Khách nhận <b>8 thông báo</b>. Mọi thứ đúng.', o.off ? 0 : 8, 8);

    rows.push({ from: C, to: C, label: '⚡ reset-offsets --to-earliest', cls: 'fail' });
    push('<span class="hit">Giờ ép Kafka giao lại từ đầu.</span> Đây <b>không phải trò bịa</b> — nó mô phỏng đúng chuyện xảy ra thật khi consumer chết trước lúc commit, khi rebalance consumer group, hoặc khi outbox dispatcher phát trùng. <b>Kafka chỉ hứa at-least-once.</b>',
      o.off ? 0 : 8, 8);

    rows.push({ from: K, to: C, label: 'giao lại 8 event (offset 0…7)', cls: 'wait' });
    push('Đúng 8 sự kiện <b>cũ</b> được giao lại. <code>event_id</code> của chúng <b>y hệt lần trước</b>.', o.off ? 0 : 8, 8);

    if (o.off) {
      rows.push({ from: C, to: D, label: 'INSERT 8 thông báo NỮA', cls: 'fail' });
      steps.push({
        view: VF.lanes(L, rows, rows.length - 1) + idemState(0, 16, true),
        note: '<b>16 thông báo cho 8 đơn hàng.</b> Khách bị spam gấp đôi. Và nếu consumer này là <b>Payment</b> thay vì Notification, thì đó là <b>trừ tiền hai lần</b>. Đây chính là cái giá của at-least-once khi phía nhận không idempotent.'
      });
    } else {
      rows.push({ from: C, to: D, label: 'INSERT processed_events → TRÙNG KHOÁ ✔', cls: 'comp' });
      push('Consumer thử ghi <code>processed_events</code>, khoá chính <b>trùng</b> ⇒ INSERT hỏng ⇒ consumer hiểu <b>“event này tôi xử lý rồi”</b> và <b>bỏ qua</b>, không ghi thông báo.',
        8, 8, true);

      steps.push({
        view: VF.lanes(L, rows, rows.length - 1) + idemState(8, 8, true),
        note: '<b>Vẫn đúng 8 thông báo, không phải 16.</b> Câu để nói: <i>“Kafka <b>không</b> cho exactly-once xuyên biên service — nó giao <b>ít nhất một lần</b>. Ta đạt <b>hiệu ứng</b> đúng-một-lần bằng bảng <code>processed_events</code> ghi <b>cùng transaction</b> với business write. Giao lại bao nhiêu lần cũng chỉ có tác dụng một lần.”</i>'
      });
    }
    return steps;
  }
});

function idemState(processed, notifs, hot) {
  var wrong = notifs === 16;
  return VF.stat([
    {
      label: 'Kafka đã giao',
      value: hot ? '16 lần' : '8 lần',
      tone: 'mute',
      hint: hot ? 'giao lại toàn bộ sau khi tua offset' : 'lần chạy đầu tiên'
    },
    {
      label: 'processed_events',
      value: processed === 0 ? '—' : processed + ' dòng',
      tone: processed === 0 ? 'bad' : 'mute',
      hint: processed === 0 ? 'bảng bị TẮT — không có gì chặn trùng' : 'khoá chính = event_id'
    },
    {
      label: 'Thông báo gửi khách',
      value: notifs + (wrong ? ' ✖' : hot ? ' ✔' : ''),
      tone: wrong ? 'bad' : (hot ? 'ok' : 'mute'),
      hot: hot,
      hint: wrong ? 'nhân đôi — khách bị spam gấp đôi'
                  : (hot ? 'giao 16, tác dụng 8 — exactly-once' : 'đúng một thông báo mỗi đơn')
    }
  ]);
}
