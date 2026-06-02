'use client';

import { useState } from 'react';
import Link from 'next/link';
import { Minus, Plus, Trash2, ShoppingBag, ArrowRight } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { Card, CardContent } from '@/shared/ui/card';
import { Skeleton } from '@/shared/ui/skeleton';
import { RoleGuard } from '@/features/auth/components/role-guard';
import { formatVnd } from '@/shared/lib/format-vnd';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { ROUTES } from '@/shared/config/constants';
import { useMyCart, useCartMutations } from '../hooks/use-cart';
import type { CartItem } from '../types/cart';

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

function CartContent() {
  const { data: cart, isLoading, isError } = useMyCart();
  const { removeItem, clear } = useCartMutations();
  const storeId = cart?.StoreID ?? '';

  const handleClear = async () => {
    if (!storeId) return;
    try {
      await clear.mutateAsync(storeId);
      toast.success('Đã xoá giỏ hàng');
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'Không xoá được giỏ hàng'));
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

  const items = cart?.Items ?? [];

  if (items.length === 0) {
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

  const subtotal = items.reduce((s, it) => s + it.PriceSnapshot * it.Qty, 0);

  return (
    <div className="space-y-4">
      {cart?.StoreName && (
        <p className="text-sm text-muted-foreground">
          Cửa hàng: <span className="font-medium text-foreground">{cart.StoreName}</span>
        </p>
      )}

      <Card>
        <CardContent className="divide-y divide-border p-0">
          {items.map((item) => (
            <CartItemRow
              key={item.MenuItemID}
              item={item}
              storeId={storeId}
              onRemove={() => {
                removeItem.mutate(
                  { menuItemId: item.MenuItemID, storeId },
                  { onError: (e) => toast.error(getApiErrorMessage(e, 'Không xoá được món')) },
                );
              }}
            />
          ))}
        </CardContent>
      </Card>

      <Card>
        <CardContent className="flex items-center justify-between py-4">
          <span className="font-medium">Tạm tính</span>
          <span className="font-semibold text-primary">{formatVnd(subtotal)}</span>
        </CardContent>
      </Card>

      <div className="flex gap-3">
        <Button
          variant="outline"
          size="sm"
          onClick={handleClear}
          disabled={clear.isPending}
          className="text-destructive border-destructive/50 hover:bg-destructive/5"
        >
          <Trash2 className="h-4 w-4 mr-1.5" />
          Xoá giỏ
        </Button>
        <Button asChild className="flex-1">
          <Link href={ROUTES.checkout}>
            Thanh toán
            <ArrowRight className="h-4 w-4 ml-1.5" />
          </Link>
        </Button>
      </div>
    </div>
  );
}

function CartItemRow({ item, storeId, onRemove }: { item: CartItem; storeId: string; onRemove: () => void }) {
  const { updateItem } = useCartMutations();
  const [qty, setQty] = useState(item.Qty);

  const changeQty = async (next: number) => {
    if (next < 1) { onRemove(); return; }
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
      <div className="min-w-0 flex-1">
        <p className="text-sm font-medium leading-snug">{item.NameSnapshot}</p>
        <p className="text-xs text-muted-foreground mt-0.5">{formatVnd(item.PriceSnapshot)} / món</p>
      </div>

      <div className="flex items-center gap-1.5 shrink-0">
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

      <p className="w-20 text-right text-sm font-semibold text-primary shrink-0">
        {formatVnd(item.PriceSnapshot * qty)}
      </p>

      <Button
        variant="ghost"
        size="icon"
        className="h-7 w-7 text-muted-foreground hover:text-destructive shrink-0"
        onClick={onRemove}
        aria-label="Xoá món"
      >
        <Trash2 className="h-3.5 w-3.5" />
      </Button>
    </div>
  );
}
