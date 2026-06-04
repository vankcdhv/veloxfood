import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse } from '@/shared/api/api-response';
import type { Cart, AddOrUpdateCartItemBody } from '../types/cart';

const CART = `${API_PREFIX}/me/cart`;
const CARTS = `${API_PREFIX}/me/carts`;

function unwrap<T>(res: { data: ApiResponse<T> }): T {
  if (res.data.data === undefined || res.data.data === null) {
    throw new Error(res.data.error ?? 'Empty response');
  }
  return res.data.data;
}

// The cart is account-bound server-side, one row per (customer, store). A
// customer may hold items from several stores at once: getAll returns every
// store's cart; the per-store endpoints address one store via store_id.
export const cartApi = {
  // All of the customer's carts (one per store), grouped per store.
  getAll: async (): Promise<Cart[]> =>
    unwrap(await http.get<ApiResponse<{ carts: Cart[] }>>(CARTS)).carts ?? [],

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
