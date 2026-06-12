import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse } from '@/shared/api/api-response';
import type {
  AdminIncident,
  AvailableDelivery,
  ClaimResult,
  MyDeliveriesData,
  ReportIncidentBody,
  UpdateStatusBody,
} from '../types/delivery';

const DELIVERIES = `${API_PREFIX}/deliveries`;

function unwrap<T>(res: { data: ApiResponse<T> }): T {
  if (res.data.data === undefined || res.data.data === null) {
    throw new Error(res.data.error ?? 'Empty response');
  }
  return res.data.data;
}

export const deliveryApi = {
  // GET /api/v1/deliveries/available
  listAvailable: async (): Promise<AvailableDelivery[]> =>
    unwrap(await http.get<ApiResponse<AvailableDelivery[]>>(`${DELIVERIES}/available`)),

  // POST /api/v1/deliveries/:orderId/claim
  claim: async (orderId: string): Promise<ClaimResult> =>
    unwrap(await http.post<ApiResponse<ClaimResult>>(`${DELIVERIES}/${orderId}/claim`, {})),

  // PATCH /api/v1/deliveries/:orderId/status
  updateStatus: async (orderId: string, body: UpdateStatusBody): Promise<void> => {
    await http.patch<ApiResponse>(`${DELIVERIES}/${orderId}/status`, body);
  },

  // GET /api/v1/me/deliveries
  myDeliveries: async (): Promise<MyDeliveriesData> =>
    unwrap(await http.get<ApiResponse<MyDeliveriesData>>(`${API_PREFIX}/me/deliveries`)),

  // POST /api/v1/deliveries/:orderId/incident
  reportIncident: async (orderId: string, body: ReportIncidentBody): Promise<void> => {
    await http.post<ApiResponse>(`${DELIVERIES}/${orderId}/incident`, body);
  },

  // GET /api/v1/deliveries/:orderId/shipper — the shipper who delivered the
  // caller's order (for rating). Returns '' if none / not the owner.
  orderShipper: async (orderId: string): Promise<string> => {
    const res = await http.get<ApiResponse<{ shipper_id: string }>>(
      `${DELIVERIES}/${orderId}/shipper`,
    );
    return res.data.data?.shipper_id ?? '';
  },

  // GET /api/v1/admin/incidents — all delivery incidents for admin review.
  adminListIncidents: async (): Promise<AdminIncident[]> =>
    unwrap(await http.get<ApiResponse<AdminIncident[]>>(`${API_PREFIX}/admin/incidents`)),

  // POST /api/v1/deliveries/:orderId/incident-photo — upload evidence, returns its URL.
  uploadIncidentPhoto: async (orderId: string, file: File): Promise<string> => {
    const form = new FormData();
    form.append('image', file);
    const res = await http.post<ApiResponse<{ photo_url: string }>>(
      `${DELIVERIES}/${orderId}/incident-photo`,
      form,
      { headers: { 'Content-Type': 'multipart/form-data' } },
    );
    if (!res.data.data?.photo_url) throw new Error(res.data.error ?? 'Upload failed');
    return res.data.data.photo_url;
  },
};
