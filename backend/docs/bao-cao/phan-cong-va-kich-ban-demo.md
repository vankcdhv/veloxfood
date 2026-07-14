# Phân công & kịch bản demo — Đồ án VeloxFood (INT4409)

Nhóm: **Văn** · **Đức** · **Nam**

## Nguyên tắc chia việc

Môn này chấm **kiến thức hệ phân tán**, không chấm số dòng code. Vì vậy chia theo **tầng khái niệm**, không chia theo "ai gõ file nào":

| Người | Tầng sở hữu | Một câu định nghĩa |
|---|---|---|
| **Văn** | **Kênh truyền & quan sát** | Mọi thứ nằm **giữa** các service: saga/DTM, outbox → Kafka, DLQ, `pkg/grpcx` (deadline/retry/breaker), Redis, **và toàn bộ tầng quan sát** (tracing, EFK, metrics) |
| **Đức** | **Service & vận hành** | Mọi thứ **bên trong** một service: 10 service, REST, DB-per-service, RBAC, gateway, probe, Docker |
| **Nam** | **Bên nhận & biên client** | Cơ chế **nhận về**: idempotent receiver, versioning sự kiện, CQRS, notification, frontend, kiểm thử |

Câu giải thích khi thầy hỏi ai làm gì:

> *"Văn làm tầng giao tiếp giữa các service — saga, hàng đợi, gRPC client, và tầng quan sát xuyên hệ. Đức làm tầng nghiệp vụ bên trong mỗi service. Nam làm bên nhận sự kiện và biên client — khử trùng lặp, dựng read model, giao diện."*

---

## Văn — Kênh truyền & quan sát *(phần lớn nhất)*

**Khái niệm sở hữu:** Saga (orchestration, không dùng 2PC) · sub-transaction barrier · null / durable / timeout compensation · Transactional Outbox · Kafka producer + retry/DLQ + bảo toàn thứ tự partition · RPC đồng bộ (deadline / retry / circuit breaker) · cache-aside · fail-closed authorization · **distributed tracing** · **centralized logging** · **metrics & dashboard**.

| Chủ đề | File |
|---|---|
| Saga engine + barrier | `pkg/saga/saga.go`, `pkg/saga/barrier.go` |
| Saga đặt hàng (compensating) | `services/order/internal/usecase/place_order_dtm.go` |
| Saga huỷ đơn (idempotent-retry) | `services/order/internal/usecase/order_cancel_dtm.go` |
| Bù trừ bền vững | `services/order/cmd/wire_compensation_worker.go` + bảng `pending_compensations` |
| Janitor giữ chỗ voucher (TTL 15') | `services/promotion/cmd/wire_reservation_janitor.go` |
| Transactional Outbox + dispatcher | `pkg/outbox/` |
| Kafka: producer, retry, DLQ, ordering | `pkg/messaging/kafka/` |
| **gRPC client chuẩn** | `pkg/grpcx/client.go` — chain: trace → **breaker** → timeout 5s → retry ×2 |
| **Circuit breaker** | `pkg/grpcx/breaker.go` (`sony/gobreaker`) |
| Redis | `pkg/auth/store/` (JTI whitelist **fail-closed**), `rbac_usecase.go` (cache-aside + invalidation) |
| **Tracing (cơ chế + UI)** | `pkg/otel/otel.go` (OTLP → Jaeger), `pkg/trace/` (lan truyền qua HTTP ↔ gRPC ↔ Kafka) |
| **Logging tập trung** | `docker-compose.logging.yml`, `docker/logging-init/init.sh` (ILM 7 ngày, index template) |
| **Metrics & dashboard** | `pkg/metrics/metrics.go` (4 metric), `docker/grafana/dashboards/` |

**Mục báo cáo:** §6, §7 (phần phát), §9, §11 (toàn bộ), phần Redis + JWT của §10.

**Phải nói được:**
- Vì sao **saga chứ không phải 2PC** (2PC khoá tài nguyên xuyên service, không chịu được partition).
- Barrier giải quyết **hai** bài toán: duplicate branch call **và** null compensation.
- Hai kiểu saga: **compensating** (đặt hàng) vs **idempotent-retry** (huỷ đơn, forward-only).
- **Transactional Outbox**: sự kiện ghi **cùng transaction** với business write ⇒ không bao giờ có chuyện "đơn tạo rồi mà sự kiện mất".
- Consumer hết retry (3 lần, backoff 0,5/1/2s) → **DLQ** kèm header truy vết; nếu vừa không xử lý được vừa không park được DLQ thì **cố ý không commit**, dừng ở message đó để **giữ thứ tự partition**.
- Circuit breaker **chỉ trip trên lỗi hạ tầng** (`Unavailable`, `DeadlineExceeded`), **không** trip trên lỗi nghiệp vụ (`NotFound`). Đây là chi tiết phân biệt người hiểu và người copy.
- Vì sao deadline 5s: participant treo thì **fail-fast để saga bù trừ**, thay vì treo cả chuỗi.
- JTI dùng **whitelist** chứ không blacklist ⇒ Redis chết thì từ chối hết (**fail-closed**), không fail-open.
- **Tracing**: trace-id sinh từ trình duyệt → Kong → HTTP → gRPC → Kafka; OTel span đẩy qua OTLP về Jaeger. Đo được **1 trace, 9 span, 5 service**. **Hạn chế tự nêu:** ngữ cảnh trace **chưa nối qua biên Kafka** — span consumer là span gốc mới, hiện chỉ liên kết qua `trace_id`.
- **Logging**: log JSON (slog) kèm `trace_id` → Fluentd → Elasticsearch → Kibana; ILM tự xoá sau 7 ngày.
- **Metrics**: 4 metric qua `/metrics` mỗi service. Lưu ý gauge `circuit_breaker_state` **chỉ publish khi breaker đổi trạng thái** ⇒ panel trống cho tới khi có breaker trip.

> **Bàn giao sang Nam:** Văn đảm bảo sự kiện **được phát đi, không mất, và quan sát được**. Từ lúc message vào topic trở đi là phần của Nam (khử trùng lặp, dựng read model).

---

## Đức — Tầng service & vận hành *(phần nhì)*

**Khái niệm sở hữu:** phân rã theo bounded-context · RESTful API · database-per-service + soft reference · RBAC tập trung (và cái giá của nó) · API Gateway · rate limiting · health/readiness probe · graceful shutdown · graceful degradation · sổ cái kép · chống tranh chấp trong service.

| Chủ đề | File |
|---|---|
| 10 service, kiến trúc sạch | `services/*/internal/{entity,repository,usecase,handler}` |
| REST (180 endpoint) | `services/*/internal/handler/http/router.go` |
| Hợp đồng gRPC (định nghĩa + gọi) | `proto/**`, `services/*/internal/infrastructure/grpcclient/` |
| Bootstrap dùng chung | `pkg/app/app.go` — `/health`, `/readyz`, graceful shutdown |
| RBAC tập trung | `pkg/auth/remotechecker/` → gRPC về user-service |
| Gateway | `kong/kong.yml` — ingress duy nhất, rate-limit 2 tầng |
| Sổ cái kép | `services/payment/internal/usecase/ledger_helper.go` |
| Chống tranh chấp | `promotion_gorm_repository.go` (`FOR UPDATE`), `delivery_gorm_repository.go` (atomic claim CAS) |
| Degrade nhiều tầng | `services/store/cmd/main.go` (noop resolver/uploader khi downstream chết) |
| Docker | `docker-compose.yml`, `docker/services/Dockerfile` |

**Mục báo cáo:** §4, §5, §8, phần RBAC/gateway của §10, §13.

**Phải nói được:**
- Database-per-service ⇒ **không FK liên DB**, tham chiếu chéo bằng UUID mềm. **Trả giá**: mất ràng buộc toàn vẹn ở tầng DB, phải xử lý ở tầng ứng dụng.
- **Trade-off phải thừa nhận thẳng:** user-service là **SPOF** — mọi route protected của 9 service kia gọi gRPC `CheckPermission` mỗi request. Đổi lại: một nguồn sự thật RBAC duy nhất. Giảm nhẹ bằng cache 5 phút.
- Vì sao rate-limit **2 tầng**: 300/phút toàn cục, nhưng `/auth/*` và đăng ký shipper chỉ **10/phút** — chống brute-force mật khẩu/OTP.
- `/readyz` **cố ý không probe Kafka**: consumer tự retry, một nhịp lỗi broker không nên kéo service khỏi rotation.
- Sổ cái kép: hai bút toán tổng = 0, khoá `FOR UPDATE` **cả hai ví**; chỉ SYSTEM (clearing) được âm vì số âm là **nghĩa vụ phải trả lại**.

---

## Nam — Bên nhận & biên client *(phần ba)*

**Khái niệm sở hữu:** **luồng nghiệp vụ dùng MQ (fan-out phía consumer)** · **idempotent receiver → hiệu ứng exactly-once** · **versioning sự kiện** · CQRS read model · event-driven choreography · điều phối đồng thời phía client · kiểm thử.

**Ranh giới với Đức:** Đức sở hữu **8 service có đường ghi**. Hai service **chỉ tiêu thụ event** — `reporting` và `notification` — thuộc Nam, vì bản chất chúng là **bên nhận**: không có API ghi nghiệp vụ, toàn bộ trạng thái dựng từ sự kiện.

| Chủ đề | File |
|---|---|
| **Idempotent receiver** | bảng `processed_events` ở mọi service tiêu thụ (order, payment, promotion, delivery, review, reporting, notification) |
| **Versioning sự kiện** | `pkg/outbox/outbox.go` (`EnvelopeVersion`) + `envelope_version_test.go` |
| Xử lý sự kiện (handler) | `services/*/internal/handler/event/**` |
| CQRS read model | `services/reporting/**` — dựng **100% từ event**, không có API ghi |
| Fan-out choreography | `services/notification/**` — 7 topic → in-app/email/FCM |
| **Biên client** | `frontend/src/shared/lib/auth-refresh.ts` (**single-flight refresh**), `trace-id.ts`, `use-favorites.ts` (optimistic + rollback) |
| Toàn bộ frontend | `frontend/src/**` (Next.js, 15 feature slice) |
| Kiểm thử | 21 package, 264 test-case |

**Mục báo cáo:** §7 (phần nhận), §12, phần CQRS của §4/§8, phần frontend của §5.

**Phải nói được:**
- **Luồng nghiệp vụ dùng MQ** (yêu cầu bắt buộc #3): khi tài xế bấm *"Đã giao"*, sự kiện `order.delivered` khiến **Payment** ghi sổ COD, **Order** tự chuyển COMPLETED, **Review** mở khoá đánh giá, **Notification** báo khách — **bốn service phản ứng độc lập, không service nào gọi service nào**. Đây là giá trị cốt lõi của hàng đợi: decoupling + chịu lỗi từng phần.
- **Notification consume 7/9 topic** (thiếu `user.events` và `store.events`) — nói đúng con số, đừng nói "tất cả".
- **At-least-once + idempotent receiver = hiệu ứng exactly-once.** Kafka **không** cho exactly-once xuyên biên service — nó giao **ít nhất một lần**. Ta đạt hiệu ứng đúng-một-lần bằng bảng `processed_events` (PK = `event_id`) ghi **cùng transaction** với business write. Chứng minh bằng **Demo 3**.
- **Versioning sự kiện**: envelope mang trường `version`; envelope cũ (chưa có version) vẫn decode được ⇒ tiến hoá schema sự kiện mà **không phá vỡ consumer cũ**.
- **CQRS**: reporting là read model thuần, chấp nhận **nhất quán cuối** (thường < vài giây) để không làm chậm đường ghi.
- **Choreography vs orchestration**: notification/reporting phản ứng với sự kiện **mà không ai gọi chúng** — khác với saga (orchestration) nơi Order chỉ huy từng bước.
- **Single-flight refresh**: refresh token là **single-use**, nên N request cùng nhận 401 mà mỗi request tự gọi `/auth/refresh` sẽ **tự vô hiệu hoá lẫn nhau**. Frontend gom về **một** lời gọi duy nhất. Đây là bài toán **điều phối đồng thời ở biên client**.
- Bộ test chứng minh cơ chế phân tán: `place_order_compensation_test.go` (inject lỗi từng bước saga), `refresh_logout_test.go` (replay refresh token → 401), `envelope_version_test.go`.

---

## Bản đồ yêu cầu đề bài → ai nói

Đây là hai bảng §14 trong báo cáo Word. Mỗi ô gán **một người chịu trách nhiệm nói**, để không sót yêu cầu nào và không ai nói chồng lên nhau.

### Yêu cầu BẮT BUỘC (Bảng 7 — 7 dòng)

| # | Yêu cầu | Ta có gì | Ai nói |
|---|---|---|---|
| 1 | ≥ 3 microservice | **10** service | Đức |
| 2 | RESTful API đúng chuẩn | **180 endpoint** qua Kong | Đức |
| 3 | ≥ 1 luồng nghiệp vụ dùng MQ | Luồng `order.delivered` → **Payment, Order, Review, Notification cùng phản ứng độc lập** (9 topic, 21 consumer group) | **Nam** |
| 4 | DB riêng mỗi service | **10 DB**, auto-migrate | Đức |
| 5 | Xử lý lỗi (retry/log) | outbox retry · consumer retry + DLQ · gRPC timeout · circuit breaker · idempotent | **Văn** |
| 6 | Kiểm thử luồng liên service | **21 package, 264 test PASS, 0 FAIL** | Nam |
| 7 | Service chạy độc lập + phối hợp | Docker + saga + sự kiện | **Văn** |

**Văn 2 · Đức 3 · Nam 2**

> Dòng 3 là **fan-out phía consumer** — một sự kiện, bốn service phản ứng mà **không ai gọi ai**. Nam sở hữu bên nhận (`notification`, `reporting`) nên trình bày luồng này. Văn vẫn giữ toàn bộ **cơ chế** phía phát (outbox → Kafka → DLQ).

### Yêu cầu NÂNG CAO — cộng điểm (Bảng 8 — 14 dòng)

Bảng 8 trong Word **đã tách thành 14 dòng riêng** (trước đây 6 mục bị nhồi chung một ô — thầy chấm theo bảng sẽ lướt qua và chỉ tick một cái).

| # | Dòng trong Bảng 8 | Ai nói | Bằng chứng |
|---|---|---|---|
| 1 | **API Gateway** ✅ | Đức | Kong declarative, 46 route, ingress duy nhất |
| 2 | **Rate limiting** ✅ | Đức | 2 tầng: 300/phút · **10/phút** cho `/auth/*` |
| 3 | **Centralized logging** ✅ | **Văn** | EFK + **ILM tự xoá sau 7 ngày** |
| 4 | **Saga / event-driven** ✅ | **Văn** | **Demo 1** + **Demo 3** |
| 5 | **JWT & bảo mật xác thực** ✅ | **Văn** | RS256 chống algorithm-confusion · JTI whitelist **fail-closed** · rotation single-use |
| 6 | **Docker** ✅ | Đức | `docker compose up -d` từ **máy sạch** |
| 7 | **Service discovery** ◐ | **Văn** | Địa chỉ **config-driven** (sang K8s chỉ đổi DNS trong YAML). **Chưa** dùng registry động |
| 8 | **Retry policy** ✅ | **Văn** | Kafka retry 3 lần → DLQ; gRPC retry ×2 chỉ trên `Unavailable` + deadline 5s |
| 9 | **Health / readiness probe** ✅ | Đức | Dừng Postgres → `/readyz` **503**, `/health` vẫn **200** |
| 10 | **Graceful shutdown** ✅ | Đức | HTTP drain 15s → gRPC → consumer → flush OTel → DB/Redis |
| 11 | **Tracing giữa service** ✅ | **Văn** | 1 trace, **9 span, 5 service**. Tự nêu hạn chế: chưa nối qua biên Kafka |
| 12 | **Circuit breaker** ✅ | **Văn** | **Demo 2** — số đo thật **5.055ms → 25ms** |
| 13 | **Dashboard giám sát** ✅ | **Văn** | Prometheus 10/10 UP; Grafana 4 panel |
| 14 | **Kubernetes** ✗ | Đức | Nói thẳng: chỉ Docker Compose. Nhưng auto-migrate dùng **advisory lock** ⇒ sẵn sàng scale |

**Văn 8 · Đức 6 · Nam 0**

> ⚠️ **Nam không có dòng nào ở Bảng 8.** Nếu thầy đi từng dòng của bảng nâng cao để hỏi, Nam sẽ **im lặng suốt**. Xem phần "Rủi ro còn lại" ở cuối để chọn cách khắc phục.
>
> Dòng 8 (**Retry policy**) là chỗ bàn giao: Văn nói **retry của gRPC client** và **retry → DLQ của consumer framework**; nếu thầy hỏi sâu *"message vào DLQ rồi thì sao"* hoặc *"redeliver thì có nhân đôi không"*, Nam tiếp lời bằng phần khử trùng lặp.

### Vượt khung đề bài — bảng riêng trong §14 (12 dòng)

Trước đây các cơ chế mạnh nhất **nằm rải trong thân báo cáo** nhưng **không có mặt ở §14** — đúng cái bảng thầy dùng để tick barem. Giờ chúng có bảng riêng ngay trong §14.

| Cơ chế | Ai nói | Câu chốt |
|---|---|---|
| **Sub-transaction barrier** | **Văn** | Chống **null compensation** + **duplicate branch call**; dòng chặn ghi **cùng transaction** với business write |
| **Bù trừ bền vững** | **Văn** | *"Nếu chính bước rollback cũng lỗi thì sao?"* → `pending_compensations`, worker quét 30s, tối đa 20 lần |
| **Bù trừ theo thời gian** | **Văn** | Janitor void voucher `RESERVED` mồ côi sau 15 phút |
| **Bù trừ một bước đã commit** | **Văn** | Huỷ đơn đã thành công vẫn void được usage **CONFIRMED** |
| **Bảo toàn thứ tự partition** | **Văn** | Không park được DLQ thì **cố ý không commit**, dừng ở message đó |
| **Fail-closed authorization** | **Văn** | Whitelist chứ không blacklist ⇒ Redis chết thì từ chối hết |
| **Sổ cái kép** | Đức | Hai bút toán tổng = 0; SYSTEM được âm vì là tài khoản clearing |
| **Chống tranh chấp** | Đức | `FOR UPDATE` siết `usage_limit`; atomic claim (CAS) cho shipper; payout loại đơn đã trong batch |
| **Idempotent receiver → exactly-once** | **Nam** | **Demo 3**: tua offset về 0, giao lại 8 event, thông báo vẫn **8 chứ không phải 16** |
| **Versioning sự kiện** | **Nam** | Envelope có `version`; envelope cũ vẫn decode được ⇒ không phá consumer cũ |
| **CQRS read model** | Nam | Reporting dựng **100% từ event**, không có API ghi |
| **Single-flight refresh** | Nam | Refresh token single-use ⇒ N request 401 song song **tự huỷ nhau** |

**Văn 6 · Đức 2 · Nam 4**

### Tổng kết phân bổ

| | Bảng 7 | Bảng 8 | Vượt khung | **Tổng** |
|---|---|---|---|---|
| **Văn** | 2 | 8 | 6 | **16** |
| **Đức** | 3 | 6 | 2 | **11** |
| **Nam** | 2 | 0 | 4 | **6** |

Thứ tự đúng ý: **Văn > Đức > Nam**.

### Lưu ý: Nam không có dòng nào ở Bảng 8

Đây là hệ quả của việc dồn toàn bộ tầng quan sát (tracing, logging, dashboard) về Văn. Nam bù lại bằng **2 dòng Bảng 7** (luồng MQ + kiểm thử), **4 mục Vượt khung** (exactly-once, versioning, CQRS, single-flight), **2 demo** (Demo 3, Demo 6), toàn bộ frontend và bộ test 264 case.

**Cách khắc phục trong lúc bảo vệ:** khi Văn trình bày **dòng 8 Bảng 8 (Retry policy)** và nói xong phần *"hết retry thì message vào DLQ"*, **Nam tiếp lời ngay**:

> *"Và khi Kafka giao lại message — vì nó chỉ đảm bảo at-least-once — thì `processed_events` chặn trùng, nên hiệu ứng nghiệp vụ vẫn đúng một lần. Em có demo tua offset về 0 để chứng minh."*

Câu đó vừa kéo Nam vào Bảng 8, vừa mở đường tự nhiên sang **Demo 3**.

---

## Kịch bản demo — mỗi pha hỏng là một bằng chứng

**Không demo happy path. Demo lúc hệ thống gãy.**

### Demo 0 — Happy path (5 phút, cả nhóm)

Đăng nhập `cust@velox.test` / `Password123!` → chọn quán → thêm món → checkout (Toà A → Tầng 1 → P101, voucher `SAGA10`, trả bằng Ví) → đặt hàng → trang đơn hiện "Đã thanh toán".

Mở song song: **DTM UI** (`:17789`) → saga `succeed` 3 nhánh. **Jaeger** (`:17093`) → 1 trace 9 span xuyên 5 service.

### Demo 1 — Văn: Saga bù trừ ngược *(đã chạy thật)*

**Dựng lỗi:** đặt đơn bằng **Ví** với số tiền **vượt số dư** (xem số dư ở `/account/wallet` trước).

**Kết quả:** khách nhận `400` — *"thanh toán thất bại"*. Mở **DTM UI**, dòng mới nhất `status = failed`:

| branch | op | status | RPC |
|---|---|---|---|
| 01 | action | **succeed** | `ApplyPromotion` |
| 01 | compensate | **succeed** | `ReleaseUsage` ← **đã chạy** |
| 02 | action | **failed** | `Capture` |
| 02 | compensate | **succeed** | `Refund` ← **đã chạy** (no-op: chưa trừ tiền nào) |
| 03 | action | `prepared` | `ConfirmUsage` ← **chưa từng chạy** |
| 03 | compensate | `prepared` | `ReleaseUsage` |

**Chốt hạ:** ví **không đổi**, voucher `used_count` **không tăng**, **không đẻ ra đơn**, sổ cái **không có bút toán mới**.

**Câu quan trọng nhất:** compensate của nhánh 02 (`Refund`) `succeed` **dù chưa hề trừ tiền** — đó là **null compensation**, và nó **bắt buộc phải là no-op idempotent**, không được báo lỗi.

```bash
docker compose exec -T postgres psql -U dev_user -d dtm_db -c \
  "select b.branch_id, b.op, b.status, split_part(b.url,'/',3) as rpc
   from trans_branch_op b join trans_global g on g.gid=b.gid
   where g.id=(select max(id) from trans_global) order by b.id;"
```

### Demo 2 — Văn: Circuit breaker + fail-fast *(đã chạy thật)*

**PHẢI dừng đúng service.** Với engine DTM, order-service **KHÔNG gọi thẳng payment** (DTM server mới gọi participant). Dừng payment sẽ **không** làm breaker trip. Phải dừng **store-service** — đó là lời gọi gRPC trực tiếp của order (`GetStoreForOrder`).

**Dựng lỗi:** `docker compose stop store-service` rồi bắn liên tiếp **≥5 request** đặt hàng (breaker cần `≥5 request/30s` mới trip — bấm 1–2 lần sẽ **không** thấy gì, demo trông như hỏng).

| Request | HTTP | Độ trễ | `circuit_breaker_state` |
|---|---|---|---|
| #1 | 400 | **5.055 ms** | chưa set |
| #2 | 400 | 3.432 ms | chưa set |
| #3–#4 | 400 | ~360 ms | chưa set |
| **#5** | 400 | 360 ms | **2 = OPEN** |
| #6–#8 | 400 | **25 ms** | 2 |

Hai chỗ chỉ tay:
- **#1 mất đúng 5s** → `deadline` 5s cắt, **không** phải TCP treo vô hạn. Fail-fast.
- **Từ #6 chỉ còn 25ms** → mạch mở, request bị **từ chối tức thì**, không đánh vào service chết nữa. Độ trễ giảm **200 lần**.

**Phục hồi:** `docker compose start store-service` → vẫn `2` (chưa hết Timeout 15s) → qua mốc → `1` half-open → 3 probe thành công (HTTP **201**, đơn tạo thật) → `0` closed.

**Cảnh báo Grafana:** gauge `circuit_breaker_state` **chỉ publish khi breaker ĐỔI trạng thái**. Lúc `closed` ban đầu **không có mẫu nào** ⇒ panel **trống trơn**. Mở panel *sau* khi đã bắn đủ 5 request.

### Demo 3 — Nam: Idempotent receiver *(đã chạy thật)*

**Tầng 1 — decoupling:** `docker compose stop notification-service` → đặt 1 đơn → **đơn vẫn tạo thành công (201)** dù consumer đã chết. Bật lại → consumer bắt kịp từ offset cũ.

**Tầng 2 — idempotency (đây mới là bằng chứng thật):** tầng 1 chỉ chứng minh *catch-up*. Muốn chứng minh idempotency phải **ép Kafka giao lại toàn bộ**:

```bash
docker compose stop notification-service
docker compose exec -T kafka /opt/kafka/bin/kafka-consumer-groups.sh \
  --bootstrap-server localhost:9092 \
  --group notification-service-order --topic order.events \
  --reset-offsets --to-earliest --execute
docker compose start notification-service
```

| | Trước | Sau khi giao lại toàn bộ |
|---|---|---|
| `notifications` | 8 | **8** ← nếu KHÔNG idempotent thì phải là **16** |
| `processed_events` | 8 | **8** |

**Câu chốt:** Kafka giao **ít nhất một lần** — nó đã giao lại đúng 8 event. Nhưng hiệu ứng nghiệp vụ vẫn **đúng một lần**, nhờ `processed_events` ghi cùng transaction với business write.

### Demo 4 — Đức: Readiness + degrade nhiều tầng

**Dựng lỗi 1:** `docker compose stop postgres` → `curl localhost:17083/readyz` trả **503**, nhưng `/health` vẫn **200**. Giải thích: liveness (process sống) khác readiness (sẵn sàng nhận traffic) — orchestrator dùng readyz để rút service khỏi rotation mà **không** giết process.

**Dựng lỗi 2:** `docker compose stop location-service` → store-service **không sập**, chỉ degrade: ship-fee resolver rơi về `noop`, trả "không phục vụ" thay vì 5xx. Giải thích **graceful degradation**.

### Demo 5 — Văn: Quan sát xuyên hệ (trace + dashboard + log)

Chạy **ngay sau Demo 2** để panel breaker có dữ liệu.

**Trace:** Jaeger (`:17093`) → `order-service` → `POST /api/v1/orders` → **1 trace, 9 span, 5 service** (`order → store → location`, `order → promotion`, `order → payment`). Chỉ ra: một request của khách chạm 5 service, và ta **nhìn thấy toàn bộ** thay vì đoán.

**Dashboard:** Grafana (`:17096`) → 4 panel. Panel **Circuit breaker state** vẽ đủ 3 trạng thái vừa tạo ra ở Demo 2 (closed → open → half-open → closed).

**Log:** Kibana → lọc theo `trace_id` của chính request vừa trace → thấy log của **cả 5 service** trong một truy vấn. Đây là lý do phải lan truyền trace-id.

**Hạn chế tự nêu:** ngữ cảnh trace **chưa nối qua biên Kafka** — span consumer là span gốc mới, chỉ liên kết qua `trace_id`.

### Demo 6 — Nam: CQRS read model

Đặt 1 đơn → chờ vài giây → mở `/admin` → số liệu analytics **đã cập nhật**, dù **reporting-service không hề có API ghi nghiệp vụ nào**. Toàn bộ read model dựng từ event.

**Câu chốt:** đây là **nhất quán cuối** trong thực tế — có độ trễ vài giây, đổi lại đường ghi không bị chậm bởi việc tính toán báo cáo. Và nếu muốn dựng lại read model từ đầu, chỉ cần replay event.

---

## Chuẩn bị trước buổi bảo vệ

```bash
cd backend
make jwt-keys                              # BẮT BUỘC — thiếu là auth tắt, mọi route protected trả 404
docker compose up -d                       # 10 service + Postgres/Redis/Kafka/DTM/MinIO/Kong
docker compose --profile monitoring up -d  # Prometheus :17095, Grafana :17096, Jaeger :17093
go run ./services/store/cmd/seed-catalog   # 20 quán, ~950 món
cd ../frontend && npm run dev              # :17070
```

> **Không chạy trộn Docker và `go run`.** `config/*.docker.yaml` địa chỉ hoá DTM và saga branch theo DNS của compose, còn `config/*.yaml` dùng `localhost` — trộn hai chế độ sẽ khiến DTM không gọi được vào nhánh và saga gãy.

**Tab mở sẵn:** Frontend `:17070` · Kong `:17000` · **DTM UI `:17789`** · **Jaeger `:17093`** · **Grafana `:17096`** · Prometheus `:17095`

**Tài khoản seed** (mật khẩu chung `Password123!`): `cust@` · `shop@` · `ship@` · `admin@` · `super-admin@` — tất cả `@velox.test`.

---

## Ba việc phải chốt, không thì mất điểm oan

1. **RabbitMQ — đừng ai nhận.** `pkg/messaging/rabbitmq` có code nhưng **0 call site**; toàn hệ chạy Kafka. Nếu thầy hỏi *"RabbitMQ dùng ở luồng nào"* thì không có câu trả lời. Hoặc xoá hẳn, hoặc khai đúng như §15 đang ghi: *đã hiện thực, chưa wire, để dành mở rộng*.

2. **Chia commit theo đúng ranh giới trên.** Nếu thầy nhìn `git log` để đánh giá đóng góp cá nhân, mọi thứ dồn vào một commit của một người là hỏng.

3. **Ai cũng phải trả lời được câu hỏi liên vùng.** Thầy hay hỏi chéo. Tối thiểu:
   - Nam biết: *saga là gì, vì sao không dùng 2PC.*
   - Đức biết: *vì sao at-least-once + idempotent receiver cho ra hiệu ứng exactly-once.*
   - Văn biết: *vì sao user-service là SPOF và ta chấp nhận cái giá đó.*

## Câu hỏi chưa giải quyết

- Có commit sẵn keypair JWT demo không? Hiện `secrets/` bị gitignore ⇒ người chấm **bắt buộc** chạy `make jwt-keys` trước.
- Dữ liệu seed demo (hiện đã có 8 đơn thật, offset Kafka đã bị tua) — giữ hay dọn sạch trước khi nộp?
- MoMo: có dựng địa chỉ công khai (ngrok) để demo trọn vòng IPN không?
