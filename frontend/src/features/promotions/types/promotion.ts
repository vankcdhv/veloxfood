// Mirrors backend promotion-service entities.
// All fields are PascalCase to match Go's default JSON encoding.

export type PromotionType = 'ORDER_DISCOUNT' | 'SHIP_DISCOUNT';
export type PromotionValueKind = 'PERCENT' | 'AMOUNT';
export type PromotionStatus = 'ACTIVE' | 'INACTIVE';

export interface Promotion {
  ID: string;
  StoreID: string;
  Code: string;
  Type: PromotionType;
  ValueKind: PromotionValueKind;
  // Money fields are int64 VND (number in JSON).
  Value: number;
  MinOrder: number;
  MaxDiscount: number | null;
  StartsAt: string;
  EndsAt: string;
  UsageLimit: number | null;
  UsedCount: number;
  Status: PromotionStatus;
  CreatedAt: string;
  UpdatedAt: string;
}

// Request bodies are snake_case (repo convention for inbound DTOs); responses
// above stay PascalCase to match Go's default entity encoding.

// Body for POST /stores/:storeId/promotions
export interface CreatePromotionBody {
  code: string;
  type: PromotionType;
  value_kind: PromotionValueKind;
  value: number;
  min_order: number;
  max_discount?: number;
  starts_at: string;
  ends_at: string;
  usage_limit?: number;
}

// Body for PATCH /stores/:storeId/promotions/:promoId — code is immutable
export interface UpdatePromotionBody {
  value?: number;
  min_order?: number;
  max_discount?: number | null;
  starts_at?: string;
  ends_at?: string;
  usage_limit?: number | null;
  status?: PromotionStatus;
}

// POST /promotions/validate response (customer-facing dry-run)
export interface ValidatePromotionResult {
  Applicable: boolean;
  ItemDiscount: number;
  ShipDiscount: number;
  ErrorReason: string;
}
