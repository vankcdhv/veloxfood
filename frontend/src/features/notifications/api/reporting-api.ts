import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse } from '@/shared/api/api-response';

// Analytics summary returned by GET /api/v1/admin/analytics?period=day
export interface AnalyticsSummary {
  OrdersTotal: number;
  RevenueTotal: number;
  NewUsersTotal: number;
  NewVendors: number;
  CancelledTotal: number;
  AvgOrderValue: number;
}

// Recent order row returned by GET /api/v1/admin/orders/recent?limit=N
export interface RecentOrderRow {
  OrderID: string;
  StoreID: string;
  GrandTotal: number;
  Status: string;
  CreatedAt: string;
  // StoreName may be present in enriched responses
  StoreName?: string;
}

function unwrap<T>(res: { data: ApiResponse<T> }): T {
  if (res.data.data === undefined || res.data.data === null) {
    throw new Error(res.data.error ?? 'Empty response');
  }
  return res.data.data;
}

export const reportingApi = {
  // GET /api/v1/admin/analytics?period=day
  analytics: async (period: 'day' | 'week' | 'month' = 'day'): Promise<AnalyticsSummary> =>
    unwrap(
      await http.get<ApiResponse<AnalyticsSummary>>(`${API_PREFIX}/admin/analytics`, {
        params: { period },
      }),
    ),

  // GET /api/v1/admin/orders/recent?limit=N
  recentOrders: async (limit = 20): Promise<RecentOrderRow[]> =>
    unwrap(
      await http.get<ApiResponse<RecentOrderRow[]>>(`${API_PREFIX}/admin/orders/recent`, {
        params: { limit },
      }),
    ),
};
