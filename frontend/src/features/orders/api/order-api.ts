import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse, PaginatedResponse } from '@/shared/api/api-response';
import type {
  AdvanceOrderStatusBody,
  Order,
  PickupVerifyBody,
  PlaceOrderBody,
  PlaceOrderResult,
} from '../types/order';

const ORDERS = `${API_PREFIX}/orders`;

function unwrap<T>(res: { data: ApiResponse<T> }): T {
  if (res.data.data === undefined || res.data.data === null) {
    throw new Error(res.data.error ?? 'Empty response');
  }
  return res.data.data;
}

// ---- Customer ----
export const customerOrderApi = {
  list: async (page = 1, pageSize = 20) =>
    unwrap(
      await http.get<PaginatedResponse<Order>>(ORDERS, {
        params: { page, page_size: pageSize },
      }),
    ),

  get: async (id: string) =>
    unwrap(await http.get<ApiResponse<Order>>(`${ORDERS}/${id}`)),

  place: async (body: PlaceOrderBody) =>
    unwrap(await http.post<ApiResponse<PlaceOrderResult>>(ORDERS, body)),

  cancel: async (id: string) => {
    await http.post<ApiResponse>(`${ORDERS}/${id}/cancel`, {});
  },

  reorder: async (id: string) =>
    unwrap(await http.post<ApiResponse<PlaceOrderResult>>(`${ORDERS}/${id}/reorder`, {})),
};

// ---- Owner / staff ----
export const ownerOrderApi = {
  list: async (storeId: string, page = 1, pageSize = 30) =>
    unwrap(
      await http.get<PaginatedResponse<Order>>(
        `${API_PREFIX}/stores/${storeId}/orders`,
        { params: { page, page_size: pageSize } },
      ),
    ),

  get: async (storeId: string, orderId: string) =>
    unwrap(
      await http.get<ApiResponse<Order>>(
        `${API_PREFIX}/stores/${storeId}/orders/${orderId}`,
      ),
    ),

  confirm: async (storeId: string, orderId: string) => {
    await http.post<ApiResponse>(
      `${API_PREFIX}/stores/${storeId}/orders/${orderId}/confirm`,
      {},
    );
  },

  reject: async (storeId: string, orderId: string) => {
    await http.post<ApiResponse>(
      `${API_PREFIX}/stores/${storeId}/orders/${orderId}/reject`,
      {},
    );
  },

  advanceStatus: async (storeId: string, orderId: string, body: AdvanceOrderStatusBody) => {
    await http.patch<ApiResponse>(
      `${API_PREFIX}/stores/${storeId}/orders/${orderId}/status`,
      body,
    );
  },

  verifyPickup: async (storeId: string, orderId: string, body: PickupVerifyBody) => {
    await http.post<ApiResponse>(
      `${API_PREFIX}/stores/${storeId}/orders/${orderId}/pickup-verify`,
      body,
    );
  },
};
