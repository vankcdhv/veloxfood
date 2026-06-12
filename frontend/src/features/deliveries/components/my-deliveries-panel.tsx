'use client';

import { useState } from 'react';
import { RefreshCw, AlertTriangle, TrendingUp, Clock } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Badge } from '@/shared/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { formatVnd } from '@/shared/lib/format-vnd';
import { formatLateBy } from '@/shared/lib/format-late-by';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useMyDeliveries, useUpdateDeliveryStatus } from '../hooks/use-deliveries';
import { DeliveryStatusBadge } from './delivery-status-badge';
import { ReportIncidentDialog } from './report-incident-dialog';
import type { MyDelivery, UpdateStatusBody } from '../types/delivery';

// Format RFC3339 to "HH:MM". Returns '' on invalid input.
function formatHHMM(rfc3339: string | undefined): string {
  if (!rfc3339) return '';
  const d = new Date(rfc3339);
  if (Number.isNaN(d.getTime())) return '';
  return `${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}

// Status transitions a shipper can trigger manually.
const NEXT_STATUS: Partial<Record<string, UpdateStatusBody['status']>> = {
  CLAIMED:   'PICKED_UP',
  PICKED_UP: 'DELIVERING',
  DELIVERING: 'DELIVERED',
};

const NEXT_STATUS_LABEL: Partial<Record<string, string>> = {
  CLAIMED:    'Đã lấy hàng',
  PICKED_UP:  'Đang giao',
  DELIVERING: 'Đã giao',
};

// Only show active deliveries in the action panel; hide terminal states.
const ACTIVE_STATUSES = new Set(['CLAIMED', 'PICKED_UP', 'DELIVERING']);

export function MyDeliveriesPanel() {
  const { data, isLoading, isError, refetch, isFetching } = useMyDeliveries();
  const updateStatus = useUpdateDeliveryStatus();
  const [incidentOrderId, setIncidentOrderId] = useState<string | null>(null);

  const deliveries = data?.deliveries ?? [];
  const earnings = data?.earnings ?? 0;

  const active = deliveries.filter((d) => ACTIVE_STATUSES.has(d.status));
  const history = deliveries.filter((d) => !ACTIVE_STATUSES.has(d.status));

  const handleAdvance = async (delivery: MyDelivery) => {
    const next = NEXT_STATUS[delivery.status];
    if (!next) return;
    try {
      await updateStatus.mutateAsync({ orderId: delivery.order_id, body: { status: next } });
      toast.success('Đã cập nhật trạng thái');
    } catch (err) {
      toast.error(getApiErrorMessage(err, 'Không cập nhật được'));
    }
  };

  return (
    <div className="space-y-4">
      {/* Earnings summary */}
      <Card className="border-primary/20 bg-primary/5">
        <CardContent className="flex items-center gap-3 px-4 py-3">
          <TrendingUp className="h-5 w-5 text-primary shrink-0" />
          <div>
            <p className="text-xs text-muted-foreground">Tổng thu nhập (đã giao)</p>
            <p className="text-lg font-bold text-primary">{formatVnd(earnings)}</p>
          </div>
        </CardContent>
      </Card>

      {/* Active deliveries */}
      <div className="space-y-3">
        <div className="flex items-center justify-between">
          <h3 className="font-semibold text-sm">Đang thực hiện ({active.length})</h3>
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
            {Array.from({ length: 2 }).map((_, i) => (
              <Skeleton key={i} className="h-24 rounded-lg" />
            ))}
          </div>
        )}

        {isError && (
          <p className="text-destructive text-sm">Không tải được danh sách đơn.</p>
        )}

        {!isLoading && active.length === 0 && (
          <p className="text-muted-foreground text-sm text-center py-4">
            Không có đơn đang thực hiện.
          </p>
        )}

        <div className="space-y-2">
          {active.map((delivery) => (
            <ActiveDeliveryCard
              key={delivery.id}
              delivery={delivery}
              onAdvance={handleAdvance}
              onReportIncident={(orderId) => setIncidentOrderId(orderId)}
              isAdvancing={
                updateStatus.isPending &&
                updateStatus.variables?.orderId === delivery.order_id
              }
            />
          ))}
        </div>
      </div>

      {/* Delivery history */}
      {history.length > 0 && (
        <div className="space-y-3">
          <h3 className="font-semibold text-sm">Lịch sử ({history.length})</h3>
          <div className="space-y-2">
            {history.map((delivery) => (
              <HistoryDeliveryRow key={delivery.id} delivery={delivery} />
            ))}
          </div>
        </div>
      )}

      {/* Incident dialog */}
      {incidentOrderId && (
        <ReportIncidentDialog
          orderId={incidentOrderId}
          open={!!incidentOrderId}
          onOpenChange={(open) => { if (!open) setIncidentOrderId(null); }}
        />
      )}
    </div>
  );
}

function ActiveDeliveryCard({
  delivery,
  onAdvance,
  onReportIncident,
  isAdvancing,
}: {
  delivery: MyDelivery;
  onAdvance: (d: MyDelivery) => void;
  onReportIncident: (orderId: string) => void;
  isAdvancing: boolean;
}) {
  const next = NEXT_STATUS[delivery.status];
  const nextLabel = NEXT_STATUS_LABEL[delivery.status];

  return (
    <Card>
      <CardHeader className="pb-2 pt-3 px-3">
        <CardTitle className="text-sm flex items-center justify-between gap-2">
          <span className="font-mono truncate text-xs text-muted-foreground">
            {delivery.order_code || 'Đơn chưa có mã'}
          </span>
          <DeliveryStatusBadge status={delivery.status} />
        </CardTitle>
      </CardHeader>
      <CardContent className="px-3 pb-3 space-y-2">
        <p className="text-sm font-semibold text-primary">{formatVnd(delivery.ship_fee)}</p>
        {delivery.desired_time && (
          <div className="flex items-center gap-1 text-xs text-muted-foreground">
            <Clock className="h-3 w-3 shrink-0" />
            <span>
              Giờ mong muốn:{' '}
              <span className="font-medium text-foreground">{formatHHMM(delivery.desired_time)}</span>
            </span>
          </div>
        )}
        {delivery.claimed_at && (
          <p className="text-xs text-muted-foreground">
            Nhận lúc: {new Date(delivery.claimed_at).toLocaleString('vi-VN')}
          </p>
        )}
        <div className="flex flex-wrap gap-2">
          {next && nextLabel && (
            <Button
              size="sm"
              onClick={() => onAdvance(delivery)}
              disabled={isAdvancing}
              className="flex-1"
            >
              {isAdvancing ? 'Đang cập nhật…' : nextLabel}
            </Button>
          )}
          <Button
            size="sm"
            variant="outline"
            onClick={() => onReportIncident(delivery.order_id)}
            className="flex-1"
          >
            <AlertTriangle className="h-3.5 w-3.5 mr-1.5" />
            Báo sự cố
          </Button>
        </div>
      </CardContent>
    </Card>
  );
}

function HistoryDeliveryRow({ delivery }: { delivery: MyDelivery }) {
  const lateBy = delivery.late_by_minutes ?? 0;
  const desiredTimeStr = formatHHMM(delivery.desired_time);

  return (
    <div className="flex items-center justify-between rounded-lg border border-border px-3 py-2 text-sm">
      <div className="min-w-0 space-y-0.5">
        <p className="font-mono text-xs text-muted-foreground truncate">
          {delivery.order_code || `#${delivery.order_id.slice(0, 8)}…`}
        </p>
        <p className="font-medium">{formatVnd(delivery.ship_fee)}</p>
        {desiredTimeStr && (
          <div className="flex items-center gap-1 text-xs text-muted-foreground">
            <Clock className="h-3 w-3 shrink-0" />
            <span>Giờ mong muốn: {desiredTimeStr}</span>
          </div>
        )}
      </div>
      <div className="flex flex-col items-end gap-1 shrink-0">
        <div className="flex items-center gap-2">
          {delivery.delivered_at && (
            <span className="text-xs text-muted-foreground hidden sm:inline">
              {new Date(delivery.delivered_at).toLocaleDateString('vi-VN')}
            </span>
          )}
          <DeliveryStatusBadge status={delivery.status} />
        </div>
        {delivery.status === 'DELIVERED' && desiredTimeStr && (
          lateBy > 0
            ? <Badge variant="destructive" className="text-xs px-1.5 py-0">Trễ {formatLateBy(lateBy)}</Badge>
            : <Badge variant="success" className="text-xs px-1.5 py-0">Đúng giờ</Badge>
        )}
      </div>
    </div>
  );
}
