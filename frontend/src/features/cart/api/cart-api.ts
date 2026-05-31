import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse } from '@/shared/api/api-response';
import type { Cart, UpdateCartItemBody } from '../types/cart';

const CART = `${API_PREFIX}/me/cart`;

function unwrap<T>(res: { data: ApiResponse<T> }): T {
  if (res.data.data === undefined || res.data.data === null) {
    throw new Error(res.data.error ?? 'Empty response');
  }
  return res.data.data;
}

export const cartApi = {
  get: async () =>
    unwrap(await http.get<ApiResponse<Cart>>(CART)),

  updateItem: async (menuItemId: string, body: UpdateCartItemBody) =>
    unwrap(await http.put<ApiResponse<Cart>>(`${CART}/items/${menuItemId}`, body)),

  removeItem: async (menuItemId: string) => {
    await http.delete<ApiResponse>(`${CART}/items/${menuItemId}`);
  },

  clear: async () => {
    await http.delete<ApiResponse>(CART);
  },
};
