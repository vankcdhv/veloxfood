-- Generalize a saved customer location from room-only to any tree level.
-- A saved location now records (location_level, location_id) where location_id
-- points at a building, floor, or room — matching how checkout/orders already
-- carry location_level. Existing rows are room-level and backfill cleanly.
ALTER TABLE customer_locations ADD COLUMN location_level VARCHAR(10) NOT NULL DEFAULT 'ROOM';
ALTER TABLE customer_locations ADD COLUMN location_id UUID;
UPDATE customer_locations SET location_id = room_id;
ALTER TABLE customer_locations ALTER COLUMN location_id SET NOT NULL;

-- Drop the room-only FK + column. location_id can no longer FK a single table
-- (it may reference buildings/floors/rooms), so existence is validated in the
-- usecase per level.
ALTER TABLE customer_locations DROP CONSTRAINT customer_locations_room_id_fkey;
ALTER TABLE customer_locations DROP COLUMN room_id;

ALTER TABLE customer_locations
  ADD CONSTRAINT chk_customer_location_level CHECK (location_level IN ('BUILDING','FLOOR','ROOM'));
