'use client';

import { useEffect, useRef, useState } from 'react';
import { useRouter } from 'next/navigation';
import { ArrowLeft, RotateCcw, XCircle, KeyRound, CheckCircle2, Circle, Copy } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Badge } from '@/shared/ui/badge';
import { Skeleton } from '@/shared/ui/skeleton';
import { ConfirmDialog } from '@/shared/ui/confirm-dialog';
import { RoleGuard } from '@/features/auth/components/role-guard';
import { ROUTES } from '@/shared/config/constants';
import { formatVnd } from '@/shared/lib/format-vnd';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useAuth } from '@/features/auth/context/auth-provider';
import { OrderReviewForm } from '@/features/reviews/components/order-review-form';
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
  const { user } = useAuth();
  const { data: order, isLoading, isError } = useMyOrder(orderId);
  const cancelOrder = useCancelOrder(orderId);
  const reorder = useReorder();

  // Firestore realtime tracking — subscribes to orders/{userId}/orders/{orderId}
  // when Firebase is configured. Falls back to the existing 10s poll otherwise.
  const [realtimeStatus, setRealtimeStatus] = useState<OrderStatus | null>(null);
  const unsubscribeRef = useRef<(() => void) | null>(null);

  useEffect(() => {
    if (!user?.id || !orderId) return;

    let cancelled = false;

    (async () => {
      try {
        const { getFirestoreDb } = await import('@/lib/firebase');
        const db = await getFirestoreDb();
        if (!db || cancelled) return;

        // Dynamic imports so tsc doesn't require `firebase` at compile time.
        // Indirect import key so tsc doesn't require firebase types when package is absent.
        const firestorePkg = 'firebase/firestore';
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const { doc, onSnapshot } = await import(/* @vite-ignore */ firestorePkg) as any;
        const docRef = doc(db, `orders/${user.id}/orders/${orderId}`);
        // eslint-disable-next-line @typescript-eslint/no-explicit-any
        const unsub = onSnapshot(docRef, (snap: any) => {
          if (snap.exists()) {
            const data = snap.data() as { status?: OrderStatus };
            if (data.status) setRealtimeStatus(data.status);
          }
        });
        unsubscribeRef.current = unsub;
      } catch {
        // Firebase unavailable — polling fallback already active via useMyOrder.
      }
    })();

    return () => {
      cancelled = true;
      unsubscribeRef.current?.();
      unsubscribeRef.current = null;
    };
  }, [user?.id, orderId]);

  const [reviewSubmitted, setReviewSubmitted] = useState(false);
  const [confirmCancel, setConfirmCancel] = useState(false);

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

  // Use realtime Firestore status when available; otherwise use polled REST data.
  const liveStatus: OrderStatus = realtimeStatus ?? order.Status;
  const displayOrder = realtimeStatus ? { ...order, Status: realtimeStatus } : order;
  const canCancel = liveStatus === 'PENDING';
  // CANCELLED and REJECTED both end the order — no timeline, show a closed note.
  const isClosed = liveStatus === 'CANCELLED' || liveStatus === 'REJECTED';

  const handleCancel = async () => {
    setConfirmCancel(false);
    try {
      await cancelOrder.mutateAsync();
      toast.success('Đã hủy đơn hàng');
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'Không hủy được đơn hàng'));
    }
  };

  // Reorder re-populates the cart (it does NOT place a new order); send the
  // customer to the cart to review + check out.
  const handleReorder = async () => {
    try {
      await reorder.mutateAsync(orderId);
      toast.success('Đã thêm các món vào giỏ hàng');
      router.push(ROUTES.cart);
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
            <h1 className="font-serif font-mono text-xl font-bold">{displayOrder.Code}</h1>
            {displayOrder.Code && (
              <button
                type="button"
                aria-label="Sao chép mã đơn"
                onClick={() => {
                  navigator.clipboard?.writeText(displayOrder.Code).then(
                    () => toast.success('Đã sao chép mã đơn'),
                    () => toast.error('Không sao chép được'),
                  );
                }}
                className="text-muted-foreground hover:text-primary transition-colors"
              >
                <Copy className="h-4 w-4" />
              </button>
            )}
            <OrderStatusBadge status={liveStatus} />
            {realtimeStatus && (
              <span className="text-xs text-success font-medium">● realtime</span>
            )}
          </div>
          {displayOrder.StoreName && (
            <p className="text-sm text-muted-foreground mt-0.5">{displayOrder.StoreName}</p>
          )}
        </div>
      </div>

      {/* Status timeline */}
      {!isClosed && <StatusTimeline order={displayOrder} />}

      {isClosed && (
        <Card className="border-destructive/50">
          <CardContent className="flex items-center gap-2 py-4 text-destructive text-sm">
            <XCircle className="h-5 w-5 shrink-0" />
            {liveStatus === 'REJECTED' ? 'Đơn hàng đã bị cửa hàng từ chối.' : 'Đơn hàng đã bị hủy.'}
          </CardContent>
        </Card>
      )}

      {/* Pickup PIN */}
      {displayOrder.Fulfillment === 'PICKUP' && displayOrder.PickupPin && (
        <Card className="border-primary/50 bg-primary/5">
          <CardContent className="flex items-center gap-3 py-4">
            <KeyRound className="h-6 w-6 text-primary shrink-0" />
            <div>
              <p className="text-sm font-medium">Mã PIN tự lấy</p>
              <p className="font-serif text-3xl font-bold tracking-widest text-primary">
                {displayOrder.PickupPin}
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
          {displayOrder.Items?.map((it) => (
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
            <span>Tạm tính</span><span>{formatVnd(displayOrder.ItemsTotal)}</span>
          </div>
          {displayOrder.ShipFee > 0 && (
            <div className="flex justify-between text-sm text-muted-foreground">
              <span>Phí ship</span><span>{formatVnd(displayOrder.ShipFee)}</span>
            </div>
          )}
          {displayOrder.Discount > 0 && (
            <div className="flex justify-between text-sm text-success">
              <span>Giảm giá</span><span>-{formatVnd(displayOrder.Discount)}</span>
            </div>
          )}
          <div className="flex justify-between font-semibold border-t border-border pt-1.5">
            <span>Tổng cộng</span>
            <span className="text-primary">{formatVnd(displayOrder.GrandTotal)}</span>
          </div>
        </CardContent>
      </Card>

      {/* Meta */}
      <Card>
        <CardContent className="space-y-1.5 py-4 text-sm">
          <div className="flex justify-between">
            <span className="text-muted-foreground">Thanh toán</span>
            <span>{paymentLabel(displayOrder)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Hình thức</span>
            <span>{displayOrder.Fulfillment === 'DELIVERY' ? 'Giao tận nơi' : 'Tự lấy'}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Giờ nhận mong muốn</span>
            <span>{desiredTimeLabel(displayOrder.DesiredTime)}</span>
          </div>
          <div className="flex justify-between">
            <span className="text-muted-foreground">Đặt lúc</span>
            <span>{new Date(displayOrder.PlacedAt).toLocaleString('vi-VN')}</span>
          </div>
          {displayOrder.PaymentStatus && (
            <div className="flex justify-between">
              <span className="text-muted-foreground">Trạng thái TT</span>
              <Badge variant={paymentStatusVariant(displayOrder.PaymentStatus)} className="text-xs">
                {paymentStatusLabel(displayOrder.PaymentStatus)}
              </Badge>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Review form — only shown for COMPLETED orders that haven't been reviewed yet */}
      {liveStatus === 'COMPLETED' && !reviewSubmitted && displayOrder.StoreID && (
        <OrderReviewForm
          orderId={orderId}
          storeId={displayOrder.StoreID}
          items={displayOrder.Items}
          onSubmitted={() => setReviewSubmitted(true)}
        />
      )}
      {liveStatus === 'COMPLETED' && reviewSubmitted && (
        <Card className="border-success/30 bg-success/5">
          <CardContent className="flex items-center gap-2 py-3 text-success text-sm">
            <CheckCircle2 className="h-4 w-4 shrink-0" />
            Cảm ơn bạn đã đánh giá đơn hàng này!
          </CardContent>
        </Card>
      )}

      {/* Actions */}
      <div className="flex gap-3">
        {canCancel && (
          <Button
            variant="outline"
            className="flex-1 text-destructive border-destructive/50 hover:bg-destructive/5"
            onClick={() => setConfirmCancel(true)}
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

      <ConfirmDialog
        open={confirmCancel}
        onOpenChange={setConfirmCancel}
        title="Hủy đơn hàng?"
        description="Đơn sẽ bị hủy và không thể khôi phục. Nếu đã thanh toán online, tiền sẽ được hoàn về ví."
        confirmLabel="Hủy đơn"
        cancelLabel="Không"
        destructive
        loading={cancelOrder.isPending}
        onConfirm={handleCancel}
      />
    </div>
  );
}

// "Giờ nhận mong muốn": a stored RFC3339 → "HH:MM hôm nay"; empty ⇒ ASAP.
function desiredTimeLabel(rfc3339?: string): string {
  if (!rfc3339) return 'Sớm nhất có thể';
  const d = new Date(rfc3339);
  if (Number.isNaN(d.getTime())) return 'Sớm nhất có thể';
  return `${d.toLocaleTimeString('vi-VN', { hour: '2-digit', minute: '2-digit' })} hôm nay`;
}

function paymentStatusLabel(status: string): string {
  const MAP: Record<string, string> = {
    UNPAID: 'Chưa thanh toán',
    PENDING: 'Chưa thanh toán',
    PAID: 'Đã thanh toán',
    REFUNDED: 'Đã hoàn tiền',
    FAILED: 'Thanh toán thất bại',
  };
  return MAP[status] ?? status;
}

function paymentStatusVariant(status: string): 'success' | 'warning' | 'destructive' | 'accent' | 'outline' {
  switch (status) {
    case 'PAID':
      return 'success';
    case 'REFUNDED':
      return 'accent';
    case 'FAILED':
      return 'destructive';
    default:
      return 'warning'; // UNPAID / PENDING
  }
}

function StatusTimeline({ order }: { order: Order }) {
  // Pickup orders never go through a delivery leg — drop that step from the timeline.
  const pipeline =
    order.Fulfillment === 'PICKUP' ? PIPELINE.filter((s) => s !== 'DELIVERING') : PIPELINE;
  // Map intermediate statuses that aren't their own pipeline step onto one.
  const NORMALIZE: Partial<Record<OrderStatus, OrderStatus>> = {
    READY_PICKUP: 'READY',
    SHIPPER_ASSIGNED: 'READY',
  };
  const normStatus = NORMALIZE[order.Status] ?? order.Status;
  const currentIdx = pipeline.indexOf(normStatus);
  const historySet = new Set(order.StatusHistory?.map((h) => h.Status) ?? []);

  return (
    <Card>
      <CardContent className="py-4">
        <ol className="relative space-y-3 pl-6 before:absolute before:left-[11px] before:top-2 before:bottom-2 before:w-0.5 before:bg-border">
          {pipeline.map((step, idx) => {
            const done = idx < currentIdx || historySet.has(step);
            const active = step === normStatus;
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
