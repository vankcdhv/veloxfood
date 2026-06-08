import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  createUser,
  deleteUser,
  getUser,
  listUsers,
  reactivateUser,
  suspendUser,
  updateUser,
  type ListUsersParams,
} from '../api/user-api';
import type { CreateUserInput, UpdateUserInput } from '../schemas/user-schema';

export const userKeys = {
  all: ['users'] as const,
  list: (params: ListUsersParams) => [...userKeys.all, 'list', params] as const,
  detail: (id: string) => [...userKeys.all, 'detail', id] as const,
};

export function useUsers(params: ListUsersParams = { page: 1, page_size: 20 }) {
  return useQuery({
    queryKey: userKeys.list(params),
    // Pass all params (search, status, page, page_size) to the admin list endpoint.
    queryFn: () => listUsers(params),
  });
}

export function useUser(id: string) {
  return useQuery({
    queryKey: userKeys.detail(id),
    queryFn: () => getUser(id),
    enabled: Boolean(id),
  });
}

export function useCreateUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: CreateUserInput) => createUser(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: userKeys.all }),
  });
}

export function useUpdateUser(id: string) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (input: UpdateUserInput) => updateUser(id, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: userKeys.all }),
  });
}

export function useDeleteUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => deleteUser(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: userKeys.all }),
  });
}

export function useSuspendUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (v: { userId: string; reason?: string }) => suspendUser(v.userId, v.reason),
    onSuccess: () => qc.invalidateQueries({ queryKey: userKeys.all }),
  });
}

export function useReactivateUser() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (userId: string) => reactivateUser(userId),
    onSuccess: () => qc.invalidateQueries({ queryKey: userKeys.all }),
  });
}
