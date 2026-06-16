import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { customerOrderApi, ownerOrderApi } from '../api/order-api';
import type {
  AdvanceOrderStatusBody,
  PickupVerifyBody,
  PlaceOrderBody,
} from '../types/order';

export const orderKeys = {
  all: ['orders'] as const,
  myList: (page: number) => [...orderKeys.all, 'my', 'list', page] as const,
  myDetail: (id: string) => [...orderKeys.all, 'my', 'detail', id] as const,
  storeList: (storeId: string, page: number) =>
    [...orderKeys.all, 'store', storeId, 'list', page] as const,
  storeDetail: (storeId: string, orderId: string) =>
    [...orderKeys.all, 'store', storeId, 'detail', orderId] as const,
};

// ---- Customer hooks ----

export function useMyOrders(page = 1) {
  return useQuery({
    queryKey: orderKeys.myList(page),
    queryFn: () => customerOrderApi.list(page),
  });
}

export function useMyOrder(id: string) {
  return useQuery({
    queryKey: orderKeys.myDetail(id),
    queryFn: () => customerOrderApi.get(id),
    enabled: !!id,
    // Poll every 10 s when on track page so status stays fresh.
    refetchInterval: 10_000,
  });
}

export function usePlaceOrder() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (body: PlaceOrderBody) => customerOrderApi.place(body),
    onSuccess: () => {
      // Invalidate order list so the new order appears immediately.
      qc.invalidateQueries({ queryKey: orderKeys.all });
    },
  });
}

export function useCancelOrder(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => customerOrderApi.cancel(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: orderKeys.myDetail(id) });
      qc.invalidateQueries({ queryKey: orderKeys.all });
    },
  });
}

export function useReorder() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => customerOrderApi.reorder(id),
    // Reorder re-populates the cart — refresh cart (and orders) views.
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: orderKeys.all });
      qc.invalidateQueries({ queryKey: ['cart'] });
    },
  });
}

// ---- Owner / staff hooks ----

export function useStoreOrders(storeId: string, page = 1, pageSize = 20) {
  return useQuery({
    queryKey: orderKeys.storeList(storeId, page),
    queryFn: () => ownerOrderApi.list(storeId, page, pageSize),
    enabled: !!storeId,
    refetchInterval: 15_000,
  });
}

export function useStoreOrder(storeId: string, orderId: string) {
  return useQuery({
    queryKey: orderKeys.storeDetail(storeId, orderId),
    queryFn: () => ownerOrderApi.get(storeId, orderId),
    enabled: !!storeId && !!orderId,
  });
}

export function useOwnerOrderMutations(storeId: string) {
  const qc = useQueryClient();
  const invalidateList = () =>
    qc.invalidateQueries({ queryKey: [...orderKeys.all, 'store', storeId] });

  return {
    confirm: useMutation({
      mutationFn: (orderId: string) => ownerOrderApi.confirm(storeId, orderId),
      onSuccess: invalidateList,
    }),
    reject: useMutation({
      mutationFn: (orderId: string) => ownerOrderApi.reject(storeId, orderId),
      onSuccess: invalidateList,
    }),
    advanceStatus: useMutation({
      mutationFn: (v: { orderId: string; body: AdvanceOrderStatusBody }) =>
        ownerOrderApi.advanceStatus(storeId, v.orderId, v.body),
      onSuccess: invalidateList,
    }),
    verifyPickup: useMutation({
      mutationFn: (v: { orderId: string; body: PickupVerifyBody }) =>
        ownerOrderApi.verifyPickup(storeId, v.orderId, v.body),
      onSuccess: invalidateList,
    }),
  };
}
