'use client';

import { useMemo, useState } from 'react';
import { Loader2, DollarSign } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { ConfirmDialog } from '@/shared/ui/confirm-dialog';
import { formatVnd } from '@/shared/lib/format-vnd';
import { formatDate } from '@/shared/lib/format-date';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useSettlement, usePayoutMutations } from '../hooks/use-payouts';

interface Props {
  storeId: string;
  storeName: string;
}

type Period = 'all' | 'today' | 'week' | 'month' | 'year';

const PERIODS: { value: Period; label: string }[] = [
  { value: 'all', label: 'Tất cả' },
  { value: 'today', label: 'Hôm nay' },
  { value: 'week', label: 'Tuần này' },
  { value: 'month', label: 'Tháng này' },
  { value: 'year', label: 'Năm nay' },
];

// inPeriod tells whether an order's created date falls within the chosen period,
// relative to "now". Week = current calendar week starting Monday.
function inPeriod(iso: string, period: Period): boolean {
  if (period === 'all') return true;
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return false;
  const now = new Date();
  switch (period) {
    case 'today':
      return d.toDateString() === now.toDateString();
    case 'week': {
      const start = new Date(now);
      start.setHours(0, 0, 0, 0);
      start.setDate(start.getDate() - ((start.getDay() + 6) % 7)); // back to Monday
      return d >= start;
    }
    case 'month':
      return d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth();
    case 'year':
      return d.getFullYear() === now.getFullYear();
    default:
      return true;
  }
}

export function SettlementPreviewPanel({ storeId, storeName }: Props) {
  const { data, isLoading, isError } = useSettlement(storeId);
  const { createBatch } = usePayoutMutations(storeId);
  const [confirmOpen, setConfirmOpen] = useState(false);
  const [period, setPeriod] = useState<Period>('all');
  // Track explicitly DESELECTED orders (default: everything selected). This way
  // freshly loaded / filtered-in orders are selected without a state-sync effect.
  const [deselected, setDeselected] = useState<Set<string>>(new Set());

  const allOrders = useMemo(() => data?.Orders ?? [], [data]);
  const filtered = useMemo(
    () => allOrders.filter((o) => inPeriod(o.CreatedAt, period)),
    [allOrders, period],
  );
  const selected = useMemo(
    () => filtered.filter((o) => !deselected.has(o.OrderID)),
    [filtered, deselected],
  );
  const selectedTotal = useMemo(() => selected.reduce((s, o) => s + o.Amount, 0), [selected]);
  const allChecked = filtered.length > 0 && selected.length === filtered.length;

  const toggleOne = (orderId: string) => {
    setDeselected((prev) => {
      const next = new Set(prev);
      if (next.has(orderId)) next.delete(orderId);
      else next.add(orderId);
      return next;
    });
  };

  const toggleAll = () => {
    setDeselected((prev) => {
      const next = new Set(prev);
      if (allChecked) {
        filtered.forEach((o) => next.add(o.OrderID)); // deselect all currently shown
      } else {
        filtered.forEach((o) => next.delete(o.OrderID)); // select all currently shown
      }
      return next;
    });
  };

  const confirmCreateBatch = () => {
    if (selected.length === 0) return;
    const dates = selected.map((o) => o.CreatedAt).sort();
    const periodFrom = dates[0].slice(0, 10);
    const periodTo = (dates[dates.length - 1] ?? dates[0]).slice(0, 10);

    createBatch.mutate(
      {
        store_id: storeId,
        period_from: periodFrom,
        period_to: periodTo,
        order_ids: selected.map((o) => o.OrderID),
        total_amount: selectedTotal,
      },
      {
        onSuccess: () => {
          toast.success(`Đã tạo đợt chi trả cho "${storeName}"`);
          setConfirmOpen(false);
          setDeselected(new Set());
        },
        onError: (err) => toast.error(getApiErrorMessage(err, 'Tạo đợt chi trả thất bại')),
      },
    );
  };

  if (isLoading) {
    return (
      <Card>
        <CardContent className="p-5 space-y-2">
          <Skeleton className="h-6 w-40" />
          <Skeleton className="h-4 w-56" />
          <Skeleton className="h-32 rounded-lg" />
        </CardContent>
      </Card>
    );
  }

  if (isError || !data) {
    return (
      <Card>
        <CardContent className="p-5">
          <p className="text-destructive text-sm">Không tải được thông tin thanh toán.</p>
        </CardContent>
      </Card>
    );
  }

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm flex items-center gap-2">
          <DollarSign className="h-4 w-4" />
          Số dư chưa chi trả — {storeName}
        </CardTitle>
      </CardHeader>
      <CardContent className="space-y-4">
        {/* Balance highlight — reflects the current selection */}
        <div className="rounded-lg border border-primary/20 bg-primary/5 px-4 py-3">
          <p className="text-xs text-muted-foreground">Tổng số dư khả dụng</p>
          <p className="text-2xl font-bold text-primary mt-0.5">{formatVnd(data.PayableBalance)}</p>
          <p className="text-xs text-muted-foreground mt-1">
            Đã chọn: <span className="font-medium text-foreground">{formatVnd(selectedTotal)}</span>{' '}
            · {selected.length}/{allOrders.length} đơn
          </p>
        </div>

        {/* Period filter chips */}
        <div className="flex flex-wrap gap-1.5">
          {PERIODS.map((p) => (
            <button
              key={p.value}
              type="button"
              onClick={() => setPeriod(p.value)}
              className={`rounded-full border px-3 py-1 text-xs transition-colors ${
                period === p.value
                  ? 'border-primary bg-primary text-primary-foreground'
                  : 'border-border bg-background text-muted-foreground hover:bg-muted'
              }`}
            >
              {p.label}
            </button>
          ))}
        </div>

        {/* Settleable orders list */}
        {filtered.length === 0 ? (
          <p className="text-muted-foreground text-sm text-center py-4">
            Không có đơn nào trong kỳ đã chọn.
          </p>
        ) : (
          <div className="space-y-1">
            {/* Select-all header */}
            <label className="flex items-center gap-2 px-3 py-1.5 text-xs text-muted-foreground border-b cursor-pointer">
              <input
                type="checkbox"
                className="h-4 w-4 accent-primary"
                checked={allChecked}
                onChange={toggleAll}
              />
              Chọn tất cả ({filtered.length} đơn trong kỳ)
            </label>

            <div className="space-y-1 max-h-72 overflow-y-auto">
              {filtered.map((order, i) => (
                <label
                  key={order.OrderID}
                  className="flex items-center gap-3 rounded-md px-3 py-1.5 text-sm hover:bg-muted/50 cursor-pointer"
                >
                  <input
                    type="checkbox"
                    className="h-4 w-4 accent-primary shrink-0"
                    checked={!deselected.has(order.OrderID)}
                    onChange={() => toggleOne(order.OrderID)}
                  />
                  <span className="text-xs text-muted-foreground w-12 shrink-0">Đơn {i + 1}</span>
                  <span className="text-xs text-muted-foreground flex-1">
                    {formatDate(order.CreatedAt)}
                  </span>
                  <span className="font-semibold text-sm">{formatVnd(order.Amount)}</span>
                </label>
              ))}
            </div>
          </div>
        )}

        <Button
          className="w-full"
          onClick={() => setConfirmOpen(true)}
          disabled={createBatch.isPending || selected.length === 0}
        >
          {createBatch.isPending ? (
            <><Loader2 className="h-4 w-4 mr-2 animate-spin" />Đang tạo…</>
          ) : (
            `Tạo đợt chi trả (${selected.length} đơn)`
          )}
        </Button>
      </CardContent>

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title="Tạo đợt chi trả?"
        description={`Gộp ${selected.length} đơn thành một đợt chi trả ${formatVnd(selectedTotal)} cho "${storeName}". Bạn vẫn cần bấm "Thực hiện chi trả" để thanh toán thật.`}
        confirmLabel="Tạo đợt"
        loading={createBatch.isPending}
        onConfirm={confirmCreateBatch}
      />
    </Card>
  );
}
