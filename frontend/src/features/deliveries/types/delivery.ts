// Mirrors backend delivery-service entities.
// Entity fields are PascalCase (Go JSON default). Request bodies use snake_case.

export type DeliveryStatus =
  | 'AVAILABLE'
  | 'CLAIMED'
  | 'PICKED_UP'
  | 'DELIVERING'
  | 'DELIVERED'
  | 'CANCELLED'
  | 'STORE_DELIVERING';

// GET /api/v1/deliveries/available — one item in the pool
export interface AvailableDelivery {
  order_id: string;
  order_code?: string;
  store_id: string;
  location_id: string;
  room_path: string; // resolved by grpcclient/location; fallback = raw UUID
  ship_fee: number;
  desired_time?: string; // RFC3339
  created_at: string;
}

// Delivery record from GET /api/v1/me/deliveries .deliveries[] — snake_case DTO.
export interface MyDelivery {
  id: string;
  order_id: string;
  order_code?: string;
  store_id: string;
  location_id: string;
  location_level?: string;
  ship_fee: number;
  status: DeliveryStatus;
  batch_id?: string;
  claimed_at?: string;
  delivered_at?: string;
  desired_time?: string;    // RFC3339 — customer's requested receive time
  late_by_minutes?: number; // > 0 means delivery was late
  created_at: string;
}

// GET /api/v1/me/deliveries response data shape
export interface MyDeliveriesData {
  deliveries: MyDelivery[];
  earnings: number;
}

// POST /api/v1/deliveries/:orderId/claim → 201
export interface ClaimResult {
  batch_id: string;
}

// PATCH /api/v1/deliveries/:orderId/status body
export interface UpdateStatusBody {
  status: 'PICKED_UP' | 'DELIVERING' | 'DELIVERED';
}

// POST /api/v1/deliveries/:orderId/incident body
export interface ReportIncidentBody {
  type: string;
  note: string;
  photo_url?: string;
}

// Incident types for the dropdown
export const INCIDENT_TYPES = [
  { value: 'WRONG_ADDRESS', label: 'Sai địa chỉ' },
  { value: 'CUSTOMER_ABSENT', label: 'Khách không có mặt' },
  { value: 'ITEM_DAMAGED', label: 'Hàng bị hỏng' },
  { value: 'OTHER', label: 'Khác' },
] as const;

// Map an incident type code → Vietnamese label (falls back to the raw code).
export function incidentTypeLabel(type: string): string {
  return INCIDENT_TYPES.find((t) => t.value === type)?.label ?? type;
}

// GET /api/v1/admin/incidents — admin-facing incident (resolved order code + shipper name).
export interface AdminIncident {
  id: string;
  order_code: string;
  shipper_name: string;
  type: string;
  note: string;
  photo_url?: string | null;
  status: 'OPEN' | 'RESOLVED';
  created_at: string;
}
