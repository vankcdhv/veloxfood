// Mirrors backend order-service entities.
// All entity fields are PascalCase to match Go's default JSON encoding.
// Request bodies use snake_case.

export type OrderStatus =
  | 'PENDING'
  | 'CONFIRMED'
  | 'PREPARING'
  | 'READY'
  | 'READY_PICKUP'
  | 'SHIPPER_ASSIGNED'
  | 'DELIVERING'
  | 'DELIVERED'
  | 'COMPLETED'
  | 'CANCELLED'
  | 'REJECTED';

export type FulfillmentType = 'DELIVERY' | 'PICKUP';
export type PaymentMethod = 'COD' | 'MOMO' | 'WALLET';
export type PaymentStatus = 'PENDING' | 'PAID' | 'REFUNDED' | 'FAILED';

export interface OrderItem {
  MenuItemID: string;
  NameSnapshot: string;
  PriceSnapshot: number;
  Qty: number;
}

export interface StatusHistoryEntry {
  Status: OrderStatus;
  OccurredAt: string;
}

export interface Order {
  ID: string;
  Code: string;
  StoreID: string;
  StoreName?: string;
  CustomerID: string;
  Status: OrderStatus;
  Fulfillment: FulfillmentType;
  LocationID?: string;
  ItemsTotal: number;
  ShipFee: number;
  Discount: number;
  GrandTotal: number;
  PaymentMethod: PaymentMethod;
  PaymentStatus: PaymentStatus;
  PickupPin?: string;
  PlacedAt: string;
  DesiredTime?: string; // RFC3339 — customer's requested receive time
  Items: OrderItem[];
  StatusHistory?: StatusHistoryEntry[];
  VoucherCodes?: string[];
  // Owner-facing enrichment (store order list/detail): customer contact + path.
  CustomerName?: string;
  CustomerPhone?: string;
  LocationPath?: string;
}

// POST /api/v1/orders
export interface PlaceOrderBody {
  store_id: string;
  location_id?: string;
  location_level?: 'BUILDING' | 'FLOOR' | 'ROOM';
  fulfillment: FulfillmentType;
  payment_method: PaymentMethod;
  voucher_codes: string[];
  desired_time?: string; // RFC3339 — customer's requested receive time (today only)
  items: Array<{
    menu_item_id: string;
    qty: number;
  }>;
}

export interface PlaceOrderResult {
  order_id: string;
  code: string;
  grand_total: number;
  pay_url?: string;
}

// PATCH /api/v1/stores/:storeId/orders/:id/status
export interface AdvanceOrderStatusBody {
  status: OrderStatus;
}

// POST /api/v1/stores/:storeId/orders/:id/pickup-verify
export interface PickupVerifyBody {
  pin: string;
}
