# VeloxFood — Smart Canteen Platform

Nền tảng đặt món **căng-tin đa nhà hàng** (multi-vendor) cho khuôn viên trường — một giỏ hàng, nhiều nhà hàng. Đây là đồ án môn **Hệ thống phân tán**: hiện thực hoá các chủ đề RPC/gRPC, messaging, saga/outbox, RBAC, observability… trong một hệ microservice thực tế.

Repo gồm 2 phần:

| Thư mục | Vai trò |
|---------|---------|
| `backend/` | Microservices Go (Go module `project`). Hiện có **User Service**: xác thực, RBAC động, hồ sơ người dùng, onboarding nhà hàng, quản trị. REST (Gin) + gRPC. |
| `frontend/` | Web app **Next.js** (App Router): storefront cho khách + dashboard quản trị, dùng chung luồng xác thực qua cookie. |

> Xác thực dùng **JWT RS256** + **cookie httpOnly do backend phát hành** (access + refresh), refresh single-use có Redis whitelist; frontend gọi `/api/*` same-origin qua Next.js rewrite nên không cần CORS ở dev. Chi tiết API User Service xem [`backend/docs/user-service.md`](backend/docs/user-service.md).

---

## Công nghệ sử dụng

### Backend (`backend/`)
- **Go 1.24**, kiến trúc clean (entity → repository → usecase → handler).
- **Gin** (REST) + **gRPC** (`google.golang.org/grpc`), Protobuf.
- **PostgreSQL** (GORM) + **Redis** (cache-aside + JWT jti whitelist).
- **Kafka** (`segmentio/kafka-go`) + **RabbitMQ** (`amqp091-go`) cho messaging; **DTM** cho saga/2PC.
- **Transactional outbox** dispatcher; **JWT RS256** (`golang-jwt/jwt/v5`); **bcrypt**; SMTP mailer (OTP/email).
- Observability: `slog` có trace-id, ship log qua **EFK** (Elasticsearch + Kibana + Fluentd).
- Migrations: `golang-migrate`; config: **Viper** (`config/config.yaml`).

### Frontend (`frontend/`)
- **Next.js 16** (App Router) + **React 19** + **TypeScript**.
- **axios** (interceptor trace-id + auto-refresh 401), **TanStack React Query** (session/state).
- **react-hook-form** + **zod** (form & validation).
- **Tailwind CSS v4** + **shadcn/ui** (Radix), **lucide-react**, **sonner**, **next-themes**.
- Test: **Vitest** + Testing Library.

### Hạ tầng dev (Docker)
PostgreSQL · Redis · Kafka · RabbitMQ · DTM (+ tuỳ chọn EFK logging stack).

---

## Chạy môi trường dev

### Yêu cầu
- **Go 1.24+**, **Node.js 20+** (npm), **Docker** + Docker Compose, **openssl**, `make`.

### 1) Khởi động hạ tầng (Docker)

Tất cả lệnh backend chạy trong thư mục `backend/`.

```bash
cd backend

# Chỉ hạ tầng (khuyến nghị khi chạy backend local để bật được auth):
docker compose up -d postgres redis kafka rabbitmq dtm

# Hoặc bật toàn bộ kể cả user-service container:
make docker-up        # docker compose up -d
make docker-down      # dừng (giữ volume)
make docker-reset     # xoá volume + dựng lại
```

> Lưu ý: container `user-service` trong compose **không kèm khoá JWT** nên route auth bị tắt. Để test đầy đủ luồng xác thực, hãy **chạy backend ở local** (mục 2) và bỏ qua / dừng container service: `docker compose stop user-service`.

### 2) Chạy User Service (local)

```bash
cd backend

make jwt-keys     # sinh secrets/jwt_private.pem + jwt_public.pem (RS256)
make migrate-up   # chạy migrations (đọc DB từ config/config.yaml)
make run          # go run ./services/user/cmd  → HTTP :8080, gRPC :50051
```

Backend đọc `config/config.yaml` (trỏ tới các cổng host đã map: Postgres 5433, Redis 6380, Kafka 19092). Kiểm tra: `curl http://localhost:8080/health` → `200`.

> SMTP trong `config/config.yaml` để trống credential → OTP đăng ký **không gửi được email thật**. Khi cần test register, điền `smtp.username/password` rồi chạy lại.

### 3) Chạy Frontend

```bash
cd frontend
cp .env.local.example .env.local   # tuỳ chọn; mặc định proxy /api → http://localhost:8080
npm install
npm run dev                        # http://localhost:3000
```

Next.js rewrite `/api/*` → backend `:8080` (same-origin, không vướng CORS). Mở **http://localhost:3000**, đăng nhập → cookie `access_token`/`refresh_token` (HttpOnly) do backend set.

### Cổng dịch vụ

| Dịch vụ | Host | Container |
|---------|------|-----------|
| Frontend (Next.js) | 3000 | — |
| User Service HTTP | 8080 | 8080 |
| User Service gRPC | 50051 | 50051 |
| PostgreSQL | 5433 | 5432 |
| Redis | 6380 | 6379 |
| Kafka | 19092 | 19092 |
| RabbitMQ AMQP / UI | 5672 / 15672 | 5672 / 15672 |
| DTM gRPC / HTTP | 36790 / 36789 | 36790 / 36789 |

---

## Lệnh thường dùng

**Backend** (`backend/`):
```bash
make build        # bin/user-service
make test         # go test ./... -v -cover
make lint         # golangci-lint
make proto        # sinh *.pb.go
make migrate-up | migrate-down | migrate-version
make efk-up       # bật stack logging (Kibana :5601)
```

**Frontend** (`frontend/`):
```bash
npm run dev | build | start
npm run lint | type-check
npm run test          # vitest run
npm run format
```

---

## Tài liệu
- API & kiến trúc User Service: [`backend/docs/user-service.md`](backend/docs/user-service.md)
