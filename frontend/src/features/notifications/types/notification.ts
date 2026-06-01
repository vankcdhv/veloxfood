// Mirrors notification-service entities. Fields PascalCase (Go JSON default).
// Request bodies use snake_case.

export interface Notification {
  ID: string;
  Type: string;
  Title: string;
  Body: string;
  // Data holds arbitrary JSON metadata (order ID, store ID, etc.)
  Data?: Record<string, unknown>;
  ReadAt?: string | null;
  CreatedAt: string;
}

export interface NotificationListResponse {
  Items: Notification[];
  Total: number;
}

export interface UnreadCountResponse {
  Count: number;
}

// POST /api/v1/me/device-tokens
export interface RegisterDeviceTokenBody {
  fcm_token: string;
  platform: 'web' | 'ios' | 'android';
}
