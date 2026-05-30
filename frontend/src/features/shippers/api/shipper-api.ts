import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse } from '@/shared/api/api-response';
import type { ShipperListResponse, ShipperStatus } from '../types/shipper';

const SHIPPER_PATH = `${API_PREFIX}/shipper`;
const ADMIN_SHIPPERS_PATH = `${API_PREFIX}/admin/shippers`;

// registerShipper uploads the two verification photos as multipart/form-data.
export async function registerShipper(idDocument: File, portrait: File): Promise<void> {
  const form = new FormData();
  form.append('id_document', idDocument);
  form.append('portrait', portrait);
  await http.post<ApiResponse>(`${SHIPPER_PATH}/register`, form, {
    headers: { 'Content-Type': 'multipart/form-data' },
  });
}

export interface ListShippersParams {
  status?: ShipperStatus;
  page?: number;
  page_size?: number;
}

export async function listShippers(params: ListShippersParams = {}): Promise<ShipperListResponse> {
  const res = await http.get<ApiResponse<ShipperListResponse>>(ADMIN_SHIPPERS_PATH, { params });
  if (!res.data.data) throw new Error(res.data.error ?? 'Không tải được danh sách shipper');
  return res.data.data;
}

export async function approveShipper(userId: string): Promise<void> {
  await http.post<ApiResponse>(`${ADMIN_SHIPPERS_PATH}/${userId}/approve`, {});
}

export async function rejectShipper(userId: string, reason: string): Promise<void> {
  await http.post<ApiResponse>(`${ADMIN_SHIPPERS_PATH}/${userId}/reject`, { reason });
}
