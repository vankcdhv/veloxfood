import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { walletApi } from '../api/wallet-api';
import type { TopupBody } from '../types/wallet';

export const walletKeys = {
  all: ['wallet'] as const,
  mine: () => [...walletKeys.all, 'mine'] as const,
};

export function useMyWallet() {
  return useQuery({
    queryKey: walletKeys.mine(),
    queryFn: walletApi.getMyWallet,
  });
}

export function useTopupWallet() {
  const qc = useQueryClient();

  return useMutation({
    mutationFn: (body: TopupBody) => walletApi.topup(body),
    onSuccess: () => {
      // Invalidate wallet data after a successful topup initiation so balance
      // refreshes when the user returns from the payment page.
      qc.invalidateQueries({ queryKey: walletKeys.mine() });
    },
  });
}
