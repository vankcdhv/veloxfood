-- Revert to room-only saved locations. Non-room saved locations cannot be
-- represented as a room_id, so they are dropped.
ALTER TABLE customer_locations DROP CONSTRAINT IF EXISTS chk_customer_location_level;
ALTER TABLE customer_locations ADD COLUMN room_id UUID;
UPDATE customer_locations SET room_id = location_id WHERE location_level = 'ROOM';
DELETE FROM customer_locations WHERE room_id IS NULL;
ALTER TABLE customer_locations ALTER COLUMN room_id SET NOT NULL;
ALTER TABLE customer_locations
  ADD CONSTRAINT customer_locations_room_id_fkey FOREIGN KEY (room_id) REFERENCES rooms(id);
ALTER TABLE customer_locations DROP COLUMN location_id;
ALTER TABLE customer_locations DROP COLUMN location_level;
