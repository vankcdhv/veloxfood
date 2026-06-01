import Link from 'next/link';
import { ArrowRight, Clock, Star, Truck, UtensilsCrossed } from 'lucide-react';
import { Button } from '@/shared/ui/button';
import { Badge } from '@/shared/ui/badge';
import { Card, CardContent } from '@/shared/ui/card';
import { ROUTES } from '@/shared/config/constants';

const HIGHLIGHTS = [
  { icon: Truck, title: 'Giao 30 phút', description: 'Cam kết giao tận nơi nội thành.' },
  { icon: Star, title: '4.9 từ 12k đánh giá', description: 'Bếp nổi tiếng, nguyên liệu chọn lọc.' },
  { icon: Clock, title: 'Mở 10:00 – 22:00', description: 'Phục vụ cả tuần, kể cả lễ tết.' },
];

const COLLECTIONS = [
  { name: 'Món hôm nay', tag: 'Hot', subtitle: '12 món bán chạy' },
  { name: 'Bữa trưa nhanh', tag: 'Quick', subtitle: 'Sẵn sàng dưới 20 phút' },
  { name: 'Chay & lành mạnh', tag: 'Healthy', subtitle: 'Calo dưới 500' },
  { name: 'Tráng miệng', tag: 'Sweet', subtitle: 'Bánh & đồ uống' },
];

export default function ShopHomePage() {
  return (
    <div>
      <section className="from-primary/5 to-background bg-gradient-to-b">
        <div className="mx-auto max-w-7xl px-4 py-16 sm:px-6 sm:py-24 lg:px-8">
          <div className="grid items-center gap-12 lg:grid-cols-2">
            <div className="space-y-6">
              <Badge variant="secondary">Ưu đãi tháng này · giảm tới 30%</Badge>
              <h1 className="font-serif text-4xl leading-tight font-bold tracking-tight sm:text-5xl lg:text-6xl">
                Đặt món
                <br />
                <span className="text-primary">ngon — nhanh — chuẩn vị</span>
              </h1>
              <p className="text-muted-foreground max-w-lg text-lg">
                Hơn 200 món từ các bếp uy tín, giao tận nơi trong 30 phút. Đặt một lần, ngon mãi.
              </p>
              <div className="flex flex-wrap gap-3">
                <Button asChild size="lg">
                  <Link href={ROUTES.stores.root}>
                    Xem thực đơn
                    <ArrowRight className="h-4 w-4" />
                  </Link>
                </Button>
                <Button asChild size="lg" variant="outline">
                  <Link href={ROUTES.cart}>Giỏ hàng</Link>
                </Button>
              </div>
            </div>

            <div className="bg-card border-border relative aspect-[4/3] overflow-hidden rounded-2xl border shadow-lg">
              <div className="from-primary/25 via-secondary/15 absolute inset-0 bg-gradient-to-br to-transparent" />
              <div className="bg-primary/10 absolute -top-10 -right-10 h-48 w-48 rounded-full blur-2xl" />
              <div className="bg-secondary/10 absolute -bottom-12 -left-8 h-40 w-40 rounded-full blur-2xl" />
              <div className="absolute inset-0 flex flex-col items-center justify-center gap-4">
                <div className="bg-primary/15 text-primary flex h-24 w-24 items-center justify-center rounded-full">
                  <UtensilsCrossed className="h-12 w-12" />
                </div>
                <p className="text-foreground/80 font-serif text-xl">Bữa ngon đang chờ bạn</p>
              </div>
              <div className="bg-card/90 border-border absolute top-6 left-6 rounded-xl border px-3 py-2 text-sm font-medium shadow-sm backdrop-blur">
                🍚 Cơm sườn · 45.000đ
              </div>
              <div className="bg-card/90 border-border absolute right-6 bottom-6 rounded-xl border px-3 py-2 text-sm font-medium shadow-sm backdrop-blur">
                ⭐ 4.9 · Giao 30′
              </div>
            </div>
          </div>

          <div className="mt-12 grid gap-4 sm:grid-cols-3">
            {HIGHLIGHTS.map((h) => {
              const Icon = h.icon;
              return (
                <Card key={h.title}>
                  <CardContent className="flex items-start gap-3 p-5">
                    <span className="bg-primary/10 text-primary inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-lg">
                      <Icon className="h-5 w-5" />
                    </span>
                    <div>
                      <p className="font-semibold">{h.title}</p>
                      <p className="text-muted-foreground text-sm">{h.description}</p>
                    </div>
                  </CardContent>
                </Card>
              );
            })}
          </div>
        </div>
      </section>

      <section className="mx-auto max-w-7xl px-4 py-16 sm:px-6 lg:px-8">
        <div className="mb-8 flex items-end justify-between gap-4">
          <div>
            <h2 className="font-serif text-3xl font-bold tracking-tight">Bộ sưu tập</h2>
            <p className="text-muted-foreground mt-1">
              Chọn theo dịp, theo nhu cầu, theo tâm trạng.
            </p>
          </div>
          <Button asChild variant="ghost">
            <Link href={ROUTES.stores.root}>
              Xem tất cả
              <ArrowRight className="h-4 w-4" />
            </Link>
          </Button>
        </div>

        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
          {COLLECTIONS.map((c) => (
            <Card key={c.name} className="group cursor-pointer overflow-hidden">
              <div className="from-primary/15 to-secondary/10 aspect-[4/3] bg-gradient-to-br transition-transform duration-300 group-hover:scale-[1.02]" />
              <CardContent className="space-y-1 p-5">
                <div className="flex items-center justify-between">
                  <p className="font-serif text-lg font-semibold">{c.name}</p>
                  <Badge variant="accent">{c.tag}</Badge>
                </div>
                <p className="text-muted-foreground text-sm">{c.subtitle}</p>
              </CardContent>
            </Card>
          ))}
        </div>
      </section>
    </div>
  );
}
