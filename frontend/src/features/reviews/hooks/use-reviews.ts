import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { adminReviewApi, reviewApi } from '../api/review-api';
import type { CreateReviewBody, ReplyReviewBody, ReportReviewBody } from '../types/review';

export const reviewKeys = {
  all: ['reviews'] as const,
  byStore: (storeId: string) => [...reviewKeys.all, 'store', storeId] as const,
  reported: () => [...reviewKeys.all, 'admin', 'reported'] as const,
};

export function useStoreReviews(storeId: string, skip = 0, limit = 20) {
  return useQuery({
    queryKey: [...reviewKeys.byStore(storeId), skip, limit],
    queryFn: () => reviewApi.listByStore(storeId, skip, limit),
    enabled: !!storeId,
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
