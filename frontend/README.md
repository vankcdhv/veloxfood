# Frontend — Distributed System

Frontend cho project **Hệ thống phân tán** — Next.js 16 App Router + React 19 + Tailwind v4, consume backend Go microservices qua REST `/api/v1/*` với trace-id propagation tới EFK.

## Stack

| Layer | Library |
|-------|---------|
| Framework | Next.js 16.2.6 (App Router) + React 19.2.4 |
| Styling | Tailwind CSS v4 + shadcn/ui (manual copy) |
| Data fetching | TanStack Query v5 + Axios (interceptor inject `X-Trace-Id`) |
| Forms | React Hook Form + Zod (`@hookform/resolvers`) |
| Testing | Vitest + Testing Library |
| Lint/Format | ESLint 9 (`eslint-config-next`) + Prettier + `prettier-plugin-tailwindcss` |

## Folder structure (Feature-Sliced lite)

```
src/
├── app/                          # Next.js routes + providers
│   ├── layout.tsx
│   ├── page.tsx                  # Landing
│   ├── providers.tsx             # QueryClientProvider wrapper
│   ├── globals.css               # Tailwind v4 + shadcn CSS vars
│   └── (dashboard)/users/page.tsx
├── features/                     # Self-contained feature modules
│   └── users/
│       ├── api/                  # Backend calls
│       ├── components/           # Feature UI
│       ├── hooks/                # TanStack Query hooks
│       ├── schemas/              # Zod schemas
│       └── types/                # TS types mirror backend
├── widgets/                      # Composed UI dùng nhiều route
│   └── header/
└── shared/
    ├── ui/                       # shadcn primitives
    ├── lib/                      # http-client, utils, env, trace-id
    ├── api/                      # QueryClient factory, ApiResponse types
    ├── config/                   # Constants, routes
    └── test/                     # Vitest setup
```

**Dependency rule:** `app → widgets → features → shared`. Features không import lẫn nhau (cross-feature qua `shared/` hoặc `widgets/`). `shared/` không import lên trên.

## Quick start

```bash
# 1. Copy env example
cp .env.local.example .env.local

# 2. Install + run
npm install
npm run dev                # http://localhost:3000

# 3. Backend đang chạy?
cd ../backend && make docker-up   # Postgres :5433, user-service :8080
```

Truy cập http://localhost:3000/users để thấy demo end-to-end (TanStack Query → Axios → backend `GET /api/v1/users` với `X-Trace-Id`).

## Scripts

```bash
npm run dev               # Dev server
npm run build             # Production build
npm run start             # Serve production build
npm run lint              # ESLint
npm run type-check        # tsc --noEmit
npm test                  # Vitest (run mode)
npm run test:watch        # Vitest watch
npm run format            # Prettier write
npm run format:check      # Prettier check
```

## Trace-ID propagation

- Axios request interceptor (`src/shared/lib/http-client.ts`) tự generate UUID v4 và set header `X-Trace-Id` mỗi request.
- Backend middleware (`pkg/middleware.RequestLogging`) extract header → propagate qua context → slog auto-inject `trace_id` vào log → Fluentd ship lên Kibana.
- Verify: mở DevTools Network, request `GET /api/v1/users` → header `X-Trace-Id: <uuid>`. Search trong Kibana `service-logs-*` theo `trace_id` thấy cùng id.

## Path aliases

```ts
import { Button } from '@/shared/ui/button';
import { useUsers } from '@/features/users/hooks/use-users';
import { Header } from '@/widgets/header/header';
```

## Notes

- Tailwind v4 dùng `@theme inline` (CSS vars syntax mới) — không có `tailwind.config.ts`.
- shadcn/ui được copy thủ công (CLI v0 chưa stable với Tailwind v4) — components nằm trong `src/shared/ui/`.
- Auth chưa setup (backend chưa expose `/auth/login`); `http-client.ts` đã chừa chỗ inject `Authorization` header.
