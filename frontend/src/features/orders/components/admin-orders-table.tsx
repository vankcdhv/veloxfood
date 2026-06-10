'use client';

import { useMemo, useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Search } from 'lucide-react';
import { Input } from '@/shared/ui/input';
import { Card, CardContent } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { formatVnd } from '@/shared/lib/format-vnd';
import { reportingApi } from '@/features/notifications/api/reporting-api';
import { useStores } from '@/features/stores/hooks/use-stores';
import { OrderStatusBadge } from './order-status-badge';
import type { OrderStatus } from '../types/order';

const STATUS_FILTERS: { value: string; label: string }[] = [
  { value: '', label: 'Tất cả trạng thái' },
  { value: 'PENDING', label: 'Chờ xác nhận' },
  { value: 'CONFIRMED', label: 'Đã xác nhận' },
  { value: 'PREPARING', label: 'Đang chuẩn bị' },
  { value: 'READY', label: 'Sẵn sàng' },
  { value: 'DELIVERING', label: 'Đang giao' },
  { value: 'COMPLETED', label: 'Hoàn thành' },
  { value: 'CANCELLED', label: 'Đã hủy' },
  { value: 'REJECTED', label: 'Bị từ chối' },
];

function formatDatetime(iso: string): string {
  return new Date(iso).toLocaleString('vi-VN', {
    day: '2-digit', month: '2-digit', hour: '2-digit', minute: '2-digit',
  });
}

// Admin order oversight. Reuses GET /admin/orders/recent (reporting). Recent-only
// (no server pagination) — filters apply client-side; enough for monitoring.
export function AdminOrdersTable() {
  const [search, setSearch] = useState('');
  const [status, setStatus] = useState('');

  const { data, isLoading, isError } = useQuery({
    queryKey: ['admin', 'orders', 'recent', 100],
    queryFn: () => reportingApi.recentOrders(100),
    refetchInterval: 20_000,
  });

  // Resolve store names client-side (reporting facts only carry store_id).
  const { data: stores } = useStores();
  const storeName = useMemo(() => {
    const m = new Map<string, string>();
    for (const s of stores ?? []) m.set(s.ID, s.Name);
    return m;
  }, [stores]);

  const rows = useMemo(() => {
    const all = data ?? [];
    const q = search.trim().toUpperCase();
    return all.filter(
      (o) =>
        (!status || o.Status === status) &&
        (!q || (o.Code ?? '').toUpperCase().includes(q)),
    );
  }, [data, search, status]);

  return (
    <div className="space-y-4">
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
        <div className="relative flex-1">
          <Search className="text-muted-foreground pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2" />
          <Input value={search} onChange={(e) => setSearch(e.target.value)} placeholder="Tìm theo mã đơn…" className="pl-9" />
        </div>
        <select value={status} onChange={(e) => setStatus(e.target.value)}
          className="border-input bg-background h-9 rounded-md border px-3 text-sm shrink-0">
          {STATUS_FILTERS.map((f) => <option key={f.value} value={f.value}>{f.label}</option>)}
        </select>
      </div>

      <Card>
        <CardContent className="p-0">
          {isLoading && (
            <div className="space-y-2 p-4">
              {Array.from({ length: 6 }).map((_, i) => <Skeleton key={i} className="h-10 rounded" />)}
            </div>
          )}
          {isError && <p className="text-destructive text-sm p-6 text-center">Không tải được danh sách đơn.</p>}
          {!isLoading && !isError && rows.length === 0 && (
            <p className="text-muted-foreground text-sm p-8 text-center">Không có đơn nào khớp.</p>
          )}
          {!isLoading && !isError && rows.length > 0 && (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border text-muted-foreground text-left">
                    <th className="px-4 py-2 font-medium">Mã đơn</th>
                    <th className="px-4 py-2 font-medium">Cửa hàng</th>
                    <th className="px-4 py-2 font-medium text-right">Tổng</th>
                    <th className="px-4 py-2 font-medium">Trạng thái</th>
                    <th className="px-4 py-2 font-medium whitespace-nowrap">Thời gian</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {rows.map((o) => (
                    <tr key={o.OrderID} className="hover:bg-muted/40 transition-colors">
                      <td className="px-4 py-2 font-mono font-medium">{o.Code ?? o.OrderID.slice(0, 8)}</td>
                      <td className="px-4 py-2 text-muted-foreground truncate max-w-[180px]">{o.StoreName ?? storeName.get(o.StoreID) ?? '—'}</td>
                      <td className="px-4 py-2 text-right font-semibold text-primary tabular-nums">{formatVnd(o.GrandTotal)}</td>
                      <td className="px-4 py-2"><OrderStatusBadge status={o.Status as OrderStatus} /></td>
                      <td className="px-4 py-2 text-muted-foreground whitespace-nowrap">{formatDatetime(o.CreatedAt)}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
