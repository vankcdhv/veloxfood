import { http } from '@/shared/lib/http-client';
import { API_PREFIX } from '@/shared/config/constants';
import type { ApiResponse } from '@/shared/api/api-response';
import type {
  NotificationListResponse,
  UnreadCountResponse,
  RegisterDeviceTokenBody,
} from '../types/notification';

function unwrap<T>(res: { data: ApiResponse<T> }): T {
  if (res.data.data === undefined || res.data.data === null) {
    throw new Error(res.data.error ?? 'Empty response');
  }
  return res.data.data;
}

const BASE = `${API_PREFIX}/me/notifications`;

export const notificationApi = {
  // GET /api/v1/me/notifications?skip=&limit=
  list: async (skip = 0, limit = 20): Promise<NotificationListResponse> =>
    unwrap(
      await http.get<ApiResponse<NotificationListResponse>>(BASE, {
        params: { skip, limit },
      }),
    ),

  // GET /api/v1/me/notifications/unread-count
  unreadCount: async (): Promise<UnreadCountResponse> =>
    unwrap(
      await http.get<ApiResponse<UnreadCountResponse>>(`${BASE}/unread-count`),
    ),

  // PATCH /api/v1/me/notifications/:id/read
  markRead: async (id: string): Promise<void> => {
    await http.patch<ApiResponse>(`${BASE}/${id}/read`, {});
  },

  // POST /api/v1/me/notifications/read-all
  markAllRead: async (): Promise<void> => {
    await http.post<ApiResponse>(`${BASE}/read-all`, {});
  },

  // POST /api/v1/me/device-tokens
  registerDeviceToken: async (body: RegisterDeviceTokenBody): Promise<void> => {
    await http.post<ApiResponse>(`${API_PREFIX}/me/device-tokens`, body);
  },
};
