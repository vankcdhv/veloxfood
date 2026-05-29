# Design System — Food Ordering Platform

> Tài liệu base UI cho frontend project Hệ thống phân tán.
> **Source of truth chi tiết:** [`../design-system/food-ordering-platform/MASTER.md`](../design-system/food-ordering-platform/MASTER.md)
> File ở đây tóm tắt cách design system được hiện thực hoá trong code.

---

## Định hướng

| | |
|---|---|
| **Product type** | Cả admin dashboard + SaaS end-user (food ordering) |
| **Pattern** | Vibrant & Block-based, ấm áp kích thích thèm ăn |
| **Theme mode** | Light (mặc định) + Dark — `next-themes`, class-based |

## Color tokens

Map sang Tailwind v4 CSS variables trong `src/app/globals.css` (dùng `oklch` cho gamut rộng + smooth dark mode).

| Token | Light | Dark | Use |
|-------|-------|------|-----|
| `--primary` / `bg-primary` | `#EA580C` orange-600 | brighter orange | CTA chính, nav active |
| `--secondary` / `bg-secondary` | `#F97316` orange-500 | warm orange | Accent ấm, badge |
| `--accent` / `bg-accent` | `#2563EB` blue-600 | lighter blue | Trust CTA (đặt, xác nhận) |
| `--background` | `#FFF7ED` cream | warm dark brown-slate | Body |
| `--foreground` | slate-900 | warm off-white | Text |
| `--muted` | `#FDF4F0` | dark muted | Subtle bg, hover |
| `--success` | emerald-600 | lighter emerald | "Đã giao", healthy state |
| `--warning` | amber-500 | lighter amber | "Chờ xác nhận" |
| `--destructive` | red-600 | brighter red | "Hủy đơn", danger |
| `--border` / `--input` | warm cream border | warm dark border | Divider, input |

Mỗi token có cặp `*-foreground` để đảm bảo contrast ≥ 4.5:1.

## Typography

| Role | Font | Variable | Tailwind class |
|------|------|----------|----------------|
| Heading | **Playfair Display SC** | `--font-playfair` | `font-serif` |
| Body | **Karla** | `--font-karla` | `font-sans` (default) |
| Mono | system mono stack | — | `font-mono` |

Load qua `next/font/google` trong `src/app/layout.tsx` (no FOIT, auto-swap, latin + latin-ext).

Heading mood: restaurant, menu, culinary, elegant, foodie.

## Spacing & shadow

- Spacing: theo Tailwind v4 scale `4/8/16/24/32/48/64` (rhythm 4–8px).
- Radius: `--radius: 0.75rem` (cards), `sm/md/lg/xl` tính từ radius gốc.
- Shadow: warm-tinted (alpha trên slate-900) — `shadow-sm/md/lg/xl`.

## Theme switching

- `next-themes` class-based (`<html class="dark">`).
- `ThemeProvider` ở `src/shared/providers/theme-provider.tsx` — `defaultTheme="light"`, `enableSystem`, `disableTransitionOnChange`.
- Toggle UI: `src/shared/ui/theme-toggle.tsx` — dropdown Sun/Moon/Monitor.
- Hydration-safe: dùng `useSyncExternalStore` cho mount detection (tránh `set-state-in-effect`).

## Component primitives (shadcn manual copy, Tailwind v4 compat)

| File | Vai trò |
|------|---------|
| `button.tsx` | 7 variants (default/accent/secondary/outline/ghost/destructive/link), 4 sizes, `asChild` |
| `input.tsx` | Form input + focus ring |
| `label.tsx` | Radix Label, semantic `<label htmlFor>` |
| `card.tsx` | Card + Header/Title/Description/Content/Footer, hover-shadow |
| `badge.tsx` | 7 variants (default/secondary/accent/success/warning/destructive/outline) |
| `separator.tsx` | Radix Separator, horizontal/vertical |
| `skeleton.tsx` | Animated loading placeholder |
| `avatar.tsx` | Radix Avatar + Image + Fallback |
| `dropdown-menu.tsx` | Full Radix DropdownMenu API |
| `dialog.tsx` | Radix Dialog, backdrop blur, ESC + close button |
| `sheet.tsx` | Side sheet (top/bottom/left/right) cho mobile nav |
| `theme-toggle.tsx` | Light/Dark/System switcher |

Anti-pattern checklist (đã enforce qua tokens & utilities):
- Không emoji làm icon → dùng `lucide-react`.
- Tất cả interactive có `cursor-pointer` + `focus-visible:ring-2`.
- Transition 200ms cho hover (theo MASTER.md range 150–300ms).
- `prefers-reduced-motion` được respect qua `@media` global trong `globals.css`.

## Layout architecture

```
src/app/
├── layout.tsx              # Root: fonts + ThemeProvider + QueryClient
├── page.tsx                # Landing + UI kit showcase
├── shop/                   # End-user layout (header + footer ấm)
│   ├── layout.tsx          # ShopHeader + ShopFooter
│   └── page.tsx            # Shop home hero + collections
└── admin/                  # Admin layout (sidebar + topbar)
    ├── layout.tsx          # AdminSidebar (sticky)
    ├── page.tsx            # Dashboard tổng quan + metrics
    └── users/page.tsx      # Quản lý người dùng (consume backend)
```

## Widgets shell

| Widget | File | Vai trò |
|--------|------|---------|
| `BrandMark` | `widgets/brand-mark/` | Logo + tên brand "VeloxFood" — dùng chung cả 2 layout |
| `ShopHeader` | `widgets/shop-header/` | Top nav shop, badge giỏ hàng, mobile Sheet, CTA "Đăng nhập" |
| `ShopFooter` | `widgets/shop-footer/` | 4-col footer + brand + copyright |
| `AdminSidebar` | `widgets/admin-sidebar/` | Sticky sidebar 64w, primary + secondary nav |
| `AdminTopbar` | `widgets/admin-topbar/` | Page title + search + avatar + theme toggle |
| `AuthHero` | `widgets/auth-hero/` | Left panel split-screen auth: gradient orange→amber, brand invert, tagline + 3 highlights |

## Auth pages

Routes nằm trong route group `src/app/(auth)/` (không xuất hiện trên URL):

| Path | Page | Form |
|------|------|------|
| `/login` | Đăng nhập | `LoginForm` (email + password + remember) |
| `/register` | Tạo tài khoản | `RegisterForm` (5 fields + đồng ý điều khoản) |
| `/forgot-password` | Quên mật khẩu | `ForgotPasswordForm` (email) |
| `/reset-password?token=...` | Đặt lại mật khẩu | `ResetPasswordForm` (password + confirm) |

**Layout:** split-screen `lg:grid-cols-2` — trái `AuthHero` (gradient ấm, ẩn `<lg`), phải form column với theme toggle góc trên + brand mark compact mobile + form card max-w-md.

**Form rules:**
- RHF + Zod (`features/auth/schemas/auth-schema.ts`), validate `onBlur`.
- Password rule (register/reset): min 8 + chữ hoa + số. Login chỉ check non-empty.
- `PasswordInput` (Eye/EyeOff toggle) ở `shared/ui/password-input.tsx`.
- Submit chưa gọi API thật — chạy `notifyComingSoon(flow)` (toast info "Tính năng đang phát triển") tại `features/auth/lib/notify-coming-soon.ts`. Sonner mount ở `app/providers.tsx`.
- Reset-password thiếu `?token=` → render error block + CTA "Yêu cầu link mới".

**Khi backend `/auth/*` sẵn sàng**, swap `notifyComingSoon()` thành mutation TanStack Query gọi `httpClient.post('/api/v1/auth/...')`. UI và validation giữ nguyên.

## Persistent design system file

Khi build page mới, kiểm tra theo workflow của ui-ux-pro-max skill:
1. `design-system/food-ordering-platform/pages/<page-slug>.md` (nếu có) **override** Master.
2. Nếu không, follow `design-system/food-ordering-platform/MASTER.md`.

## Pre-delivery checklist (per page)

- [ ] Contrast ≥ 4.5:1 (light + dark)
- [ ] Touch target ≥ 44px (mobile)
- [ ] Focus ring visible
- [ ] No horizontal scroll @ 375px
- [ ] Loading state có Skeleton
- [ ] Empty state có message + CTA
- [ ] Form: label + helper text + error position rõ ràng
- [ ] No emoji as icon (dùng `lucide-react`)
- [ ] `prefers-reduced-motion` ok
