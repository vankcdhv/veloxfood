'use client';

import { useQuery } from '@tanstack/react-query';
import { Wallet, Clock } from 'lucide-react';
import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse } from '@/shared/api/api-response';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Badge } from '@/shared/ui/badge';
import { Skeleton } from '@/shared/ui/skeleton';
import { formatVnd } from '@/shared/lib/format-vnd';

interface SettleableOrder {
  OrderID: string;
  Amount: number;
  CreatedAt: string;
}
interface RevenueSummary {
  StoreID: string;
  PayableBalance: number;
  Orders: SettleableOrder[] | null;
  Total: number;
}
interface PayoutBatch {
  ID: string;
  PeriodFrom?: string;
  PeriodTo?: string;
  TotalAmount: number;
  Status: string;
  SettledAt?: string | null;
  CreatedAt: string;
}
interface RevenueResponse {
  summary: RevenueSummary;
  batches: PayoutBatch[] | null;
}

const BATCH_STATUS: Record<string, { label: string; variant: 'success' | 'warning' | 'outline' }> = {
  SETTLED: { label: 'Đã chi trả', variant: 'success' },
  PENDING: { label: 'Chờ chi trả', variant: 'warning' },
};

function formatDate(iso?: string | null): string {
  if (!iso) return '—';
  return new Date(iso).toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit', year: 'numeric' });
}

export function VendorRevenuePanel({ storeId }: { storeId: string }) {
  const { data, isLoading, isError } = useQuery({
    queryKey: ['store-revenue', storeId],
    queryFn: async () => {
      const res = await http.get<ApiResponse<RevenueResponse>>(`${API_PREFIX}/me/store-revenue`, {
        params: { store_id: storeId },
      });
      if (!res.data.data) throw new Error(res.data.error ?? 'Empty');
      return res.data.data;
    },
    enabled: !!storeId,
  });

  if (isLoading) return <Skeleton className="h-48 rounded-xl" />;
  if (isError || !data) return <p className="text-destructive text-sm">Không tải được doanh thu.</p>;

  const orders = data.summary.Orders ?? [];
  const batches = data.batches ?? [];

  return (
    <div className="space-y-4">
      <Card className="border-primary/30 bg-primary/5">
        <CardContent className="flex items-center gap-4 py-5">
          <span className="bg-primary/15 text-primary flex h-12 w-12 items-center justify-center rounded-xl">
            <Wallet className="h-6 w-6" />
          </span>
          <div>
            <p className="text-muted-foreground text-sm">Số dư chờ chi trả</p>
            <p className="text-primary font-serif text-2xl font-bold">{formatVnd(data.summary.PayableBalance)}</p>
            <p className="text-muted-foreground text-xs">{orders.length} đơn chưa đối soát</p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Đơn chưa đối soát</CardTitle>
        </CardHeader>
        <CardContent>
          {orders.length === 0 ? (
            <p className="text-muted-foreground text-sm py-4 text-center">Chưa có khoản nào chờ đối soát.</p>
          ) : (
            <ul className="divide-y divide-border">
              {orders.map((o) => (
                <li key={o.OrderID} className="flex items-center justify-between py-2 text-sm">
                  <span className="text-muted-foreground flex items-center gap-1.5">
                    <Clock className="h-3.5 w-3.5" /> {formatDate(o.CreatedAt)}
                  </span>
                  <span className="font-semibold text-primary tabular-nums">{formatVnd(o.Amount)}</span>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Lịch sử chi trả</CardTitle>
        </CardHeader>
        <CardContent>
          {batches.length === 0 ? (
            <p className="text-muted-foreground text-sm py-4 text-center">Chưa có đợt chi trả nào.</p>
          ) : (
            <ul className="divide-y divide-border">
              {batches.map((b) => (
                <li key={b.ID} className="flex items-center justify-between gap-2 py-2.5 text-sm">
                  <div className="text-muted-foreground">
                    {formatDate(b.PeriodFrom)} – {formatDate(b.PeriodTo)}
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="font-semibold tabular-nums">{formatVnd(b.TotalAmount)}</span>
                    <Badge variant={BATCH_STATUS[b.Status]?.variant ?? 'outline'}>
                      {BATCH_STATUS[b.Status]?.label ?? b.Status}
                    </Badge>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
