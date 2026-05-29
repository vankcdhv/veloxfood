import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse, PaginatedData } from '@/shared/api/api-response';
import type { User } from '../types/user';
import type { CreateUserInput, UpdateUserInput } from '../schemas/user-schema';

const USERS_PATH = `${API_PREFIX}/users`;

export interface ListUsersParams {
  page?: number;
  page_size?: number;
}

export async function listUsers(params: ListUsersParams = {}): Promise<PaginatedData<User>> {
  const res = await http.get<ApiResponse<PaginatedData<User>>>(USERS_PATH, { params });
  if (!res.data.data) {
    throw new Error(res.data.error ?? 'Empty response');
  }
  return res.data.data;
}

export async function getUser(id: string): Promise<User> {
  const res = await http.get<ApiResponse<User>>(`${USERS_PATH}/${id}`);
  if (!res.data.data) {
    throw new Error(res.data.error ?? 'User not found');
  }
  return res.data.data;
}

export async function createUser(input: CreateUserInput): Promise<User> {
  const res = await http.post<ApiResponse<User>>(USERS_PATH, input);
  if (!res.data.data) {
    throw new Error(res.data.error ?? 'Create failed');
  }
  return res.data.data;
}

export async function updateUser(id: string, input: UpdateUserInput): Promise<User> {
  const res = await http.put<ApiResponse<User>>(`${USERS_PATH}/${id}`, input);
  if (!res.data.data) {
    throw new Error(res.data.error ?? 'Update failed');
  }
  return res.data.data;
}

export async function deleteUser(id: string): Promise<void> {
  await http.delete<ApiResponse<null>>(`${USERS_PATH}/${id}`);
}
