/* Tab Tổng quan — bản đồ hệ thống, ranh giới sở hữu, lịch demo, checklist. */
VF.content = VF.content || {};
VF.content.tongquan = {
  label: 'Tổng quan',
  accent: 'var(--stamp)',
  blurb: 'Ai sở hữu tầng nào, ai nói mục nào, và buổi bảo vệ chạy theo trình tự ra sao.',
  cards: [

    {
      id: 'ov-map',
      title: 'Bản đồ hệ thống và ranh giới sở hữu',
      eyebrow: ['đọc cái này trước'],
      lede: 'Môn này chấm <b>kiến thức hệ phân tán</b>, không chấm số dòng code. Nên nhóm chia theo <b>tầng khái niệm</b>, không chia theo “ai gõ file nào”.',
      html: [
        '<div class="tablewrap"><table style="min-width:560px">',
        '<thead><tr><th>Người</th><th>Tầng sở hữu</th><th>Một câu định nghĩa</th></tr></thead><tbody>',
        '<tr><td style="color:var(--owner-van);font-weight:700">Văn</td>',
        '<td><b>Kênh truyền &amp; quan sát</b></td>',
        '<td>Mọi thứ nằm <b>giữa</b> các service: saga/DTM, outbox → Kafka, DLQ, gRPC client, Redis, và toàn bộ tầng quan sát</td></tr>',
        '<tr><td style="color:var(--owner-duc);font-weight:700">Đức</td>',
        '<td><b>Service &amp; vận hành</b></td>',
        '<td>Mọi thứ <b>bên trong</b> một service: 8 service có đường ghi, REST, DB riêng, RBAC, gateway, probe, Docker</td></tr>',
        '<tr><td style="color:var(--owner-nam);font-weight:700">Nam</td>',
        '<td><b>Bên nhận &amp; biên client</b></td>',
        '<td>Cơ chế <b>nhận về</b>: idempotent receiver, versioning, CQRS, hai service thuần-tiêu-thụ (<code>reporting</code>, <code>notification</code>), frontend, kiểm thử</td></tr>',
        '</tbody></table></div>',

        '<div class="say" style="margin-top:24px;max-width:none">',
        '<span class="label">Câu trả lời khi thầy hỏi “ai làm gì”</span>',
        '<p>“Văn làm <b>tầng giao tiếp giữa các service</b> — saga, hàng đợi, gRPC client, và tầng quan sát xuyên hệ. ',
        'Đức làm <b>tầng nghiệp vụ bên trong</b> mỗi service. ',
        'Nam làm <b>bên nhận sự kiện và biên client</b> — khử trùng lặp, dựng read model, giao diện.”</p>',
        '</div>',

        '<p style="margin-top:24px"><b>Ranh giới hay bị hỏi vặn nhất:</b> vì sao <code>reporting</code> và <code>notification</code> là của Nam chứ không phải Đức? ',
        'Vì chúng <b>không có API ghi nghiệp vụ</b> — toàn bộ trạng thái dựng từ sự kiện. Chúng <b>là</b> bên nhận. ',
        'Tám service còn lại có đường ghi, và đó là của Đức.</p>'
      ].join('')
    },

    {
      id: 'ov-tally',
      title: 'Ai nói mục nào — bản đồ tới barem chấm điểm',
      eyebrow: ['§14 của báo cáo Word'],
      lede: 'Ba bảng trong §14 chính là thứ thầy dùng để tick điểm. Mỗi ô đã gán <b>một</b> người chịu trách nhiệm nói — để không sót yêu cầu nào và không ai nói chồng lên nhau.',
      html: [
        '<div class="tablewrap"><table style="min-width:420px">',
        '<thead><tr><th></th><th style="text-align:right">Bảng 7<br>(bắt buộc)</th><th style="text-align:right">Bảng 8<br>(cộng điểm)</th>',
        '<th style="text-align:right">Vượt khung</th><th style="text-align:right">Tổng</th></tr></thead><tbody>',
        '<tr><td style="color:var(--owner-van);font-weight:700">Văn</td>',
        '<td class="num">2</td><td class="num">8</td><td class="num">6</td><td class="num" style="font-weight:700">16</td></tr>',
        '<tr><td style="color:var(--owner-duc);font-weight:700">Đức</td>',
        '<td class="num">3</td><td class="num">6</td><td class="num">2</td><td class="num" style="font-weight:700">11</td></tr>',
        '<tr><td style="color:var(--owner-nam);font-weight:700">Nam</td>',
        '<td class="num">2</td><td class="num" style="color:var(--stamp)">0</td><td class="num">4</td><td class="num" style="font-weight:700">6</td></tr>',
        '</tbody></table></div>',

        '<div class="say" style="margin-top:24px;max-width:none">',
        '<span class="label">Rủi ro đã biết — Nam không có dòng nào ở Bảng 8</span>',
        '<p>Nếu thầy đi <b>từng dòng</b> của bảng nâng cao để hỏi, Nam sẽ <b>im lặng suốt</b>. ',
        'Cách chữa: khi Văn trình bày <b>dòng 8 (Retry policy)</b> và vừa nói xong <i>“hết retry thì message vào DLQ”</i>, <b>Nam tiếp lời ngay</b>:</p>',
        '<p style="border-left:2px solid var(--stamp);padding-left:16px;font-style:italic">',
        '“Và khi Kafka giao lại message — vì nó chỉ đảm bảo at-least-once — thì <code>processed_events</code> chặn trùng, ',
        'nên hiệu ứng nghiệp vụ vẫn đúng một lần. Em có demo tua offset về 0 để chứng minh.”</p>',
        '<p>Câu đó vừa kéo Nam vào Bảng 8, vừa mở đường tự nhiên sang <b>Demo 3</b>.</p>',
        '</div>'
      ].join('')
    },

    {
      id: 'ov-demos',
      title: 'Kịch bản demo — không demo happy path',
      eyebrow: ['6 demo', 'đã chạy thật'],
      lede: '<b>Demo lúc hệ thống gãy.</b> Happy path không chứng minh được gì về hệ phân tán — mọi đồ án đều có happy path.',
      html: [
        '<div class="tablewrap"><table style="min-width:600px">',
        '<thead><tr><th>Demo</th><th>Ai</th><th>Chứng minh điều gì</th></tr></thead><tbody>',
        '<tr><td><b>0</b> · Happy path</td><td>cả nhóm</td><td>Đặt đơn qua UI → DTM UI hiện saga <code>succeed</code> 3 nhánh → Jaeger hiện 1 trace 9 span</td></tr>',
        '<tr><td><b>1</b> · Saga bù trừ ngược</td><td style="color:var(--owner-van);font-weight:700">Văn</td>',
        '<td>Đặt đơn <b>vượt số dư ví</b> → saga <code>failed</code>, bù trừ chạy ngược, <b>null compensation</b> vẫn succeed</td></tr>',
        '<tr><td><b>2</b> · Circuit breaker</td><td style="color:var(--owner-van);font-weight:700">Văn</td>',
        '<td>Dừng <b>store-service</b> (KHÔNG phải payment!), bắn <b>≥5 request</b> → mạch mở, <b>5.055ms → 25ms</b></td></tr>',
        '<tr><td><b>3</b> · Idempotent receiver</td><td style="color:var(--owner-nam);font-weight:700">Nam</td>',
        '<td>Tua offset Kafka về 0 → giao lại 8 event → thông báo vẫn <b>8, không phải 16</b></td></tr>',
        '<tr><td><b>4</b> · Readiness + degrade</td><td style="color:var(--owner-duc);font-weight:700">Đức</td>',
        '<td>Dừng Postgres → <code>/readyz</code> <b>503</b> nhưng <code>/health</code> vẫn <b>200</b>; dừng location → store degrade, không sập</td></tr>',
        '<tr><td><b>5</b> · Quan sát xuyên hệ</td><td style="color:var(--owner-van);font-weight:700">Văn</td>',
        '<td>Chạy <b>ngay sau Demo 2</b> để panel breaker có dữ liệu. Jaeger + Grafana + Kibana lọc theo một <code>trace_id</code></td></tr>',
        '<tr><td><b>6</b> · CQRS read model</td><td style="color:var(--owner-nam);font-weight:700">Nam</td>',
        '<td>Đặt đơn → vài giây sau <code>/admin</code> cập nhật, dù reporting <b>không có API ghi nào</b></td></tr>',
        '</tbody></table></div>',

        '<div class="say" style="margin-top:24px;max-width:none">',
        '<span class="label">Ba cái bẫy đã vấp phải khi chạy thử — đừng vấp lại</span>',
        '<p style="font-family:var(--font-body);font-size:16px">',
        '<b>Demo 2:</b> dừng <b>payment-service</b> thì breaker <b>KHÔNG</b> trip — vì với engine DTM, <b>DTM server</b> gọi participant, không phải order-service. Phải dừng <b>store-service</b>. Và phải bắn <b>≥5 request</b>; bấm 1–2 lần thì demo trông như hỏng.<br><br>',
        '<b>Demo 3:</b> chỉ restart consumer thì <b>chỉ chứng minh catch-up</b>, không chứng minh idempotency. Phải <b>tua offset</b> bằng <code>kafka-consumer-groups.sh --reset-offsets --to-earliest</code>.<br><br>',
        '<b>Demo 5:</b> panel <code>circuit_breaker_state</code> <b>trống trơn</b> nếu mở trước Demo 2 — vì gauge <b>chỉ publish khi breaker đổi trạng thái</b>.</p>',
        '</div>'
      ].join('')
    },

    {
      id: 'ov-setup',
      title: 'Chuẩn bị trước buổi bảo vệ',
      eyebrow: ['checklist'],
      lede: 'Chạy đúng thứ tự này. Bỏ bước đầu là <b>auth tắt im lặng</b> và mọi route protected trả 404 — rất khó đoán ra giữa buổi.',
      html: [
        '<pre style="background:var(--paper-sunken);border:1px solid var(--rule);padding:16px;overflow-x:auto;',
        'font-family:var(--font-mono);font-size:13px;line-height:1.7"><code>cd backend\n',
        'make jwt-keys                              <span style="color:var(--stamp)"># BẮT BUỘC — thiếu là auth tắt, mọi route protected trả 404</span>\n',
        'docker compose up -d                       <span style="color:var(--ink-faint)"># 10 service + Postgres/Redis/Kafka/DTM/MinIO/Kong</span>\n',
        'docker compose --profile monitoring up -d  <span style="color:var(--ink-faint)"># Prometheus :17095, Grafana :17096, Jaeger :17093</span>\n',
        'go run ./services/store/cmd/seed-catalog   <span style="color:var(--ink-faint)"># 20 quán, ~950 món</span>\n',
        'cd ../frontend &amp;&amp; npm run dev              <span style="color:var(--ink-faint)"># :17070</span></code></pre>',

        '<p style="margin-top:20px"><b>Không chạy trộn Docker và <code>go run</code>.</b> <code>config/*.docker.yaml</code> địa chỉ hoá DTM và các nhánh saga theo <b>DNS của compose</b>, ',
        'còn <code>config/*.yaml</code> dùng <code>localhost</code> — trộn hai chế độ sẽ khiến DTM <b>không gọi được vào nhánh</b> và saga gãy.</p>',

        '<p style="margin-top:16px"><b>Tab mở sẵn:</b> Frontend <code>:17070</code> · Kong <code>:17000</code> · ',
        '<b>DTM UI <code>:17789</code></b> · <b>Jaeger <code>:17093</code></b> · <b>Grafana <code>:17096</code></b> · Prometheus <code>:17095</code></p>',

        '<p style="margin-top:16px"><b>Tài khoản seed</b> (mật khẩu chung <code>Password123!</code>): ',
        '<code>cust@</code> · <code>shop@</code> · <code>ship@</code> · <code>admin@</code> · <code>super-admin@</code> — tất cả <code>@velox.test</code></p>',

        '<div class="say" style="margin-top:24px;max-width:none">',
        '<span class="label">Một mục phải thống nhất trước — đừng ai nhận RabbitMQ</span>',
        '<p style="font-family:var(--font-body);font-size:16px"><code>pkg/messaging/rabbitmq</code> <b>có code nhưng 0 call site</b> — toàn hệ chạy Kafka. ',
        'Nếu thầy hỏi <i>“RabbitMQ dùng ở luồng nào”</i> mà có người ậm ừ nhận bừa là <b>mất điểm ngay</b>. ',
        'Khai đúng như §15 của báo cáo: <b>đã hiện thực, chưa wire, để dành mở rộng</b>.</p>',
        '</div>'
      ].join('')
    }
  ]
};
