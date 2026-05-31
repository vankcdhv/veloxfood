'use client';

import { useState } from 'react';
import { ShoppingCart, Plus } from 'lucide-react';
import { toast } from 'sonner';
import { Button } from '@/shared/ui/button';
import { getApiErrorMessage } from '@/shared/lib/api-error';
import { useCartMutations, useMyCart } from '../hooks/use-cart';
import type { MenuItem } from '@/features/stores/types/store';

interface AddToCartButtonProps {
  item: MenuItem;
  // Optional slot fields for pre-order stores; if omitted, item is added without slot.
  cutoffId?: string;
  date?: string;
  className?: string;
}

export function AddToCartButton({ item, cutoffId, date, className }: AddToCartButtonProps) {
  const { data: cart } = useMyCart();
  const { updateItem } = useCartMutations();
  const [busy, setBusy] = useState(false);

  // Current quantity already in cart for this item.
  const existingQty = cart?.Items?.find((it) => it.MenuItemID === item.ID)?.Qty ?? 0;
  // Warn if cart belongs to a different store.
  const differentStore = !!cart?.StoreID && cart.StoreID !== item.StoreID && cart.Items?.length;

  const handleAdd = async () => {
    if (item.Status === 'off') {
      toast.error('Món này hiện đã hết hàng.');
      return;
    }

    if (differentStore) {
      // Ask for confirmation to clear and start fresh.
      const confirmed = window.confirm(
        `Giỏ hàng hiện tại chứa món từ cửa hàng khác. Xác nhận để xoá giỏ hàng cũ và thêm món này?`,
      );
      if (!confirmed) return;
    }

    setBusy(true);
    try {
      await updateItem.mutateAsync({
        menuItemId: item.ID,
        body: {
          qty: existingQty + 1,
          cutoff_id: cutoffId,
          date,
        },
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
