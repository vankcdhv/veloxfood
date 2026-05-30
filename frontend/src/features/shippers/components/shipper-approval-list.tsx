'use client';

import { useState } from 'react';
import { Check, Loader2, X } from 'lucide-react';
import { toast } from 'sonner';
import { Badge } from '@/shared/ui/badge';
import { Button } from '@/shared/ui/button';
import { Skeleton } from '@/shared/ui/skeleton';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog';
import { Input } from '@/shared/ui/input';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useApproveShipper, useRejectShipper, useShippers } from '../hooks/use-shippers';
import type { ShipperProfile, ShipperStatus } from '../types/shipper';

const STATUS_TABS: { key: ShipperStatus; label: string }[] = [
  { key: 'pending', label: 'Chờ duyệt' },
  { key: 'approved', label: 'Đã duyệt' },
  { key: 'rejected', label: 'Từ chối' },
];

const STATUS_BADGE: Record<ShipperStatus, { variant: 'warning' | 'success' | 'destructive'; label: string }> = {
  pending: { variant: 'warning', label: 'Chờ duyệt' },
  approved: { variant: 'success', label: 'Đã duyệt' },
  rejected: { variant: 'destructive', label: 'Từ chối' },
};

export function ShipperApprovalList() {
  const [status, setStatus] = useState<ShipperStatus>('pending');
  const { data, isLoading, isError } = useShippers({ status, page: 1, page_size: 50 });
  const approve = useApproveShipper();
  const [rejectTarget, setRejectTarget] = useState<ShipperProfile | null>(null);

  const onApprove = async (s: ShipperProfile) => {
    try {
      await approve.mutateAsync(s.user_id);
      toast.success('Đã duyệt shipper.');
    } catch (err) {
      toast.error(getApiErrorMessage(err, 'Duyệt thất bại'));
    }
  };

  return (
    <div className="space-y-4">
      {/* Status filter */}
      <div className="bg-muted inline-flex rounded-lg p-1" role="tablist" aria-label="Lọc theo trạng thái">
        {STATUS_TABS.map((t) => (
          <button
            key={t.key}
            role="tab"
            aria-selected={status === t.key}
            onClick={() => setStatus(t.key)}
            className={`focus-visible:ring-ring rounded-md px-3 py-1.5 text-sm font-medium transition-colors duration-200 focus-visible:ring-2 focus-visible:outline-none ${
              status === t.key ? 'bg-background text-foreground shadow-sm' : 'text-muted-foreground hover:text-foreground'
            }`}
          >
            {t.label}
          </button>
        ))}
      </div>

      <div className="border-border overflow-hidden rounded-lg border">
        <table className="w-full text-sm">
          <thead className="bg-muted/50 text-muted-foreground">
            <tr>
              <th className="px-4 py-3 text-left font-medium">Người dùng</th>
              <th className="px-4 py-3 text-left font-medium">Ngày nộp</th>
              <th className="px-4 py-3 text-left font-medium">Trạng thái</th>
              <th className="px-4 py-3 text-right font-medium">Hành động</th>
            </tr>
          </thead>
          <tbody className="divide-border divide-y">
            {isLoading && (
              <tr>
                <td colSpan={4} className="px-4 py-4">
                  <Skeleton className="h-6 w-full" />
                </td>
              </tr>
            )}
            {isError && (
              <tr>
                <td colSpan={4} className="text-destructive px-4 py-6 text-center">
                  Không tải được danh sách.
                </td>
              </tr>
            )}
            {!isLoading && !isError && (data?.items.length ?? 0) === 0 && (
              <tr>
                <td colSpan={4} className="text-muted-foreground px-4 py-10 text-center">
                  Không có hồ sơ nào ở trạng thái này.
                </td>
              </tr>
            )}
            {data?.items.map((s) => (
              <tr key={s.user_id} className="hover:bg-accent/30 transition-colors">
                <td className="px-4 py-3">
                  <p className="text-sm font-medium">{s.full_name || '—'}</p>
                  {s.email && <p className="text-muted-foreground text-xs">{s.email}</p>}
                </td>
                <td className="px-4 py-3">{new Date(s.created_at).toLocaleString('vi-VN')}</td>
                <td className="px-4 py-3">
                  <Badge variant={STATUS_BADGE[s.status].variant}>{STATUS_BADGE[s.status].label}</Badge>
                </td>
                <td className="px-4 py-3">
                  {s.status === 'pending' ? (
                    <div className="flex justify-end gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => onApprove(s)}
                        disabled={approve.isPending}
                      >
                        <Check className="h-4 w-4" />
                        Duyệt
                      </Button>
                      <Button size="sm" variant="destructive" onClick={() => setRejectTarget(s)}>
                        <X className="h-4 w-4" />
                        Từ chối
                      </Button>
                    </div>
                  ) : (
                    <span className="text-muted-foreground block text-right">—</span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>

      <RejectDialog target={rejectTarget} onClose={() => setRejectTarget(null)} />
    </div>
  );
}

function RejectDialog({ target, onClose }: { target: ShipperProfile | null; onClose: () => void }) {
  const reject = useRejectShipper();
  const [reason, setReason] = useState('');

  const submit = async () => {
    if (!target) return;
    try {
      await reject.mutateAsync({ userId: target.user_id, reason });
      toast.success('Đã từ chối hồ sơ.');
      setReason('');
      onClose();
    } catch (err) {
      toast.error(getApiErrorMessage(err, 'Từ chối thất bại'));
    }
  };

  return (
    <Dialog open={!!target} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Từ chối hồ sơ shipper</DialogTitle>
          <DialogDescription>Nêu lý do để người dùng biết cần bổ sung gì.</DialogDescription>
        </DialogHeader>
        <Input
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          placeholder="Lý do từ chối (tuỳ chọn)"
          aria-label="Lý do từ chối"
        />
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={reject.isPending}>
            Huỷ
          </Button>
          <Button variant="destructive" onClick={submit} disabled={reject.isPending}>
            {reject.isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : 'Xác nhận từ chối'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
