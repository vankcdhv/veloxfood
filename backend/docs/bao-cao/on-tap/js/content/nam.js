/* Nam — bên nhận & biên client. Cơ chế NHẬN VỀ. */
VF.content = VF.content || {};
VF.content.nam = {
  label: 'Nam',
  tally: 6,
  accent: 'var(--owner-nam)',
  blurb: 'Bên nhận &amp; biên client — cơ chế <b>nhận về</b>: khử trùng lặp, dựng read model, versioning sự kiện, và điều phối đồng thời phía trình duyệt.',
  cards: [

    {
      id: 'nam-mq-flow',
      title: 'Luồng nghiệp vụ chạy trên hàng đợi',
      eyebrow: ['Bảng 7 · #3', 'yêu cầu BẮT BUỘC'],
      lede: 'Đây là <b>yêu cầu bắt buộc</b> của đề bài, và cũng là chỗ dễ trả lời hời hợt nhất. Đừng chỉ nói “em có Kafka”. Phải chỉ ra <b>giá trị</b> mà hàng đợi mang lại.',
      say: [
        '“Luồng minh hoạ: tài xế bấm <b>‘Đã giao’</b>. Order phát <b>một</b> sự kiện <code>order.delivered</code>. Rồi <b>bốn</b> service phản ứng, <b>độc lập với nhau</b>: <b>Payment</b> ghi sổ tiền mặt COD · <b>Order</b> tự chuyển sang COMPLETED · <b>Review</b> mở khoá cho khách đánh giá · <b>Notification</b> báo khách.”',
        '“<b>Điểm cốt lõi: không service nào gọi service nào.</b> Order <b>không hề biết</b> Review tồn tại. Ngày mai muốn thêm một service tính điểm thưởng, chỉ cần <b>subscribe thêm</b> — <b>không sửa một dòng nào</b> trong Order.”',
        '“Và <b>chịu lỗi từng phần</b>: Notification chết thì Payment <b>vẫn</b> ghi sổ, Review <b>vẫn</b> mở khoá. Sự kiện nằm chờ trong Kafka; Notification sống lại là bắt kịp từ offset cũ. Nếu là gọi đồng bộ thì <b>một service chết kéo cả chuỗi chết</b>.”',
        '“Toàn hệ: <b>9 topic, 21 consumer group</b>. Notification consume <b>7 trên 9</b> topic — thiếu <code>user.events</code> và <code>store.events</code>.”'
      ],
      probes: [
        ['Vì sao không gọi gRPC cho nhanh?', 'Vì gọi đồng bộ tạo ra <b>ràng buộc thời gian</b>: Order phải <b>chờ</b> cả bốn service trả lời mới xong việc. Bốn service ⇒ bốn cơ hội hỏng ⇒ độ trễ cộng dồn. Và Order phải <b>biết tên</b> cả bốn. Hàng đợi cắt đứt cả hai ràng buộc: Order chỉ <b>công bố một sự thật</b> (“đơn đã giao”), ai quan tâm thì tự xử.'],
        ['Nếu Payment xử lý sự kiện đó bị lỗi thì sao?', 'Nó <b>retry 3 lần</b>, rồi vào <b>DLQ</b> — cơ chế đó là phần của Văn. Điều quan trọng ở đây: <b>lỗi của Payment không ảnh hưởng Review hay Notification</b>. Mỗi consumer group có <b>offset riêng</b>, chúng <b>hoàn toàn độc lập</b>.'],
        ['Sao lại là "21 consumer group"?', 'Vì <b>mỗi cặp (service, topic) là một consumer group riêng</b> — ví dụ <code>notification-service-order</code>, <code>reporting-service-order</code>. Nhờ vậy Notification và Reporting cùng đọc <code>order.events</code> mà <b>không tranh nhau message</b>: mỗi group nhận <b>đầy đủ</b> mọi message, giữ offset riêng. Đó là khác biệt giữa <b>pub/sub</b> và <b>hàng đợi công việc</b>.']
      ],
      srcs: ['services/notification/**', 'services/*/internal/handler/event/**']
    },

    {
      id: 'nam-idempotent',
      title: 'At-least-once + idempotent receiver = exactly-once',
      eyebrow: ['vượt khung', 'Demo 3', 'câu để chen vào Bảng 8'],
      lede: 'Câu hỏi bẫy kinh điển: <i>“Kafka có exactly-once không?”</i>. Trả lời “có” là <b>mất điểm</b>. Đây là chỗ Nam ghi điểm mạnh nhất.',
      scene: 'idempotent-receiver',
      say: [
        '“<b>Kafka không cho exactly-once xuyên biên service.</b> Nó giao <b>ít nhất một lần</b> — at-least-once. Nguyên nhân đơn giản: consumer xử lý xong rồi chết <b>trước khi kịp commit offset</b>, thì message đó <b>sẽ được giao lại</b>. Và outbox dispatcher cũng có thể phát trùng vì đúng lý do đó.”',
        '“Nên chúng em <b>không cố ép Kafka</b> làm điều nó không làm được. Thay vào đó, chúng em làm cho <b>hiệu ứng nghiệp vụ</b> đúng một lần: bảng <code>processed_events</code>, khoá chính là <code>event_id</code>, ghi trong <b>CÙNG transaction</b> với business write. Giao lại lần hai ⇒ trùng khoá ⇒ bỏ qua.”',
        '“<b>Demo 3</b>: em tua offset consumer về 0, ép Kafka giao lại <b>toàn bộ 8 sự kiện</b>. Số thông báo vẫn là <b>8</b> — <b>không phải 16</b>.”'
      ],
      probes: [
        ['Vì sao phải cùng transaction? Ghi processed_events trước có được không?', '<b>Không.</b> Ghi trước rồi crash ⇒ đánh dấu “đã xử lý” nhưng <b>thông báo chưa hề được tạo</b> ⇒ event đó <b>mất vĩnh viễn</b>, và retry cũng bị chặn. Ghi sau rồi crash ⇒ thông báo tạo rồi nhưng chưa đánh dấu ⇒ giao lại ⇒ <b>nhân đôi</b>. <b>Chỉ có chung một transaction mới đúng.</b>'],
        ['Kafka quảng cáo có exactly-once mà?', 'Có — nhưng chỉ <b>trong phạm vi Kafka</b> (transaction giữa topic với topic, đọc-xử lý-ghi khép kín trong Kafka Streams). Còn khi consumer phải <b>ghi vào Postgres</b> — một hệ thống <b>bên ngoài</b> — thì Kafka <b>không thể</b> đảm bảo hai bên cùng commit. Đó là <b>vấn đề two-generals</b>, và cách duy nhất là <b>phía nhận tự idempotent</b>.'],
        ['Bảng processed_events có phình mãi không?', '<b>Có</b>, và em nói thẳng: hiện <b>chưa dọn</b>. Đúng ra nên xoá các dòng cũ hơn khoảng thời gian giao lại tối đa (ví dụ giữ 7 ngày). Nếu không, sau vài triệu event thì bảng thành gánh nặng. <b>Đây là hạn chế em biết và chưa xử lý.</b>'],
        ['Cái này khác gì barrier của saga?', 'Cùng <b>nguyên lý</b> — khử trùng lặp bằng khoá chính, ghi cùng transaction. Khác <b>ngữ cảnh</b>: barrier bảo vệ <b>nhánh saga đồng bộ</b> và còn phải xử lý cả chuyện <b>bù trừ đến trước action</b>. <code>processed_events</code> bảo vệ <b>consumer bất đồng bộ</b>, chỉ có một chiều.']
      ],
      srcs: ['bảng processed_events (7 service)', 'services/*/internal/handler/event/**']
    },

    {
      id: 'nam-versioning',
      title: 'Versioning sự kiện',
      eyebrow: ['vượt khung'],
      lede: 'Sự kiện là một <b>hợp đồng công khai</b> giữa các service. Đổi nó mà không nghĩ là <b>phá hỏng consumer của người khác</b>.',
      say: [
        '“Envelope sự kiện mang trường <code>version</code>. Consumer đọc version để biết cách giải mã.”',
        '“<b>Chỗ đáng nói:</b> envelope <b>cũ</b> — loại chưa hề có trường <code>version</code> — <b>vẫn decode được</b>. Nghĩa là có thể <b>tiến hoá schema sự kiện</b> mà <b>không phá vỡ</b> consumer đang chạy và không phải deploy đồng loạt toàn bộ 10 service cùng lúc. Có <b>test riêng</b> chứng minh (<code>envelope_version_test.go</code>).”',
        '“Vì sao quan trọng: producer và consumer <b>deploy độc lập</b>. Sẽ luôn có một khoảng thời gian producer đã lên version mới còn consumer vẫn ở bản cũ — và ngược lại. Hệ thống <b>phải sống được</b> trong khoảng đó.”'
      ],
      probes: [
        ['Đổi schema thì làm sao cho an toàn?', 'Nguyên tắc: <b>chỉ thêm, không xoá, không đổi nghĩa</b>. Thêm trường mới thì consumer cũ <b>bỏ qua</b> — vẫn chạy. Xoá hay đổi nghĩa một trường thì consumer cũ <b>hiểu sai</b> — vỡ. Muốn xoá thì phải qua <b>hai nhịp deploy</b>: nhịp một cho consumer thôi dùng trường đó, nhịp hai mới bỏ khỏi producer.'],
        ['Sao không dùng schema registry (Avro/Protobuf)?', 'Đó là cách <b>chuẩn công nghiệp</b> — registry <b>ép</b> kiểm tra tương thích ngay lúc publish, không dựa vào kỷ luật của người viết code. Ở quy mô đồ án thì envelope có version là đủ, và em <b>biết</b> hướng nâng cấp là gì. Nói được điều này là chứng minh mình hiểu <b>vấn đề</b>, không chỉ hiểu <b>giải pháp mình đã làm</b>.']
      ],
      srcs: ['pkg/outbox/outbox.go (EnvelopeVersion)', 'envelope_version_test.go']
    },

    {
      id: 'nam-cqrs',
      title: 'CQRS read model',
      eyebrow: ['§4, §8', 'vượt khung', 'Demo 6'],
      lede: '<code>reporting-service</code> <b>không có một API ghi nghiệp vụ nào</b>. Toàn bộ trạng thái của nó dựng <b>100% từ sự kiện</b>.',
      scene: 'cqrs-read-model',
      say: [
        '“CQRS là <b>tách đường ghi khỏi đường đọc</b>. Đường ghi (order, payment…) tối ưu cho <b>ghi nhanh, đúng</b>. Đường đọc (reporting) tối ưu cho <b>truy vấn tổng hợp</b> — doanh thu, top món, biểu đồ.”',
        '“Nếu gộp lại — tính báo cáo <b>ngay trong</b> đường ghi — thì <b>mọi khách đặt hàng</b> đều phải trả giá cho việc <b>admin thỉnh thoảng mở dashboard</b>. Các truy vấn tổng hợp quét cả bảng và <b>chỉ chậm dần</b> khi dữ liệu lớn lên.”',
        '“Cái giá: <b>nhất quán cuối</b>. Đơn vừa đặt thì dashboard <b>chưa thấy ngay</b> — trễ khoảng <b>vài giây</b>. Chúng em <b>chấp nhận có ý thức</b>: báo cáo trễ 2 giây <b>không ai chết</b>, còn đặt hàng chậm 1,4 giây thì khách bỏ đi.”',
        '“Và một hệ quả đẹp: nếu read model hỏng, chỉ cần <b>replay lại event</b> từ đầu là <b>dựng lại được toàn bộ</b>.”'
      ],
      probes: [
        ['Reporting mất dữ liệu thì làm sao khôi phục?', '<b>Replay event từ Kafka.</b> Vì read model là <b>hàm thuần</b> của dòng sự kiện — cùng một chuỗi event thì luôn cho ra cùng một trạng thái. Đây là ưu điểm lớn nhất của kiến trúc hướng sự kiện, và nó <b>chỉ đúng khi consumer idempotent</b> — nối thẳng sang thẻ trên.'],
        ['Vài giây trễ, admin có phàn nàn không?', 'Với báo cáo doanh thu thì không — <b>không ai ra quyết định dựa trên số liệu của 2 giây trước</b>. Nếu là màn hình cần realtime (ví dụ theo dõi đơn đang giao) thì <b>không dùng CQRS</b> — phải đọc thẳng từ đường ghi. <b>Chọn đúng công cụ cho đúng bài toán.</b>'],
        ['Choreography khác orchestration chỗ nào?', 'Reporting và Notification <b>tự phản ứng với sự kiện, không ai gọi chúng</b> — đó là <b>choreography</b>: điều phối <b>phi tập trung</b>, không có nhạc trưởng. Ngược lại, saga đặt hàng là <b>orchestration</b>: Order (qua DTM) <b>chỉ huy từng bước</b> và biết phải bù trừ cái gì. Hệ thống dùng <b>cả hai</b>, mỗi cái cho đúng việc của nó — và đó là câu trả lời tốt.']
      ],
      srcs: ['services/reporting/** (không có API ghi)']
    },

    {
      id: 'nam-single-flight',
      title: 'Single-flight refresh — đồng bộ ở biên client',
      eyebrow: ['§10', 'vượt khung'],
      lede: 'Bài toán <b>điều phối đồng thời</b> — chỉ khác là nó nằm trong <b>trình duyệt</b>, không phải trong server. Đây là chỗ chứng minh frontend cũng là hệ phân tán.',
      scene: 'single-flight-refresh',
      say: [
        '“Refresh token là <b>single-use</b>: dùng một lần là bị thu hồi ngay. Đó là <b>tính năng an ninh</b> — nó chặn replay.”',
        '“Nhưng nó tạo ra một cái bẫy ở phía client: khi access token hết hạn, <b>N request đang bay cùng nhận <code>401</code></b> <b>cùng lúc</b>. Nếu mỗi request tự gọi <code>/auth/refresh</code>, cả N cùng mang <b>một</b> refresh token. Cái đầu tiên thắng và <b>thu hồi</b> token đó ⇒ N−1 cái còn lại bị coi là <b>replay attack</b> ⇒ <b>đăng xuất sạch</b>. Người dùng bị văng ra <b>không vì lý do gì cả</b>.”',
        '“Chữa bằng <b>single-flight</b>: lớp <code>auth-refresh.ts</code> gom tất cả về <b>đúng một</b> lời gọi refresh; các request khác <b>chờ chung một Promise</b> rồi cùng retry bằng token mới.”',
        '“Nói cách khác: <b>tính năng an ninh của server biến thành bug, chỉ vì client thiếu điều phối</b>. Bài toán race condition này <b>giống hệt</b> race condition trong server — chỉ đổi chỗ.”'
      ],
      probes: [
        ['Sao không để refresh token dùng nhiều lần cho gọn?', 'Thì mất <b>khả năng phát hiện replay</b>. Refresh token sống rất lâu (7 ngày) — nếu bị đánh cắp mà <b>dùng lại được nhiều lần</b>, kẻ tấn công dùng thoải mái. Với single-use, nạn nhân và kẻ tấn công <b>sẽ va nhau</b>: một trong hai gặp token đã bị thu hồi ⇒ hệ thống <b>phát hiện được</b> và đăng xuất toàn bộ phiên.'],
        ['Frontend còn cơ chế phân tán nào nữa không?', '<b>Optimistic update + rollback</b> ở tính năng yêu thích (<code>use-favorites.ts</code>): UI cập nhật <b>ngay lập tức</b>, nếu server từ chối thì <b>hoàn tác</b>. Về bản chất đó là <b>bù trừ</b> — <b>cùng một ý tưởng với saga</b>, chỉ khác quy mô.']
      ],
      srcs: ['frontend/src/shared/lib/auth-refresh.ts', 'trace-id.ts', 'use-favorites.ts']
    },

    {
      id: 'nam-tests',
      title: 'Kiểm thử luồng liên service',
      eyebrow: ['Bảng 7 · #6', '§12'],
      lede: '<b>21 package, 264 test-case, 0 FAIL, 0 SKIP.</b> Nhưng con số không phải là điểm — <b>test cái gì</b> mới là.',
      say: [
        '“<b>264 test PASS, 0 FAIL.</b> Quan trọng hơn con số: ba bộ test <b>chứng minh trực tiếp các cơ chế phân tán</b>.”',
        '“<code>place_order_compensation_test.go</code> — <b>inject lỗi vào từng bước saga</b> rồi kiểm tra bù trừ có chạy đủ không. Đây là test <b>khó viết nhất</b>, vì phải giả lập một service hỏng <b>đúng lúc</b>.”',
        '“<code>refresh_logout_test.go</code> — <b>replay refresh token</b> đã dùng, kiểm tra hệ thống trả <code>401</code>. Test một <b>thuộc tính an ninh</b>, không phải một tính năng.”',
        '“<code>envelope_version_test.go</code> — envelope <b>cũ, chưa có version</b>, vẫn decode được. Test <b>khả năng tương thích ngược</b>.”'
      ],
      probes: [
        ['Test saga mà không cần dựng cả hệ thống?', 'Được, vì các gateway là <b>interface</b> (<code>PaymentGateway</code>, <code>PromotionGateway</code>). Test <b>inject stub</b> có thể ném lỗi <b>theo ý muốn</b>, đúng ở bước mình chọn. Đó chính là <b>lý do</b> phải tách interface — không phải để cho “sạch kiến trúc”, mà để <b>test được các đường hỏng</b>.'],
        ['Có integration test thật không hay chỉ mock?', '<b>Có thật.</b> Chúng dùng <code>pkg/testutil</code> để tạo database <code>&lt;svc&gt;_db_test</code> riêng, auto-migrate, truncate giữa các test. Chạy thật với Postgres và Redis — không phải mock. (Chi tiết đáng nhớ: trước đây <code>go test</code> trả về “ok” từ <b>cache</b> và <b>che mất</b> việc test đang bị SKIP. Phải chạy <code>-count=1</code> mới ra kết quả thật.)']
      ],
      srcs: ['21 package · 264 test-case', 'pkg/testutil/db.go']
    }
  ]
};
