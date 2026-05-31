import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse } from '@/shared/api/api-response';
import type { TopupBody, TopupResult, WalletData } from '../types/wallet';

function unwrap<T>(res: { data: ApiResponse<T> }): T {
  if (res.data.data === undefined || res.data.data === null) {
    throw new Error(res.data.error ?? 'Empty response');
  }
  return res.data.data;
}

export const walletApi = {
  // GET /api/v1/me/wallet — returns wallet + ledger entries for the current user.
  getMyWallet: async () =>
    unwrap(await http.get<ApiResponse<WalletData>>(`${API_PREFIX}/me/wallet`)),

  // POST /api/v1/wallet/topup — initiate a MoMo top-up; returns redirect URL.
  topup: async (body: TopupBody) =>
    unwrap(await http.post<ApiResponse<TopupResult>>(`${API_PREFIX}/wallet/topup`, body)),
};
