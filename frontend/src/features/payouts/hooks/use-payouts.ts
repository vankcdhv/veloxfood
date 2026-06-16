import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { payoutApi } from '../api/payout-api';
import type { CreatePayoutBody } from '../types/payout';

export const payoutKeys = {
  all: ['payouts'] as const,
  settlement: (storeId: string) => [...payoutKeys.all, 'settlement', storeId] as const,
  batches: (storeId: string, page: number) => [...payoutKeys.all, 'batches', storeId, page] as const,
};

export const useSettlement = (storeId: string) =>
  useQuery({
    queryKey: payoutKeys.settlement(storeId),
    queryFn: () => payoutApi.getSettlement(storeId),
    enabled: !!storeId,
  });

const BATCHES_PAGE_SIZE = 20;

export const usePayoutBatches = (storeId: string, page = 1) =>
  useQuery({
    queryKey: payoutKeys.batches(storeId, page),
    queryFn: () => payoutApi.listBatches(storeId, page, BATCHES_PAGE_SIZE),
    enabled: !!storeId,
  });

export function usePayoutMutations(storeId: string) {
  const qc = useQueryClient();

  const invalidate = () => {
    qc.invalidateQueries({ queryKey: payoutKeys.settlement(storeId) });
    // Invalidate all batch pages by matching without the page segment.
    qc.invalidateQueries({ queryKey: [...payoutKeys.all, 'batches', storeId] });
  };

  return {
    createBatch: useMutation({
      mutationFn: (body: CreatePayoutBody) => payoutApi.createBatch(body),
      onSuccess: invalidate,
    }),
    executeBatch: useMutation({
      mutationFn: (batchId: string) => payoutApi.executeBatch(batchId),
      onSuccess: invalidate,
    }),
  };
}
