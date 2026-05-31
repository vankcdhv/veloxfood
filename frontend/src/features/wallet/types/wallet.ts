// Mirrors backend payment-service wallet entities.
// All entity fields are PascalCase to match Go's default JSON encoding.
// Request bodies use snake_case (inbound DTO convention).

export type EntryType = 'TOPUP' | 'PAYMENT' | 'REFUND' | 'PAYOUT';

export interface Wallet {
  ID: string;
  OwnerType: string;
  OwnerID: string;
  Balance: number;
  CreatedAt: string;
  UpdatedAt: string;
}

export interface LedgerEntry {
  ID: string;
  EntryType: EntryType;
  Amount: number;
  RefType: string;
  BalanceAfter: number;
  CreatedAt: string;
}

// Response shape for GET /api/v1/me/wallet — the wallet fields are returned
// flat with the ledger embedded as `Ledger` (not nested under `Wallet`).
export interface WalletData {
  ID: string;
  OwnerType: string;
  Balance: number;
  Ledger: LedgerEntry[];
}

// POST /api/v1/wallet/topup response
export interface TopupResult {
  PayUrl: string;
  TopupID: string;
}

// Request body for POST /api/v1/wallet/topup
export interface TopupBody {
  amount: number;
}
