// Mirrors backend payment-service payout entities.
// Response fields are PascalCase (Go JSON default). Request bodies use snake_case.

export type PayoutStatus = 'PENDING' | 'SETTLED';

// GET /api/v1/admin/settlements?store_id=<uuid>
export interface SettlementPreview {
  StoreID: string;
  PayableBalance: number;
  Orders: SettlementOrder[];
  Total: number;
}

export interface SettlementOrder {
  OrderID: string;
  Amount: number;
  CreatedAt: string;
}

// One payout batch from GET /api/v1/admin/payouts?store_id=...
export interface PayoutBatch {
  ID: string;
  StoreID: string;
  PeriodFrom: string;
  PeriodTo: string;
  OrderIDs: string[];
  TotalAmount: number;
  Status: PayoutStatus;
  SettledBy?: string;
  SettledAt?: string;
  CreatedAt: string;
}

// POST /api/v1/admin/payouts request body
export interface CreatePayoutBody {
  store_id: string;
  period_from: string;
  period_to: string;
  order_ids: string[];
  total_amount: number;
}
