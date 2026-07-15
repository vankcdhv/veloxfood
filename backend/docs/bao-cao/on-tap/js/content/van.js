/* Văn — kênh truyền & quan sát. Mọi thứ nằm GIỮA các service. */
VF.content = VF.content || {};
VF.content.van = {
  label: 'Văn',
  tally: 16,
  accent: 'var(--owner-van)',
  blurb: 'Kênh truyền &amp; quan sát — mọi thứ nằm <b>giữa</b> các service: saga, hàng đợi, gRPC client, và toàn bộ tầng nhìn thấy hệ thống.',
  cards: [

    {
      id: 'van-saga-vs-2pc',
      title: 'Vì sao saga, không phải 2PC',
      eyebrow: ['§6', 'nền tảng — hỏi là chắc chắn'],
      lede: 'Đây là câu hỏi mở đầu kinh điển. Trả lời được câu này là đã chứng minh hiểu bản chất giao dịch phân tán; trả lời trượt thì mọi thứ sau đó đều lung lay.',
      scene: 'two-pc-vs-saga',
      say: [
        '“Giao dịch của chúng em trải qua <b>ba database khác nhau</b> — order_db, promotion_db, payment_db. Không có transaction nào của Postgres phủ được cả ba.”',
        '“2PC làm được, nhưng nó là giao thức <b>blocking</b>: trong lúc chờ pha 2, mọi tài nguyên đều <b>bị khoá</b>. Nếu coordinator chết hoặc mạng đứt giữa hai pha, khoá đó <b>treo vô thời hạn</b>. Với hệ phân tán thật, nơi partition là chuyện thường ngày, cái giá đó quá đắt.”',
        '“Saga đổi <b>nhất quán tức thời</b> lấy <b>tính sẵn sàng</b>: mỗi bước commit ngay và nhả khoá ngay; nếu bước sau hỏng thì <b>bù trừ ngược</b> các bước trước. Hệ thống có thể <b>tạm thời</b> không nhất quán, nhưng <b>không bao giờ khoá ai</b>.”'
      ],
      probes: [
        ['Vậy saga có đảm bảo ACID không?', 'Không. Saga <b>hy sinh tính cô lập</b> (Isolation). Có một khoảng ngắn mà voucher đã <code>RESERVED</code> nhưng đơn chưa chắc tồn tại — người khác nhìn vào thấy trạng thái “nửa vời”. Đó là cái giá phải thừa nhận thẳng. Ta bù lại bằng cách <b>rút ngắn khoảng đó</b> (deadline 5s) và <b>đảm bảo cuối cùng luôn hội tụ</b> (bù trừ bền vững + janitor).'],
        ['Sao không dùng transaction phân tán của Postgres (PREPARE TRANSACTION)?', 'Vẫn là 2PC, chỉ đổi tên. Và nó chỉ hoạt động giữa các Postgres — trong khi nhánh <code>Capture</code> có thể phải gọi <b>MoMo</b>, một hệ thống bên ngoài hoàn toàn không biết PREPARE TRANSACTION là gì.'],
        ['Orchestration hay choreography?', '<b>Orchestration</b> — có một bên chỉ huy (DTM, do Order khởi xướng) gọi từng nhánh theo thứ tự và biết phải bù trừ cái gì. Choreography (mỗi service tự phản ứng với event) thì ta <b>cũng có</b>, nhưng dùng cho luồng sau khi đặt hàng thành công — phần đó là của Nam.']
      ],
      srcs: ['pkg/saga/saga.go', 'services/order/internal/usecase/place_order_dtm.go', 'config/order.yaml:38']
    },

    {
      id: 'van-saga-compensation',
      title: 'Saga đặt hàng và chuỗi bù trừ ngược',
      eyebrow: ['§6', 'Bảng 8 · dòng 4', 'Demo 1'],
      lede: 'Ba nhánh, mỗi nhánh một cặp action / compensate. Khi một nhánh hỏng, DTM chạy bù trừ <b>ngược thứ tự</b> qua đúng những nhánh đã thành công.',
      scene: 'saga-compensation',
      say: [
        '“Saga đặt hàng có <b>ba nhánh</b>: ApplyPromotion / ReleaseUsage · Capture / Refund · ConfirmUsage / ReleaseUsage. Order <b>không tự gọi</b> ai — nó nộp saga cho DTM, và DTM gọi từng nhánh.”',
        '“Nếu nhánh 2 hỏng, DTM chạy bù trừ <b>ngược</b>: Refund trước, rồi ReleaseUsage. Kết quả: ví không đổi, voucher không tăng lượt dùng, không đẻ ra đơn nào, sổ cái không có bút toán mới.”',
        '“<b>Chi tiết em muốn thầy chú ý:</b> compensate của nhánh 2 — <code>Refund</code> — trả về <b>succeed</b> dù <b>chưa hề trừ đồng nào</b>. Đó là <b>null compensation</b>: bù trừ cho một hành động chưa từng chạy. Nó <b>bắt buộc phải là no-op idempotent</b>, không được báo lỗi — vì nếu nó báo lỗi, DTM sẽ retry mãi và saga treo vĩnh viễn.”'
      ],
      probes: [
        ['Store có phải một nhánh saga không?', '<b>Không.</b> Store chỉ được <b>đọc</b> (validate quán còn mở, tính phí ship) trước khi mở saga — thao tác <b>read-only</b> thì không cần bù trừ. Chỉ những bước <b>làm thay đổi trạng thái</b> mới thành nhánh.'],
        ['Bù trừ có thể hỏng không? Thì sao?', 'Có, và đây là chỗ hệ thống đi xa hơn saga sách vở. Ý định bù trừ được ghi <b>bền</b> vào bảng <code>pending_compensations</code>; một worker quét <b>mỗi 30 giây</b>, gọi lại tối đa <b>20 lần</b>. Vì lời gọi bù trừ <b>idempotent</b> (khoá theo order-id), retry không gây tác dụng phụ. Xem thẻ “Bù trừ bền vững”.'],
        ['Đơn COD thì Capture làm gì?', 'Bỏ qua — không trừ tiền lúc đặt. Nhưng nhánh <b>vẫn tồn tại</b> trong saga, và compensate của nó vẫn chạy khi rollback. Đây chính là ca <b>null compensation trong đời thật</b>, không phải tình huống giả định: <code>order.cancelled</code> của đơn COD gọi <code>Refund</code> cho một payment <b>chưa bao giờ tồn tại</b>.'],
        ['Saga huỷ đơn có giống saga đặt đơn không?', 'Không — nó là kiểu <b>khác</b>. Đặt đơn là <b>compensating saga</b> (hỏng thì lùi). Huỷ đơn là <b>idempotent-retry saga</b> (forward-only): không có đường lùi, chỉ có <b>tiến tới cho bằng được</b>, retry đến khi thành công. Vì bạn <b>không thể “bù trừ” một cú huỷ đơn</b> — khách đã bấm huỷ rồi.']
      ],
      srcs: ['services/order/internal/usecase/place_order_dtm.go', 'services/order/internal/usecase/order_cancel_dtm.go', 'services/payment/internal/usecase/refund_usecase.go']
    },

    {
      id: 'van-barrier',
      title: 'Sub-transaction barrier',
      eyebrow: ['§6', 'vượt khung', 'cơ chế khó nhất'],
      lede: 'Cơ chế tinh vi nhất trong cả hệ thống, và cũng là thứ dễ ghi điểm nhất — vì rất ít đồ án sinh viên có nó. Nó giải <b>hai</b> bài toán mà retry sinh ra.',
      scene: 'sub-transaction-barrier',
      say: [
        '“Trong hệ phân tán, <b>không thể phân biệt</b> ‘service chết’ với ‘service làm xong nhưng trả lời bị mất’. Nên DTM <b>bắt buộc phải retry</b>. Mà retry thì sinh ra hai bệnh.”',
        '“Bệnh một — <b>gọi trùng nhánh</b>: <code>Capture</code> chạy hai lần, khách bị trừ tiền hai lần. Bệnh hai — <b>bù trừ đến trước</b>: <code>Refund</code> tới trước <code>Capture</code>, rồi <code>Capture</code> lết tới sau và trừ tiền của một đơn đã huỷ, không ai hoàn lại.”',
        '“Barrier chữa cả hai bằng <b>một</b> ý tưởng: mỗi nhánh, trước khi làm việc, <b>ghi một dòng chặn</b> <code>(gid, branch, op)</code> vào bảng <code>dtm_barrier</code> — trong <b>CÙNG transaction</b> với thao tác nghiệp vụ. Khoá chính trùng ⇒ biết là đã chạy rồi ⇒ bỏ qua. Thấy dòng <code>compensate</code> có trước ⇒ biết action đến muộn ⇒ từ chối.”'
      ],
      probes: [
        ['Vì sao phải CÙNG transaction? Ghi riêng hai lệnh có được không?', '<b>Không.</b> Nếu ghi barrier ở transaction riêng, sẽ có khe hở: ghi barrier xong → crash → tiền chưa trừ nhưng barrier nói “đã trừ rồi” ⇒ retry sẽ bị chặn ⇒ <b>đơn mất tiền vĩnh viễn</b>. Ghi chung một transaction thì <b>cả hai cùng sống hoặc cùng chết</b> — không có trạng thái nửa vời. Đây là toàn bộ lý do cơ chế này hoạt động.'],
        ['Barrier khác gì bảng processed_events của Nam?', 'Cùng <b>nguyên lý</b> (khử trùng lặp bằng khoá chính, ghi cùng transaction), khác <b>ngữ cảnh</b>. Barrier bảo vệ <b>nhánh saga</b> (gọi gRPC đồng bộ, có cả action lẫn compensate, nên còn phải xử lý chuyện <b>thứ tự đảo ngược</b>). <code>processed_events</code> bảo vệ <b>consumer Kafka</b> (bất đồng bộ, chỉ có một chiều). Nói được cả hai giống và khác chỗ nào là ghi điểm.'],
        ['Ai ghi bảng dtm_barrier — DTM hay service?', '<b>Service tự ghi</b>, vào <b>chính database của mình</b> (payment_db, promotion_db). DTM không đụng vào DB nghiệp vụ. Đó là điều bắt buộc — vì chỉ khi barrier nằm cùng DB với dữ liệu nghiệp vụ thì mới ghi chung một transaction được.']
      ],
      srcs: ['pkg/saga/barrier.go', 'bảng dtm_barrier trong payment_db & promotion_db']
    },

    {
      id: 'van-durable-compensation',
      title: 'Khi chính bước bù trừ cũng hỏng',
      eyebrow: ['§9', 'vượt khung ×3'],
      lede: 'Saga trong sách dừng ở “hỏng thì bù trừ”. Câu hỏi bỏ ngỏ: <b>nếu bù trừ cũng hỏng thì sao?</b> Hệ thống trả lời câu đó bằng ba cơ chế khác nhau.',
      say: [
        '“Bù trừ cũng là một lời gọi mạng — nó cũng hỏng được. Nếu bỏ qua thì tiền của khách mắc kẹt vĩnh viễn. Nên chúng em có <b>ba tầng</b>.”',
        '“<b>Bù trừ bền vững:</b> ý định rollback được ghi <b>bền</b> vào bảng <code>pending_compensations</code>. Một worker quét <b>mỗi 30 giây</b>, gọi lại, tối đa <b>20 lần</b>. Vì lời gọi idempotent (khoá theo order-id) nên retry an toàn.”',
        '“<b>Bù trừ theo thời gian:</b> nếu saga chết hẳn giữa chừng, nó để lại voucher <code>RESERVED</code> mồ côi — không ai bù trừ nữa. Một janitor void chúng sau <b>TTL 15 phút</b> và trả lại <code>used_count</code>. Không có nó thì hạn mức voucher <b>rò rỉ vĩnh viễn</b>.”',
        '“<b>Bù trừ một bước đã commit:</b> huỷ một đơn <b>đã đặt thành công</b> vẫn void được usage ở trạng thái <code>CONFIRMED</code> — khác hẳn với việc chỉ nhả một chỗ giữ tạm.”'
      ],
      probes: [
        ['Sao lại 20 lần? Sao không retry vô hạn?', 'Vì có những lỗi <b>không bao giờ tự khỏi</b> (dữ liệu hỏng, hợp đồng API đổi). Retry vô hạn cho những ca đó chỉ đốt tài nguyên và <b>giấu sự cố</b>. Hết 20 lần thì nó <b>đứng lại và ồn ào</b> để con người vào xem — thà báo động còn hơn im lặng thử mãi.'],
        ['Vì sao TTL 15 phút, không phải 1 phút?', 'Vì saga bình thường chỉ mất <b>vài trăm mili-giây</b>. 15 phút là biên <b>rất rộng</b> — đủ để chắc chắn saga đó <b>thật sự đã chết</b> chứ không phải đang chậm. Void nhầm một voucher đang được dùng hợp lệ thì tệ hơn nhiều so với việc giữ chỗ thừa thêm vài phút.'],
        ['Worker chạy ở đâu? Nhiều replica thì có đua nhau không?', 'Worker nằm trong chính <code>order-service</code> (<code>wire_compensation_worker.go</code>). Nhiều replica thì mỗi lần quét đều lấy bản ghi <b>có khoá</b>, và bản thân lời gọi bù trừ <b>idempotent</b> — nên kể cả hai replica cùng gọi, kết quả vẫn đúng một lần.']
      ],
      srcs: ['services/order/cmd/wire_compensation_worker.go', 'services/promotion/cmd/wire_reservation_janitor.go', 'bảng pending_compensations']
    },

    {
      id: 'van-outbox',
      title: 'Transactional Outbox',
      eyebrow: ['§7', 'Bảng 8 · dòng 4'],
      lede: 'Bài toán: ghi DB và phát Kafka là <b>hai hệ thống khác nhau</b>, không có transaction chung. Crash đúng khe giữa hai thao tác là mất sự kiện vĩnh viễn.',
      scene: 'transactional-outbox',
      say: [
        '“Không thể ghi Postgres và publish Kafka trong <b>một</b> transaction — chúng là hai hệ thống khác nhau. Nếu ghi đơn xong, commit, rồi mới publish, mà process chết đúng khe đó, thì <b>đơn tồn tại nhưng sự kiện không bao giờ được phát</b>: khách không được báo, doanh thu không được ghi nhận, đơn không được giao. Hệ thống lệch nhau <b>vĩnh viễn</b> và không ai biết.”',
        '“Outbox lật ngược vấn đề: sự kiện được ghi thành <b>một dòng trong cùng database</b>, trong <b>cùng transaction</b> với đơn hàng. Rồi một dispatcher chạy nền (mỗi 2 giây) quét bảng đó và phát lên Kafka.”',
        '“Đánh đổi: dispatcher có thể phát <b>trùng</b> — nếu nó publish xong rồi chết trước khi kịp đánh dấu <code>SENT</code>. Nên đây <b>chính xác</b> là <b>at-least-once</b>, và phía nhận <b>bắt buộc</b> phải idempotent. Đó là lý do phần của Nam tồn tại.”'
      ],
      probes: [
        ['Vì sao không publish Kafka trước rồi mới ghi DB?', 'Còn tệ hơn: publish xong rồi ghi DB hỏng ⇒ <b>sự kiện nói về một đơn không tồn tại</b>. Notification báo khách “đơn đã đặt” trong khi <b>không có đơn nào</b>. Mất sự kiện còn cứu được (phát lại); <b>bịa</b> ra sự kiện thì không.'],
        ['Dispatcher chết thì sao?', 'Sự kiện vẫn nằm nguyên trong bảng <code>outbox</code> ở trạng thái <code>PENDING</code>. Dispatcher sống lại là quét thấy và phát tiếp. <b>Không mất gì</b> — chỉ trễ.'],
        ['Có bị trễ không? 2 giây là nhiều đấy.', 'Có, và đó là <b>đánh đổi có ý thức</b>: outbox theo kiểu polling thì độ trễ = chu kỳ quét. Muốn tức thì thì phải dùng CDC (đọc WAL của Postgres, ví dụ Debezium) — phức tạp hơn nhiều. Với đồ án này, 2 giây là quá đủ, và em <b>biết</b> hướng nâng cấp là gì.']
      ],
      srcs: ['pkg/outbox/', 'bảng outbox trong mỗi service']
    },

    {
      id: 'van-kafka-dlq',
      title: 'Retry, DLQ, và bảo toàn thứ tự partition',
      eyebrow: ['Bảng 7 · #5', 'Bảng 8 · dòng 8', 'vượt khung'],
      lede: 'Consumer retry 3 lần rồi park vào dead-letter. Nhưng chi tiết đáng giá nhất nằm ở chỗ <b>khi DLQ cũng chết</b> — lúc đó hệ thống <b>cố ý</b> chọn tắc nghẽn.',
      scene: 'kafka-retry-dlq',
      say: [
        '“Consumer retry <b>3 lần</b>, backoff <b>0,5 → 1 → 2 giây</b>. Hết retry thì message được park sang <code><topic>.dlq</code>, kèm header truy vết: topic gốc, partition, offset, lỗi, thời điểm. Rồi mới commit offset — nên một message độc <b>không làm tắc cả partition</b>.”',
        '“<b>Chỗ em muốn nói kỹ:</b> nếu <b>vừa không xử lý được, vừa không park được sang DLQ</b> — thì consumer <b>cố ý KHÔNG commit</b> và <b>dừng lại ngay tại message đó</b>.”',
        '“Vì sao không bỏ qua để chạy tiếp? Vì Kafka đảm bảo <b>thứ tự trong partition</b>, và các sự kiện của <b>cùng một đơn</b> nằm <b>cùng một partition</b>. Bỏ qua <code>order.status_changed</code> để xử lý <code>order.cancelled</code> trước là <b>phá vỡ nhân quả</b>. Thà tắc nghẽn và ồn ào, còn hơn chạy tiếp và <b>sai âm thầm</b>.”'
      ],
      probes: [
        ['Vì sao 3 lần retry, không phải 10?', 'Vì retry chỉ chữa được lỗi <b>tạm thời</b> (DB nghẽn, mạng chớp). Lỗi <b>vĩnh viễn</b> (JSON hỏng, dữ liệu sai) thì retry 10 lần cũng vô ích, chỉ làm <b>chậm cả partition</b> phía sau. 3 lần là đủ vượt qua các nhịp trục trặc ngắn, còn lại thì cách ly ngay vào DLQ.'],
        ['Message vào DLQ rồi thì làm gì?', 'Nó nằm đó, <b>không mất</b>, kèm đủ ngữ cảnh để điều tra (<code>x-original-topic</code>, <code>x-original-offset</code>, <code>x-error</code>). Sau khi sửa nguyên nhân, có thể phát lại từ DLQ về topic gốc. Ở quy mô đồ án, ta chưa dựng công cụ replay tự động — <b>và em nói thẳng là chưa có</b>.'],
        ['Tại sao lại quan tâm thứ tự? Kafka có đảm bảo thứ tự toàn cục không?', '<b>Không</b> — Kafka chỉ đảm bảo thứ tự <b>trong một partition</b>. Toàn cục thì không. Đó là lý do phải chọn <b>khoá partition</b> cẩn thận: các sự kiện cùng một đơn hàng dùng cùng khoá ⇒ vào cùng partition ⇒ được xử lý đúng thứ tự. Đây là <b>quyết định thiết kế</b>, không phải mặc định.']
      ],
      srcs: ['pkg/messaging/kafka/']
    },

    {
      id: 'van-grpc-breaker',
      title: 'gRPC client: deadline, retry, circuit breaker',
      eyebrow: ['Bảng 8 · dòng 12', 'Demo 2'],
      lede: 'Mọi lời gọi gRPC đi qua <code>pkg/grpcx.Dial</code>, với chuỗi bảo vệ: trace → <b>breaker</b> → <b>deadline 5s</b> → <b>retry ×2</b>. Số đo thật: <b>5.055ms → 25ms</b>.',
      scene: 'circuit-breaker',
      say: [
        '“Breaker mở khi thoả <b>cả hai</b> điều kiện: <b>≥5 request trong 30 giây</b> <b>VÀ</b> <b>≥60% lỗi hạ tầng</b>. Mở rồi thì request bị <b>từ chối ngay tại client</b>, không chạm tới service chết nữa. Sau <b>15 giây</b> nó chuyển <b>half-open</b>, cho <b>3 request thăm dò</b>; qua hết thì đóng lại.”',
        '“<b>Chi tiết phân biệt người hiểu và người copy:</b> breaker <b>chỉ trip trên lỗi hạ tầng</b> — <code>Unavailable</code>, <code>DeadlineExceeded</code>. Lỗi <b>nghiệp vụ</b> như <code>NotFound</code> hay <code>InvalidArgument</code> thì <b>không</b>. Vì service lúc đó vẫn <b>hoàn toàn khoẻ mạnh</b> — nó chỉ đang từ chối <b>đúng</b>. Nếu để lỗi nghiệp vụ trip breaker, thì chỉ cần 5 khách tra nhầm mã voucher là cả hệ thống ngắt kết nối tới promotion-service.”',
        '“Và deadline <b>5 giây</b> có lý do: participant treo thì phải <b>fail-fast để saga kịp bù trừ</b>, chứ không treo cả chuỗi. Số đo thật: request đầu tiên mất <b>đúng 5,055 giây</b> — đó là deadline cắt, không phải TCP treo vô hạn.”'
      ],
      probes: [
        ['Demo breaker thì dừng service nào?', '<b>store-service.</b> KHÔNG phải payment. Vì với engine DTM, <b>order-service không gọi thẳng payment</b> — <b>DTM server</b> mới là bên gọi participant. Lời gọi gRPC <b>trực tiếp</b> của order là <code>GetStoreForOrder</code> sang store. Dừng nhầm payment thì breaker <b>không trip</b> và demo trông như hỏng.'],
        ['Vì sao breaker giúp ích, nếu request vẫn hỏng?', 'Nó <b>không cứu</b> request đó. Nó làm hai việc khác: (1) <b>trả lời thất bại trong 25ms thay vì 5 giây</b> — khách không phải ngồi chờ; (2) <b>ngừng đập vào một service đang hấp hối</b>, cho nó cơ hội hồi phục. Không có breaker, service chết sẽ bị hàng nghìn request dồn vào và <b>không bao giờ đứng dậy nổi</b> — đó là <b>cascading failure</b>.'],
        ['Retry ×2 có làm hỏng thêm không?', 'Có nguy cơ, nên retry <b>chỉ trên <code>Unavailable</code></b> — nghĩa là “chưa chắc đã tới nơi”. Không retry trên lỗi nghiệp vụ (retry cũng vô ích) và không retry trên <code>DeadlineExceeded</code> (có thể đã tới nơi và đang xử lý — retry là <b>gọi trùng</b>). Còn nếu vẫn trùng, thì đã có <b>barrier</b> đỡ.'],
        ['Panel Grafana của breaker trống, sao vậy?', 'Vì gauge <code>circuit_breaker_state</code> <b>chỉ publish khi breaker ĐỔI trạng thái</b>. Lúc <code>closed</code> ban đầu <b>không có mẫu nào</b>. Phải bắn đủ 5 request cho breaker trip <b>trước</b>, rồi mới mở panel. <b>Biết trước điều này</b> để không lúng túng giữa buổi demo.']
      ],
      srcs: ['pkg/grpcx/client.go', 'pkg/grpcx/breaker.go', 'sony/gobreaker']
    },

    {
      id: 'van-fail-closed',
      title: 'Redis: JTI whitelist fail-closed, cache-aside',
      eyebrow: ['§10', 'Bảng 8 · dòng 5', 'vượt khung'],
      lede: 'JWT tự nó <b>không thu hồi được</b>. Cách giải quyết là tra Redis — và <b>chọn whitelist hay blacklist</b> quyết định hệ thống hành xử thế nào <b>khi Redis chết</b>.',
      scene: 'fail-closed-auth',
      say: [
        '“JWT là stateless: đã ký thì <b>hợp lệ tới khi hết hạn</b>, kể cả khi người dùng đã logout hay token bị đánh cắp. Muốn thu hồi thì phải có <b>trạng thái</b> ở đâu đó — chúng em để trong Redis.”',
        '“Và chúng em dùng <b>whitelist</b> chứ không phải blacklist: Redis lưu các JTI <b>còn sống</b>. Hệ quả: Redis chết ⇒ không thấy JTI nào ⇒ <b>từ chối tất cả</b>. Đó là <b>fail-closed</b>.”',
        '“Nếu dùng blacklist — cách phổ biến hơn — thì Redis chết ⇒ sổ đen rỗng ⇒ <b>mọi token đều được coi là sạch</b>, kể cả token đã bị thu hồi. Đó là <b>fail-open</b>, và nó là một <b>lỗ hổng an ninh thật</b>: kẻ tấn công chỉ cần làm Redis quá tải là mở được cửa. Chúng em chấp nhận mất tính sẵn sàng để đổi lấy an toàn.”'
      ],
      probes: [
        ['Redis chết là cả hệ thống mất đăng nhập? Nghe rất tệ.', '<b>Đúng, và đó là lựa chọn có ý thức.</b> Khi không thể xác minh, hệ thống phải <b>từ chối</b>, không phải cho qua. Mất đăng nhập tạm thời thì khôi phục được; cho token bị đánh cắp đi qua thì <b>không</b>. Giảm nhẹ thì có: chạy Redis replica. Nhưng <b>hướng fail</b> thì không đổi.'],
        ['Cache-aside dùng ở đâu?', 'Ở <b>RBAC</b>: quyền của người dùng được cache 5 phút trong Redis, vì mỗi request protected đều phải hỏi. Khi quyền bị đổi thì <b>chủ động invalidate</b>. Đây là thứ giảm nhẹ cho vấn đề <b>SPOF của user-service</b> — vấn đề mà Đức sẽ thừa nhận thẳng.'],
        ['Redis chết thì cache-aside cũng chết chứ?', 'Có, nhưng hành vi <b>ngược lại</b> với JTI: cache RBAC <b>degrade về không cache</b> — vẫn gọi gRPC sang user-service, chỉ chậm hơn. Nghĩa là cùng một Redis, nhưng <b>một chỗ fail-closed, một chỗ fail-open</b>. Đó là <b>chủ ý</b>: cache là để tăng tốc (mất thì chậm), whitelist là để bảo vệ (mất thì phải khoá).']
      ],
      srcs: ['pkg/auth/store/', 'services/user/internal/usecase/rbac_usecase.go']
    },

    {
      id: 'van-jwt',
      title: 'JWT và bảo mật xác thực',
      eyebrow: ['Bảng 8 · dòng 5'],
      lede: 'RS256, chống algorithm-confusion, refresh token xoay vòng single-use, khoá tài khoản, OTP băm bcrypt, Google OAuth2.',
      say: [
        '“Token ký <b>RS256</b> — bất đối xứng: user-service giữ khoá riêng để ký, 9 service kia chỉ cần khoá công khai để xác minh. Không service nào ngoài user-service có khả năng <b>tạo</b> token.”',
        '“Có chặn <b>algorithm confusion</b>: nếu kẻ tấn công đổi header thành <code>alg: HS256</code> và ký bằng chính khoá công khai (thứ ai cũng biết), hệ thống sẽ <b>từ chối</b> — vì nó <b>chỉ chấp nhận RS256</b>, không tin vào cái mà token tự khai.”',
        '“Refresh token <b>xoay vòng single-use</b>: mỗi lần refresh sinh token mới và <b>thu hồi ngay</b> token cũ. Dùng lại token cũ ⇒ nghi replay ⇒ <b>đăng xuất sạch</b>. Có test riêng chứng minh. Ngoài ra: khoá tài khoản 15 phút sau 5 lần sai, OTP <b>băm bcrypt</b> chứ không lưu thô.”'
      ],
      probes: [
        ['Vì sao RS256 chứ không HS256?', 'HS256 <b>đối xứng</b> — muốn xác minh thì phải biết <b>khoá ký</b>. Nghĩa là <b>cả 10 service</b> đều giữ khoá tạo token. Một service bị chiếm là <b>toàn hệ thống</b> bị giả mạo. RS256 tách bạch: <b>một</b> nơi ký, <b>mọi</b> nơi xác minh.'],
        ['Single-use rotation gây bug gì ở phía client không?', '<b>Có — và đó chính là bài toán của Nam.</b> N request cùng hết hạn, mỗi cái tự gọi <code>/auth/refresh</code> với cùng một token ⇒ cái đầu thắng và thu hồi token ⇒ các cái sau bị coi là replay ⇒ <b>người dùng bị đăng xuất oan</b>. Phải gom về <b>một</b> lời gọi refresh — <b>single-flight</b>.']
      ],
      srcs: ['services/user/internal/usecase/auth_usecase.go', 'pkg/auth/', 'refresh_logout_test.go']
    },

    {
      id: 'van-tracing',
      title: 'Distributed tracing',
      eyebrow: ['Bảng 8 · dòng 11', 'Demo 5'],
      lede: 'Một request của khách chạm <b>5 service</b>. Không có tracing thì khi nó chậm, bạn chỉ có thể <b>đoán</b> xem chậm ở đâu.',
      scene: 'trace-waterfall',
      say: [
        '“<code>trace_id</code> sinh ở <b>Kong</b>, rồi lan truyền qua HTTP → gRPC → Kafka. OpenTelemetry gom span, đẩy qua OTLP về <b>Jaeger</b>. Đo được: <b>1 trace, 9 span, 5 service</b>.”',
        '“Giá trị thật nằm ở chỗ này: nhìn từ order-service, bạn <b>không hề biết</b> location-service có tham gia — vì order gọi store, và <b>store</b> mới gọi location. Chỉ có trace mới lộ ra chuỗi đó.”',
        '“<b>Hạn chế em xin nêu trước:</b> ngữ cảnh trace <b>chưa nối qua biên Kafka</b>. Span của consumer là một span <b>gốc mới</b>, hiện chỉ liên kết được với luồng gốc qua <code>trace_id</code> trong log, chứ chưa thành một cây span liền mạch trong Jaeger. Muốn nối thì phải truyền <code>traceparent</code> qua header của Kafka message.”'
      ],
      probes: [
        ['Span là gì, khác trace chỗ nào?', '<b>Trace</b> là toàn bộ hành trình của <b>một</b> request. <b>Span</b> là <b>một đoạn công việc</b> trong hành trình đó — có thời điểm bắt đầu, thời lượng, và biết <b>span cha</b> của mình là ai. Ghép các quan hệ cha–con lại thì ra <b>cây</b>, vẽ ra chính là biểu đồ thác nước.'],
        ['Vì sao tự nêu hạn chế? Không sợ mất điểm à?', 'Ngược lại. Thầy <b>sẽ</b> hỏi “trace có xuyên qua Kafka không”. Tự nêu trước thì chứng minh mình <b>hiểu giới hạn của chính hệ thống mình</b>. Bị hỏi mới lúng túng thì <b>tệ hơn nhiều</b>.']
      ],
      srcs: ['pkg/otel/otel.go', 'pkg/trace/', 'Jaeger :17093']
    },

    {
      id: 'van-logging-metrics',
      title: 'Log tập trung và dashboard',
      eyebrow: ['Bảng 8 · dòng 3 & 13', 'Demo 5'],
      lede: 'Mười service, mười luồng log. Không gom lại thì mỗi lần điều tra một sự cố là <code>docker logs</code> mười lần rồi tự ghép bằng mắt.',
      say: [
        '“Log là <b>JSON</b> (slog) và <b>luôn kèm <code>trace_id</code></b> → Fluentd → Elasticsearch → Kibana. Nhờ vậy, lọc theo <b>một</b> <code>trace_id</code> là thấy log của <b>cả 5 service</b> trong <b>một</b> truy vấn — <b>đây chính là lý do phải lan truyền trace-id</b>, chứ không chỉ để vẽ biểu đồ đẹp.”',
        '“Có <b>ILM</b> tự xoá log sau <b>7 ngày</b> và index template khai kiểu tường minh (<code>trace_id</code>, <code>latency_ms</code>, <code>client_ip</code> kiểu <code>ip</code>). Không có ILM thì Elasticsearch <b>ăn hết đĩa</b> — đó là chuyện xảy ra thật ở hệ thống chạy lâu.”',
        '“<b>Metrics:</b> mỗi service phơi <code>/metrics</code>, Prometheus scrape <b>10/10</b>. Grafana provision sẵn dashboard <b>4 panel</b>: request rate, latency p95, lỗi gRPC client, và trạng thái circuit breaker.”'
      ],
      probes: [
        ['Ba thứ này khác nhau thế nào — log, metric, trace?', 'Ba lăng kính khác nhau của cùng một sự cố. <b>Metric</b> trả lời “<b>có</b> đang hỏng không?” (một con số, rẻ, lưu lâu). <b>Trace</b> trả lời “hỏng <b>ở đâu</b> trong chuỗi?”. <b>Log</b> trả lời “<b>vì sao</b> hỏng?” (chi tiết, đắt, lưu ngắn). Ba trụ cột của observability — và ta có đủ cả ba.'],
        ['Vì sao log phải là JSON?', 'Để <b>máy</b> đọc được. Log dạng văn xuôi thì phải viết regex để bóc trường ra — mong manh và chậm. JSON thì Elasticsearch index thẳng, lọc theo <code>trace_id</code> hay <code>latency_ms > 1000</code> là chuyện tức thì.'],
        ['Vì sao 7 ngày?', 'Đủ để điều tra một sự cố (thường phát hiện trong vài giờ), và đủ ngắn để không ngốn đĩa. Đây là <b>đánh đổi dung lượng ↔ khả năng điều tra</b>, và ta chọn con số một cách <b>có chủ ý</b>, không phải để mặc định.']
      ],
      srcs: ['docker-compose.logging.yml', 'docker/logging-init/init.sh', 'pkg/metrics/metrics.go', 'docker/grafana/dashboards/']
    },

    {
      id: 'van-discovery',
      title: 'Service discovery — mục ◐ duy nhất',
      eyebrow: ['Bảng 8 · dòng 7', 'thừa nhận thẳng'],
      lede: 'Đây là mục ta <b>không</b> ghi ✅. Nói thẳng và nói đúng còn ghi điểm hơn là gồng lên nhận.',
      say: [
        '“Chúng em <b>chưa</b> có service registry động kiểu Consul hay etcd. Định tuyến vào là <b>tập trung qua Kong</b>.”',
        '“Nhưng có một điểm em muốn nêu: <b>mọi</b> địa chỉ downstream — gRPC peer, DTM, saga branch, Kafka, MoMo — đều nằm trong <code>config/*.yaml</code>, <b>không một dòng nào hard-code trong Go</b>. Chuyển sang Kubernetes chỉ cần đổi giá trị config sang DNS của service (<code>payment-service.default.svc:50056</code>) — <b>không sửa một dòng code</b>.”',
        '“Nghĩa là chúng em <b>chưa làm</b> service discovery động, nhưng <b>đã chuẩn bị sẵn đường</b> cho nó.”'
      ],
      probes: [
        ['Vậy Kong có phải service discovery không?', 'Kong là <b>ingress</b> — nó giải quyết “từ ngoài vào thì gõ cửa ở đâu”. Service discovery giải quyết “<b>service này tìm service kia ở đâu</b>” <b>bên trong</b> hệ thống, và <b>tự cập nhật</b> khi instance lên/xuống. Chúng em làm việc thứ hai bằng <b>config tĩnh</b> — chạy được ở quy mô cố định, nhưng <b>không</b> tự co giãn.']
      ],
      srcs: ['kong/kong.yml', 'config/*.yaml']
    }
  ]
};
