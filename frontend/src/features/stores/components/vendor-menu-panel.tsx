'use client';

import { useState } from 'react';
import { ImagePlus, Pencil, Plus, Sliders, Trash2, ToggleLeft, ToggleRight, BarChart2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Badge } from '@/shared/ui/badge';
import { Skeleton } from '@/shared/ui/skeleton';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useStoreMenu, useVendorStoreMutations } from '../hooks/use-stores';
import { AddMenuItemDialog, EditMenuItemDialog, ImageUploadButton } from './vendor-menu-item-dialogs';
import { VendorMenuItemQuotaDialog } from './vendor-menu-item-quota-dialog';
import { VendorMenuItemOptionGroupsDialog } from './vendor-menu-item-option-groups-dialog';
import type { Category, MenuItem } from '../types/store';

interface Props {
  storeId: string;
}

export function VendorMenuPanel({ storeId }: Props) {
  const { data: menu, isLoading } = useStoreMenu(storeId);
  const m = useVendorStoreMutations(storeId);
  const [addTarget, setAddTarget] = useState<Category | null>(null);
  const [editTarget, setEditTarget] = useState<MenuItem | null>(null);
  const [quotaTarget, setQuotaTarget] = useState<MenuItem | null>(null);
  const [optGroupTarget, setOptGroupTarget] = useState<MenuItem | null>(null);

  const toggleStatus = (item: MenuItem) => {
    const next = item.Status === 'on' ? 'off' : 'on';
    m.setMenuItemStatus.mutate(
      { itemId: item.ID, status: next },
      {
        onSuccess: () => toast.success(next === 'on' ? 'Đã bật món.' : 'Đã tắt món.'),
        onError: (e) => toast.error(getApiErrorMessage(e, 'Cập nhật thất bại')),
      },
    );
  };

  const deleteItem = (itemId: string) => {
    m.deleteMenuItem.mutate(itemId, {
      onSuccess: () => toast.success('Đã xoá món.'),
      onError: (e) => toast.error(getApiErrorMessage(e, 'Xoá thất bại')),
    });
  };

  return (
    <>
      <div className="space-y-4">
        {isLoading && <Skeleton className="h-32 rounded-xl" />}
        {!isLoading && (!menu || menu.length === 0) && (
          <p className="text-muted-foreground text-sm">Chưa có danh mục nào. Thêm danh mục trước.</p>
        )}
        {menu?.map(({ Category: cat, Items: items }) => (
          <Card key={cat.ID}>
            <CardHeader className="pb-2">
              <div className="flex items-center justify-between">
                <CardTitle className="text-sm font-semibold">{cat.Name}</CardTitle>
                <Button
                  size="sm"
                  variant="outline"
                  onClick={() => setAddTarget(cat)}
                  className="h-7 gap-1 text-xs"
                >
                  <Plus className="h-3.5 w-3.5" />
                  Thêm món
                </Button>
              </div>
            </CardHeader>
            <CardContent className="space-y-2 pt-0">
              {items.length === 0 && (
                <p className="text-muted-foreground text-xs">Chưa có món nào.</p>
              )}
              {items.map((item) => (
                <MenuItemRow
                  key={item.ID}
                  storeId={storeId}
                  item={item}
                  onToggle={() => toggleStatus(item)}
                  onEdit={() => setEditTarget(item)}
                  onDelete={() => deleteItem(item.ID)}
                  onQuota={() => setQuotaTarget(item)}
                  onOptionGroups={() => setOptGroupTarget(item)}
                  toggling={m.setMenuItemStatus.isPending}
                  deleting={m.deleteMenuItem.isPending}
                />
              ))}
            </CardContent>
          </Card>
        ))}
      </div>

      <AddMenuItemDialog
        storeId={storeId}
        category={addTarget}
        onClose={() => setAddTarget(null)}
      />
      {/* key remounts the dialog each time a different item is selected, resetting form state. */}
      <EditMenuItemDialog
        key={editTarget?.ID ?? 'none'}
        storeId={storeId}
        item={editTarget}
        onClose={() => setEditTarget(null)}
      />
      <VendorMenuItemQuotaDialog
        key={quotaTarget?.ID ?? 'quota-none'}
        storeId={storeId}
        item={quotaTarget}
        onClose={() => setQuotaTarget(null)}
      />
      <VendorMenuItemOptionGroupsDialog
        key={optGroupTarget?.ID ?? 'og-none'}
        storeId={storeId}
        item={optGroupTarget}
        onClose={() => setOptGroupTarget(null)}
      />
    </>
  );
}

// ---------- Menu item row ----------

function MenuItemRow({
  storeId, item, onToggle, onEdit, onDelete, onQuota, onOptionGroups, toggling, deleting,
}: {
  storeId: string;
  item: MenuItem;
  onToggle: () => void;
  onEdit: () => void;
  onDelete: () => void;
  onQuota: () => void;
  onOptionGroups: () => void;
  toggling: boolean;
  deleting: boolean;
}) {
  return (
    <div className="group flex items-center gap-3 rounded-lg border border-border px-3 py-2 hover:bg-accent/30 transition-colors">
      {item.ImageURL ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img
          src={item.ImageURL}
          alt={item.Name}
          className="h-10 w-10 shrink-0 rounded-md object-cover"
        />
      ) : (
        <div className="h-10 w-10 shrink-0 rounded-md bg-muted flex items-center justify-center text-muted-foreground">
          <ImagePlus className="h-4 w-4" />
        </div>
      )}

      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium truncate">{item.Name}</p>
        <p className="text-primary text-xs font-semibold">{item.Price.toLocaleString('vi-VN')}đ</p>
      </div>

      <Badge variant={item.Status === 'on' ? 'success' : 'outline'} className="shrink-0 text-xs">
        {item.Status === 'on' ? 'Đang bán' : 'Tạm ẩn'}
      </Badge>

      <button
        type="button"
        aria-label="Chỉnh sửa món"
        onClick={onEdit}
        className="text-muted-foreground hover:text-primary shrink-0 p-1 opacity-0 group-hover:opacity-100 transition-opacity"
      >
        <Pencil className="h-4 w-4" />
      </button>

      <button
        type="button"
        aria-label="Nhóm tuỳ chọn"
        onClick={onOptionGroups}
        className="text-muted-foreground hover:text-primary shrink-0 p-1 opacity-0 group-hover:opacity-100 transition-opacity"
      >
        <Sliders className="h-4 w-4" />
      </button>

      <button
        type="button"
        aria-label="Số lượng"
        onClick={onQuota}
        className="text-muted-foreground hover:text-primary shrink-0 p-1 opacity-0 group-hover:opacity-100 transition-opacity"
      >
        <BarChart2 className="h-4 w-4" />
      </button>

      <ImageUploadButton storeId={storeId} itemId={item.ID} />

      <button
        type="button"
        aria-label={item.Status === 'on' ? 'Tắt món' : 'Bật món'}
        onClick={onToggle}
        disabled={toggling}
        className="text-muted-foreground hover:text-primary shrink-0 disabled:opacity-30"
      >
        {item.Status === 'on'
          ? <ToggleRight className="h-5 w-5" />
          : <ToggleLeft className="h-5 w-5" />}
      </button>

      <button
        type="button"
        aria-label="Xoá món"
        onClick={onDelete}
        disabled={deleting}
        className="text-muted-foreground hover:text-destructive opacity-0 group-hover:opacity-100 shrink-0 p-1 transition-opacity disabled:opacity-30"
      >
        <Trash2 className="h-4 w-4" />
      </button>
    </div>
  );
}

