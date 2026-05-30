'use client';

import { useState } from 'react';
import { ChevronDown, ChevronRight, Loader2, Pencil, Plus, Trash2 } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Input } from '@/shared/ui/input';
import { Card, CardContent, CardHeader, CardTitle } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import {
  Dialog,
  DialogContent,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/shared/ui/dialog';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import {
  useCombos,
  useComboItems,
  useComboMutations,
} from '../hooks/use-stores';
import { useStoreMenu } from '../hooks/use-stores';
import type { Combo, MenuItem } from '../types/store';

interface Props {
  storeId: string;
}

export function VendorComboPanel({ storeId }: Props) {
  const { data: combos, isLoading, isError } = useCombos(storeId);
  const { data: menuGroups } = useStoreMenu(storeId);
  const m = useComboMutations(storeId);
  const [expanded, setExpanded] = useState<string | null>(null);
  const [editCombo, setEditCombo] = useState<Combo | null>(null);
  const [showCreate, setShowCreate] = useState(false);

  // Flat list of all menu items for name resolution and picker.
  const allItems: MenuItem[] = menuGroups?.flatMap((g) => g.Items) ?? [];

  const deleteCombo = (comboId: string) => {
    m.deleteCombo.mutate(comboId, {
      onSuccess: () => toast.success('Đã xoá combo.'),
      onError: (e) => toast.error(getApiErrorMessage(e, 'Xoá thất bại')),
    });
  };

  return (
    <>
      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-sm font-semibold">Combo</h2>
          <Button size="sm" onClick={() => setShowCreate(true)} className="gap-1 h-8 text-xs">
            <Plus className="h-3.5 w-3.5" />
            Thêm combo
          </Button>
        </div>

        {isLoading && (
          <div className="space-y-2">
            {[1, 2].map((i) => <Skeleton key={i} className="h-14 rounded-xl" />)}
          </div>
        )}

        {isError && (
          <p className="text-destructive text-sm">Không tải được danh sách combo.</p>
        )}

        {!isLoading && !isError && (!combos || combos.length === 0) && (
          <p className="text-muted-foreground text-sm">Chưa có combo nào.</p>
        )}

        {combos?.map((combo) => (
          <Card key={combo.ID}>
            <CardHeader
              className="pb-2 cursor-pointer select-none"
              onClick={() => setExpanded(expanded === combo.ID ? null : combo.ID)}
            >
              <div className="flex items-center gap-2">
                {expanded === combo.ID
                  ? <ChevronDown className="h-4 w-4 shrink-0 text-muted-foreground" />
                  : <ChevronRight className="h-4 w-4 shrink-0 text-muted-foreground" />}
                <CardTitle className="text-sm font-semibold flex-1">{combo.Name}</CardTitle>
                <span className="text-primary text-sm font-semibold shrink-0">
                  {combo.Price.toLocaleString('vi-VN')}đ
                </span>
                <div className="flex items-center gap-1 shrink-0" onClick={(e) => e.stopPropagation()}>
                  <button
                    type="button"
                    aria-label="Chỉnh sửa combo"
                    onClick={() => setEditCombo(combo)}
                    className="text-muted-foreground hover:text-primary p-1"
                  >
                    <Pencil className="h-3.5 w-3.5" />
                  </button>
                  <button
                    type="button"
                    aria-label="Xoá combo"
                    onClick={() => deleteCombo(combo.ID)}
                    disabled={m.deleteCombo.isPending}
                    className="text-muted-foreground hover:text-destructive p-1 disabled:opacity-30"
                  >
                    <Trash2 className="h-3.5 w-3.5" />
                  </button>
                </div>
              </div>
            </CardHeader>

            {expanded === combo.ID && (
              <CardContent className="pt-0">
                <ComboItemList
                  storeId={storeId}
                  combo={combo}
                  allItems={allItems}
                  mutations={m}
                />
              </CardContent>
            )}
          </Card>
        ))}
      </div>

      <ComboDialog
        open={showCreate}
        onClose={() => setShowCreate(false)}
        mutations={m}
      />

      <ComboDialog
        key={editCombo?.ID ?? 'none'}
        open={!!editCombo}
        combo={editCombo ?? undefined}
        onClose={() => setEditCombo(null)}
        mutations={m}
      />
    </>
  );
}

// ---------- Combo items list ----------

function ComboItemList({
  storeId, combo, allItems, mutations,
}: {
  storeId: string;
  combo: Combo;
  allItems: MenuItem[];
  mutations: ReturnType<typeof useComboMutations>;
}) {
  const { data: comboItems, isLoading } = useComboItems(storeId, combo.ID);
  const [selectedItemId, setSelectedItemId] = useState('');
  const [quantity, setQuantity] = useState('1');

  // Items not already in the combo.
  const attachedIds = new Set(comboItems?.map((ci) => ci.MenuItemID) ?? []);
  const availableItems = allItems.filter((it) => !attachedIds.has(it.ID));

  const addItem = () => {
    const qty = parseInt(quantity, 10);
    if (!selectedItemId || isNaN(qty) || qty < 1) {
      toast.error('Vui lòng chọn món và nhập số lượng hợp lệ.');
      return;
    }
    mutations.addComboItem.mutate(
      { comboId: combo.ID, menu_item_id: selectedItemId, quantity: qty },
      {
        onSuccess: () => {
          toast.success('Đã thêm món vào combo.');
          setSelectedItemId('');
          setQuantity('1');
        },
        onError: (e) => toast.error(getApiErrorMessage(e, 'Thêm thất bại')),
      },
    );
  };

  const removeItem = (menuItemId: string) => {
    mutations.removeComboItem.mutate(
      { comboId: combo.ID, menuItemId },
      {
        onSuccess: () => toast.success('Đã xoá món khỏi combo.'),
        onError: (e) => toast.error(getApiErrorMessage(e, 'Xoá thất bại')),
      },
    );
  };

  const resolveName = (menuItemId: string) =>
    allItems.find((it) => it.ID === menuItemId)?.Name ?? '—';

  return (
    <div className="space-y-2 border-t border-border pt-3">
      {isLoading && <Skeleton className="h-10" />}

      {!isLoading && (!comboItems || comboItems.length === 0) && (
        <p className="text-muted-foreground text-xs">Chưa có món nào trong combo.</p>
      )}

      <ul className="space-y-1">
        {comboItems?.map((ci) => (
          <li
            key={ci.ID}
            className="group flex items-center gap-3 rounded-md px-2 py-1.5 hover:bg-accent/40 transition-colors"
          >
            <span className="flex-1 text-sm truncate">{resolveName(ci.MenuItemID)}</span>
            <span className="text-xs text-muted-foreground shrink-0">x{ci.Quantity}</span>
            <button
              type="button"
              aria-label="Xoá món khỏi combo"
              onClick={() => removeItem(ci.MenuItemID)}
              disabled={mutations.removeComboItem.isPending}
              className="text-muted-foreground hover:text-destructive opacity-0 group-hover:opacity-100 p-1 transition-opacity disabled:opacity-30"
            >
              <Trash2 className="h-3.5 w-3.5" />
            </button>
          </li>
        ))}
      </ul>

      {/* Add item row */}
      <div className="flex gap-2 items-center pt-1">
        <select
          value={selectedItemId}
          onChange={(e) => setSelectedItemId(e.target.value)}
          className="border-input bg-background flex-1 h-9 rounded-md border px-2 text-sm min-w-0"
          disabled={availableItems.length === 0}
        >
          <option value="">
            {availableItems.length === 0 ? 'Không còn món để thêm' : 'Chọn món…'}
          </option>
          {availableItems.map((it) => (
            <option key={it.ID} value={it.ID}>{it.Name}</option>
          ))}
        </select>
        <Input
          value={quantity}
          type="number"
          min={1}
          onChange={(e) => setQuantity(e.target.value)}
          className="w-16 shrink-0"
          placeholder="SL"
        />
        <Button
          size="icon"
          onClick={addItem}
          disabled={mutations.addComboItem.isPending || !selectedItemId}
          aria-label="Thêm món vào combo"
          className="shrink-0"
        >
          {mutations.addComboItem.isPending
            ? <Loader2 className="h-4 w-4 animate-spin" />
            : <Plus className="h-4 w-4" />}
        </Button>
      </div>
    </div>
  );
}

// ---------- Create / Edit combo dialog ----------

function ComboDialog({
  open, combo, onClose, mutations,
}: {
  open: boolean;
  combo?: Combo;
  onClose: () => void;
  mutations: ReturnType<typeof useComboMutations>;
}) {
  const isEdit = !!combo;
  const [name, setName] = useState(combo?.Name ?? '');
  const [price, setPrice] = useState(String(combo?.Price ?? ''));

  const isPending = mutations.createCombo.isPending || mutations.updateCombo.isPending;

  const submit = () => {
    const p = parseInt(price, 10);
    if (!name.trim() || isNaN(p) || p < 0) {
      toast.error('Tên và giá combo không hợp lệ.');
      return;
    }

    if (isEdit && combo) {
      mutations.updateCombo.mutate(
        { comboId: combo.ID, body: { name: name.trim(), price: p } },
        {
          onSuccess: () => { toast.success('Đã cập nhật combo.'); onClose(); },
          onError: (e) => toast.error(getApiErrorMessage(e, 'Cập nhật thất bại')),
        },
      );
    } else {
      mutations.createCombo.mutate(
        { name: name.trim(), price: p },
        {
          onSuccess: () => { toast.success('Đã thêm combo.'); onClose(); },
          onError: (e) => toast.error(getApiErrorMessage(e, 'Thêm thất bại')),
        },
      );
    }
  };

  return (
    <Dialog open={open} onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{isEdit ? 'Chỉnh sửa combo' : 'Thêm combo'}</DialogTitle>
        </DialogHeader>
        <div className="space-y-3">
          <div>
            <label className="text-foreground mb-1.5 block text-sm font-medium">Tên combo</label>
            <Input value={name} onChange={(e) => setName(e.target.value)} placeholder="VD: Combo cơm + nước" />
          </div>
          <div>
            <label className="text-foreground mb-1.5 block text-sm font-medium">Giá combo (VNĐ)</label>
            <Input value={price} type="number" min={0} onChange={(e) => setPrice(e.target.value)} placeholder="VD: 55000" />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose} disabled={isPending}>Huỷ</Button>
          <Button onClick={submit} disabled={isPending}>
            {isPending ? <Loader2 className="h-4 w-4 animate-spin" /> : (isEdit ? 'Lưu' : 'Thêm combo')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
