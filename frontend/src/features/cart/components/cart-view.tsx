'use client';

import { useMemo, useState } from 'react';
import Link from 'next/link';
import { Minus, Plus, Trash2, ShoppingBag, ArrowRight, Store as StoreIcon } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Card, CardContent } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { RoleGuard } from '@/features/auth/components/role-guard';
import { StoreOpenBadge } from '@/features/stores/components/sale-status-badge';
import { formatVnd } from '@/shared/lib/format-vnd';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { ROUTES } from '@/shared/config/constants';
import { useStores } from '@/features/stores/hooks/use-stores';
import type { SaleStatus } from '@/features/stores/types/store';
import { useMyCarts, useCartMutations } from '../hooks/use-cart';
import type { Cart, CartItem } from '../types/cart';

export function CartView() {
  return (
    <RoleGuard allow={(a) => a.isAuthenticated}>
      <main className="bg-background min-h-dvh px-4 py-10">
        <div className="mx-auto w-full max-w-2xl space-y-6">
          <h1 className="font-serif text-2xl font-bold">Giỏ hàng</h1>
          <CartContent />
        </div>
      </main>
    </RoleGuard>
  );
}

interface StoreMeta {
  name: string;
  openNow: boolean;
  status: SaleStatus;
}

function CartContent() {
  const { data: carts, isLoading, isError } = useMyCarts();
  const { data: stores } = useStores();
  const { removeItem } = useCartMutations();

  // Store name + open-now lookup, reused from the public browse list.
  const storeMeta = useMemo(() => {
    const m = new Map<string, StoreMeta>();
    (stores ?? []).forEach((s) => m.set(s.ID, { name: s.Name, openNow: !!s.OpenNow, status: s.SaleStatus }));
    return m;
  }, [stores]);

  // Carts that actually have items, grouped per store.
  const groups = useMemo(() => (carts ?? []).filter((c) => (c.Items?.length ?? 0) > 0), [carts]);

  // Checkout selection is locked to a single store: picking an item from store A
  // disables every other store until the selection is cleared.
  const [selectedStoreId, setSelectedStoreId] = useState<string | null>(null);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());

  const toggleItem = (storeId: string, menuItemId: string) => {
    if (selectedStoreId && selectedStoreId !== storeId) return; // other store locked
    const next = new Set(selectedStoreId === storeId ? selectedIds : []);
    if (next.has(menuItemId)) next.delete(menuItemId);
    else next.add(menuItemId);
    setSelectedIds(next);
    setSelectedStoreId(next.size === 0 ? null : storeId);
  };

  const handleRemove = (storeId: string, menuItemId: string) => {
    removeItem.mutate(
      { menuItemId, storeId },
      { onError: (e) => toast.error(getApiErrorMessage(e, 'Không xoá được món')) },
    );
    if (selectedIds.has(menuItemId)) {
      const next = new Set(selectedIds);
      next.delete(menuItemId);
      setSelectedIds(next);
      if (next.size === 0) setSelectedStoreId(null);
    }
  };

  if (isLoading) {
    return (
      <div className="space-y-3">
        {Array.from({ length: 3 }).map((_, i) => (
          <Skeleton key={i} className="h-20 rounded-xl" />
        ))}
      </div>
    );
  }

  if (isError) {
    return <p className="text-destructive text-sm text-center py-12">Không tải được giỏ hàng.</p>;
  }

  if (groups.length === 0) {
    return (
      <div className="flex flex-col items-center gap-4 py-20 text-muted-foreground">
        <ShoppingBag className="h-12 w-12 opacity-30" />
        <p className="text-sm">Giỏ hàng trống. Hãy thêm món từ cửa hàng!</p>
        <Button asChild variant="outline">
          <Link href={ROUTES.stores.root}>Xem cửa hàng</Link>
        </Button>
      </div>
    );
  }

  // Selected store summary (drives the checkout bar).
  const selectedGroup = groups.find((g) => g.StoreID === selectedStoreId);
  const selectedItems = selectedGroup
    ? selectedGroup.Items.filter((it) => selectedIds.has(it.MenuItemID))
    : [];
  const selectedSubtotal = selectedItems.reduce((s, it) => s + it.PriceSnapshot * it.Qty, 0);
  const selectedStoreName = selectedStoreId ? storeMeta.get(selectedStoreId)?.name : undefined;
  const checkoutHref =
    selectedGroup && selectedItems.length > 0
      ? `${ROUTES.checkout}?store_id=${selectedGroup.StoreID}&items=${selectedItems
          .map((it) => it.MenuItemID)
          .join(',')}`
      : '';

  return (
    <div className="space-y-4">
      <p className="text-muted-foreground text-sm">
        Tích chọn các món của <span className="font-medium text-foreground">một cửa hàng</span> để thanh
        toán. Mỗi đơn hàng thuộc về một cửa hàng.
      </p>

      {groups.map((group) => (
        <StoreCartGroup
          key={group.StoreID}
          group={group}
          meta={storeMeta.get(group.StoreID)}
          locked={selectedStoreId !== null && selectedStoreId !== group.StoreID}
          selectedIds={selectedIds}
          onToggle={(menuItemId) => toggleItem(group.StoreID, menuItemId)}
          onRemove={(menuItemId) => handleRemove(group.StoreID, menuItemId)}
        />
      ))}

      {/* Sticky checkout bar — only when a store has items selected */}
      {checkoutHref && (
        <div className="bg-background/90 sticky bottom-0 -mx-4 border-t border-border px-4 py-3 backdrop-blur-md">
          <div className="mx-auto flex max-w-2xl items-center justify-between gap-3">
            <div className="min-w-0">
              <p className="truncate text-sm font-medium">{selectedStoreName}</p>
              <p className="text-muted-foreground text-xs">
                {selectedItems.length} món · {formatVnd(selectedSubtotal)}
              </p>
            </div>
            <Button asChild>
              <Link href={checkoutHref}>
                Thanh toán
                <ArrowRight className="ml-1.5 h-4 w-4" />
              </Link>
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

const CART_GROUP_INITIAL = 8;

function StoreCartGroup({
  group,
  meta,
  locked,
  selectedIds,
  onToggle,
  onRemove,
}: {
  group: Cart;
  meta?: StoreMeta;
  locked: boolean;
  selectedIds: Set<string>;
  onToggle: (menuItemId: string) => void;
  onRemove: (menuItemId: string) => void;
}) {
  const [visible, setVisible] = useState(CART_GROUP_INITIAL);
  const subtotal = group.Items.reduce((s, it) => s + it.PriceSnapshot * it.Qty, 0);
  const visibleItems = group.Items.slice(0, visible);
  const remaining = group.Items.length - visible;

  return (
    <Card className={locked ? 'opacity-50' : ''}>
      <CardContent className="p-0">
        {/* Store header */}
        <div className="flex items-center gap-2 border-b border-border px-4 py-3">
          <StoreIcon className="text-muted-foreground h-4 w-4 shrink-0" />
          <Link
            href={ROUTES.stores.detail(group.StoreID)}
            className="truncate text-sm font-semibold hover:underline"
          >
            {meta?.name ?? 'Cửa hàng'}
          </Link>
          {meta && <StoreOpenBadge openNow={meta.openNow} status={meta.status} />}
        </div>

        <div className="divide-y divide-border">
          {visibleItems.map((item) => (
            <CartItemRow
              key={item.MenuItemID}
              item={item}
              storeId={group.StoreID}
              checked={selectedIds.has(item.MenuItemID)}
              disabled={locked}
              onToggle={() => onToggle(item.MenuItemID)}
              onRemove={() => onRemove(item.MenuItemID)}
            />
          ))}
        </div>

        {remaining > 0 && (
          <div className="flex justify-center px-4 py-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setVisible((v) => v + remaining)}
            >
              Xem thêm ({remaining})
            </Button>
          </div>
        )}

        <div className="flex items-center justify-between px-4 py-2.5 text-sm">
          <span className="text-muted-foreground">Tạm tính cửa hàng</span>
          <span className="text-primary font-semibold">{formatVnd(subtotal)}</span>
        </div>
      </CardContent>
    </Card>
  );
}

function CartItemRow({
  item,
  storeId,
  checked,
  disabled,
  onToggle,
  onRemove,
}: {
  item: CartItem;
  storeId: string;
  checked: boolean;
  disabled: boolean;
  onToggle: () => void;
  onRemove: () => void;
}) {
  const { updateItem } = useCartMutations();
  const [qty, setQty] = useState(item.Qty);

  const changeQty = async (next: number) => {
    if (next < 1) {
      onRemove();
      return;
    }
    setQty(next);
    try {
      await updateItem.mutateAsync({
        store_id: storeId,
        menu_item_id: item.MenuItemID,
        name_snapshot: item.NameSnapshot,
        price_snapshot: item.PriceSnapshot,
        qty: next,
      });
    } catch (e) {
      setQty(item.Qty);
      toast.error(getApiErrorMessage(e, 'Không cập nhật được số lượng'));
    }
  };

  return (
    <div className="flex items-center gap-3 px-4 py-3">
      <input
        type="checkbox"
        checked={checked}
        disabled={disabled}
        onChange={onToggle}
        aria-label={`Chọn ${item.NameSnapshot}`}
        className="accent-primary h-4 w-4 shrink-0 cursor-pointer disabled:cursor-not-allowed"
      />

      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium leading-snug">{item.NameSnapshot}</p>
        <p className="text-muted-foreground mt-0.5 text-xs">{formatVnd(item.PriceSnapshot)} / món</p>
      </div>

      <div className="flex shrink-0 items-center gap-1.5">
        <Button
          variant="outline"
          size="icon"
          className="h-7 w-7"
          onClick={() => changeQty(qty - 1)}
          disabled={updateItem.isPending}
          aria-label="Giảm số lượng"
        >
          <Minus className="h-3 w-3" />
        </Button>
        <span className="w-6 text-center text-sm font-medium">{qty}</span>
        <Button
          variant="outline"
          size="icon"
          className="h-7 w-7"
          onClick={() => changeQty(qty + 1)}
          disabled={updateItem.isPending}
          aria-label="Tăng số lượng"
        >
          <Plus className="h-3 w-3" />
        </Button>
      </div>

      <p className="text-primary w-20 shrink-0 text-right text-sm font-semibold">
        {formatVnd(item.PriceSnapshot * qty)}
      </p>

      <Button
        variant="ghost"
        size="icon"
        className="text-muted-foreground hover:text-destructive h-7 w-7 shrink-0"
        onClick={onRemove}
        aria-label="Xoá món"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  );
}
