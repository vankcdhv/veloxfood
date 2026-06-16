'use client';

import { useState } from 'react';
import { ImagePlus, Pencil, Plus, Sliders, Trash2, ToggleLeft, ToggleRight } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Badge } from '@/shared/ui/badge';
import { Skeleton } from '@/shared/ui/skeleton';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { formatVnd } from '@/shared/lib/format-vnd';
import { ConfirmDialog } from '@/shared/ui/confirm-dialog';
import { useStoreMenu, useVendorStoreMutations } from '../hooks/use-stores';
import { AddMenuItemDialog, EditMenuItemDialog, ImageUploadButton } from './vendor-menu-item-dialogs';
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
  const [optGroupTarget, setOptGroupTarget] = useState<MenuItem | null>(null);
  const [deleteTarget, setDeleteTarget] = useState<MenuItem | null>(null);

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

  const confirmDelete = () => {
    if (!deleteTarget) return;
    m.deleteMenuItem.mutate(deleteTarget.ID, {
      onSuccess: () => {
        toast.success('Đã xoá món.');
        setDeleteTarget(null);
      },
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
          <VendorCategoryCard
            key={cat.ID}
            storeId={storeId}
            cat={cat}
            items={items}
            onAddItem={() => setAddTarget(cat)}
            onEditItem={(item) => setEditTarget(item)}
            onDeleteItem={(item) => setDeleteTarget(item)}
            onOptionGroups={(item) => setOptGroupTarget(item)}
            toggling={m.setMenuItemStatus.isPending}
            deleting={m.deleteMenuItem.isPending}
            onToggleStatus={toggleStatus}
          />
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
      <VendorMenuItemOptionGroupsDialog
        key={optGroupTarget?.ID ?? 'og-none'}
        storeId={storeId}
        item={optGroupTarget}
        onClose={() => setOptGroupTarget(null)}
      />
      <ConfirmDialog
        open={!!deleteTarget}
        onOpenChange={(o) => !o && setDeleteTarget(null)}
        title="Xoá món này?"
        description={
          deleteTarget
            ? `Món "${deleteTarget.Name}" sẽ bị xoá khỏi thực đơn. Thao tác không thể hoàn tác.`
            : undefined
        }
        confirmLabel="Xoá món"
        destructive
        loading={m.deleteMenuItem.isPending}
        onConfirm={confirmDelete}
      />
    </>
  );
}

// ---------- Category card with per-category show-more cap ----------

const VENDOR_CAT_INITIAL = 6;

function VendorCategoryCard({
  storeId, cat, items, onAddItem, onEditItem, onDeleteItem, onOptionGroups,
  toggling, deleting, onToggleStatus,
}: {
  storeId: string;
  cat: Category;
  items: MenuItem[];
  onAddItem: () => void;
  onEditItem: (item: MenuItem) => void;
  onDeleteItem: (item: MenuItem) => void;
  onOptionGroups: (item: MenuItem) => void;
  toggling: boolean;
  deleting: boolean;
  onToggleStatus: (item: MenuItem) => void;
}) {
  const [visible, setVisible] = useState(VENDOR_CAT_INITIAL);
  const visibleItems = items.slice(0, visible);
  const remaining = items.length - visible;

  return (
    <Card>
      <CardHeader className="pb-2">
        <div className="flex items-center justify-between">
          <CardTitle className="text-sm font-semibold">{cat.Name}</CardTitle>
          <Button
            size="sm"
            variant="outline"
            onClick={onAddItem}
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
        {visibleItems.map((item) => (
          <MenuItemRow
            key={item.ID}
            storeId={storeId}
            item={item}
            onToggle={() => onToggleStatus(item)}
            onEdit={() => onEditItem(item)}
            onDelete={() => onDeleteItem(item)}
            onOptionGroups={() => onOptionGroups(item)}
            toggling={toggling}
            deleting={deleting}
          />
        ))}
        {remaining > 0 && (
          <div className="flex justify-center pt-1">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setVisible(items.length)}
            >
              Xem thêm ({remaining} món)
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  );
}

// ---------- Menu item row ----------

function MenuItemRow({
  storeId, item, onToggle, onEdit, onDelete, onOptionGroups, toggling, deleting,
}: {
  storeId: string;
  item: MenuItem;
  onToggle: () => void;
  onEdit: () => void;
  onDelete: () => void;
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
        <p className="text-primary text-xs font-semibold">{formatVnd(item.Price)}</p>
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

