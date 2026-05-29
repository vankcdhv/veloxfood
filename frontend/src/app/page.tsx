import Link from 'next/link';
import { ArrowRight, ChefHat, LayoutDashboard, Palette } from 'lucide-react';
import { Button } from '@/shared/ui/button';
import { Badge } from '@/shared/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/card';
import { ThemeToggle } from '@/shared/ui/theme-toggle';
import { BrandMark } from '@/widgets/brand-mark/brand-mark';
import { ROUTES } from '@/shared/config/constants';

const ENTRY_POINTS = [
  {
    title: 'Shop (End-user)',
    description:
      'Trải nghiệm khách hàng — duyệt thực đơn, đặt món, theo dõi đơn. Layout có header + footer ấm áp.',
    href: ROUTES.shop.root,
    icon: ChefHat,
    cta: 'Vào shop',
  },
  {
    title: 'Admin Dashboard',
    description:
      'Bảng điều khiển vận hành — quản lý đơn hàng, sản phẩm, người dùng. Layout có sidebar + topbar.',
    href: ROUTES.admin.root,
    icon: LayoutDashboard,
    cta: 'Vào dashboard',
  },
  {
    title: 'UI Kit Showcase',
    description: 'Xem nhanh design tokens, typography, màu sắc, components base đã build.',
    href: '#ui-kit',
    icon: Palette,
    cta: 'Xem UI kit',
  },
] as const;

const PALETTE = [
  { name: 'Primary', token: 'bg-primary', text: 'text-primary-foreground', hex: '#EA580C' },
  {
    name: 'Secondary',
    token: 'bg-secondary',
    text: 'text-secondary-foreground',
    hex: '#F97316',
  },
  { name: 'Accent', token: 'bg-accent', text: 'text-accent-foreground', hex: '#2563EB' },
  { name: 'Muted', token: 'bg-muted', text: 'text-muted-foreground', hex: '#FDF4F0' },
  { name: 'Success', token: 'bg-success', text: 'text-success-foreground', hex: '#0E9F6E' },
  { name: 'Warning', token: 'bg-warning', text: 'text-warning-foreground', hex: '#F59E0B' },
  {
    name: 'Destructive',
    token: 'bg-destructive',
    text: 'text-destructive-foreground',
    hex: '#DC2626',
  },
  {
    name: 'Foreground',
    token: 'bg-foreground',
    text: 'text-background',
    hex: '#0F172A',
  },
];

export default function HomePage() {
  return (
    <div className="min-h-screen">
      <header className="border-border bg-background/80 sticky top-0 z-30 border-b backdrop-blur-md">
        <div className="mx-auto flex h-16 max-w-6xl items-center justify-between px-4 sm:px-6">
          <BrandMark href={ROUTES.home} />
          <div className="flex items-center gap-2">
            <Button asChild variant="ghost" size="sm">
              <Link href={ROUTES.shop.root}>Shop</Link>
            </Button>
            <Button asChild variant="ghost" size="sm">
              <Link href={ROUTES.admin.root}>Admin</Link>
            </Button>
            <ThemeToggle />
          </div>
        </div>
      </header>

      <section className="mx-auto max-w-6xl px-4 py-16 sm:px-6 sm:py-24">
        <div className="flex flex-col items-start gap-6">
          <Badge variant="accent">Base UI Kit · v0.1.0</Badge>
          <h1 className="font-serif text-4xl leading-tight font-bold tracking-tight sm:text-5xl lg:text-6xl">
            Một nền tảng đặt món
            <br />
            <span className="text-primary">ấm áp</span> và{' '}
            <span className="text-accent">đáng tin cậy</span>.
          </h1>
          <p className="text-muted-foreground max-w-2xl text-lg">
            Frontend base cho project Hệ thống phân tán — Next.js 16 · React 19 · Tailwind v4 ·
            shadcn/ui. Hai layout (shop end-user + admin dashboard) cùng một design system ấm áp lấy
            cảm hứng từ ẩm thực.
          </p>
          <div className="flex flex-wrap gap-3">
            <Button asChild size="lg">
              <Link href={ROUTES.shop.root}>
                Vào shop
                <ArrowRight className="h-4 w-4" />
              </Link>
            </Button>
            <Button asChild size="lg" variant="outline">
              <Link href={ROUTES.admin.root}>Vào dashboard</Link>
            </Button>
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-6xl px-4 pb-16 sm:px-6 sm:pb-24">
        <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
          {ENTRY_POINTS.map((item) => {
            const Icon = item.icon;
            return (
              <Card key={item.title} className="flex flex-col">
                <CardHeader>
                  <span className="bg-primary/10 text-primary inline-flex h-10 w-10 items-center justify-center rounded-lg">
                    <Icon className="h-5 w-5" />
                  </span>
                  <CardTitle className="mt-3">{item.title}</CardTitle>
                  <CardDescription>{item.description}</CardDescription>
                </CardHeader>
                <CardContent className="mt-auto">
                  <Button asChild variant="ghost" className="px-0">
                    <Link href={item.href}>
                      {item.cta}
                      <ArrowRight className="h-4 w-4" />
                    </Link>
                  </Button>
                </CardContent>
              </Card>
            );
          })}
        </div>
      </section>

      <section id="ui-kit" className="border-border border-t">
        <div className="mx-auto max-w-6xl px-4 py-16 sm:px-6 sm:py-20">
          <div className="mb-10">
            <h2 className="font-serif text-3xl font-bold tracking-tight sm:text-4xl">UI Kit</h2>
            <p className="text-muted-foreground mt-2 max-w-2xl">
              Design tokens & components base — toàn bộ feature page sẽ kế thừa từ đây.
            </p>
          </div>

          <div className="space-y-12">
            <TokenSection title="Color Palette">
              <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                {PALETTE.map((c) => (
                  <div key={c.name} className={`${c.token} ${c.text} rounded-lg p-4 shadow-sm`}>
                    <p className="text-sm font-semibold">{c.name}</p>
                    <p className="text-xs opacity-80">{c.hex}</p>
                  </div>
                ))}
              </div>
            </TokenSection>

            <TokenSection title="Typography">
              <div className="space-y-3">
                <p className="font-serif text-4xl font-bold tracking-tight">
                  Playfair Display SC — Heading
                </p>
                <p className="text-base">
                  Karla — Body text. Phù hợp cho UI dày dữ liệu, dễ đọc ở 14–16px.
                </p>
                <p className="text-muted-foreground text-sm">
                  Sm muted — caption, metadata, helper text.
                </p>
              </div>
            </TokenSection>

            <TokenSection title="Buttons">
              <div className="flex flex-wrap gap-3">
                <Button>Đặt món ngay</Button>
                <Button variant="accent">Trust CTA</Button>
                <Button variant="secondary">Secondary</Button>
                <Button variant="outline">Outline</Button>
                <Button variant="ghost">Ghost</Button>
                <Button variant="destructive">Hủy đơn</Button>
                <Button variant="link">Link</Button>
                <Button disabled>Disabled</Button>
              </div>
            </TokenSection>

            <TokenSection title="Badges">
              <div className="flex flex-wrap gap-2">
                <Badge>Mới</Badge>
                <Badge variant="accent">Khuyến mãi</Badge>
                <Badge variant="success">Đã giao</Badge>
                <Badge variant="warning">Chờ xác nhận</Badge>
                <Badge variant="destructive">Đã hủy</Badge>
                <Badge variant="outline">Outline</Badge>
              </div>
            </TokenSection>
          </div>
        </div>
      </section>
    </div>
  );
}

function TokenSection({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div>
      <h3 className="text-muted-foreground mb-4 text-xs font-semibold tracking-widest uppercase">
        {title}
      </h3>
      {children}
    </div>
  );
}
