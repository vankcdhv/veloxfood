'use client';

import { useState } from 'react';
import { Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import { Skeleton } from '@/shared/ui/skeleton';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useStoreHours, useSlotQuota, useSlotQuotaMutations } from '../hooks/use-stores';
import type { MenuItem } from '../types/store';

interface Props {
  storeId: string;
  item: MenuItem | null;
  onClose: () => void;
}

// Format "HH:MM" cutoff label with lead minutes hint.
function cutoffLabel(cutoffTime: string, leadMinutes: number) {
  return `${cutoffTime} (còn ${leadMinutes} phút)`;
}

// Returns today's date as "YYYY-MM-DD" in local time.
function todayStr() {
  const d = new Date();
  const yyyy = d.getFullYear();
  const mm = String(d.getMonth() + 1).padStart(2, '0');
  const dd = String(d.getDate()).padStart(2, '0');
  return `${yyyy}-${mm}-${dd}`;
}

export function VendorMenuItemQuotaDialog({ storeId, item, onClose }: Props) {
  const [date, setDate] = useState(todayStr());
  const [cutoffId, setCutoffId] = useState('');
  const [quota, setQuota] = useState('');

  const { data: hoursData, isLoading: hoursLoading } = useStoreHours(storeId);
  const cutoffs = hoursData?.ship_cutoffs ?? [];

  // Fetch quotas for the selected date (only when item is open).
  const {
    data: quotas,
    isLoading: quotasLoading,
  } = useSlotQuota(storeId, item?.ID ?? '', date);

  const { createQuota } = useSlotQuotaMutations(storeId, item?.ID ?? '');

  const submit = () => {
    const q = parseInt(quota, 10);
    if (!cutoffId || isNaN(q) || q < 1) {
      toast.error('Vui lòng chọn mốc giờ và nhập số lượng hợp lệ.');
      return;
    }
    createQuota.mutate(
      { date, cutoff_id: cutoffId, quota: q },
      {
        onSuccess: () => {
          toast.success('Đã thiết lập số lượng.');
          setQuota('');
          setCutoffId('');
        },
        onError: (e) => toast.error(getApiErrorMessage(e, 'Thiết lập thất bại')),
      },
    );
  };

  // Resolve cutoff label for a given ID.
  const resolveCutoffLabel = (id: string) => {
    const c = cutoffs.find((ct) => ct.ID === id);
    return c ? cutoffLabel(c.CutoffTime, c.LeadMinutes) : id;
  };

  return (
    <Dialog open={!!item} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Số lượng — {item?.Name}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4">
          {/* Date picker */}
          <div>
            <label className="text-foreground mb-1.5 block text-sm font-medium">Ngày</label>
            <Input
              type="date"
              value={date}
              onChange={(e) => setDate(e.target.value)}
            />
          </div>

          {/* Cutoff picker */}
          {hoursLoading ? (
            <Skeleton className="h-9" />
          ) : cutoffs.length === 0 ? (
            <p className="text-muted-foreground text-sm rounded-lg border border-border px-3 py-2">
              Hãy thiết lập mốc giờ chốt (tab Giờ hoạt động) trước.
            </p>
          ) : (
            <div>
              <label className="text-foreground mb-1.5 block text-sm font-medium">Mốc giờ chốt</label>
              <select
                value={cutoffId}
                onChange={(e) => setCutoffId(e.target.value)}
                className="border-input bg-background w-full h-9 rounded-md border px-2 text-sm"
              >
                <option value="">Chọn mốc giờ…</option>
                {cutoffs.map((c) => (
                  <option key={c.ID} value={c.ID}>
                    {cutoffLabel(c.CutoffTime, c.LeadMinutes)}
                  </option>
                ))}
              </select>
            </div>
          )}

          {/* Quota input */}
          <div>
            <label className="text-foreground mb-1.5 block text-sm font-medium">Số lượng tối đa</label>
            <Input
              type="number"
              min={1}
              value={quota}
              onChange={(e) => setQuota(e.target.value)}
              placeholder="VD: 50"
              disabled={cutoffs.length === 0}
            />
          </div>

          {/* Existing quotas for selected date */}
          <div className="border-t border-border pt-3 space-y-1">
            <p className="text-xs font-medium text-muted-foreground mb-2">
              Đã thiết lập ngày {date}
            </p>
            {quotasLoading && <Skeleton className="h-8" />}
            {!quotasLoading && (!quotas || quotas.length === 0) && (
              <p className="text-muted-foreground text-xs">Chưa có mốc nào.</p>
            )}
            {quotas?.map((q) => (
              <div
                key={q.ID}
                className="flex items-center justify-between rounded-md border border-border px-3 py-1.5 text-sm"
              >
                <span className="text-muted-foreground text-xs">
                  Đợt {resolveCutoffLabel(q.CutoffID)}
                </span>
                <span className="text-xs font-medium">
                  đã bán {q.SoldCount}/{q.Quota}
                </span>
              </div>
            ))}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Đóng</Button>
          {cutoffs.length > 0 && (
            <Button onClick={submit} disabled={createQuota.isPending || !cutoffId}>
              {createQuota.isPending
                ? <Loader2 className="h-4 w-4 animate-spin" />
                : 'Thiết lập'}
            </Button>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
