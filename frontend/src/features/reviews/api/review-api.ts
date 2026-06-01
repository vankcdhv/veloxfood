import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse } from '@/shared/api/api-response';
import type {
  CreateReviewBody,
  ItemRatingSummary,
  RatingSummary,
  ReplyReviewBody,
  ReportReviewBody,
  Review,
  ReviewListResponse,
} from '../types/review';

function unwrap<T>(res: { data: ApiResponse<T> }): T {
  if (res.data.data === undefined || res.data.data === null) {
    throw new Error(res.data.error ?? 'Empty response');
  }
  return res.data.data;
}

export const reviewApi = {
  // POST /api/v1/orders/:orderId/reviews
  create: async (orderId: string, body: CreateReviewBody): Promise<Review> =>
    unwrap(
      await http.post<ApiResponse<Review>>(
        `${API_PREFIX}/orders/${orderId}/reviews`,
        body,
      ),
    ),

  // GET /api/v1/stores/:storeId/reviews
  listByStore: async (storeId: string, page = 1, pageSize = 20): Promise<ReviewListResponse> =>
    unwrap(
      await http.get<ApiResponse<ReviewListResponse>>(
        `${API_PREFIX}/stores/${storeId}/reviews`,
        { params: { page, page_size: pageSize } },
      ),
    ),

  // GET /api/v1/stores/:storeId/reviews/summary — store avg rating + count
  storeSummary: async (storeId: string): Promise<RatingSummary> =>
    unwrap(
      await http.get<ApiResponse<RatingSummary>>(
        `${API_PREFIX}/stores/${storeId}/reviews/summary`,
      ),
    ),

  // GET /api/v1/stores/:storeId/reviews/item-summaries — per-item avg+count
  itemSummaries: async (storeId: string): Promise<ItemRatingSummary[]> =>
    unwrap(
      await http.get<ApiResponse<ItemRatingSummary[]>>(
        `${API_PREFIX}/stores/${storeId}/reviews/item-summaries`,
      ),
    ),

  // GET /api/v1/stores/:storeId/reviews/items/:itemId — reviews of one menu item
  listByItem: async (storeId: string, itemId: string, page = 1, pageSize = 20): Promise<ReviewListResponse> =>
    unwrap(
      await http.get<ApiResponse<ReviewListResponse>>(
        `${API_PREFIX}/stores/${storeId}/reviews/items/${itemId}`,
        { params: { page, page_size: pageSize } },
      ),
    ),

  // POST /api/v1/reviews/:id/reply
  reply: async (reviewId: string, body: ReplyReviewBody): Promise<Review> =>
    unwrap(
      await http.post<ApiResponse<Review>>(
        `${API_PREFIX}/reviews/${reviewId}/reply`,
        body,
      ),
    ),

  // POST /api/v1/reviews/:id/report
  report: async (reviewId: string, body: ReportReviewBody): Promise<void> => {
    await http.post<ApiResponse>(`${API_PREFIX}/reviews/${reviewId}/report`, body);
  },
};

export const adminReviewApi = {
  // GET /api/v1/admin/reviews/reported
  listReported: async (skip = 0, limit = 20): Promise<ReviewListResponse> =>
    unwrap(
      await http.get<ApiResponse<ReviewListResponse>>(
        `${API_PREFIX}/admin/reviews/reported`,
        { params: { skip, limit } },
      ),
    ),

  // PATCH /api/v1/admin/reviews/:id/hide
  hide: async (reviewId: string): Promise<void> => {
    await http.patch<ApiResponse>(`${API_PREFIX}/admin/reviews/${reviewId}/hide`, {});
  },

  // PATCH /api/v1/admin/reviews/:id/restore
  restore: async (reviewId: string): Promise<void> => {
    await http.patch<ApiResponse>(`${API_PREFIX}/admin/reviews/${reviewId}/restore`, {});
  },
};
