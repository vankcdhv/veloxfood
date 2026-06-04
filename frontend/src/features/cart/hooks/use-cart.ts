import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { cartApi } from '../api/cart-api';
import type { AddOrUpdateCartItemBody } from '../types/cart';

export const cartKeys = {
  all: ['cart'] as const,
  mine: (storeId: string | null) => [...cartKeys.all, 'mine', storeId] as const,
  mineAll: () => [...cartKeys.all, 'mine-all'] as const,
};

// All of the customer's carts across every store (account-bound, multi-store).
export function useMyCarts() {
  return useQuery({
    queryKey: cartKeys.mineAll(),
    queryFn: () => cartApi.getAll(),
    staleTime: 30_000,
  });
}

// One store's cart. Used where a single store is in scope (add-to-cart, checkout).
export function useStoreCart(storeId: string) {
  return useQuery({
    queryKey: cartKeys.mine(storeId),
    queryFn: () => cartApi.get(storeId),
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
