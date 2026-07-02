import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse } from '@/shared/api/api-response';
import type { MenuItemSearchResult, Store } from '@/features/stores/types/store';

const FAVORITES = `${API_PREFIX}/me/favorites`;

export type FavoriteType = 'store' | 'item';

function unwrap<T>(res: { data: ApiResponse<T> }): T {
  if (res.data.data === undefined || res.data.data === null) {
    throw new Error(res.data.error ?? 'Empty response');
  }
  return res.data.data;
}

export const favoriteApi = {
  // Bookmarked target ids of one type — cheap heart-state hydration.
  ids: async (type: FavoriteType): Promise<string[]> =>
    unwrap(await http.get<ApiResponse<string[]>>(`${FAVORITES}/ids`, { params: { type } })),

  // Bookmarked stores (enriched with OpenNow), most recent first.
  stores: async (): Promise<Store[]> =>
    unwrap(await http.get<ApiResponse<Store[]>>(FAVORITES, { params: { type: 'store' } })),

  // Bookmarked menu items (same row shape as global search), most recent first.
  items: async (): Promise<MenuItemSearchResult[]> =>
    unwrap(
      await http.get<ApiResponse<MenuItemSearchResult[]>>(FAVORITES, { params: { type: 'item' } }),
    ),

  add: async (type: FavoriteType, id: string): Promise<void> => {
    await http.put(`${FAVORITES}/${type}/${id}`);
  },

  remove: async (type: FavoriteType, id: string): Promise<void> => {
    await http.delete(`${FAVORITES}/${type}/${id}`);
  },
};
