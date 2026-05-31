'use client';

import { RefreshCw, MapPin, Package } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Card, CardContent } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { formatVnd } from '@/features/wallet/lib/format-vnd';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useAvailableDeliveries, useClaimDelivery } from '../hooks/use-deliveries';
import type { AvailableDelivery } from '../types/delivery';

export function AvailableDeliveriesList() {
  const { data: deliveries, isLoading, isError, refetch, isFetching } = useAvailableDeliveries();
  const claimMutation = useClaimDelivery();

  const handleClaim = async (orderId: string) => {
    try {
      await claimMutation.mutateAsync(orderId);
      toast.success('Đã nhận đơn thành công!');
    } catch (err) {
      const msg = getApiErrorMessage(err, 'Không nhận được đơn');
      // 409 = already claimed by another shipper
      if (msg.toLowerCase().includes('409') || msg.toLowerCase().includes('already') || msg.toLowerCase().includes('claimed')) {
        toast.error('Đơn đã có người nhận');
      } else {
        toast.error(msg);
      }
    }
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h3 className="font-semibold text-sm">
          Đơn có sẵn ({deliveries?.length ?? 0})
        </h3>
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
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-20 rounded-lg" />
          ))}
        </div>
      )}

      {isError && (
        <p className="text-destructive text-sm">Không tải được danh sách đơn.</p>
      )}

      {!isLoading && deliveries?.length === 0 && (
        <p className="text-muted-foreground text-sm text-center py-8">
          Hiện không có đơn nào để nhận.
        </p>
      )}

      <div className="space-y-2">
        {deliveries?.map((delivery) => (
          <AvailableDeliveryCard
            key={delivery.order_id}
            delivery={delivery}
            onClaim={handleClaim}
            isClaiming={claimMutation.isPending && claimMutation.variables === delivery.order_id}
          />
        ))}
      </div>
    </div>
  );
}

function AvailableDeliveryCard({
  delivery,
  onClaim,
  isClaiming,
}: {
  delivery: AvailableDelivery;
  onClaim: (orderId: string) => void;
  isClaiming: boolean;
}) {
  return (
    <Card className="border-border">
      <CardContent className="px-3 py-2.5">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0 space-y-1">
            <div className="flex items-center gap-1.5 text-sm font-medium">
              <MapPin className="h-3.5 w-3.5 shrink-0 text-muted-foreground" />
              <span className="truncate">{delivery.room_path}</span>
            </div>
            <div className="flex items-center gap-1.5 text-xs text-muted-foreground">
              <Package className="h-3 w-3 shrink-0" />
              <span className="truncate">Cửa hàng: {delivery.store_id.slice(0, 8)}…</span>
            </div>
            <p className="text-sm font-semibold text-primary">
              Phí ship: {formatVnd(delivery.ship_fee)}
            </p>
          </div>
          <Button
            size="sm"
            className="shrink-0"
            onClick={() => onClaim(delivery.order_id)}
            disabled={isClaiming}
          >
            {isClaiming ? 'Đang nhận…' : 'Nhận đơn'}
          </Button>
        </div>
        <p className="mt-1.5 text-xs text-muted-foreground">
          {new Date(delivery.created_at).toLocaleString('vi-VN')}
        </p>
      </CardContent>
    </Card>
  );
}
