'use client';

import { ShoppingCart, Users, DollarSign, TrendingDown } from 'lucide-react';
import { useQuery } from '@tanstack/react-query';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/shared/ui/card';
import { Badge } from '@/shared/ui/badge';
import { Skeleton } from '@/shared/ui/skeleton';
import { AdminTopbar } from '@/widgets/admin-topbar/admin-topbar';
import { formatVnd } from '@/shared/lib/format-vnd';
import { reportingApi } from '@/features/notifications/api/reporting-api';
import type { RecentOrderRow } from '@/features/notifications/api/reporting-api';

// ── Analytics hooks (inline — lightweight, not shared elsewhere) ──

function useAnalytics() {
  return useQuery({
    queryKey: ['admin', 'analytics', 'day'],
    queryFn: () => reportingApi.analytics('day'),
    // Refresh every 2 min on the dashboard.
    refetchInterval: 120_000,
  });
}

function useRecentOrders() {
  return useQuery({
    queryKey: ['admin', 'recent-orders'],
    queryFn: () => reportingApi.recentOrders(20),
    refetchInterval: 120_000,
  });
}

// ── Status badge for recent orders ──

const STATUS_VARIANT: Record<string, 'success' | 'warning' | 'destructive' | 'outline'> = {
  COMPLETED: 'success',
  DELIVERED: 'success',
  DELIVERING: 'warning',
  PREPARING: 'warning',
  CONFIRMED: 'warning',
  PENDING: 'outline',
  CANCELLED: 'destructive',
  READY: 'warning',
};

const STATUS_LABEL: Record<string, string> = {
  PENDING: 'Chờ xác nhận',
  CONFIRMED: 'Đã xác nhận',
  PREPARING: 'Đang chuẩn bị',
  READY: 'Sẵn sàng',
  DELIVERING: 'Đang giao',
  DELIVERED: 'Đã giao',
  COMPLETED: 'Hoàn thành',
  CANCELLED: 'Đã hủy',
};

export default function AdminDashboardPage() {
  return (
    <>
      <AdminTopbar title="Tổng quan" description="Snapshot vận hành 24h qua" />
      <div className="space-y-6 p-4 sm:p-6 lg:p-8">
        <MetricsGrid />
        <div className="grid gap-6 lg:grid-cols-3">
          <RecentOrdersCard />
          <Card>
            <CardHeader>
              <CardTitle>Dịch vụ đã triển khai</CardTitle>
              <CardDescription>Các microservice hiện có của hệ thống.</CardDescription>
            </CardHeader>
            <CardContent>
              <ul className="space-y-3 text-sm">
                {[
                  { name: 'Tài khoản & phân quyền', svc: 'user-service' },
                  { name: 'Vị trí giao', svc: 'location-service' },
                  { name: 'Cửa hàng & thực đơn', svc: 'store-service' },
                  { name: 'Đơn hàng & giỏ hàng', svc: 'order-service' },
                  { name: 'Thanh toán & ví', svc: 'payment-service' },
                  { name: 'Giao hàng', svc: 'delivery-service' },
                  { name: 'Đánh giá', svc: 'review-service' },
                  { name: 'Thông báo', svc: 'notification-service' },
                ].map((s) => (
                  <li key={s.svc} className="flex items-center justify-between gap-2">
                    <span>{s.name}</span>
                    <Badge variant="success">Đang chạy</Badge>
                  </li>
                ))}
              </ul>
            </CardContent>
          </Card>
        </div>
      </div>
    </>
  );
}

function MetricsGrid() {
  const { data, isLoading, isError } = useAnalytics();

  if (isLoading) {
    return (
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-28 rounded-xl" />
        ))}
      </div>
    );
  }

  if (isError || !data) {
    return (
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        {METRIC_CONFIGS.map((m) => (
          <MetricCard key={m.label} label={m.label} value="—" delta="Lỗi tải dữ liệu" Icon={m.icon} />
        ))}
      </div>
    );
  }

  return (
    <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
      <MetricCard
        label="Đơn hôm nay"
        value={String(data.OrdersTotal)}
        delta={data.CancelledTotal > 0 ? `${data.CancelledTotal} đã hủy` : 'Không có hủy'}
        Icon={ShoppingCart}
      />
      <MetricCard
        label="Doanh thu"
        value={formatVnd(data.RevenueTotal)}
        delta={data.AvgOrderValue > 0 ? `TB: ${formatVnd(Math.round(data.AvgOrderValue))}` : '—'}
        Icon={DollarSign}
      />
      <MetricCard
        label="Người dùng mới"
        value={String(data.NewUsersTotal)}
        delta={data.NewVendors > 0 ? `${data.NewVendors} vendor mới` : 'Không có vendor mới'}
        Icon={Users}
      />
      <MetricCard
        label="Đơn hủy"
        value={String(data.CancelledTotal)}
        delta={
          data.OrdersTotal > 0
            ? `${Math.round((data.CancelledTotal / data.OrdersTotal) * 100)}% tổng đơn`
            : '—'
        }
        Icon={TrendingDown}
      />
    </div>
  );
}

const METRIC_CONFIGS = [
  { label: 'Đơn hôm nay', icon: ShoppingCart },
  { label: 'Doanh thu', icon: DollarSign },
  { label: 'Người dùng mới', icon: Users },
  { label: 'Đơn hủy', icon: TrendingDown },
];

function MetricCard({
  label,
  value,
  delta,
  Icon,
}: {
  label: string;
  value: string;
  delta: string;
  Icon: React.ComponentType<{ className?: string }>;
}) {
  return (
    <Card>
      <CardContent className="p-5">
        <div className="flex items-start justify-between">
          <div>
            <p className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
              {label}
            </p>
            <p className="mt-2 font-serif text-3xl font-bold">{value}</p>
          </div>
          <span className="bg-primary/10 text-primary inline-flex h-9 w-9 items-center justify-center rounded-lg">
            <Icon className="h-4 w-4" />
          </span>
        </div>
        <Badge variant="outline" className="mt-3">
          {delta}
        </Badge>
      </CardContent>
    </Card>
  );
}

function RecentOrdersCard() {
  const { data: orders, isLoading, isError } = useRecentOrders();

  return (
    <Card className="lg:col-span-2">
      <CardHeader>
        <CardTitle>Đơn hàng gần đây</CardTitle>
        <CardDescription>20 đơn mới nhất trong hệ thống.</CardDescription>
      </CardHeader>
      <CardContent>
        {isLoading && (
          <div className="space-y-2">
            {Array.from({ length: 5 }).map((_, i) => (
              <Skeleton key={i} className="h-10 rounded-lg" />
            ))}
          </div>
        )}

        {isError && (
          <div className="border-border bg-muted/40 text-muted-foreground flex h-32 items-center justify-center rounded-lg border border-dashed text-sm">
            Không tải được dữ liệu đơn hàng.
          </div>
        )}

        {!isLoading && !isError && (!orders || orders.length === 0) && (
          <div className="border-border bg-muted/40 text-muted-foreground flex h-32 items-center justify-center rounded-lg border border-dashed text-sm">
            Chưa có đơn hàng nào.
          </div>
        )}

        {!isLoading && !isError && orders && orders.length > 0 && (
          <div className="space-y-1">
            {orders.map((order) => (
              <RecentOrderRow key={order.OrderID} order={order} />
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

function RecentOrderRow({ order }: { order: RecentOrderRow }) {
  const variant = STATUS_VARIANT[order.Status] ?? 'outline';
  const label = STATUS_LABEL[order.Status] ?? order.Status;

  return (
    <div className="flex items-center justify-between gap-3 rounded-lg px-3 py-2 hover:bg-muted/50 text-sm">
      <div className="min-w-0 flex-1">
        <p className="truncate font-medium font-mono text-xs">{order.Code || 'Đơn chưa có mã'}</p>
        {order.StoreName && (
          <p className="text-muted-foreground text-xs truncate">{order.StoreName}</p>
        )}
      </div>
      <span className="text-muted-foreground text-xs shrink-0">
        {new Date(order.CreatedAt).toLocaleTimeString('vi-VN', { hour: '2-digit', minute: '2-digit' })}
      </span>
      <span className="font-semibold shrink-0">{formatVnd(order.GrandTotal)}</span>
      <Badge variant={variant} className="shrink-0 text-xs">
        {label}
      </Badge>
    </div>
  );
}
