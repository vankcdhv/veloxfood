'use client';

import Link from 'next/link';
import { Package } from 'lucide-react';
import { Card, CardContent } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { RoleGuard } from '@/features/auth/components/role-guard';
import { formatVnd } from '@/features/wallet/lib/format-vnd';
import { useMyOrders } from '../hooks/use-orders';
import { OrderStatusBadge } from './order-status-badge';
import type { Order } from '../types/order';

export function MyOrdersView() {
  return (
    <RoleGuard allow={(a) => a.isAuthenticated}>
      <main className="bg-background min-h-dvh px-4 py-10">
        <div className="mx-auto w-full max-w-2xl space-y-6">
          <h1 className="font-serif text-2xl font-bold">Lịch sử đơn hàng</h1>
          <OrderListContent />
        </div>
      </main>
    </RoleGuard>
  );
}

function OrderListContent() {
  const { data, isLoading, isError } = useMyOrders();
  const orders = data?.items ?? [];

  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-24 rounded-xl" />
        ))}
      </div>
    );
  }

  if (isError) {
    return <p className="text-destructive text-sm text-center py-12">Không tải được đơn hàng.</p>;
  }

  if (orders.length === 0) {
    return (
      <div className="flex flex-col items-center gap-3 py-20 text-muted-foreground">
        <Package className="h-10 w-10 opacity-30" />
        <p className="text-sm">Bạn chưa có đơn hàng nào.</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {orders.map((order) => (
        <OrderCard key={order.ID} order={order} />
      ))}
    </div>
  );
}

function OrderCard({ order }: { order: Order }) {
  const itemCount = order.Items?.reduce((s, it) => s + it.Qty, 0) ?? 0;
  return (
    <Link href={`/account/orders/${order.ID}`} className="block group">
      <Card className="transition-shadow group-hover:shadow-md">
        <CardContent className="flex items-center justify-between gap-4 py-4">
          <div className="min-w-0 space-y-1">
            <p className="font-medium text-sm">{order.Code}</p>
            {order.StoreName && (
              <p className="text-xs text-muted-foreground truncate">{order.StoreName}</p>
            )}
            <p className="text-xs text-muted-foreground">
              {itemCount} món · {new Date(order.PlacedAt).toLocaleDateString('vi-VN')}
            </p>
          </div>
          <div className="flex flex-col items-end gap-1.5 shrink-0">
            <OrderStatusBadge status={order.Status} />
            <span className="text-sm font-semibold text-primary">{formatVnd(order.GrandTotal)}</span>
          </div>
        </CardContent>
      </Card>
    </Link>
  );
}
