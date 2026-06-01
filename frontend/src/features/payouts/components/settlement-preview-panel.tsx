'use client';

import { useState } from 'react';
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

export function SettlementPreviewPanel({ storeId, storeName }: Props) {
  const { data, isLoading, isError } = useSettlement(storeId);
  const { createBatch } = usePayoutMutations(storeId);
  const [confirmOpen, setConfirmOpen] = useState(false);

  const confirmCreateBatch = () => {
    if (!data || data.Orders.length === 0) return;

    // Period = min/max of order dates; fall back to today if only one order.
    const dates = data.Orders.map((o) => o.CreatedAt).sort();
    const periodFrom = dates[0].slice(0, 10);
    const periodTo = (dates[dates.length - 1] ?? dates[0]).slice(0, 10);

    createBatch.mutate(
      {
        store_id: storeId,
        period_from: periodFrom,
        period_to: periodTo,
        order_ids: data.Orders.map((o) => o.OrderID),
        total_amount: data.Total,
      },
      {
        onSuccess: () => {
          toast.success(`Đã tạo đợt chi trả cho "${storeName}"`);
          setConfirmOpen(false);
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
        {/* Balance highlight */}
        <div className="rounded-lg border border-primary/20 bg-primary/5 px-4 py-3">
          <p className="text-xs text-muted-foreground">Số dư khả dụng</p>
          <p className="text-2xl font-bold text-primary mt-0.5">{formatVnd(data.PayableBalance)}</p>
          <p className="text-xs text-muted-foreground mt-1">
            Tổng đợt này: {formatVnd(data.Total)} · {data.Orders.length} đơn
          </p>
        </div>

        {/* Settleable orders list */}
        {data.Orders.length === 0 ? (
          <p className="text-muted-foreground text-sm text-center py-4">
            Không có đơn nào cần chi trả.
          </p>
        ) : (
          <div className="space-y-1 max-h-56 overflow-y-auto">
            {data.Orders.map((order, i) => (
              <div
                key={order.OrderID}
                className="flex items-center justify-between rounded-md px-3 py-1.5 text-sm hover:bg-muted/50"
              >
                <span className="text-xs text-muted-foreground">Đơn {i + 1}</span>
                <span className="text-xs text-muted-foreground">
                  {formatDate(order.CreatedAt)}
                </span>
                <span className="font-semibold text-sm">{formatVnd(order.Amount)}</span>
              </div>
            ))}
          </div>
        )}

        <Button
          className="w-full"
          onClick={() => setConfirmOpen(true)}
          disabled={createBatch.isPending || data.Orders.length === 0}
        >
          {createBatch.isPending ? (
            <><Loader2 className="h-4 w-4 mr-2 animate-spin" />Đang tạo…</>
          ) : (
            'Tạo đợt chi trả'
          )}
        </Button>
      </CardContent>

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title="Tạo đợt chi trả?"
        description={`Gộp ${data.Orders.length} đơn thành một đợt chi trả ${formatVnd(data.Total)} cho "${storeName}". Bạn vẫn cần bấm "Thực hiện chi trả" để thanh toán thật.`}
        confirmLabel="Tạo đợt"
        loading={createBatch.isPending}
        onConfirm={confirmCreateBatch}
      />
    </Card>
  );
}
