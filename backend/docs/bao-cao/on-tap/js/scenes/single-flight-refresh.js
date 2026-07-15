/* Single-flight refresh — bài toán điều phối đồng thời ở BIÊN CLIENT.
   Refresh token là single-use, nên N request cùng hết hạn mà mỗi cái tự gọi
   /auth/refresh sẽ tự vô hiệu hoá lẫn nhau. */
VF.scene('single-flight-refresh', {
  title: 'Single-flight refresh ở biên client',
  controls: [
    { kind: 'toggle', id: 'off', label: '⚡ TẮT single-flight (mỗi request tự refresh)', value: false }
  ],

  build: function (o) {
    var L = [
      { name: 'Trang', sub: '3 request song song' },
      { name: 'Client', sub: 'lớp auth-refresh' },
      { name: 'user-service', sub: '/auth/refresh' }
    ];
    var P = 0, C = 1, S = 2;
    var steps = [], rows = [];
    function push(note, extra) {
      steps.push({ view: VF.lanes(L, rows.slice(), rows.length - 1) + (extra || ''), note: note });
    }

    rows.push({ from: P, to: C, label: '3 request cùng lúc: /orders · /cart · /profile', cls: 'wait' });
    push('Người dùng mở trang. Ba lời gọi API chạy <b>song song</b>. Access token vừa hết hạn.',
      rtState([['RT-1', 'còn sống']], null));

    rows.push({ from: C, to: C, label: 'cả 3 nhận 401', cls: 'fail' });
    push('Cả ba cùng nhận <code>401</code>. Cả ba cùng muốn làm <b>một việc giống hệt nhau</b>: đi làm mới token.',
      rtState([['RT-1', 'còn sống']], null));

    if (o.off) {
      rows.push({ from: C, to: S, label: 'refresh(RT-1) ×3 — cùng lúc', cls: 'fail' });
      push('Không có single-flight ⇒ <b>ba lời gọi <code>/auth/refresh</code> cùng bắn đi</b>, cùng mang <b>một</b> refresh token <code>RT-1</code>.',
        rtState([['RT-1', 'còn sống']], null));

      rows.push({ from: S, to: C, label: '#1 → RT-2 ✔ · RT-1 bị thu hồi', cls: 'ok' });
      push('Lời gọi thứ nhất thắng: đổi được <code>RT-2</code>. Và vì refresh token là <b>single-use</b>, server <b>thu hồi ngay <code>RT-1</code></b>.',
        rtState([['RT-1', 'ĐÃ THU HỒI'], ['RT-2', 'còn sống']], null));

      rows.push({ from: S, to: C, label: '#2, #3 → 401 (RT-1 chết rồi)', cls: 'fail' });
      steps.push({
        view: VF.lanes(L, rows, rows.length - 1) + rtState([['RT-1', 'ĐÃ THU HỒI'], ['RT-2', 'còn sống']], 'Đăng xuất oan'),
        note: '<span class="hit">Ba request tự vô hiệu hoá lẫn nhau.</span> Lời gọi #2 và #3 vẫn cầm <code>RT-1</code> — token vừa bị chính #1 thu hồi. Server thấy refresh token đã dùng ⇒ nghi <b>replay attack</b> ⇒ <b>đăng xuất sạch</b>. Người dùng bị <b>văng ra khỏi ứng dụng vì chính hệ thống của mình</b>, không phải vì lỗi gì cả. Trớ trêu: <b>tính năng an ninh</b> (rotation single-use) lại thành <b>bug</b>, chỉ vì phía client thiếu điều phối.'
      });
    } else {
      rows.push({ from: C, to: S, label: 'refresh(RT-1) — CHỈ MỘT lời gọi', cls: 'ok' });
      push('Có single-flight: lớp <code>auth-refresh.ts</code> thấy <b>đã có một refresh đang bay</b> ⇒ <b>request #2 và #3 không gọi nữa, chúng chờ chung vào cùng một Promise</b>.',
        rtState([['RT-1', 'còn sống']], null));

      rows.push({ from: S, to: C, label: '→ RT-2 ✔ (RT-1 thu hồi bình thường)', cls: 'ok' });
      push('Refresh thành công. <code>RT-1</code> bị thu hồi — nhưng <b>không ai còn cầm nó</b> để dùng lại.',
        rtState([['RT-1', 'đã thu hồi (không ai dùng)'], ['RT-2', 'còn sống']], null));

      rows.push({ from: C, to: P, label: 'retry cả 3 request bằng token mới', cls: 'ok' });
      steps.push({
        view: VF.lanes(L, rows, rows.length - 1) + rtState([['RT-2', 'còn sống']], 'Cả 3 request thành công'),
        note: '<b>Một lời gọi refresh, ba request được cứu.</b> Câu để nói: <i>“Refresh token là <b>single-use</b>, nên nhiều request cùng hết hạn phải được <b>gom về một lời gọi refresh duy nhất</b>. Đây là bài toán <b>điều phối đồng thời</b> — chỉ khác là nó nằm ở <b>biên client</b> chứ không phải trong server.”</i>'
      });
    }
    return steps;
  }
});

function rtState(tokens, verdict) {
  var h = '<table class="grid" style="margin-top:24px"><thead><tr><th>Refresh token</th><th>trạng thái trên server</th></tr></thead><tbody>';
  tokens.forEach(function (t) {
    var dead = /THU HỒI|thu hồi/.test(t[1]);
    h += '<tr><td class="mono' + (dead ? ' strike' : '') + '">' + t[0] + '</td>' +
         '<td><span class="chip ' + (dead ? 'dead' : 'ok') + '">' + VF.esc(t[1]) + '</span></td></tr>';
  });
  h += '</tbody></table>';
  if (verdict) {
    var bad = /oan/.test(verdict);
    h += '<p style="font-family:var(--font-mono);font-size:14px;margin-top:12px;font-weight:700;color:' +
         (bad ? 'var(--stamp)' : 'var(--ok)') + '">Kết cục: ' + VF.esc(verdict) + '</p>';
  }
  return h;
}
