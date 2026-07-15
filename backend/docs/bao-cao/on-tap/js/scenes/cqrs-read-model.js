/* CQRS — reporting là read model thuần, dựng 100% từ event, không có API ghi.
   Điểm phải nói: nhất quán CUỐI là cái ta CHỌN, không phải cái ta chịu đựng. */
VF.scene('cqrs-read-model', {
  title: 'CQRS — tách đường ghi khỏi đường đọc',
  controls: [
    { kind: 'toggle', id: 'sync', label: '⚡ Gộp lại: tính báo cáo ngay trong đường ghi', value: false }
  ],

  build: function (o) {
    var L = [
      { name: 'Khách', sub: 'đặt đơn' },
      { name: 'Order', sub: 'đường GHI' },
      { name: 'Kafka', sub: 'order.placed' },
      { name: 'Reporting', sub: 'đường ĐỌC' }
    ];
    var U = 0, O = 1, K = 2, R = 3;
    var steps = [], rows = [];
    function push(note, extra) {
      steps.push({ view: VF.lanes(L, rows.slice(), rows.length - 1) + (extra || ''), note: note });
    }

    if (o.sync) {
      rows.push({ from: U, to: O, label: 'POST /orders', cls: 'ok' });
      push('Khách đặt đơn.', cqrsLatency(null, null));

      rows.push({ from: O, to: R, label: 'gọi thẳng: tính lại doanh thu, top món, biểu đồ…', cls: 'wait' });
      push('Cách gộp: đường ghi <b>tự tính luôn báo cáo</b> trước khi trả lời khách. Các truy vấn tổng hợp này <b>quét cả bảng</b>.',
        cqrsLatency(null, null));

      rows.push({ from: O, to: U, label: '201 — sau 1.400 ms', cls: 'fail' });
      steps.push({
        view: VF.lanes(L, rows, rows.length - 1) + cqrsLatency(1400, 0),
        note: '<span class="hit">Khách phải chờ 1,4 giây để đặt một bát phở.</span> Báo cáo thì luôn chính xác tuyệt đối — nhưng <b>mọi khách hàng</b> đều trả giá cho việc <b>admin thỉnh thoảng mở dashboard</b>. Và khi lượng đơn tăng, con số này chỉ có tăng.'
      });
      return steps;
    }

    rows.push({ from: U, to: O, label: 'POST /orders', cls: 'ok' });
    push('Khách đặt đơn.', cqrsLatency(null, null));

    rows.push({ from: O, to: U, label: '201 — sau 120 ms', cls: 'ok' });
    push('<b>Trả lời khách ngay.</b> Đường ghi chỉ làm đúng việc của nó: ghi đơn, ghi outbox, xong.',
      cqrsLatency(120, null));

    rows.push({ from: O, to: K, label: 'order.placed', cls: 'ok' });
    push('Sự kiện đi vào Kafka. Đường ghi <b>không quan tâm</b> ai đọc nó — nó đã trả lời khách xong rồi.',
      cqrsLatency(120, null));

    rows.push({ from: K, to: R, label: 'consume → cập nhật read model', cls: 'ok' });
    steps.push({
      view: VF.lanes(L, rows, rows.length - 1) + cqrsLatency(120, 2500),
      note: '<b>Read model cập nhật sau ~2 giây.</b> Trong 2 giây đó, dashboard admin <b>chưa thấy đơn này</b> — đó là <b>nhất quán cuối</b>, và ta <b>chấp nhận có ý thức</b>. Câu để nói: <i>“Reporting <b>không có một API ghi nghiệp vụ nào</b>. Toàn bộ trạng thái dựng từ sự kiện. Nếu read model hỏng, ta chỉ cần <b>replay lại event</b> là dựng lại được từ đầu.”</i> Đổi lại: khách đặt hàng <b>không bao giờ</b> phải chờ vì admin muốn xem báo cáo.'
    });
    return steps;
  }
});

function cqrsLatency(write, read) {
  return VF.stat([
    {
      label: 'Khách chờ bao lâu',
      value: write === null ? '…' : write.toLocaleString('vi-VN') + ' ms',
      tone: write === null ? 'mute' : (write > 1000 ? 'bad' : 'ok'),
      hint: write === null ? 'đang xử lý'
          : (write > 1000 ? 'mọi khách trả giá cho báo cáo của admin' : 'ghi đơn xong là trả lời ngay')
    },
    {
      label: 'Dashboard trễ bao lâu',
      value: read === null ? '…' : (read === 0 ? '0 ms' : '~' + (read / 1000) + ' giây'),
      tone: 'mute',
      hint: read === 0 ? 'chính xác tuyệt đối — nhưng phải trả giá bên trái'
          : (read === null ? 'chưa cập nhật' : 'nhất quán cuối — chấp nhận có ý thức')
    }
  ]);
}
