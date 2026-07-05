-- MoMo checkout URL, persisted at capture time. Under the DTM saga engine the
-- branch response is not forwarded to the saga opener, so the order service
-- reads the pay_url back via GetPaymentStatus after the saga completes.
ALTER TABLE payments ADD COLUMN IF NOT EXISTS pay_url TEXT NOT NULL DEFAULT '';
