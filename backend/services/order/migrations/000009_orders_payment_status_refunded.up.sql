-- Allow REFUNDED so a cancelled/rejected online-paid order reflects its refund.
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_payment_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_payment_status_check
  CHECK (payment_status IN ('UNPAID','PAID','REFUNDED'));
