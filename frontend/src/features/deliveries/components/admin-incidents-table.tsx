'use client';

import { AlertTriangle, ImageIcon } from 'lucide-react';
import { Badge } from '@/shared/ui/badge';
import { Card, CardContent } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { formatDateTime } from '@/shared/lib/format-date';
import { useAdminIncidents } from '../hooks/use-deliveries';
import { incidentTypeLabel } from '../types/delivery';

const STATUS_BADGE: Record<string, { variant: 'warning' | 'success'; label: string }> = {
  OPEN: { variant: 'warning', label: 'Chưa xử lý' },
  RESOLVED: { variant: 'success', label: 'Đã xử lý' },
};

// AdminIncidentsTable lists every incident shippers reported, with the resolved
// order code + shipper name and a link to the evidence photo (no raw UUIDs).
export function AdminIncidentsTable() {
  const { data: incidents, isLoading, isError } = useAdminIncidents();

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
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}
