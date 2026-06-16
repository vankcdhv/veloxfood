import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { publicPromotionApi, vendorPromotionApi } from '../api/promotion-api';
import type { CreatePromotionBody, UpdatePromotionBody } from '../types/promotion';

export const promotionKeys = {
  all: ['promotions'] as const,
  list: (storeId: string, page: number) => [...promotionKeys.all, 'list', storeId, page] as const,
};

const PROMOTIONS_PAGE_SIZE = 20;

export const usePromotions = (storeId: string, page = 1) =>
  useQuery({
    queryKey: promotionKeys.list(storeId, page),
    queryFn: () => vendorPromotionApi.list(storeId, page, PROMOTIONS_PAGE_SIZE),
    enabled: !!storeId,
  });

export function usePromotionMutations(storeId: string) {
  const qc = useQueryClient();
  // Invalidate all pages by matching the base prefix (storeId without page).
  const invalidate = () => qc.invalidateQueries({ queryKey: [...promotionKeys.all, 'list', storeId] });

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
