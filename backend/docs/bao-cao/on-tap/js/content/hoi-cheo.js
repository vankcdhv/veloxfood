/* Tab Hỏi chéo — thầy hay hỏi người này về phần người kia.
   Đây là chỗ cả nhóm dễ vỡ trận nhất, nên tách riêng. */
VF.content = VF.content || {};
VF.content.hoicheo = {
  label: 'Hỏi chéo',
  accent: 'var(--stamp)',
  blurb: 'Thầy sẽ hỏi Nam về saga và hỏi Đức về exactly-once. <b>Ba câu bắt buộc</b> ai cũng phải trả lời được, cộng các câu bẫy hay gặp.',
  cards: [

    {
      id: 'x-mandatory',
      title: 'Ba câu bắt buộc — ai cũng phải trả lời được',
      eyebrow: ['không được trượt câu nào'],
      lede: 'Không phải câu khó. Nhưng nếu người được hỏi <b>không phải chủ sở hữu</b> mà ấp úng, thì cả nhóm lộ ra là <b>chia việc chứ không chia hiểu biết</b>. Đó là thứ mất điểm nặng nhất.',
      probes: [
        ['<b>Nam</b> phải trả lời được: Saga là gì, vì sao không dùng 2PC?',
         'Giao dịch trải qua <b>3 database khác nhau</b> — không transaction nào của Postgres phủ được cả ba. <b>2PC</b> làm được nhưng nó <b>khoá tài nguyên</b> trong lúc chờ pha 2; coordinator chết hoặc mạng đứt thì khoá <b>treo vô thời hạn</b> ⇒ giao thức <b>blocking</b>. <b>Saga</b> đổi nhất quán tức thời lấy tính sẵn sàng: mỗi bước <b>commit ngay, nhả khoá ngay</b>; bước sau hỏng thì <b>bù trừ ngược</b> các bước trước. Hệ thống có thể <b>tạm thời</b> không nhất quán nhưng <b>không bao giờ khoá ai</b>.'],
        ['<b>Đức</b> phải trả lời được: Vì sao at-least-once + idempotent receiver lại cho ra hiệu ứng exactly-once?',
         'Kafka <b>không</b> cho exactly-once xuyên biên service — consumer xử lý xong rồi chết <b>trước khi commit offset</b> thì message <b>sẽ được giao lại</b>. Ta <b>không ép Kafka</b> làm điều nó không làm được; ta làm cho <b>hiệu ứng nghiệp vụ</b> đúng một lần: bảng <code>processed_events</code> (khoá chính = <code>event_id</code>) ghi trong <b>cùng transaction</b> với business write. Giao lại ⇒ trùng khoá ⇒ bỏ qua. <b>Giao nhiều lần, tác dụng một lần.</b>'],
        ['<b>Văn</b> phải trả lời được: Vì sao user-service là SPOF, và vì sao vẫn chấp nhận?',
         'Mọi route protected của <b>9 service kia</b> đều gọi gRPC <code>CheckPermission</code> sang user-service ⇒ user-service chết là <b>toàn hệ thống ngừng phục vụ route protected</b>. Chấp nhận vì phương án kia — <b>mỗi service tự giữ bản sao phân quyền</b> — tạo ra <b>10 nguồn sự thật</b>: thu quyền một người phải cập nhật 10 chỗ, và trong lúc chưa đồng bộ xong thì <b>quyền đã bị thu vẫn còn hiệu lực</b>. Với <b>phân quyền</b>, một nguồn sự thật duy nhất <b>đáng giá hơn</b> tính sẵn sàng. Giảm nhẹ bằng <b>cache 5 phút</b> + invalidate chủ động.']
      ]
    },

    {
      id: 'x-traps',
      title: 'Câu bẫy — trả lời sai là mất điểm ngay',
      eyebrow: ['đọc kỹ'],
      lede: 'Những câu này nghe như hỏi kiến thức phổ thông, nhưng câu trả lời “thuộc lòng” lại <b>sai</b>.',
      probes: [
        ['Kafka có đảm bảo exactly-once không?',
         '<b>KHÔNG</b> — trả lời “có” là mất điểm. Kafka đảm bảo <b>at-least-once</b>. Nó <b>có</b> tính năng exactly-once, nhưng chỉ <b>trong phạm vi Kafka</b> (topic → topic, Kafka Streams). Khi consumer phải ghi vào <b>Postgres</b> — một hệ thống bên ngoài — thì Kafka <b>không thể</b> đảm bảo hai bên cùng commit. Đó là lý do <b>phía nhận phải tự idempotent</b>.'],
        ['Kafka có đảm bảo thứ tự message không?',
         'Chỉ <b>trong một partition</b>, <b>không</b> phải toàn cục. Nên phải <b>chọn khoá partition</b> cho đúng: các sự kiện của <b>cùng một đơn</b> dùng cùng khoá ⇒ vào cùng partition ⇒ đúng thứ tự. Đây là <b>quyết định thiết kế</b>, không phải hành vi mặc định.'],
        ['Circuit breaker có nên trip khi service trả về lỗi 404 / NotFound không?',
         '<b>KHÔNG.</b> Chỉ lỗi <b>hạ tầng</b> (<code>Unavailable</code>, <code>DeadlineExceeded</code>) mới trip. <code>NotFound</code> nghĩa là service <b>vẫn khoẻ</b> — nó đang từ chối <b>đúng</b>. Để lỗi nghiệp vụ trip breaker thì chỉ cần <b>5 khách gõ nhầm mã voucher</b> là cả hệ thống ngắt kết nối tới promotion-service.'],
        ['Compensation của một bước chưa từng chạy thì nên báo lỗi chứ?',
         '<b>KHÔNG — phải trả về succeed.</b> Đó là <b>null compensation</b>. Nếu nó báo lỗi thì DTM sẽ retry <b>mãi mãi</b> một việc <b>không bao giờ thành công được</b>, và saga <b>treo vĩnh viễn</b>. Bù trừ <b>bắt buộc phải là no-op idempotent</b>. (Đây từng là <b>bug thật</b> trong <code>refund_usecase.go</code> và đã sửa.)'],
        ['Redis chết thì cho token đi qua để hệ thống còn chạy được chứ?',
         '<b>KHÔNG — fail-closed.</b> Đó chính là <b>lỗ hổng fail-open</b>. Nếu dùng blacklist thì Redis chết ⇒ sổ đen rỗng ⇒ <b>mọi token bị thu hồi đều sống lại</b>. Kẻ tấn công chỉ cần <b>làm Redis quá tải</b> là mở được cửa. Ta dùng <b>whitelist</b>: không xác minh được thì <b>từ chối</b>.'],
        ['Có 10 microservice thì tốt hơn 3 chứ?',
         '<b>Không nhất thiết</b> — và trả lời “càng nhiều càng tốt” là bẫy. Chia nhỏ quá thì mọi lời gọi hàm trở thành <b>lời gọi mạng</b> (chậm hơn, hỏng được, phải retry). Tiêu chí đúng là <b>bounded context</b>: cái gì <b>thay đổi cùng nhau</b> thì ở <b>cùng một chỗ</b>. Ta có 10 vì có 10 miền dữ liệu tách bạch — và ta <b>nói được ranh giới nào mỏng nhất</b> (location).']
      ]
    },

    {
      id: 'x-limits',
      title: 'Hạn chế — tự nêu trước khi bị hỏi',
      eyebrow: ['chiến thuật'],
      lede: 'Tự nêu hạn chế <b>không</b> mất điểm — nó chứng minh mình hiểu <b>giới hạn của chính hệ thống mình</b>. Bị hỏi mới lúng túng thì <b>tệ hơn nhiều</b>.',
      html: [
        '<div class="tablewrap"><table style="min-width:560px">',
        '<thead><tr><th>Hạn chế</th><th>Ai nêu</th><th>Nêu kèm hướng khắc phục</th></tr></thead><tbody>',
        '<tr><td>Trace <b>chưa nối qua biên Kafka</b> — span consumer là span gốc mới</td>',
        '<td style="color:var(--owner-van)">Văn</td><td>Truyền <code>traceparent</code> qua header của Kafka message</td></tr>',
        '<tr><td><b>Chưa có service discovery động</b> (Consul/etcd)</td>',
        '<td style="color:var(--owner-van)">Văn</td><td>Nhưng <b>mọi</b> địa chỉ đã config-driven ⇒ sang K8s chỉ đổi DNS trong YAML, <b>không sửa code</b></td></tr>',
        '<tr><td><b>user-service là SPOF</b></td>',
        '<td style="color:var(--owner-duc)">Đức</td><td>Đánh đổi có ý thức: một nguồn sự thật RBAC. Giảm nhẹ bằng cache 5 phút</td></tr>',
        '<tr><td><b>Chưa có Kubernetes</b></td>',
        '<td style="color:var(--owner-duc)">Đức</td><td>Nhưng auto-migrate dùng <b>advisory lock</b> ⇒ nhiều replica start song song không đua migration</td></tr>',
        '<tr><td>Một instance Postgres chung (10 DB logic tách biệt)</td>',
        '<td style="color:var(--owner-duc)">Đức</td><td>Giới hạn của máy đồ án. Tách instance thật chỉ là đổi host trong config</td></tr>',
        '<tr><td><b><code>processed_events</code> chưa có cơ chế dọn</b></td>',
        '<td style="color:var(--owner-nam)">Nam</td><td>Nên xoá dòng cũ hơn cửa sổ giao lại tối đa (ví dụ 7 ngày)</td></tr>',
        '<tr><td><b>Chưa có schema registry</b> cho sự kiện</td>',
        '<td style="color:var(--owner-nam)">Nam</td><td>Envelope có <code>version</code> là đủ ở quy mô này; registry (Avro/Protobuf) sẽ <b>ép</b> kiểm tra tương thích lúc publish</td></tr>',
        '<tr><td><b>RabbitMQ có code nhưng 0 call site</b></td>',
        '<td>cả nhóm</td><td>Khai đúng: <b>đã hiện thực, chưa wire</b>. <b>Đừng ai nhận bừa</b> là có dùng</td></tr>',
        '</tbody></table></div>'
      ].join('')
    }
  ]
};
