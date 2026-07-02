import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useAuth } from '@/features/auth/context/auth-provider';
import { favoriteApi, type FavoriteType } from '../api/favorite-api';

export const favoriteKeys = {
  all: ['favorites'] as const,
  ids: (type: FavoriteType) => [...favoriteKeys.all, 'ids', type] as const,
  list: (type: FavoriteType) => [...favoriteKeys.all, 'list', type] as const,
};

// Bookmarked ids of one type as a Set — O(1) heart-state lookups. Disabled
// (empty set) when logged out.
export function useFavoriteIds(type: FavoriteType) {
  const { user } = useAuth();
  const query = useQuery({
    queryKey: favoriteKeys.ids(type),
    queryFn: () => favoriteApi.ids(type),
    enabled: !!user,
  });
  return { ...query, idSet: new Set(query.data ?? []) };
}

export function useFavoriteStores() {
  return useQuery({ queryKey: favoriteKeys.list('store'), queryFn: favoriteApi.stores });
}

export function useFavoriteItems() {
  return useQuery({ queryKey: favoriteKeys.list('item'), queryFn: favoriteApi.items });
}

// Toggle with optimistic ids update so the heart flips instantly.
export function useToggleFavorite(type: FavoriteType) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, next }: { id: string; next: boolean }) =>
      next ? favoriteApi.add(type, id) : favoriteApi.remove(type, id),
    onMutate: async ({ id, next }) => {
      await qc.cancelQueries({ queryKey: favoriteKeys.ids(type) });
      const prev = qc.getQueryData<string[]>(favoriteKeys.ids(type));
      qc.setQueryData<string[]>(favoriteKeys.ids(type), (old = []) =>
        next ? [id, ...old.filter((x) => x !== id)] : old.filter((x) => x !== id),
      );
      return { prev };
    },
    onError: (_err, _vars, ctx) => {
      if (ctx?.prev) qc.setQueryData(favoriteKeys.ids(type), ctx.prev);
    },
    onSettled: () => {
      void qc.invalidateQueries({ queryKey: favoriteKeys.all });
    },
  });
}
