import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { adminReviewApi, reviewApi } from '../api/review-api';
import type { CreateReviewBody, ReplyReviewBody, ReportReviewBody } from '../types/review';

export const reviewKeys = {
  all: ['reviews'] as const,
  byStore: (storeId: string) => [...reviewKeys.all, 'store', storeId] as const,
  storeSummary: (storeId: string) => [...reviewKeys.all, 'store', storeId, 'summary'] as const,
  itemSummaries: (storeId: string) => [...reviewKeys.all, 'store', storeId, 'item-summaries'] as const,
  byItem: (storeId: string, itemId: string) => [...reviewKeys.all, 'store', storeId, 'item', itemId] as const,
  reported: () => [...reviewKeys.all, 'admin', 'reported'] as const,
};

export function useStoreReviews(storeId: string, page = 1, pageSize = 20) {
  return useQuery({
    queryKey: [...reviewKeys.byStore(storeId), page, pageSize],
    queryFn: () => reviewApi.listByStore(storeId, page, pageSize),
    enabled: !!storeId,
  });
}

// Store-level average rating + count (for the store card / header).
export function useStoreRatingSummary(storeId: string) {
  return useQuery({
    queryKey: reviewKeys.storeSummary(storeId),
    queryFn: () => reviewApi.storeSummary(storeId),
    enabled: !!storeId,
  });
}

// Per-menu-item rating summaries for a store, returned as a Map keyed by item ID.
export function useItemRatingSummaries(storeId: string) {
  return useQuery({
    queryKey: reviewKeys.itemSummaries(storeId),
    queryFn: async () => {
      const list = await reviewApi.itemSummaries(storeId);
      return new Map((list ?? []).map((s) => [s.TargetID, s]));
    },
    enabled: !!storeId,
  });
}

// Reviews for a single menu item (lazy — only fetched when enabled).
export function useItemReviews(storeId: string, itemId: string, enabled = true) {
  return useQuery({
    queryKey: reviewKeys.byItem(storeId, itemId),
    queryFn: () => reviewApi.listByItem(storeId, itemId),
    enabled: !!storeId && !!itemId && enabled,
  });
}

export function useCreateReview(orderId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: CreateReviewBody) => reviewApi.create(orderId, body),
    onSuccess: (_data, variables) => {
      // Invalidate store reviews so the new review appears immediately.
      qc.invalidateQueries({ queryKey: reviewKeys.byStore(variables.target_id) });
    },
  });
}

export function useReplyReview(storeId: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ reviewId, body }: { reviewId: string; body: ReplyReviewBody }) =>
      reviewApi.reply(reviewId, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: reviewKeys.byStore(storeId) });
    },
  });
}

export function useReportReview() {
  return useMutation({
    mutationFn: ({ reviewId, body }: { reviewId: string; body: ReportReviewBody }) =>
      reviewApi.report(reviewId, body),
  });
}

// Admin hooks
export function useReportedReviews(skip = 0, limit = 20) {
  return useQuery({
    queryKey: [...reviewKeys.reported(), skip, limit],
    queryFn: () => adminReviewApi.listReported(skip, limit),
  });
}

export function useAdminReviewActions() {
  const qc = useQueryClient();
  const invalidate = () => qc.invalidateQueries({ queryKey: reviewKeys.reported() });
  return {
    hide: useMutation({ mutationFn: (id: string) => adminReviewApi.hide(id), onSuccess: invalidate }),
    restore: useMutation({ mutationFn: (id: string) => adminReviewApi.restore(id), onSuccess: invalidate }),
  };
}
