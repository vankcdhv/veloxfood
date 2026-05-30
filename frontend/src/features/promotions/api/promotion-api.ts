import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse } from '@/shared/api/api-response';
import type {
  CreatePromotionBody,
  Promotion,
  UpdatePromotionBody,
  ValidatePromotionResult,
} from '../types/promotion';

const STORES = `${API_PREFIX}/stores`;
const PROMOTIONS = `${API_PREFIX}/promotions`;

function unwrap<T>(res: { data: ApiResponse<T> }): T {
  if (res.data.data === undefined || res.data.data === null) {
    throw new Error(res.data.error ?? 'Empty response');
  }
  return res.data.data;
}

export const vendorPromotionApi = {
  list: async (storeId: string) =>
    unwrap(await http.get<ApiResponse<Promotion[]>>(`${STORES}/${storeId}/promotions`)),

  create: async (storeId: string, body: CreatePromotionBody) =>
    unwrap(await http.post<ApiResponse<Promotion>>(`${STORES}/${storeId}/promotions`, body)),

  update: async (storeId: string, promoId: string, body: UpdatePromotionBody) =>
    unwrap(await http.patch<ApiResponse<Promotion>>(`${STORES}/${storeId}/promotions/${promoId}`, body)),

  delete: async (storeId: string, promoId: string) => {
    await http.delete<ApiResponse>(`${STORES}/${storeId}/promotions/${promoId}`);
  },
};

export const publicPromotionApi = {
  // Dry-run preview — does NOT reserve usage.
  validate: async (body: { code: string; store_id: string; subtotal: number; item_count: number }) =>
    unwrap(await http.post<ApiResponse<ValidatePromotionResult>>(`${PROMOTIONS}/validate`, body)),
};
