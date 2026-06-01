import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse } from '@/shared/api/api-response';
import type { CreatePayoutBody, PayoutBatch, SettlementPreview } from '../types/payout';

const ADMIN_SETTLEMENTS = `${API_PREFIX}/admin/settlements`;
const ADMIN_PAYOUTS = `${API_PREFIX}/admin/payouts`;

function unwrap<T>(res: { data: ApiResponse<T> }): T {
  if (res.data.data === undefined || res.data.data === null) {
    throw new Error(res.data.error ?? 'Empty response');
  }
  return res.data.data;
}

export const payoutApi = {
  // Preview payable balance + settleable orders for a store.
  getSettlement: async (storeId: string): Promise<SettlementPreview> =>
    unwrap(await http.get<ApiResponse<SettlementPreview>>(ADMIN_SETTLEMENTS, {
      params: { store_id: storeId },
    })),

  // List payout batch history for a store.
  listBatches: async (storeId: string, limit = 20, offset = 0): Promise<PayoutBatch[]> =>
    unwrap(await http.get<ApiResponse<PayoutBatch[]>>(ADMIN_PAYOUTS, {
      params: { store_id: storeId, limit, offset },
    })),

  // Create a new payout batch from settleable orders.
  createBatch: async (body: CreatePayoutBody): Promise<PayoutBatch> =>
    unwrap(await http.post<ApiResponse<PayoutBatch>>(ADMIN_PAYOUTS, body)),

  // Execute (settle) a pending payout batch.
  executeBatch: async (batchId: string): Promise<PayoutBatch> =>
    unwrap(await http.post<ApiResponse<PayoutBatch>>(`${ADMIN_PAYOUTS}/${batchId}/execute`, {})),
};
