import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useActiveStoreId } from '@/shared/lib/active-store';
import { cartApi } from '../api/cart-api';
import type { AddOrUpdateCartItemBody } from '../types/cart';

export const cartKeys = {
  all: ['cart'] as const,
  mine: (storeId: string | null) => [...cartKeys.all, 'mine', storeId] as const,
};

export function useMyCart() {
  const storeId = useActiveStoreId();
  return useQuery({
    queryKey: cartKeys.mine(storeId),
    queryFn: () => cartApi.get(storeId as string),
    // No active store → no cart to load (shows empty state, not an error).
    enabled: !!storeId,
    staleTime: 30_000,
  });
}

export function useCartMutations() {
  const qc = useQueryClient();
  const invalidate = () => qc.invalidateQueries({ queryKey: cartKeys.all });

  return {
    updateItem: useMutation({
      mutationFn: (body: AddOrUpdateCartItemBody) => cartApi.updateItem(body),
      onSuccess: invalidate,
    }),

    removeItem: useMutation({
      mutationFn: (v: { menuItemId: string; storeId: string }) =>
        cartApi.removeItem(v.menuItemId, v.storeId),
      onSuccess: invalidate,
    }),

    clear: useMutation({
      mutationFn: (storeId: string) => cartApi.clear(storeId),
      onSuccess: invalidate,
    }),
  };
}
