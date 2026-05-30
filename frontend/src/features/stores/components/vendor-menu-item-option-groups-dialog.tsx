'use client';

import { Loader2 } from 'lucide-react';
import { toast } from 'sonner';
import { Skeleton } from '@/shared/ui/skeleton';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog';
import { Button } from '@/shared/ui/button';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import {
  useOptionGroups,
  useMenuItemOptionGroups,
  useMenuItemOptionGroupMutations,
} from '../hooks/use-stores';
import type { MenuItem } from '../types/store';

interface Props {
  storeId: string;
  item: MenuItem | null;
  onClose: () => void;
}

export function VendorMenuItemOptionGroupsDialog({ storeId, item, onClose }: Props) {
  const { data: allGroups, isLoading: groupsLoading } = useOptionGroups(storeId);
  const { data: attachedGroups, isLoading: attachedLoading } =
    useMenuItemOptionGroups(storeId, item?.ID ?? '');
  const { attach, detach } = useMenuItemOptionGroupMutations(storeId, item?.ID ?? '');

  const attachedIds = new Set(attachedGroups?.map((g) => g.ID) ?? []);
  const isLoading = groupsLoading || attachedLoading;
  const isMutating = attach.isPending || detach.isPending;

  const toggle = (ogId: string, currentlyAttached: boolean) => {
    if (currentlyAttached) {
      detach.mutate(ogId, {
        onSuccess: () => toast.success('Đã gỡ nhóm tuỳ chọn.'),
        onError: (e) => toast.error(getApiErrorMessage(e, 'Gỡ thất bại')),
      });
    } else {
      attach.mutate(ogId, {
        onSuccess: () => toast.success('Đã gắn nhóm tuỳ chọn.'),
        onError: (e) => toast.error(getApiErrorMessage(e, 'Gắn thất bại')),
      });
    }
  };

  return (
    <Dialog open={!!item} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Nhóm tuỳ chọn — {item?.Name}</DialogTitle>
        </DialogHeader>

        <div className="space-y-2">
          {isLoading && (
            <div className="space-y-2">
              {[1, 2, 3].map((i) => <Skeleton key={i} className="h-10" />)}
            </div>
          )}

          {!isLoading && (!allGroups || allGroups.length === 0) && (
            <p className="text-muted-foreground text-sm">
              Chưa có nhóm tuỳ chọn nào. Hãy tạo nhóm ở tab Tuỳ chọn trước.
            </p>
          )}

          {!isLoading && allGroups?.map((group) => {
            const checked = attachedIds.has(group.ID);
            return (
              <label
                key={group.ID}
                className="flex items-center gap-3 rounded-lg border border-border px-3 py-2.5 cursor-pointer hover:bg-accent/30 transition-colors"
              >
                <input
                  type="checkbox"
                  checked={checked}
                  disabled={isMutating}
                  onChange={() => toggle(group.ID, checked)}
                  className="h-4 w-4 rounded"
                />
                <span className="flex-1 text-sm font-medium">{group.Name}</span>
                <span className="text-xs text-muted-foreground shrink-0">
                  {group.MinSelect}–{group.MaxSelect} chọn
                  {group.Required ? ' · Bắt buộc' : ''}
                </span>
                {isMutating && <Loader2 className="h-3.5 w-3.5 animate-spin text-muted-foreground" />}
              </label>
            );
          })}
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={onClose}>Đóng</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
