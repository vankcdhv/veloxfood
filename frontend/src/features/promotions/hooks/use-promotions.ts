import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { publicPromotionApi, vendorPromotionApi } from '../api/promotion-api';
import type { CreatePromotionBody, UpdatePromotionBody } from '../types/promotion';

export const promotionKeys = {
  all: ['promotions'] as const,
  list: (storeId: string) => [...promotionKeys.all, 'list', storeId] as const,
};

export const usePromotions = (storeId: string) =>
  useQuery({
    queryKey: promotionKeys.list(storeId),
    queryFn: () => vendorPromotionApi.list(storeId),
    enabled: !!storeId,
  });

export function usePromotionMutations(storeId: string) {
  const qc = useQueryClient();
  const invalidate = () => qc.invalidateQueries({ queryKey: promotionKeys.list(storeId) });

  return {
    create: useMutation({
      mutationFn: (body: CreatePromotionBody) => vendorPromotionApi.create(storeId, body),
      onSuccess: invalidate,
    }),
    update: useMutation({
      mutationFn: (v: { promoId: string; body: UpdatePromotionBody }) =>
        vendorPromotionApi.update(storeId, v.promoId, v.body),
      onSuccess: invalidate,
    }),
    remove: useMutation({
      mutationFn: (promoId: string) => vendorPromotionApi.delete(storeId, promoId),
      onSuccess: invalidate,
    }),
    toggleStatus: useMutation({
      mutationFn: (v: { promoId: string; status: 'ACTIVE' | 'INACTIVE' }) =>
        vendorPromotionApi.update(storeId, v.promoId, { status: v.status }),
      onSuccess: invalidate,
    }),
  };
}

// Dry-run validate — not tied to a specific store mutation context.
export function useValidatePromotion() {
  return useMutation({
    mutationFn: publicPromotionApi.validate,
  });
}
