'use client';

import { useState } from 'react';
import { AlertTriangle, CheckCircle2, ImageIcon } from 'lucide-react';
import { toast } from 'sonner';
import { Badge } from '@/shared/ui/badge';
import { Button } from '@/shared/ui/button';
import { Card, CardContent } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { ConfirmDialog } from '@/shared/ui/confirm-dialog';
import { formatDateTime } from '@/shared/lib/format-date';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useAdminIncidents, useResolveIncident } from '../hooks/use-deliveries';
import { incidentTypeLabel, type AdminIncident } from '../types/delivery';

const STATUS_BADGE: Record<string, { variant: 'warning' | 'success'; label: string }> = {
  OPEN: { variant: 'warning', label: 'Chưa xử lý' },
  RESOLVED: { variant: 'success', label: 'Đã xử lý' },
};

// AdminIncidentsTable lists every incident shippers reported, with the resolved
// order code + shipper name and a link to the evidence photo (no raw UUIDs).
export function AdminIncidentsTable() {
  const { data: incidents, isLoading, isError } = useAdminIncidents();
  const resolve = useResolveIncident();
  const [pendingResolve, setPendingResolve] = useState<AdminIncident | null>(null);

  const confirmResolve = () => {
    if (!pendingResolve) return;
    resolve.mutate(pendingResolve.id, {
      onSuccess: () => {
        toast.success('Đã đánh dấu xử lý.');
        setPendingResolve(null);
      },
      onError: (e) => toast.error(getApiErrorMessage(e, 'Không cập nhật được sự cố')),
    });
  };

  if (isLoading) {
    return (
      <div className="space-y-3">
        {[1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-24 rounded-xl" />
        ))}
      </div>
    );
  }

  if (isError) {
    return <p className="text-destructive text-sm py-12 text-center">Không tải được danh sách sự cố.</p>;
  }

  if (!incidents || incidents.length === 0) {
    return (
      <div className="text-muted-foreground flex flex-col items-center gap-3 py-20">
        <AlertTriangle className="h-10 w-10 opacity-30" />
        <p className="text-sm">Chưa có sự cố nào được báo cáo.</p>
      </div>
    );
  }

  return (
    <div className="space-y-3">
      {incidents.map((inc) => {
        const badge = STATUS_BADGE[inc.status] ?? { variant: 'warning' as const, label: inc.status };
        return (
          <Card key={inc.id}>
            <CardContent className="space-y-2 py-4">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-semibold">{inc.order_code || '—'}</span>
                  <Badge variant="outline" className="text-xs">{incidentTypeLabel(inc.type)}</Badge>
                  <Badge variant={badge.variant} className="text-xs">{badge.label}</Badge>
                </div>
                <span className="text-muted-foreground text-xs">{formatDateTime(inc.created_at)}</span>
              </div>

              <p className="text-sm">{inc.note}</p>

              <div className="text-muted-foreground flex items-center justify-between gap-3 text-xs">
                <span>Shipper: {inc.shipper_name || '—'}</span>
                {inc.photo_url ? (
                  <a
                    href={inc.photo_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-primary inline-flex items-center gap-1 font-medium hover:underline"
                  >
                    <ImageIcon className="h-3.5 w-3.5" />
                    Xem ảnh
                  </a>
                ) : (
                  <span>Không có ảnh</span>
                )}
              </div>

              {inc.status === 'OPEN' && (
                <div className="flex justify-end border-t border-border pt-2">
                  <Button
                    size="sm"
                    variant="outline"
                    className="h-8 gap-1 text-xs"
                    onClick={() => setPendingResolve(inc)}
                  >
                    <CheckCircle2 className="h-3.5 w-3.5" />
                    Đánh dấu đã xử lý
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>
        );
      })}

      <ConfirmDialog
        open={!!pendingResolve}
        onOpenChange={(o) => !o && setPendingResolve(null)}
        title="Đánh dấu đã xử lý?"
        description={
          pendingResolve
            ? `Đánh dấu sự cố của đơn ${pendingResolve.order_code || '—'} là đã xử lý.`
            : undefined
        }
        confirmLabel="Xác nhận"
        loading={resolve.isPending}
        onConfirm={confirmResolve}
      />
    </div>
  );
}
