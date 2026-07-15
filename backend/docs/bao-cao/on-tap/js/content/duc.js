/* Đức — tầng service & vận hành. Mọi thứ BÊN TRONG một service. */
VF.content = VF.content || {};
VF.content.duc = {
  label: 'Đức',
  tally: 11,
  accent: 'var(--owner-duc)',
  blurb: 'Service &amp; vận hành — mọi thứ nằm <b>bên trong</b> một service: 10 service, REST, DB riêng, RBAC, gateway, probe, Docker.',
  cards: [

    {
      id: 'duc-decomposition',
      title: 'Phân rã hệ thống thành 10 service',
      eyebrow: ['Bảng 7 · #1', '§4'],
      lede: 'Đề bài yêu cầu ≥ 3 microservice. Ta có <b>10</b>. Nhưng con số không phải là điểm — <b>ranh giới chia ở đâu và vì sao</b> mới là.',
      say: [
        '“Chúng em chia theo <b>bounded context</b> — mỗi service sở hữu <b>trọn vẹn</b> một miền nghiệp vụ và <b>dữ liệu</b> của miền đó: user (danh tính), location, store (quán + thực đơn), order (đơn hàng), promotion (voucher), payment (ví + sổ cái), delivery (giao vận), review, reporting, notification.”',
        '“Tiêu chí chia: <b>cái gì thay đổi cùng nhau thì ở cùng một chỗ</b>. Thực đơn và giờ mở cửa luôn đổi cùng nhau ⇒ cùng nằm trong store. Còn voucher đổi theo lịch marketing, chẳng liên quan gì tới thực đơn ⇒ tách ra.”',
        '“Mỗi service theo <b>clean architecture</b>: handler → usecase → repository (interface) ← persistence. Entity là lá, không phụ thuộc gì. Nhờ vậy usecase <b>test được mà không cần database</b> — ta inject stub vào interface.”'
      ],
      probes: [
        ['Vì sao 10 mà không phải 3? Có bị chia quá nhỏ không?', 'Câu hỏi công bằng — <b>chia quá nhỏ</b> là bệnh có thật. Nhưng mỗi service ở đây đều <b>sở hữu dữ liệu riêng</b> và có <b>vòng đời riêng</b>. Kiểm tra ngược lại: <b>reporting</b> và <b>notification</b> tách ra vì chúng chỉ <b>đọc sự kiện</b>, không có API ghi — gộp vào order sẽ làm đường ghi <b>chậm đi</b> vì phải tính báo cáo.'],
        ['Nếu phải gộp bớt thì gộp cái nào?', '<b>location</b> có thể gộp vào store — nó nhỏ, và gần như chỉ store gọi tới. Em giữ riêng vì cây Toà → Tầng → Phòng là dữ liệu <b>dùng chung</b> cho cả đơn hàng lẫn giao vận. Nhưng em <b>công nhận</b> đó là ranh giới mỏng nhất trong hệ thống.']
      ],
      srcs: ['services/*/internal/{entity,repository,usecase,handler}']
    },

    {
      id: 'duc-rest',
      title: 'RESTful API — 180 endpoint',
      eyebrow: ['Bảng 7 · #2'],
      lede: 'Tài nguyên là danh từ, hành động là động từ HTTP, mã trạng thái đúng ngữ nghĩa.',
      say: [
        '“<b>180 endpoint</b> qua Kong. Tài nguyên là <b>danh từ số nhiều</b> (<code>/orders</code>, <code>/stores</code>), hành động dùng <b>động từ HTTP</b>, và mã trạng thái mang đúng nghĩa: <code>201</code> khi tạo, <code>204</code> khi xoá, <code>409</code> khi xung đột, <code>503</code> khi chưa sẵn sàng.”',
        '“Quy ước chung toàn hệ: <b>request body snake_case, response entity PascalCase</b>. Tiền là <code>int64</code> đơn vị nhỏ nhất — <b>không dùng float</b>, vì số thực <b>không biểu diễn chính xác</b> được tiền tệ.”'
      ],
      probes: [
        ['Vì sao tiền là int64 chứ không phải float?', '<code>0.1 + 0.2 !== 0.3</code> trong dấu phẩy động. Với tiền thì sai số đó <b>tích luỹ</b> thành lệch sổ. Nên lưu <b>đơn vị nhỏ nhất</b> (đồng) dưới dạng số nguyên — <b>chính xác tuyệt đối</b>.'],
        ['Có endpoint nào không RESTful không?', 'Có, và em nói thẳng: các hành động <b>chuyển trạng thái</b> như <code>POST /orders/:id/cancel</code> không phải REST thuần (nó là <b>động từ</b> trong URL). Nhưng ép thành <code>PATCH /orders/:id {status: "CANCELLED"}</code> sẽ <b>giấu mất</b> việc đó là một thao tác <b>nghiệp vụ phức tạp</b> — nó mở cả một saga. Em <b>chọn</b> sự rõ ràng.']
      ],
      srcs: ['services/*/internal/handler/http/router.go']
    },

    {
      id: 'duc-db-per-service',
      title: 'Database riêng mỗi service',
      eyebrow: ['Bảng 7 · #4'],
      lede: 'Mười service, mười database. Không service nào đọc thẳng bảng của service khác. <b>Và cái giá phải trả là gì?</b>',
      say: [
        '“Mỗi service có <b>database riêng</b> — <code>order_db</code>, <code>payment_db</code>… Không ai được đọc thẳng bảng của người khác; muốn dữ liệu thì phải <b>gọi API</b>.”',
        '“<b>Cái giá phải trả — và em xin nói thẳng:</b> <b>không có khoá ngoại liên database</b>. <code>orders.user_id</code> chỉ là một UUID <b>mềm</b> — database <b>không thể</b> đảm bảo người dùng đó tồn tại. Ta <b>mất ràng buộc toàn vẹn ở tầng DB</b> và phải tự lo ở tầng ứng dụng.”',
        '“Đổi lại: mỗi service <b>tiến hoá schema độc lập</b>, triển khai độc lập, và một database chết <b>không kéo cả hệ thống chết theo</b>.”'
      ],
      probes: [
        ['Vậy làm sao join dữ liệu giữa các service?', '<b>Không join.</b> Hai cách: (1) <b>gọi API</b> lấy dữ liệu rồi ghép ở tầng ứng dụng — dùng khi cần dữ liệu tươi; (2) <b>dựng read model từ event</b> — đó chính là <code>reporting</code>. Cách 2 là lý do CQRS tồn tại.'],
        ['Xoá một user thì dữ liệu ở service khác thành mồ côi à?', '<b>Đúng vậy</b>, và không có DB nào chặn được. Xử lý bằng <b>sự kiện</b>: <code>user.deleted</code> được phát, các service quan tâm tự dọn phần của mình. Nhất quán <b>cuối</b>, không tức thì — và đó chính là bản chất của kiến trúc này.'],
        ['Sao vẫn để chung một instance Postgres?', 'Đó là <b>giới hạn của Docker Compose trên máy đồ án</b>, và em nói thẳng. Về mặt <b>logic</b> chúng vẫn là 10 database tách biệt (10 connection string riêng, không cross-query). Chuyển sang 10 instance thật chỉ là đổi host trong config — <b>không sửa code</b>.']
      ],
      srcs: ['config/*.yaml', 'migrations riêng từng service']
    },

    {
      id: 'duc-rbac',
      title: 'RBAC tập trung — và SPOF phải thừa nhận',
      eyebrow: ['§10', 'trade-off lớn nhất của hệ thống'],
      lede: 'Đây là <b>điểm yếu kiến trúc lớn nhất</b> của cả hệ thống. Chủ động nêu ra và giải thích được vì sao chấp nhận nó — đó là dấu hiệu của người thiết kế, không phải người gõ code.',
      say: [
        '“Phân quyền tập trung ở user-service: <b>40 permission, 6 role</b>, có cả phạm vi theo vendor. Chín service kia, mỗi khi gặp route protected, đều gọi gRPC <code>CheckPermission</code> sang user-service.”',
        '“<b>Và đây là SPOF.</b> user-service chết ⇒ <b>mọi route protected của toàn hệ thống</b> ngừng hoạt động. Em <b>không giấu</b> điều đó.”',
        '“Vì sao vẫn chọn? Vì phương án kia — <b>mỗi service tự giữ bản sao phân quyền</b> — nghe thì phi tập trung hơn, nhưng nó tạo ra <b>10 nguồn sự thật</b>. Thu quyền của một người thì phải cập nhật 10 chỗ, và trong lúc chưa kịp đồng bộ thì <b>quyền đã bị thu vẫn còn hiệu lực</b> ở đâu đó. Với <b>phân quyền</b>, em cho rằng <b>một nguồn sự thật duy nhất</b> đáng giá hơn tính sẵn sàng.”',
        '“Giảm nhẹ: cache 5 phút trong Redis, nên không phải request nào cũng đi mạng.”'
      ],
      probes: [
        ['Cache 5 phút thì quyền bị thu hồi vẫn còn hiệu lực 5 phút?', '<b>Đúng.</b> Nên khi quyền đổi, ta <b>chủ động invalidate cache</b> chứ không đợi hết hạn. Cái 5 phút chỉ là lưới an toàn cuối cùng cho trường hợp invalidate bị lỡ.'],
        ['Có cách nào bỏ SPOF mà vẫn giữ một nguồn sự thật?', 'Có — nhét quyền <b>vào chính JWT</b> lúc đăng nhập. Mỗi service tự đọc, <b>không gọi ai</b>. Nhưng thế thì <b>không thu hồi được</b> cho tới khi token hết hạn, và token <b>phình to</b>. Đó là đánh đổi <b>ngược lại</b>, và em <b>biết</b> nó tồn tại — em chọn hướng còn lại một cách có ý thức.']
      ],
      srcs: ['pkg/auth/remotechecker/', 'services/user/internal/usecase/rbac_usecase.go']
    },

    {
      id: 'duc-gateway',
      title: 'API Gateway và rate limiting hai tầng',
      eyebrow: ['Bảng 8 · dòng 1 & 2'],
      lede: 'Kong là <b>cổng vào duy nhất</b>. Khách không bao giờ biết bên trong có bao nhiêu service.',
      say: [
        '“Kong khai báo <b>declarative</b> (<code>kong.yml</code>), <b>46 route</b>. Nó là <b>ingress duy nhất</b> — nó <b>giấu toàn bộ topology bên trong</b>. Client chỉ thấy <code>:17000</code>; ta có tách hay gộp service bên trong, client <b>không cần biết</b>.”',
        '“Kong cũng lo phần <b>xuyên suốt</b>: CORS, correlation-id, và rate limiting — để 10 service <b>không phải mỗi anh tự viết một kiểu</b>.”',
        '“Rate limit <b>hai tầng</b>, và tầng thứ hai mới là chỗ có suy nghĩ: <b>300 request/phút</b> toàn cục theo IP, nhưng riêng <code>/auth/*</code> và đăng ký shipper chỉ <b>10/phút</b>. Vì đó là các route <b>đoán được</b>: mật khẩu và OTP <b>brute-force được</b>. 300 lần thử OTP 6 số mỗi phút là <b>quá đủ để dò ra</b>.”'
      ],
      probes: [
        ['Rate limit theo IP thì NAT chung IP bị chặn oan?', '<b>Đúng</b> — cả một ký túc xá sau một NAT sẽ dùng chung hạn mức. Chuẩn hơn là giới hạn theo <b>user</b> hoặc <b>API key</b>. Với đồ án thì theo IP là đủ, và em <b>biết</b> giới hạn của cách này.'],
        ['Khách bị 429 thì thấy gì?', 'Frontend bắt <code>429</code> và hiện thông báo thân thiện, chứ không quăng lỗi thô. Chi tiết nhỏ nhưng nó cho thấy giới hạn được <b>thiết kế</b> chứ không phải bật đại một plugin.']
      ],
      srcs: ['kong/kong.yml']
    },

    {
      id: 'duc-probes',
      title: 'Health, readiness, graceful shutdown, degrade',
      eyebrow: ['Bảng 8 · dòng 9 & 10', 'Demo 4'],
      lede: 'Bốn cơ chế trả lời bốn câu khác nhau. Trộn lẫn chúng là hiểu sai.',
      say: [
        '“<code>/health</code> là <b>liveness</b> — process còn sống không? Chết thì orchestrator <b>giết và khởi động lại</b>.”',
        '“<code>/readyz</code> là <b>readiness</b> — có sẵn sàng <b>nhận traffic</b> không? Nó ping DB và Redis; mất DB thì trả <code>503</code>. Orchestrator <b>rút service khỏi rotation</b> nhưng <b>không giết</b> — vì restart <b>không</b> chữa được database chết.”',
        '“<b>Chi tiết cố ý:</b> <code>/readyz</code> <b>KHÔNG</b> probe Kafka. Vì consumer <b>tự retry</b> — một nhịp trục trặc của broker <b>không nên</b> kéo cả service ra khỏi rotation trong khi API HTTP của nó vẫn phục vụ tốt.”',
        '“<b>Graceful shutdown</b> có thứ tự <b>có chủ đích</b>: HTTP drain 15s → gRPC GracefulStop → consumer → flush span OTel → đóng Redis/DB. Đảo thứ tự là mất request đang xử lý dở.”',
        '“<b>Graceful degradation:</b> location-service chết thì store <b>không sập</b> — resolver phí ship rơi về <code>noop</code>, trả lời ‘không phục vụ khu vực này’ thay vì <code>500</code>.”'
      ],
      probes: [
        ['Vì sao tách liveness và readiness? Một cái không đủ à?', 'Vì <b>hai câu hỏi khác nhau dẫn tới hai hành động khác nhau</b>. Gộp lại thì database chết ⇒ readiness fail ⇒ orchestrator tưởng process hỏng ⇒ <b>restart liên tục</b> ⇒ <b>crash loop</b>. Mà restart <b>không chữa được</b> database chết. Tách ra thì: DB chết ⇒ rút khỏi rotation, <b>giữ process sống</b>, chờ DB về.'],
        ['Vì sao drain HTTP đúng 15 giây?', 'Đủ dài để request đang xử lý dở chạy xong (p95 chỉ vài trăm ms), đủ ngắn để deploy không lê thê. Và phải drain <b>trước</b> khi đóng DB — đóng DB trước thì request đang dở <b>chết giữa chừng</b>.']
      ],
      srcs: ['pkg/app/app.go', 'services/store/cmd/main.go']
    },

    {
      id: 'duc-ledger',
      title: 'Sổ cái kép',
      eyebrow: ['§8', 'vượt khung'],
      lede: 'Tiền không được phép “bốc hơi”. Sổ kép làm cho mọi sai lệch <b>tự lộ ra ngay</b>, thay vì lộ lúc đối soát cuối tháng.',
      scene: 'double-entry-ledger',
      say: [
        '“Mỗi luồng tiền sinh <b>hai</b> bút toán, <b>tổng luôn bằng 0</b>: tiền rời ví khách thì <b>phải</b> có tài khoản nào đó nhận. Cả hai ghi trong <b>một transaction</b>, khoá <code>FOR UPDATE</code> <b>cả hai ví</b>.”',
        '“Nếu tổng khác 0 thì <b>chắc chắn có bug</b> — và ta <b>phát hiện được ngay</b>. Sổ ghi một chiều thì tiền biến mất mà <b>không ai biết</b> cho tới lúc đối soát.”',
        '“<b>Chi tiết em muốn nói:</b> tài khoản <code>SYSTEM</code> (clearing) <b>được phép âm</b>, ví người dùng thì <b>không</b>. Vì số âm ở tài khoản trung gian chính là <b>nghĩa vụ hệ thống phải trả lại</b> — nó có <b>ý nghĩa kế toán</b>, không phải lỗi.”'
      ],
      probes: [
        ['Vì sao FOR UPDATE cả hai ví, không phải một?', 'Vì cả hai đều bị <b>sửa</b>. Chỉ khoá một bên thì hai giao dịch song song có thể đọc–ghi chồng lên nhau ở bên còn lại (<b>lost update</b>). Và phải khoá theo <b>thứ tự cố định</b> (ví dụ tăng dần theo id) — nếu không thì hai transaction khoá ngược chiều nhau sẽ <b>deadlock</b>.'],
        ['Refund thì ghi ngược lại hay xoá bút toán cũ?', '<b>Ghi ngược lại.</b> Sổ cái là <b>append-only</b> — <b>không bao giờ sửa hay xoá</b> bút toán đã ghi. Hoàn tiền là <b>hai bút toán mới</b> ngược chiều. Nhờ vậy <b>toàn bộ lịch sử được giữ nguyên</b> và có thể kiểm toán.']
      ],
      srcs: ['services/payment/internal/usecase/ledger_helper.go']
    },

    {
      id: 'duc-concurrency',
      title: 'Chống tranh chấp trong service',
      eyebrow: ['§9', 'vượt khung'],
      lede: 'Ba tình huống đua nhau khác nhau, ba cách xử lý khác nhau.',
      say: [
        '“<b>Voucher:</b> khoá <code>FOR UPDATE</code> khi siết <code>usage_limit</code>. Không có nó thì 100 người bấm cùng lúc sẽ <b>vượt hạn mức</b> — vì cả 100 cùng đọc thấy ‘còn 1 lượt’.”',
        '“<b>Nhận đơn giao:</b> <b>atomic claim</b> — <code>UPDATE … WHERE shipper_id IS NULL</code>. Chỉ <b>một</b> tài xế thắng, database tự quyết định. Không cần khoá, không cần hỏi trước rồi ghi sau (kiểu đó là <b>TOCTOU</b>: kiểm tra xong mới ghi, khe hở giữa hai bước đủ cho người khác chen vào).”',
        '“<b>Chi trả cho quán:</b> đối soát <b>loại trừ</b> các đơn đã nằm trong batch <code>PENDING</code>/<code>SETTLED</code> — nếu không thì chạy payout hai lần là <b>trả tiền gấp đôi</b>.”'
      ],
      probes: [
        ['Khoá bi quan (FOR UPDATE) và lạc quan (CAS) — chọn cái nào khi nào?', '<b>Bi quan</b> khi tranh chấp <b>hay xảy ra</b> và việc làm lại thì đắt — như voucher hot 100 người tranh 1 suất. <b>Lạc quan</b> (atomic claim / CAS) khi tranh chấp <b>hiếm</b> — như tài xế nhận đơn: thường chỉ 1–2 người cùng bấm, người thua chỉ cần thấy “đơn đã có người nhận”.']
      ],
      srcs: ['promotion_gorm_repository.go', 'delivery_gorm_repository.go']
    },

    {
      id: 'duc-docker',
      title: 'Docker — và vì sao chưa có Kubernetes',
      eyebrow: ['Bảng 8 · dòng 6 & 14'],
      lede: 'Docker ✅ thật, chạy được từ máy sạch. Kubernetes ✗, và ta nói thẳng.',
      say: [
        '“<code>docker compose up -d</code> dựng <b>10 service + Postgres, Redis, Kafka, DTM, MinIO, Kong</b> từ <b>máy sạch</b>. Script init tự tạo 10 database và cả bảng của DTM.”',
        '“<b>Kubernetes: chưa có, em nói thẳng.</b> Nhưng có một chi tiết đã <b>sẵn sàng cho nó</b>: auto-migrate dùng <b>advisory lock của Postgres</b> — nên nhiều replica khởi động <b>song song</b> sẽ <b>không đua nhau chạy migration</b>. Đúng một replica thắng khoá và chạy, các replica khác chờ rồi đi tiếp.”',
        '“<b>Ràng buộc phải nêu trước khi thầy chạy thử:</b> không được chạy <b>trộn</b> Docker với <code>go run</code>. <code>config/*.docker.yaml</code> địa chỉ hoá theo <b>DNS của compose</b>, còn <code>config/*.yaml</code> dùng <code>localhost</code> — trộn hai chế độ là DTM <b>không gọi được vào nhánh saga</b>.”'
      ],
      probes: [
        ['Advisory lock là gì?', 'Một khoá <b>do ứng dụng tự đặt tên</b>, Postgres giữ hộ, <b>không gắn với bảng nào</b>. Ở đây nó dùng để đảm bảo <b>đúng một</b> process chạy migration tại một thời điểm. Không có nó, 3 replica cùng start sẽ cùng chạy <code>CREATE TABLE</code> ⇒ hai cái lỗi ⇒ <b>crash loop</b>.'],
        ['Chuyển sang K8s cần đổi gì?', 'Chủ yếu là <b>config</b>: địa chỉ downstream đổi sang DNS của K8s. Thêm manifest cho Deployment/Service, và trỏ liveness/readiness probe vào <code>/health</code>, <code>/readyz</code> — <b>hai endpoint đã có sẵn</b>. Phần khó thật sự là <b>stateful</b> (Postgres, Kafka), chứ không phải service Go.']
      ],
      srcs: ['docker-compose.yml', 'docker/services/Dockerfile', 'docker/postgres/init/']
    }
  ]
};
