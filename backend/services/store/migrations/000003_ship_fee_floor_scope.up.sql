-- ship_fee_rules.scope is VARCHAR(10) with no CHECK constraint,
-- so no schema change is needed to support the 'floor' value.
-- This migration is intentionally a no-op to mark the feature version boundary.
SELECT 1;
