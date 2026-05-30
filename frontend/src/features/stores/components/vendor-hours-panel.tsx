'use client';

import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Badge } from '@/shared/ui/badge';
import { Skeleton } from '@/shared/ui/skeleton';
import { useStoreHours, useVendorHoursChangeRequests } from '../hooks/use-stores';
import { VendorHoursSubmitForm } from './vendor-hours-submit-form';
import type { HoursChangeRequest } from '../types/store';

// Vietnamese weekday short labels: index 0=CN … 6=T7
const WEEKDAY_LABELS = ['CN', 'T2', 'T3', 'T4', 'T5', 'T6', 'T7'];

// Full Vietnamese weekday names for payload summaries.
const WEEKDAY_FULL = ['Chủ nhật', 'Thứ 2', 'Thứ 3', 'Thứ 4', 'Thứ 5', 'Thứ 6', 'Thứ 7'];

const STATUS_BADGE_MAP: Record<string, { variant: 'warning' | 'success' | 'destructive'; label: string }> = {
  pending: { variant: 'warning', label: 'Chờ duyệt' },
  approved: { variant: 'success', label: 'Đã duyệt' },
  rejected: { variant: 'destructive', label: 'Đã từ chối' },
};

interface Props {
  storeId: string;
}

export function VendorHoursPanel({ storeId }: Props) {
  return (
    <div className="space-y-6">
      <CurrentHoursCard storeId={storeId} />
      <VendorHoursSubmitForm storeId={storeId} />
      <ChangeRequestHistoryCard storeId={storeId} />
      <p className="text-muted-foreground text-xs border-t border-border pt-3">
        Thay đổi giờ cần admin duyệt mới áp dụng. Trạng thái bán (mở/đóng/tạm dừng) thì tức thì ở tab Thông tin.
      </p>
    </div>
  );
}

// ---------- Current hours (read-only) ----------

function CurrentHoursCard({ storeId }: { storeId: string }) {
  const { data, isLoading, isError } = useStoreHours(storeId);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Giờ hoạt động hiện tại</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading && <Skeleton className="h-20 rounded-lg" />}
        {isError && <p className="text-destructive text-sm">Không tải được giờ hoạt động.</p>}
        {data && data.operating_hours.length === 0 && (
          <p className="text-muted-foreground text-sm">Chưa thiết lập giờ hoạt động.</p>
        )}
        {data && data.operating_hours.length > 0 && (
          <div className="space-y-1">
            {data.operating_hours
              .slice()
              .sort((a, b) => a.Weekday - b.Weekday)
              .map((oh) => (
                <div key={oh.ID} className="flex items-center gap-3 text-sm">
                  <span className="w-8 font-medium text-muted-foreground">
                    {WEEKDAY_LABELS[oh.Weekday]}
                  </span>
                  <span>{oh.OpenTime} – {oh.CloseTime}</span>
                </div>
              ))}
          </div>
        )}
        {data && data.ship_cutoffs.length > 0 && (
          <div className="mt-4 space-y-1">
            <p className="text-xs font-medium text-muted-foreground mb-1">Giờ cắt đơn:</p>
            {data.ship_cutoffs.map((sc) => (
              <div key={sc.ID} className="text-sm">
                {sc.CutoffTime} (trước {sc.LeadMinutes} phút)
              </div>
            ))}
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// ---------- Change request history ----------

function ChangeRequestHistoryCard({ storeId }: { storeId: string }) {
  const { data: requests, isLoading, isError } = useVendorHoursChangeRequests(storeId);

  return (
    <Card>
      <CardHeader>
        <CardTitle className="text-base">Lịch sử yêu cầu</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading && <Skeleton className="h-16 rounded-lg" />}
        {isError && <p className="text-destructive text-sm">Không tải được lịch sử yêu cầu.</p>}
        {!isLoading && requests?.length === 0 && (
          <p className="text-muted-foreground text-sm">Chưa có yêu cầu nào.</p>
        )}
        <div className="space-y-3">
          {requests?.map((req) => (
            <RequestRow key={req.ID} req={req} />
          ))}
        </div>
      </CardContent>
    </Card>
  );
}

function RequestRow({ req }: { req: HoursChangeRequest }) {
  const cfg = STATUS_BADGE_MAP[req.Status] ?? { variant: 'outline' as const, label: req.Status };
  return (
    <div className="rounded-lg border border-border p-3 space-y-2">
      <div className="flex items-center justify-between gap-2 flex-wrap">
        <span className="text-xs text-muted-foreground">
          {new Date(req.CreatedAt).toLocaleString('vi-VN')}
        </span>
        <Badge variant={cfg.variant}>{cfg.label}</Badge>
      </div>
      <HoursPayloadSummary payload={req.Payload} />
    </div>
  );
}

// ---------- Shared payload summary (also exported for admin-hours-change-review) ----------

interface PayloadLike {
  operating_hours?: Array<{ weekday: number; open_time: string; close_time: string }>;
  ship_cutoffs?: Array<{ cutoff_time: string; lead_minutes: number }>;
  note?: string;
}

export function HoursPayloadSummary({ payload }: { payload: PayloadLike | string }) {
  // The backend stores the request payload in a jsonb-as-string column, so the
  // API may serialize it as a JSON string — parse it back to an object first.
  let p: PayloadLike = {};
  if (typeof payload === 'string') {
    try {
      p = JSON.parse(payload) as PayloadLike;
    } catch {
      p = {};
    }
  } else if (payload && typeof payload === 'object') {
    p = payload;
  }

  const hours = Array.isArray(p.operating_hours) ? p.operating_hours : [];
  const cutoffs = Array.isArray(p.ship_cutoffs) ? p.ship_cutoffs : [];
  const note = typeof p.note === 'string' && p.note.trim() ? p.note : null;

  if (hours.length === 0 && cutoffs.length === 0 && !note) {
    return <p className="text-xs text-muted-foreground">Yêu cầu cập nhật giờ hoạt động.</p>;
  }

  return (
    <div className="text-xs space-y-1">
      {hours.length > 0 && (
        <div className="space-y-0.5">
          {hours
            .slice()
            .sort((a, b) => a.weekday - b.weekday)
            .map((h) => (
              <p key={h.weekday}>
                <span className="font-medium">{WEEKDAY_FULL[h.weekday] ?? `Ngày ${h.weekday}`}:</span>{' '}
                {h.open_time} – {h.close_time}
              </p>
            ))}
        </div>
      )}
      {cutoffs.length > 0 && (
        <div className="space-y-0.5 mt-1">
          <p className="text-muted-foreground">Giờ cắt đơn:</p>
          {cutoffs.map((c, i) => (
            <p key={i}>{c.cutoff_time} (trước {c.lead_minutes} phút)</p>
          ))}
        </div>
      )}
      {note && <p className="text-muted-foreground mt-0.5">Ghi chú: {note}</p>}
    </div>
  );
}
