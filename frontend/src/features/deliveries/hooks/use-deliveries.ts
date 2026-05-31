import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { deliveryApi } from '../api/delivery-api';
import type { ReportIncidentBody, UpdateStatusBody } from '../types/delivery';

export const deliveryKeys = {
  all: ['deliveries'] as const,
  available: () => [...deliveryKeys.all, 'available'] as const,
  mine: () => [...deliveryKeys.all, 'mine'] as const,
};

// Poll available deliveries every 15 s so the list stays fresh.
export function useAvailableDeliveries() {
  return useQuery({
    queryKey: deliveryKeys.available(),
    queryFn: deliveryApi.listAvailable,
    refetchInterval: 15_000,
  });
}

// Poll shipper's own deliveries every 10 s.
export function useMyDeliveries() {
  return useQuery({
    queryKey: deliveryKeys.mine(),
    queryFn: deliveryApi.myDeliveries,
    refetchInterval: 10_000,
  });
}

export function useClaimDelivery() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (orderId: string) => deliveryApi.claim(orderId),
    onSuccess: () => {
      // Remove the claimed order from the available pool and refresh my list.
      qc.invalidateQueries({ queryKey: deliveryKeys.available() });
      qc.invalidateQueries({ queryKey: deliveryKeys.mine() });
    },
  });
}

export function useUpdateDeliveryStatus() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ orderId, body }: { orderId: string; body: UpdateStatusBody }) =>
      deliveryApi.updateStatus(orderId, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: deliveryKeys.mine() });
    },
  });
}

export function useReportIncident() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ orderId, body }: { orderId: string; body: ReportIncidentBody }) =>
      deliveryApi.reportIncident(orderId, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: deliveryKeys.mine() });
    },
  });
}
