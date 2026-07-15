/* Trace waterfall — dựng lại đúng màn Jaeger sẽ hiện lúc demo:
   1 trace · 9 span · 5 service. Kèm hạn chế tự nêu: chưa nối qua biên Kafka. */
VF.scene('trace-waterfall', {
  title: 'Một request, 9 span, 5 service',
  controls: [],

  build: function () {
    // [tên span, service, độ sâu, bắt đầu ms, kéo dài ms, ghi chú]
    var spans = [
      ['POST /api/v1/orders', 'kong', 0, 0, 342, 'Trace bắt đầu ở <b>gateway</b>. Kong sinh <code>trace_id</code> và nhét vào header — <b>mọi thứ sau đây đều mang theo nó</b>.'],
      ['POST /api/v1/orders', 'order', 1, 6, 330, 'Order nhận request. Span này là <b>cha</b> của tất cả lời gọi phía dưới.'],
      ['grpc GetStoreForOrder', 'store', 2, 18, 46, 'Lời gọi gRPC <b>đồng bộ</b> đầu tiên. Đây chính là lời gọi mà <b>circuit breaker</b> bảo vệ.'],
      ['grpc ResolveShipFee', 'location', 3, 30, 22, 'store gọi tiếp sang location. <b>Đây là chỗ mà không có tracing thì bạn mù</b>: nhìn từ order, bạn không hề biết location có tham gia.'],
      ['dtm SubmitSaga', 'dtm', 2, 70, 250, 'Order nộp saga. Từ đây trở đi DTM là bên gọi.'],
      ['grpc ApplyPromotion', 'promotion', 3, 80, 60, 'Nhánh 1.'],
      ['grpc Capture', 'payment', 3, 145, 95, 'Nhánh 2 — chậm nhất trong ba nhánh, vì phải khoá hai ví và ghi hai bút toán.'],
      ['grpc ConfirmUsage', 'promotion', 3, 245, 55, 'Nhánh 3.'],
      ['tx: insert order + outbox', 'order', 2, 305, 28, 'Ghi đơn và sự kiện trong <b>một transaction</b>. Span cuối cùng.']
    ];

    var colors = {
      kong: 'var(--ink-soft)', order: 'var(--owner-van)', store: 'var(--owner-duc)',
      location: 'var(--owner-duc)', dtm: 'var(--stamp)', promotion: 'var(--ok)', payment: 'var(--wait)'
    };
    var total = 342;

    var steps = spans.map(function (_, i) {
      var rows = spans.slice(0, i + 1);
      var h = '<div style="min-width:600px">';
      rows.forEach(function (s, j) {
        var last = j === i;
        var left = (s[3] / total * 100).toFixed(1);
        var w = Math.max(s[4] / total * 100, 1.2).toFixed(1);
        h += '<div style="display:grid;grid-template-columns:260px 1fr 64px;gap:12px;align-items:center;' +
             'padding:5px 0;border-bottom:1px dashed var(--rule);' + (last ? 'background:var(--paper-sunken)' : '') + '">';
        h += '<div style="padding-left:' + (s[2] * 14) + 'px;font-family:var(--font-mono);font-size:11px;' +
             'white-space:nowrap;overflow:hidden;text-overflow:ellipsis">' +
             '<span style="color:' + colors[s[1]] + ';font-weight:700">' + s[1] + '</span> ' +
             '<span style="color:var(--ink-soft)">' + VF.esc(s[0]) + '</span></div>';
        h += '<div style="position:relative;height:14px;background:var(--paper-sunken)">' +
             '<div style="position:absolute;left:' + left + '%;width:' + w + '%;top:0;bottom:0;background:' +
             colors[s[1]] + ';opacity:' + (last ? '1' : '.55') + '"></div></div>';
        h += '<div class="num" style="font-family:var(--font-mono);font-size:11px;color:var(--ink-faint);text-align:right">' +
             s[4] + 'ms</div>';
        h += '</div>';
      });
      h += '</div>';

      var note = spans[i][5];
      if (i === spans.length - 1) {
        note += '<br><b>Tổng: 1 trace · 9 span · 5 service.</b> Một request của khách chạm 5 service — và ta <b>nhìn thấy toàn bộ</b> thay vì đoán. ' +
                '<b>Hạn chế phải tự nêu trước khi thầy hỏi:</b> ngữ cảnh trace <b>chưa nối qua biên Kafka</b> — span của consumer là span <b>gốc mới</b>, hiện chỉ liên kết được qua <code>trace_id</code> trong log.';
      }
      return { view: h, note: note };
    });

    steps.unshift({
      view: '<p style="font-family:var(--font-mono);font-size:12px;color:var(--ink-faint);min-width:600px">' +
            'Trace rỗng. Bấm <b>Sau ▶</b> để dựng từng span đúng như Jaeger sẽ vẽ.</p>',
      note: 'Đây là màn <b>Jaeger</b> (<code>localhost:17093</code>) mà thầy sẽ nhìn thấy. Mỗi thanh là một <b>span</b> — một đoạn công việc có bắt đầu, có kết thúc, và biết <b>cha</b> của mình là ai.'
    });
    return steps;
  }
});
