'use client';

import { useState } from 'react';
import { RefreshCw, CheckCircle, XCircle, KeyRound, ChevronRight, Clock } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Badge } from '@/shared/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { formatVnd } from '@/shared/lib/format-vnd';
import { formatLateBy } from '@/shared/lib/format-late-by';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useStoreOrders, useOwnerOrderMutations, useStoreOrder } from '../hooks/use-orders';
import { OrderStatusBadge, orderStatusLabel } from './order-status-badge';
import type { Order, OrderStatus } from '../types/order';

// Format an RFC3339 string to "HH:MM". Returns '' on invalid input.
function formatHHMM(rfc3339: string | undefined): string {
  if (!rfc3339) return '';
  const d = new Date(rfc3339);
  if (Number.isNaN(d.getTime())) return '';
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}

// Compute lateness in minutes between two RFC3339 timestamps. Returns 0 if on time.
function lateMinutes(deliveredAt: string | undefined, desiredTime: string | undefined): number {
  if (!deliveredAt || !desiredTime) return 0;
  const delivered = new Date(deliveredAt).getTime();
  const desired = new Date(desiredTime).getTime();
  return Math.max(0, Math.round((delivered - desired) / 60000));
}

function DesiredTimeMeta({ order }: { order: Order }) {
  const timeStr = formatHHMM(order.DesiredTime);
  if (!timeStr) return null;

  // Only show late badge if the order has been delivered.
  const isDelivered = order.Status === 'DELIVERED' || order.Status === 'COMPLETED';
  // DeliveredAt is enriched by the owner detail endpoint from status history.
  const late = isDelivered ? lateMinutes(order.DeliveredAt, order.DesiredTime) : 0;

  return (
    <div className="flex items-center gap-2 text-xs text-muted-foreground">
      <Clock className="h-3.5 w-3.5 shrink-0" />
      <span>Giờ mong muốn: <span className="font-medium text-foreground">{timeStr}</span></span>
      {isDelivered && (
        late > 0
          ? <Badge variant="destructive" className="text-xs px-1.5 py-0">Trễ {formatLateBy(late)}</Badge>
          : <Badge variant="success" className="text-xs px-1.5 py-0">Đúng giờ</Badge>
      )}
    </div>
  );
}

interface VendorOrderPanelProps {
  storeId: string;
}

// Next logical status transitions an owner can trigger manually.
const NEXT_STATUS: Partial<Record<OrderStatus, OrderStatus>> = {
  CONFIRMED: 'PREPARING',
  PREPARING: 'READY',
  READY: 'DELIVERING',
  DELIVERING: 'DELIVERED',
  DELIVERED: 'COMPLETED',
};

export function VendorOrderPanel({ storeId }: VendorOrderPanelProps) {
  const [selectedId, setSelectedId] = useState<string | null>(null);

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <OrderList storeId={storeId} selectedId={selectedId} onSelect={setSelectedId} />
      {selectedId && (
        <OrderActions storeId={storeId} orderId={selectedId} />
      )}
    </div>
  );
}

function OrderList({
  storeId,
  selectedId,
  onSelect,
}: {
  storeId: string;
  selectedId: string | null;
  onSelect: (id: string) => void;
}) {
  const { data, isLoading, isError, refetch, isFetching } = useStoreOrders(storeId);
  const orders = data?.items ?? [];

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="font-semibold text-sm">Đơn hàng ({data?.total ?? 0})</h3>
        <Button
          variant="ghost"
          size="icon"
          className="h-7 w-7"
          onClick={() => refetch()}
          disabled={isFetching}
          aria-label="Làm mới"
        >
          <RefreshCw className={`h-3.5 w-3.5 ${isFetching ? 'animate-spin' : ''}`} />
        </Button>
      </div>

      {isLoading && (
        <div className="space-y-2">
          {Array.from({ length: 4 }).map((_, i) => (
            <Skeleton key={i} className="h-16 rounded-lg" />
          ))}
        </div>
      )}

      {isError && (
        <p className="text-destructive text-sm">Không tải được danh sách đơn.</p>
      )}

      {!isLoading && orders.length === 0 && (
        <p className="text-muted-foreground text-sm text-center py-8">Chưa có đơn hàng nào.</p>
      )}

      <div className="space-y-2">
        {orders.map((order) => (
          <button
            key={order.ID}
            type="button"
            onClick={() => onSelect(order.ID)}
            className={`w-full text-left rounded-lg border px-3 py-2.5 transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
              selectedId === order.ID
                ? 'border-primary bg-primary/5'
                : 'border-border hover:border-primary/40 hover:bg-muted/50'
            }`}
          >
            <div className="flex items-center justify-between gap-2">
              <div className="min-w-0">
                <p className="text-sm font-medium truncate">{order.Code}</p>
                <p className="text-xs text-muted-foreground mt-0.5">
                  {order.Items?.reduce((s, it) => s + it.Qty, 0) ?? 0} món ·{' '}
                  {formatVnd(order.GrandTotal)}
                </p>
              </div>
              <div className="flex items-center gap-1 shrink-0">
                <OrderStatusBadge status={order.Status} />
                <ChevronRight className="h-3.5 w-3.5 text-muted-foreground" />
              </div>
            </div>
          </button>
        ))}
      </div>
    </div>
  );
}

function OrderActions({ storeId, orderId }: { storeId: string; orderId: string }) {
  const { data: order, isLoading } = useStoreOrder(storeId, orderId);
  const mutations = useOwnerOrderMutations(storeId);
  const [pinInput, setPinInput] = useState('');
  const [showPinDialog, setShowPinDialog] = useState(false);

  if (isLoading || !order) {
    return <Skeleton className="h-64 rounded-xl" />;
  }

  const nextStatus = NEXT_STATUS[order.Status];

  const handleConfirm = async () => {
    try {
      await mutations.confirm.mutateAsync(orderId);
      toast.success('Đã xác nhận đơn hàng');
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'Không xác nhận được'));
    }
  };

  const handleReject = async () => {
    try {
      await mutations.reject.mutateAsync(orderId);
      toast.success('Đã từ chối đơn hàng');
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'Không từ chối được'));
    }
  };

  const handleAdvance = async () => {
    if (!nextStatus) return;
    try {
      await mutations.advanceStatus.mutateAsync({ orderId, body: { status: nextStatus } });
      toast.success('Đã cập nhật trạng thái');
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'Không cập nhật được'));
    }
  };

  const handlePickupVerify = async () => {
    if (!pinInput.trim()) return;
    try {
      await mutations.verifyPickup.mutateAsync({ orderId, body: { pin: pinInput.trim() } });
      toast.success('Xác nhận PIN thành công');
      setShowPinDialog(false);
      setPinInput('');
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'PIN không đúng'));
    }
  };

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base flex items-center justify-between gap-2">
          <span>{order.Code}</span>
          <OrderStatusBadge status={order.Status} />
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Items */}
        <div className="space-y-1">
          {order.Items?.map((it) => (
            <div key={it.MenuItemID} className="flex justify-between text-sm">
              <span>
                {it.NameSnapshot}
                <span className="text-muted-foreground ml-1">×{it.Qty}</span>
              </span>
              <span>{formatVnd(it.PriceSnapshot * it.Qty)}</span>
            </div>
          ))}
          <div className="flex justify-between text-sm font-semibold border-t border-border pt-1 mt-1">
            <span>Tổng</span>
            <span className="text-primary">{formatVnd(order.GrandTotal)}</span>
          </div>
        </div>

        {/* Meta */}
        <div className="text-xs text-muted-foreground space-y-0.5">
          {order.CustomerName && (
            <p className="text-foreground font-medium">
              Khách: {order.CustomerName}
              {order.CustomerPhone ? ` · ${order.CustomerPhone}` : ''}
            </p>
          )}
          {order.Fulfillment === 'DELIVERY' && order.LocationPath && (
            <p>Giao đến: {order.LocationPath}</p>
          )}
          <p>Hình thức: {order.Fulfillment === 'DELIVERY' ? 'Giao hàng' : 'Tự lấy'}</p>
          <p>Thanh toán: {order.PaymentMethod}</p>
          <p>Đặt lúc: {new Date(order.PlacedAt).toLocaleString('vi-VN')}</p>
        </div>
        <DesiredTimeMeta order={order} />

        {/* Fulfillment badge for PICKUP */}
        {order.Fulfillment === 'PICKUP' && order.PickupPin && (
          <div className="flex items-center gap-2 rounded-md bg-primary/5 px-3 py-2">
            <KeyRound className="h-4 w-4 text-primary" />
            <span className="text-sm font-medium">PIN: </span>
            <span className="font-mono font-bold tracking-widest text-primary">{order.PickupPin}</span>
          </div>
        )}

        {/* Action buttons */}
        <div className="flex flex-wrap gap-2">
          {order.Status === 'PENDING' && (
            <>
              <Button
                size="sm"
                onClick={handleConfirm}
                disabled={mutations.confirm.isPending}
                className="flex-1"
              >
                <CheckCircle className="h-4 w-4 mr-1.5" />
                Xác nhận
              </Button>
              <Button
                size="sm"
                variant="destructive"
                onClick={handleReject}
                disabled={mutations.reject.isPending}
                className="flex-1"
              >
                <XCircle className="h-4 w-4 mr-1.5" />
                Từ chối
              </Button>
            </>
          )}

          {nextStatus && order.Status !== 'PENDING' && (
            <Button
              size="sm"
              variant="outline"
              onClick={handleAdvance}
              disabled={mutations.advanceStatus.isPending}
              className="flex-1"
            >
              Chuyển sang: {orderStatusLabel(nextStatus)}
            </Button>
          )}

          {order.Fulfillment === 'PICKUP' && order.Status === 'READY' && (
            <Button
              size="sm"
              variant="outline"
              onClick={() => setShowPinDialog((v) => !v)}
              className="flex-1"
            >
              <KeyRound className="h-4 w-4 mr-1.5" />
              Xác nhận PIN
            </Button>
          )}
        </div>

        {/* Inline PIN dialog */}
        {showPinDialog && (
          <div className="space-y-2 rounded-lg border border-border p-3">
            <p className="text-sm font-medium">Nhập PIN từ khách hàng</p>
            <div className="flex gap-2">
              <input
                type="text"
                inputMode="numeric"
                value={pinInput}
                onChange={(e) => setPinInput(e.target.value.replace(/\D/g, ''))}
                placeholder="Mã PIN"
                className="border-input bg-background focus-visible:ring-ring h-9 flex-1 rounded-md border px-3 text-sm font-mono tracking-widest focus-visible:ring-2 focus-visible:outline-none"
                maxLength={8}
              />
              <Button
                size="sm"
                onClick={handlePickupVerify}
                disabled={!pinInput.trim() || mutations.verifyPickup.isPending}
              >
                {mutations.verifyPickup.isPending ? 'Đang kiểm tra…' : 'Xác nhận'}
              </Button>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
