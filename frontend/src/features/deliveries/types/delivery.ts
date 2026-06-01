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
  created_at: string;
}

// Delivery record from GET /api/v1/me/deliveries .deliveries[]
export interface MyDelivery {
  ID: string;
  OrderID: string;
  OrderCode?: string;
  StoreID: string;
  LocationID: string;
  ShipFee: number;
  Status: DeliveryStatus;
  BatchID?: string;
  ClaimedAt?: string;
  DeliveredAt?: string;
  CreatedAt: string;
  UpdatedAt: string;
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
