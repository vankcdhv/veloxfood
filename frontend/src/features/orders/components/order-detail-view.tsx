'use client';

import { useRouter } from 'next/navigation';
import { ArrowLeft, RotateCcw, XCircle, KeyRound, CheckCircle2, Circle } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Badge } from '@/shared/ui/badge';
import { Skeleton } from '@/shared/ui/skeleton';
import { RoleGuard } from '@/features/auth/components/role-guard';
import { formatVnd } from '@/features/wallet/lib/format-vnd';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useCancelOrder, useMyOrder, useReorder } from '../hooks/use-orders';
import { OrderStatusBadge, orderStatusLabel } from './order-status-badge';
import type { Order, OrderStatus } from '../types/order';

// Canonical pipeline order for status timeline display.
const PIPELINE: OrderStatus[] = [
  'PENDING', 'CONFIRMED', 'PREPARING', 'READY', 'DELIVERING', 'DELIVERED', 'COMPLETED',
];

interface OrderDetailViewProps {
  orderId: string;
}

export function OrderDetailView({ orderId }: OrderDetailViewProps) {
  return (
    <RoleGuard allow={(a) => a.isAuthenticated}>
      <main className="bg-background min-h-dvh px-4 py-10">
        <div className="mx-auto w-full max-w-2xl space-y-6">
          <OrderDetailContent orderId={orderId} />
        </div>
      </main>
    </RoleGuard>
  );
}

function OrderDetailContent({ orderId }: { orderId: string }) {
  const router = useRouter();
  const { data: order, isLoading, isError } = useMyOrder(orderId);
  const cancelOrder = useCancelOrder(orderId);
  const reorder = useReorder();

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-32 rounded-xl" />
        <Skeleton className="h-48 rounded-xl" />
      </div>
    );
  }

  if (isError || !order) {
    return <p className="text-destructive text-sm text-center py-12">Không tải được đơn hàng.</p>;
  }

  const canCancel = order.Status === 'PENDING';

  const handleCancel = async () => {
    try {
      await cancelOrder.mutateAsync();
      toast.success('Đã hủy đơn hàng');
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'Không hủy được đơn hàng'));
    }
  };

  const handleReorder = async () => {
    try {
      const result = await reorder.mutateAsync(orderId);
      toast.success(`Đã đặt lại! Mã đơn: ${result.code}`);
      router.push(`/account/orders/${result.order_id}`);
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'Không đặt lại được'));
    }
  };

  return (
    <div className="space-y-6">
      {/* Header */}
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" onClick={() => router.back()} aria-label="Quay lại">
          <ArrowLeft className="h-4 w-4" />
        </Button>
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <h1 className="font-serif text-xl font-bold">{order.Code}</h1>
            <OrderStatusBadge status={order.Status} />
          </div>
          {order.StoreName && (
            <p className="text-sm text-muted-foreground mt-0.5">{order.StoreName}</p>
          )}
        </div>
      </div>

      {/* Status timeline */}
      {order.Status !== 'CANCELLED' && (
        <StatusTimeline order={order} />
      )}

      {order.Status === 'CANCELLED' && (
        <Card className="border-destructive/50">
          <CardContent className="flex items-center gap-2 py-4 text-destructive text-sm">
            <XCircle className="h-5 w-5 shrink-0" />
            Đơn hàng đã bị hủy.
          </CardContent>
        </Card>
      )}

      {/* Pickup PIN */}
      {order.Fulfillment === 'PICKUP' && order.PickupPin && (
        <Card className="border-primary/50 bg-primary/5">
          <CardContent className="flex items-center gap-3 py-4">
            <KeyRound className="h-6 w-6 text-primary shrink-0" />
            <div>
              <p className="text-sm font-medium">Mã PIN tự lấy</p>
              <p className="font-serif text-3xl font-bold tracking-widest text-primary">
                {order.PickupPin}
              </p>
              <p className="text-xs text-muted-foreground mt-0.5">Xuất trình mã này tại quán khi nhận hàng.</p>
            </div>
          </CardContent>
        </Card>
      )}

      {/* Items */}
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Món đã đặt</CardTitle>
        </CardHeader>
        <CardContent className="space-y-2">
          {order.Items?.map((it) => (
            <div key={it.MenuItemID} className="flex justify-between text-sm">
              <span>
                {it.NameSnapshot}
                <span className="text-muted-foreground ml-1">×{it.Qty}</span>
              </span>
              <span className="font-medium">{formatVnd(it.PriceSnapshot * it.Qty)}</span>
            </div>
          ))}
        </CardContent>
      </Card>

      {/* Totals */}
      <Card>
        <CardContent className="space-y-1.5 py-4">
          <div className="flex justify-between text-sm text-muted-foreground">
            <span>Tạm tính</span><span>{formatVnd(order.ItemsTotal)}</span>
          </div>
          {order.ShipFee > 0 && (
            <div className="flex justify-between text-sm text-muted-foreground">
              <span>Phí ship</span><span>{formatVnd(order.ShipFee)}</span>
            </div>
          )}
          {order.Discount > 0 && (
            <div className="flex justify-between text-sm text-success">
              <span>Giảm giá</span><span>-{formatVnd(order.Discount)}</span>
            </div>
          )}
          <div className="flex justify-between font-semibold border-t border-border pt-1.5">
            <span>Tổng cộng</span>
            <span className="text-primary">{formatVnd(order.GrandTotal)}</span>
          </div>
        </CardContent>
      </Card>

      {/* Meta */}
      <Card>
        <CardContent className="space-y-1.5 py-4 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">Thanh toán</span>
            <span>{paymentLabel(order)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Hình thức</span>
            <span>{order.Fulfillment === 'DELIVERY' ? 'Giao tận nơi' : 'Tự lấy'}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Đặt lúc</span>
            <span>{new Date(order.PlacedAt).toLocaleString('vi-VN')}</span>
          </div>
          {order.PaymentStatus && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">Trạng thái TT</span>
              <Badge variant={order.PaymentStatus === 'PAID' ? 'success' : 'outline'} className="text-xs">
                {order.PaymentStatus}
              </Badge>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Actions */}
      <div className="flex gap-3">
        {canCancel && (
          <Button
            variant="outline"
            className="flex-1 text-destructive border-destructive/50 hover:bg-destructive/5"
            onClick={handleCancel}
            disabled={cancelOrder.isPending}
          >
            <XCircle className="h-4 w-4 mr-1.5" />
            {cancelOrder.isPending ? 'Đang hủy…' : 'Hủy đơn'}
          </Button>
        )}
        <Button
          variant="outline"
          className="flex-1"
          onClick={handleReorder}
          disabled={reorder.isPending}
        >
          <RotateCcw className="h-4 w-4 mr-1.5" />
          {reorder.isPending ? 'Đang đặt lại…' : 'Đặt lại'}
        </Button>
      </div>
    </div>
  );
}

function StatusTimeline({ order }: { order: Order }) {
  const currentIdx = PIPELINE.indexOf(order.Status);
  const historySet = new Set(order.StatusHistory?.map((h) => h.Status) ?? []);

  return (
    <Card>
      <CardContent className="py-4">
        <ol className="relative space-y-3 pl-6 before:absolute before:left-[11px] before:top-2 before:bottom-2 before:w-0.5 before:bg-border">
          {PIPELINE.map((step, idx) => {
            const done = idx < currentIdx || historySet.has(step);
            const active = step === order.Status;
            return (
              <li key={step} className="flex items-center gap-3 relative">
                <span className="absolute -left-6 flex h-5 w-5 items-center justify-center rounded-full bg-background">
                  {done ? (
                    <CheckCircle2 className="h-4 w-4 text-success" />
                  ) : active ? (
                    <div className="h-3 w-3 rounded-full bg-primary animate-pulse" />
                  ) : (
                    <Circle className="h-4 w-4 text-muted-foreground/40" />
                  )}
                </span>
                <span
                  className={`text-sm ${
                    active
                      ? 'font-semibold text-foreground'
                      : done
                      ? 'text-muted-foreground'
                      : 'text-muted-foreground/50'
                  }`}
                >
                  {orderStatusLabel(step)}
                </span>
              </li>
            );
          })}
        </ol>
      </CardContent>
    </Card>
  );
}

function paymentLabel(order: Order): string {
  const METHOD_LABEL: Record<string, string> = {
    COD: 'Tiền mặt',
    WALLET: 'Ví VeloxFood',
    MOMO: 'MoMo',
  };
  return METHOD_LABEL[order.PaymentMethod] ?? order.PaymentMethod;
}
