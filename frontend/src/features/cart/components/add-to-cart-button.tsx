'use client';

import { useState } from 'react';
import { ShoppingCart, Plus } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useCartMutations, useStoreCart } from '../hooks/use-cart';
import type { MenuItem } from '@/features/stores/types/store';

interface AddToCartButtonProps {
  item: MenuItem;
  className?: string;
}

export function AddToCartButton({ item, className }: AddToCartButtonProps) {
  // Cart is multi-store: items from different stores coexist, so we just read
  // this item's own store cart for the current quantity.
  const { data: cart } = useStoreCart(item.StoreID);
  const { updateItem } = useCartMutations();
  const [busy, setBusy] = useState(false);

  // Current quantity already in cart for this item.
  const existingQty = cart?.Items?.find((it) => it.MenuItemID === item.ID)?.Qty ?? 0;

  const handleAdd = async () => {
    if (item.Status === 'off') {
      toast.error('Món này hiện đã hết hàng.');
      return;
    }

    setBusy(true);
    try {
      await updateItem.mutateAsync({
        store_id: item.StoreID,
        menu_item_id: item.ID,
        name_snapshot: item.Name,
        price_snapshot: item.Price,
        qty: existingQty + 1,
      });
      toast.success(`Đã thêm "${item.Name}" vào giỏ hàng`);
    } catch (e) {
      toast.error(getApiErrorMessage(e, 'Không thêm được vào giỏ hàng'));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Button
      size="sm"
      variant={existingQty > 0 ? 'default' : 'outline'}
      className={className}
      onClick={handleAdd}
      disabled={busy || item.Status === 'off'}
      aria-label={`Thêm ${item.Name} vào giỏ hàng`}
    >
      {existingQty > 0 ? (
        <>
          <ShoppingCart className="h-3.5 w-3.5 mr-1" />
          {existingQty} trong giỏ
        </>
      ) : (
        <>
          <Plus className="h-3.5 w-3.5 mr-1" />
          Thêm
        </>
      )}
    </Button>
  );
}
