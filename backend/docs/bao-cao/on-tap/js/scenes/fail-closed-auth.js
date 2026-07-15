/* Fail-closed authorization — whitelist JTI trên Redis.
   Chi tiết đáng điểm: chọn whitelist hay blacklist quyết định hệ thống sẽ hành xử
   thế nào KHI REDIS CHẾT. Đó là một quyết định an ninh, không phải chi tiết kỹ thuật. */
VF.scene('fail-closed-auth', {
  title: 'Fail-closed: whitelist JTI, không phải blacklist',
  controls: [
    {
      kind: 'select', id: 'mode', label: 'Cách lưu', value: 'white',
      options: [['white', 'Whitelist (ta dùng)'], ['black', 'Blacklist (cách phổ biến)']]
    },
    { kind: 'toggle', id: 'down', label: '⚡ Redis chết', value: true }
  ],

  build: function (o) {
    var L = [
      { name: 'Client', sub: 'kèm JWT' },
      { name: 'Service', sub: 'middleware auth' },
      { name: 'Redis', sub: 'kho JTI' }
    ];
    var C = 0, S = 1, R = 2;
    var white = o.mode === 'white';
    var steps = [], rows = [];
    function push(note, extra) {
      steps.push({ view: VF.lanes(L, rows.slice(), rows.length - 1) + (extra || ''), note: note });
    }

    rows.push({ from: C, to: S, label: 'GET /orders + Bearer <jwt>', cls: 'ok' });
    push('Request kèm JWT. Chữ ký RS256 <b>hợp lệ</b>, chưa hết hạn.',
      authTable(white, 'Chữ ký hợp lệ ✔ — nhưng token này có thể đã bị <b>thu hồi</b> (logout, đổi mật khẩu, bị đánh cắp). JWT tự nó <b>không biết</b> điều đó.'));

    rows.push({ from: S, to: R, label: white ? 'JTI có trong whitelist?' : 'JTI có trong blacklist?', cls: o.down ? 'fail' : 'wait' });

    if (!o.down) {
      rows.push({ from: R, to: S, label: white ? 'CÓ ⇒ token còn sống' : 'KHÔNG ⇒ token chưa bị thu hồi', cls: 'ok' });
      rows.push({ from: S, to: C, label: '200 OK', cls: 'ok' });
      push('Redis khoẻ ⇒ <b>cả hai cách đều chạy đúng</b>. Khác biệt chỉ lộ ra khi Redis chết. Bật nút để thấy.',
        authTable(white, 'Redis trả lời được ⇒ quyết định đúng. Không phân biệt được whitelist hay blacklist ở đây.'));
      return steps;
    }

    push('<span class="hit">Redis chết.</span> Service <b>không thể biết</b> token này còn sống hay đã bị thu hồi. Đây là lúc phải chọn: <b>đoán an toàn</b> hay <b>đoán tiện lợi</b>.',
      authTable(white, 'Không có câu trả lời. Phải quyết định trong tình trạng mù.'));

    if (white) {
      rows.push({ from: S, to: C, label: '401 Unauthorized', cls: 'comp' });
      steps.push({
        view: VF.lanes(L, rows, rows.length - 1) +
          authTable(true, '<b>Whitelist:</b> “còn trong kho ⇒ hợp lệ”. Redis chết ⇒ <b>không thấy JTI nào</b> ⇒ <b>từ chối tất cả</b>.'),
        note: '<b>Fail-closed.</b> Redis chết thì <b>toàn hệ thống mất đăng nhập</b> — nghe rất tệ. Nhưng cái tệ hơn nằm ở phương án kia. Câu để nói: <i>“Khi không thể xác minh, hệ thống phải <b>từ chối</b>, không phải cho qua.”</i>'
      });
    } else {
      rows.push({ from: S, to: C, label: '200 OK — CHO QUA HẾT', cls: 'fail' });
      steps.push({
        view: VF.lanes(L, rows, rows.length - 1) +
          authTable(false, '<b>Blacklist:</b> “không có trong sổ đen ⇒ hợp lệ”. Redis chết ⇒ <b>sổ đen rỗng</b> ⇒ <b>mọi token đều “sạch”</b>.'),
        note: '<span class="hit">Fail-open — và đây là lỗ hổng an ninh thật.</span> Mọi token <b>đã bị thu hồi</b> đều sống lại: token của người vừa logout, token bị đánh cắp mà nạn nhân đã báo, token của nhân viên vừa bị sa thải. Kẻ tấn công chỉ cần <b>làm Redis quá tải</b> là mở được cửa. Đây là lý do ta chọn whitelist — <b>chi phí là tính sẵn sàng, nhưng đổi lấy an toàn</b>.'
      });
    }
    return steps;
  }
});

function authTable(white, msg) {
  return '<table class="grid" style="margin-top:24px"><thead><tr><th>' +
    (white ? 'Whitelist — “có mặt nghĩa là còn sống”' : 'Blacklist — “vắng mặt nghĩa là còn sống”') +
    '</th></tr></thead><tbody><tr><td>' + msg + '</td></tr></tbody></table>';
}
