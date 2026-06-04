-- Revert: map any REFUNDED back to PAID, then restore the two-value constraint.
UPDATE orders SET payment_status = 'PAID' WHERE payment_status = 'REFUNDED';
ALTER TABLE orders DROP CONSTRAINT IF EXISTS orders_payment_status_check;
ALTER TABLE orders ADD CONSTRAINT orders_payment_status_check
  CHECK (payment_status IN ('UNPAID','PAID'));
