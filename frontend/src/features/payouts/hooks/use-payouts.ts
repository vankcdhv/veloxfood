import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { payoutApi } from '../api/payout-api';
import type { CreatePayoutBody } from '../types/payout';

export const payoutKeys = {
  all: ['payouts'] as const,
  settlement: (storeId: string) => [...payoutKeys.all, 'settlement', storeId] as const,
  batches: (storeId: string) => [...payoutKeys.all, 'batches', storeId] as const,
};

export const useSettlement = (storeId: string) =>
  useQuery({
    queryKey: payoutKeys.settlement(storeId),
    queryFn: () => payoutApi.getSettlement(storeId),
    enabled: !!storeId,
  });

export const usePayoutBatches = (storeId: string) =>
  useQuery({
    queryKey: payoutKeys.batches(storeId),
    queryFn: () => payoutApi.listBatches(storeId),
    enabled: !!storeId,
  });

export function usePayoutMutations(storeId: string) {
  const qc = useQueryClient();

  const invalidate = () => {
    qc.invalidateQueries({ queryKey: payoutKeys.settlement(storeId) });
    qc.invalidateQueries({ queryKey: payoutKeys.batches(storeId) });
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
