-- Enable pgcrypto for gen_random_uuid().
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- wallets: SYSTEM (singleton), CUSTOMER (per user), STORE_PAYABLE (per store).
-- SYSTEM wallet owner_id is the zero UUID (00000000-0000-0000-0000-000000000000).
CREATE TABLE IF NOT EXISTS wallets (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  owner_type  VARCHAR(20) NOT NULL CHECK (owner_type IN ('SYSTEM','CUSTOMER','STORE_PAYABLE')),
  owner_id    UUID NOT NULL,
  -- CUSTOMER / STORE_PAYABLE wallets can never overdraw. The SYSTEM wallet is an
  -- internal clearing/float account: customer top-ups bring external (MoMo) money
  -- in, so SYSTEM may legitimately go negative — that balance is the liability
  -- owed back to customers.
  balance     BIGINT NOT NULL DEFAULT 0 CHECK (balance >= 0 OR owner_type = 'SYSTEM'),
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (owner_type, owner_id)
);

-- ledger_entries: append-only double-entry journal.
-- amount is signed: positive = credit, negative = debit.
-- balance_after must equal the wallet.balance after this entry is applied.
CREATE TABLE IF NOT EXISTS ledger_entries (
  id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  wallet_id     UUID NOT NULL REFERENCES wallets(id),
  entry_type    VARCHAR(20) NOT NULL CHECK (entry_type IN ('TOPUP','PAYMENT','REFUND','PAYOUT')),
  amount        BIGINT NOT NULL,
  ref_type      VARCHAR(20) NOT NULL CHECK (ref_type IN ('order','topup','payout_batch')),
  ref_id        UUID NOT NULL,
  balance_after BIGINT NOT NULL,
  trace_id      VARCHAR(80),
  created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ledger_wallet_created ON ledger_entries(wallet_id, created_at DESC);

-- payments: tracks each payment attempt (order or wallet top-up).
-- Exactly one of order_id / topup_id must be non-null.
CREATE TABLE IF NOT EXISTS payments (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  order_id         UUID,
  topup_id         UUID,
  customer_id      UUID NOT NULL,
  method           VARCHAR(20) NOT NULL CHECK (method IN ('COD','MOMO','WALLET')),
  provider         VARCHAR(20) NOT NULL CHECK (provider IN ('MOMO','INTERNAL')),
  env              VARCHAR(10) NOT NULL DEFAULT 'demo',
  amount           BIGINT NOT NULL,
  status           VARCHAR(20) NOT NULL DEFAULT 'PENDING'
                   CHECK (status IN ('PENDING','CAPTURED','FAILED','REFUNDED')),
  momo_trans_id    VARCHAR(100),
  idempotency_key  VARCHAR(200) UNIQUE,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  CONSTRAINT chk_payments_ref CHECK (order_id IS NOT NULL OR topup_id IS NOT NULL)
);

CREATE INDEX IF NOT EXISTS idx_payments_order_id   ON payments(order_id)   WHERE order_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_payments_customer_id ON payments(customer_id);

-- payout_batches: store settlement batches created by admins.
CREATE TABLE IF NOT EXISTS payout_batches (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  store_id     UUID NOT NULL,
  period_from  DATE NOT NULL,
  period_to    DATE NOT NULL,
  order_ids    JSONB NOT NULL DEFAULT '[]',
  total_amount BIGINT NOT NULL DEFAULT 0,
  status       VARCHAR(20) NOT NULL DEFAULT 'PENDING'
               CHECK (status IN ('PENDING','SETTLED')),
  settled_by   UUID,
  settled_at   TIMESTAMPTZ,
  created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_payout_store_status ON payout_batches(store_id, status);
