'use client';

import { useMemo } from 'react';
import Link from 'next/link';
import { useRouter } from 'next/navigation';
import { ArrowRight, RotateCcw, UtensilsCrossed } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Card, CardContent } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { ROUTES } from '@/shared/config/constants';
import { formatVnd } from '@/shared/lib/format-vnd';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useAuth } from '@/features/auth/context/auth-provider';
import { useMyOrders, useReorder } from '@/features/orders/hooks/use-orders';
import { useStores } from '../hooks/use-stores';
import { StoreCard } from './store-card';

const FEATURED_LIMIT = 6;
const CUISINE_LIMIT = 8;
const REORDER_LIMIT = 3;

// Real-data home discovery: cuisine quick-row + featured stores + reorder.
// Replaces the old hardcoded "collections". Sections self-hide when empty
// (e.g. logged-out users without store access) so the landing never looks broken.
export function HomeDiscovery() {
  const { isAuthenticated } = useAuth();
  const { data: stores, isLoading } = useStores();

  const cuisines = useMemo(() => {
    const seen = new Set<string>();
    for (const s of stores ?? []) {
      const c = s.BusinessType?.trim();
      if (c) seen.add(c);
    }
    return [...seen].slice(0, CUISINE_LIMIT);
  }, [stores]);

  // Featured = open stores first, then the rest; capped.
  const featured = useMemo(() => {
    const all = [...(stores ?? [])];
    all.sort((a, b) => Number(b.OpenNow ?? false) - Number(a.OpenNow ?? false));
    return all.slice(0, FEATURED_LIMIT);
  }, [stores]);

  return (
    <section className="mx-auto max-w-7xl space-y-12 px-4 py-12 sm:px-6 lg:px-8">
      {isAuthenticated && <ReorderRow />}

      {/* Cuisine quick-row */}
      {cuisines.length > 0 && (
        <div className="space-y-3">
          <h2 className="font-serif text-2xl font-bold tracking-tight">Khám phá theo loại món</h2>
          <div className="flex flex-wrap gap-2">
            {cuisines.map((c) => (
              <Link
                key={c}
                href={`${ROUTES.stores.root}?cuisine=${encodeURIComponent(c)}`}
                className="border-border hover:border-primary/50 hover:bg-primary/5 inline-flex items-center gap-1.5 rounded-full border px-4 py-2 text-sm font-medium transition-colors"
              >
                <UtensilsCrossed className="text-primary h-3.5 w-3.5" />
                {c}
              </Link>
            ))}
          </div>
        </div>
      )}

      {/* Featured stores */}
      <div className="space-y-4">
        <div className="flex items-end justify-between gap-4">
          <div>
            <h2 className="font-serif text-2xl font-bold tracking-tight">Quán nổi bật</h2>
            <p className="text-muted-foreground mt-1 text-sm">Những bếp đang mở, sẵn sàng phục vụ.</p>
          </div>
          <Button asChild variant="ghost">
            <Link href={ROUTES.stores.root}>
              Xem tất cả
              <ArrowRight className="h-4 w-4" />
            </Link>
          </Button>
        </div>

        {isLoading ? (
          <div className="grid gap-4 auto-rows-fr sm:grid-cols-2 lg:grid-cols-3">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-44 rounded-xl" />
            ))}
          </div>
        ) : featured.length > 0 ? (
          <div className="grid gap-4 auto-rows-fr sm:grid-cols-2 lg:grid-cols-3">
            {featured.map((s) => (
              <StoreCard key={s.ID} store={s} href={ROUTES.stores.detail(s.ID)} />
            ))}
          </div>
        ) : (
          <Card>
            <CardContent className="text-muted-foreground flex flex-col items-center gap-3 py-10 text-sm">
              <UtensilsCrossed className="h-8 w-8 opacity-30" />
              <p>Đăng nhập để khám phá các cửa hàng quanh bạn.</p>
              <Button asChild size="sm">
                <Link href={ROUTES.auth.login}>Đăng nhập</Link>
              </Button>
            </CardContent>
          </Card>
        )}
      </div>
    </section>
  );
}

// "Đặt lại" — surfaces the customer's most recent orders with a one-tap reorder.
function ReorderRow() {
  const router = useRouter();
  const { data } = useMyOrders(1);
  const reorder = useReorder();
  const recent = (data?.items ?? []).slice(0, REORDER_LIMIT);

  if (recent.length === 0) return null;

  const onReorder = (id: string) => {
    reorder.mutate(id, {
      onSuccess: () => {
        toast.success('Đã thêm vào giỏ — kiểm tra và đặt lại nhé!');
        router.push(ROUTES.cart);
      },
      onError: (e) => toast.error(getApiErrorMessage(e, 'Không đặt lại được đơn này')),
    });
  };

  return (
    <div className="space-y-3">
      <h2 className="font-serif text-2xl font-bold tracking-tight">Đặt lại</h2>
      <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
        {recent.map((o) => (
          <Card key={o.ID}>
            <CardContent className="flex items-center justify-between gap-3 py-4">
              <div className="min-w-0">
                <p className="truncate text-sm font-medium">{o.StoreName ?? o.Code}</p>
                <p className="text-muted-foreground text-xs">
                  {o.Items?.reduce((s, it) => s + it.Qty, 0) ?? 0} món · {formatVnd(o.GrandTotal)}
                </p>
              </div>
              <Button
                size="sm"
                variant="outline"
                className="shrink-0 gap-1"
                disabled={reorder.isPending}
                onClick={() => onReorder(o.ID)}
              >
                <RotateCcw className="h-3.5 w-3.5" />
                Đặt lại
              </Button>
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  );
}
