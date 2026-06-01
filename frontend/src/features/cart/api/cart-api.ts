import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse } from '@/shared/api/api-response';
import type { Cart, AddOrUpdateCartItemBody } from '../types/cart';

const CART = `${API_PREFIX}/me/cart`;

function unwrap<T>(res: { data: ApiResponse<T> }): T {
  if (res.data.data === undefined || res.data.data === null) {
    throw new Error(res.data.error ?? 'Empty response');
  }
  return res.data.data;
}

// Cart is per-store: every endpoint needs store_id (query param on GET/DELETE,
// inside the body on the upsert PUT). The active store is tracked client-side
// (see shared/lib/active-store).
export const cartApi = {
  get: async (storeId: string) =>
    unwrap(await http.get<ApiResponse<Cart>>(CART, { params: { store_id: storeId } })),

  updateItem: async (body: AddOrUpdateCartItemBody) =>
    unwrap(await http.put<ApiResponse<Cart>>(`${CART}/items`, body)),

  removeItem: async (menuItemId: string, storeId: string) => {
    await http.delete<ApiResponse>(`${CART}/items/${menuItemId}`, { params: { store_id: storeId } });
  },

  clear: async (storeId: string) => {
    await http.delete<ApiResponse>(CART, { params: { store_id: storeId } });
  },
};
