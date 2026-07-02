'use client';

import { useState } from 'react';
import { Check, X } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Badge } from '@/shared/ui/badge';
import { Skeleton } from '@/shared/ui/skeleton';
import { ConfirmDialog } from '@/shared/ui/confirm-dialog';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { formatDateTime } from '@/shared/lib/format-date';
import { useHoursChangeRequests, useHoursChangeReviewMutations } from '../hooks/use-stores';
import { HoursPayloadSummary } from './vendor-hours-panel';
import type { HoursChangeRequest } from '../types/store';

interface Props {
  storeId: string;
}

type ReviewAction = { req: HoursChangeRequest; kind: 'approve' | 'reject' };

const STATUS_BADGE: Record<string, { variant: 'warning' | 'success' | 'destructive' | 'outline'; label: string }> = {
  pending: { variant: 'warning', label: 'Chờ duyệt' },
  approved: { variant: 'success', label: 'Đã duyệt' },
  rejected: { variant: 'destructive', label: 'Đã từ chối' },
};

export function AdminHoursChangeReview({ storeId }: Props) {
  const { data: requests, isLoading, isError } = useHoursChangeRequests(storeId);
  const m = useHoursChangeReviewMutations(storeId);
  const [pendingAction, setPendingAction] = useState<ReviewAction | null>(null);

  const runAction = async () => {
    if (!pendingAction) return;
    const { req, kind } = pendingAction;
    try {
      if (kind === 'approve') {
        await m.approve.mutateAsync(req.ID);
        toast.success('Đã duyệt yêu cầu.');
      } else {
        await m.reject.mutateAsync(req.ID);
        toast.success('Đã từ chối yêu cầu.');
      }
      setPendingAction(null);
    } catch (e) {
      toast.error(getApiErrorMessage(e, kind === 'approve' ? 'Duyệt thất bại' : 'Từ chối thất bại'));
    }
  };

  if (isLoading) return <Skeleton className="h-24 rounded-xl" />;

  if (isError) {
    return <p className="text-destructive text-sm">Không tải được yêu cầu thay đổi giờ.</p>;
  }

  if (!requests?.length) {
    return <p className="text-muted-foreground text-sm">Không có yêu cầu nào.</p>;
  }

  return (
    <div className="border-border overflow-x-auto rounded-lg border">
      <table className="w-full text-sm">
        <thead className="bg-muted/50 text-muted-foreground">
          <tr>
            <th className="px-4 py-3 text-left font-medium">Ngày gửi</th>
            <th className="px-4 py-3 text-left font-medium">Nội dung</th>
            <th className="px-4 py-3 text-left font-medium">Trạng thái</th>
            <th className="px-4 py-3 text-right font-medium">Hành động</th>
          </tr>
        </thead>
        <tbody className="divide-border divide-y">
          {requests.map((req) => {
            const badgeCfg = STATUS_BADGE[req.Status] ?? { variant: 'outline' as const, label: req.Status };
            const isPending = req.Status === 'pending';
            return (
              <tr key={req.ID} className="hover:bg-accent/30 transition-colors">
                <td className="px-4 py-3 text-xs text-muted-foreground whitespace-nowrap">
                  {formatDateTime(req.CreatedAt)}
                </td>
                <td className="px-4 py-3">
                  <HoursPayloadSummary payload={req.Payload} />
                </td>
                <td className="px-4 py-3">
                  <Badge variant={badgeCfg.variant}>{badgeCfg.label}</Badge>
                </td>
                <td className="px-4 py-3">
                  {isPending ? (
                    <div className="flex justify-end gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => setPendingAction({ req, kind: 'approve' })}
                        disabled={m.approve.isPending || m.reject.isPending}
                      >
                        <Check className="h-3.5 w-3.5" />
                        Duyệt
                      </Button>
                      <Button
                        size="sm"
                        variant="destructive"
                        onClick={() => setPendingAction({ req, kind: 'reject' })}
                        disabled={m.approve.isPending || m.reject.isPending}
                      >
                        <X className="h-3.5 w-3.5" />
                        Từ chối
                      </Button>
                    </div>
                  ) : (
                    <span className="text-muted-foreground block text-right">—</span>
                  )}
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>

      <ConfirmDialog
        open={!!pendingAction}
        onOpenChange={(o) => !o && setPendingAction(null)}
        title={pendingAction?.kind === 'reject' ? 'Từ chối yêu cầu đổi giờ?' : 'Duyệt yêu cầu đổi giờ?'}
        description={
          pendingAction?.kind === 'reject'
            ? 'Yêu cầu sẽ bị từ chối và giờ bán của cửa hàng giữ nguyên.'
            : 'Giờ bán mới sẽ được áp dụng cho cửa hàng ngay sau khi duyệt.'
        }
        confirmLabel={pendingAction?.kind === 'reject' ? 'Từ chối' : 'Duyệt'}
        destructive={pendingAction?.kind === 'reject'}
        loading={m.approve.isPending || m.reject.isPending}
        onConfirm={runAction}
      />
    </div>
  );
}

