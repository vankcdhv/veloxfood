'use client';

import { useState } from 'react';
import { Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Badge } from '@/shared/ui/badge';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { ConfirmDialog } from '@/shared/ui/confirm-dialog';
import { formatVnd } from '@/shared/lib/format-vnd';
import { formatDate, formatDateTime } from '@/shared/lib/format-date';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { usePayoutBatches, usePayoutMutations } from '../hooks/use-payouts';
import type { PayoutBatch } from '../types/payout';

interface Props {
  storeId: string;
}

export function PayoutBatchHistoryTable({ storeId }: Props) {
  const { data: batches, isLoading, isError } = usePayoutBatches(storeId);
  const { executeBatch } = usePayoutMutations(storeId);
  const [executeTarget, setExecuteTarget] = useState<PayoutBatch | null>(null);

  const confirmExecute = () => {
    if (!executeTarget) return;
    executeBatch.mutate(executeTarget.ID, {
      onSuccess: () => {
        toast.success('Đã thực hiện chi trả thành công');
        setExecuteTarget(null);
      },
      onError: (err) => toast.error(getApiErrorMessage(err, 'Thực hiện chi trả thất bại')),
    });
  };

  return (
    <Card>
      <CardHeader className="pb-3">
        <CardTitle className="text-sm">Lịch sử đợt chi trả</CardTitle>
      </CardHeader>
      <CardContent>
        {isLoading && (
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-14 rounded-lg" />
            ))}
          </div>
        )}

        {isError && (
          <p className="text-destructive text-sm">Không tải được lịch sử chi trả.</p>
        )}

        {!isLoading && !isError && (!batches || batches.length === 0) && (
          <div className="border-border bg-muted/40 text-muted-foreground flex h-24 items-center justify-center rounded-lg border border-dashed text-sm">
            Chưa có đợt chi trả nào.
          </div>
        )}

        {batches && batches.length > 0 && (
          <div className="space-y-2">
            {batches.map((batch) => (
              <BatchRow
                key={batch.ID}
                batch={batch}
                onExecute={setExecuteTarget}
                isExecuting={executeBatch.isPending && executeBatch.variables === batch.ID}
              />
            ))}
          </div>
        )}
      </CardContent>

      <ConfirmDialog
        open={!!executeTarget}
        onOpenChange={(o) => !o && setExecuteTarget(null)}
        title="Thực hiện chi trả?"
        description={
          executeTarget
            ? `Chi trả ${formatVnd(executeTarget.TotalAmount)} cho ${executeTarget.OrderIDs.length} đơn. Thao tác không thể hoàn tác.`
            : undefined
        }
        confirmLabel="Chi trả ngay"
        loading={executeBatch.isPending}
        onConfirm={confirmExecute}
      />
    </Card>
  );
}

function BatchRow({
  batch,
  onExecute,
  isExecuting,
}: {
  batch: PayoutBatch;
  onExecute: (b: PayoutBatch) => void;
  isExecuting: boolean;
}) {
  const isPending = batch.Status === 'PENDING';

  return (
    <div className="flex items-center justify-between gap-3 rounded-lg border border-border px-3 py-2.5 text-sm">
      <div className="min-w-0 flex-1 space-y-0.5">
        <div className="flex items-center gap-2">
          <span className="font-semibold">{formatVnd(batch.TotalAmount)}</span>
          <Badge variant={isPending ? 'warning' : 'success'}>
            {isPending ? 'Chờ chi trả' : 'Đã chi trả'}
          </Badge>
        </div>
        <p className="text-xs text-muted-foreground">
          {formatDate(batch.PeriodFrom)} → {formatDate(batch.PeriodTo)} · {batch.OrderIDs.length} đơn
        </p>
        {batch.SettledAt && (
          <p className="text-xs text-muted-foreground">
            Thực hiện: {formatDateTime(batch.SettledAt)}
          </p>
        )}
      </div>

      {isPending && (
        <Button
          size="sm"
          variant="outline"
          className="shrink-0"
          onClick={() => onExecute(batch)}
          disabled={isExecuting}
        >
          {isExecuting ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" />
          ) : (
            'Thực hiện chi trả'
          )}
        </Button>
      )}
    </div>
  );
}
