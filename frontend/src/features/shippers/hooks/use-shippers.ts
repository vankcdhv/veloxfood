import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  approveShipper,
  listShippers,
  myShipperStatus,
  registerShipper,
  rejectShipper,
  type ListShippersParams,
} from '../api/shipper-api';

export const shipperKeys = {
  all: ['shippers'] as const,
  list: (params: ListShippersParams) => [...shipperKeys.all, 'list', params] as const,
  mine: () => [...shipperKeys.all, 'mine'] as const,
};

export function useMyShipperStatus() {
  return useQuery({ queryKey: shipperKeys.mine(), queryFn: myShipperStatus });
}

export function useShippers(params: ListShippersParams = { status: 'pending', page: 1, page_size: 20 }) {
  return useQuery({
    queryKey: shipperKeys.list(params),
    queryFn: () => listShippers(params),
  });
}

export function useRegisterShipper() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (vars: { idDocument: File; portrait: File }) =>
      registerShipper(vars.idDocument, vars.portrait),
    onSuccess: () => qc.invalidateQueries({ queryKey: shipperKeys.mine() }),
  });
}

export function useApproveShipper() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => approveShipper(userId),
    onSuccess: () => qc.invalidateQueries({ queryKey: shipperKeys.all }),
  });
}

export function useRejectShipper() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (vars: { userId: string; reason: string }) => rejectShipper(vars.userId, vars.reason),
    onSuccess: () => qc.invalidateQueries({ queryKey: shipperKeys.all }),
  });
}
