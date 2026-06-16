'use client';

import { useState } from 'react';
import { Loader2, Pencil, Plus, Trash2, ToggleLeft, ToggleRight } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { Badge } from '@/shared/ui/badge';
import { Pagination } from '@/shared/ui/pagination';
import { usePagedState } from '@/shared/hooks/use-paged-state';
import { ConfirmDialog } from '@/shared/ui/confirm-dialog';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { usePromotionMutations, usePromotions } from '../hooks/use-promotions';
import { PromotionStatusBadge } from './promotion-status-badge';
import { PromotionFormDialog } from './promotion-form-dialog';
import type { Promotion, PromotionType, PromotionValueKind } from '../types/promotion';

const PAGE_SIZE = 20;

interface Props {
  storeId: string;
}

const TYPE_LABELS: Record<PromotionType, string> = {
  ORDER_DISCOUNT: 'Đơn hàng',
  SHIP_DISCOUNT: 'Vận chuyển',
};

const KIND_LABELS: Record<PromotionValueKind, string> = {
  PERCENT: '%',
  AMOUNT: 'đ',
};

function formatValue(p: Promotion): string {
  if (p.ValueKind === 'PERCENT') return `${p.Value}%`;
  return `${p.Value.toLocaleString('vi-VN')}đ`;
}

function formatDate(iso: string): string {
  return new Date(iso).toLocaleDateString('vi-VN', { day: '2-digit', month: '2-digit', year: 'numeric' });
}

export function VendorPromotionPanel({ storeId }: Props) {
  // Page resets to 1 when the selected store changes.
  const [page, setPage] = usePagedState(storeId);

  const { data, isLoading, isError } = usePromotions(storeId, page);
  const promotions = data?.items ?? [];
  const totalPages = Math.max(1, Math.ceil((data?.total ?? 0) / PAGE_SIZE));
  const m = usePromotionMutations(storeId);
  const [dialogOpen, setDialogOpen] = useState(false);
  const [editing, setEditing] = useState<Promotion | undefined>(undefined);
  const [pendingDelete, setPendingDelete] = useState<Promotion | null>(null);

  const openCreate = () => { setEditing(undefined); setDialogOpen(true); };
  const openEdit = (p: Promotion) => { setEditing(p); setDialogOpen(true); };
  const closeDialog = () => setDialogOpen(false);

  const handleToggle = (p: Promotion) => {
    const next = p.Status === 'ACTIVE' ? 'INACTIVE' : 'ACTIVE';
    m.toggleStatus.mutate(
      { promoId: p.ID, status: next },
      {
        onSuccess: () => toast.success(next === 'ACTIVE' ? 'Đã bật voucher.' : 'Đã tắt voucher.'),
        onError: (e) => toast.error(getApiErrorMessage(e, 'Thay đổi trạng thái thất bại')),
      },
    );
  };

  const confirmDelete = () => {
    if (!pendingDelete) return;
    m.remove.mutate(pendingDelete.ID, {
      onSuccess: () => { toast.success('Đã xoá voucher.'); setPendingDelete(null); },
      onError: (e) => { toast.error(getApiErrorMessage(e, 'Xoá thất bại')); setPendingDelete(null); },
    });
  };

  return (
    <>
      <Card>
        <CardHeader className="flex flex-row items-center justify-between">
          <CardTitle className="text-base">Khuyến mãi &amp; Voucher</CardTitle>
          <Button
            size="sm"
            onClick={openCreate}
            className="bg-orange-600 hover:bg-orange-700 text-white gap-1"
          >
            <Plus className="h-4 w-4" />
            Tạo voucher
          </Button>
        </CardHeader>

        <CardContent>
          {isLoading && (
            <div className="space-y-2">
              {Array.from({ length: 3 }).map((_, i) => (
                <Skeleton key={i} className="h-12 rounded-lg" />
              ))}
            </div>
          )}

          {isError && (
            <p className="text-destructive text-sm">Không tải được danh sách voucher.</p>
          )}

          {!isLoading && !isError && promotions.length === 0 && (
            <p className="text-muted-foreground text-sm py-4 text-center">
              Chưa có voucher nào. Nhấn &quot;Tạo voucher&quot; để bắt đầu.
            </p>
          )}

          {!isLoading && !isError && promotions.length > 0 && (
            <div className="overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-border text-muted-foreground text-left">
                    <th className="pb-2 pr-3 font-medium">Mã</th>
                    <th className="pb-2 pr-3 font-medium">Loại</th>
                    <th className="pb-2 pr-3 font-medium">Giá trị</th>
                    <th className="pb-2 pr-3 font-medium">Đơn tối thiểu</th>
                    <th className="pb-2 pr-3 font-medium">Thời hạn</th>
                    <th className="pb-2 pr-3 font-medium">Đã dùng</th>
                    <th className="pb-2 font-medium">Trạng thái</th>
                    <th className="pb-2" />
                  </tr>
                </thead>
                <tbody className="divide-y divide-border">
                  {promotions.map((p) => (
                    <PromotionRow
                      key={p.ID}
                      promotion={p}
                      onEdit={openEdit}
                      onToggle={handleToggle}
                      onDelete={setPendingDelete}
                      isToggling={m.toggleStatus.isPending}
                      isDeleting={m.remove.isPending}
                    />
                  ))}
                </tbody>
              </table>
            </div>
          )}

          <Pagination page={page} totalPages={totalPages} onChange={setPage} />
        </CardContent>
      </Card>

      <PromotionFormDialog
        storeId={storeId}
        open={dialogOpen}
        onClose={closeDialog}
        promotion={editing}
      />

      <ConfirmDialog
        open={!!pendingDelete}
        onOpenChange={(o) => !o && setPendingDelete(null)}
        title="Xoá voucher?"
        description={`Xoá voucher "${pendingDelete?.Code ?? ''}". Hành động không thể hoàn tác.`}
        confirmLabel="Xoá"
        destructive
        loading={m.remove.isPending}
        onConfirm={confirmDelete}
      />
    </>
  );
}

interface RowProps {
  promotion: Promotion;
  onEdit: (p: Promotion) => void;
  onToggle: (p: Promotion) => void;
  onDelete: (p: Promotion) => void;
  isToggling: boolean;
  isDeleting: boolean;
}

function PromotionRow({ promotion: p, onEdit, onToggle, onDelete, isToggling, isDeleting }: RowProps) {
  const usageLabel = p.UsageLimit != null
    ? `${p.UsedCount}/${p.UsageLimit}`
    : `${p.UsedCount}/∞`;

  return (
    <tr className="group hover:bg-muted/40 transition-colors">
      <td className="py-2 pr-3">
        <span className="font-mono font-semibold text-orange-600">{p.Code}</span>
      </td>
      <td className="py-2 pr-3">
        <Badge variant="outline" className="text-xs whitespace-nowrap">
          {TYPE_LABELS[p.Type]}
        </Badge>
      </td>
      <td className="py-2 pr-3 font-medium">
        {formatValue(p)}
        <span className="text-muted-foreground text-xs ml-1">
          ({KIND_LABELS[p.ValueKind]})
        </span>
      </td>
      <td className="py-2 pr-3 text-muted-foreground">
        {p.MinOrder > 0 ? `${p.MinOrder.toLocaleString('vi-VN')}đ` : '—'}
      </td>
      <td className="py-2 pr-3 text-muted-foreground whitespace-nowrap">
        {formatDate(p.StartsAt)} – {formatDate(p.EndsAt)}
      </td>
      <td className="py-2 pr-3 text-muted-foreground">{usageLabel}</td>
      <td className="py-2 pr-3">
        <PromotionStatusBadge status={p.Status} />
      </td>
      <td className="py-2">
        <div className="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
          <button
            type="button"
            aria-label={p.Status === 'ACTIVE' ? 'Tắt voucher' : 'Bật voucher'}
            onClick={() => onToggle(p)}
            disabled={isToggling}
            className="text-muted-foreground hover:text-orange-600 p-1 disabled:opacity-30"
          >
            {isToggling
              ? <Loader2 className="h-4 w-4 animate-spin" />
              : p.Status === 'ACTIVE'
                ? <ToggleRight className="h-4 w-4" />
                : <ToggleLeft className="h-4 w-4" />}
          </button>
          <button
            type="button"
            aria-label="Chỉnh sửa voucher"
            onClick={() => onEdit(p)}
            className="text-muted-foreground hover:text-foreground p-1"
          >
            <Pencil className="h-4 w-4" />
          </button>
          <button
            type="button"
            aria-label="Xoá voucher"
            onClick={() => onDelete(p)}
            disabled={isDeleting}
            className="text-muted-foreground hover:text-destructive p-1 disabled:opacity-30"
          >
            <Trash2 className="h-4 w-4" />
          </button>
        </div>
      </td>
    </tr>
  );
}
