import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { cartApi } from '../api/cart-api';
import type { UpdateCartItemBody } from '../types/cart';

export const cartKeys = {
  all: ['cart'] as const,
  mine: () => [...cartKeys.all, 'mine'] as const,
};

export function useMyCart() {
  return useQuery({
    queryKey: cartKeys.mine(),
    queryFn: cartApi.get,
    // Cart is always user-specific; no need to share across tabs aggressively.
    staleTime: 30_000,
  });
}

export function useCartMutations() {
  const qc = useQueryClient();
  const invalidate = () => qc.invalidateQueries({ queryKey: cartKeys.mine() });

  return {
    updateItem: useMutation({
      mutationFn: (v: { menuItemId: string; body: UpdateCartItemBody }) =>
        cartApi.updateItem(v.menuItemId, v.body),
      onSuccess: invalidate,
    }),

    removeItem: useMutation({
      mutationFn: (menuItemId: string) => cartApi.removeItem(menuItemId),
      onSuccess: invalidate,
    }),

    clear: useMutation({
      mutationFn: cartApi.clear,
      onSuccess: invalidate,
    }),
  };
}
