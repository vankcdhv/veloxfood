DROP INDEX IF EXISTS idx_menu_items_name_trgm;
DROP FUNCTION IF EXISTS f_unaccent(text);
-- Extensions left in place intentionally (shared, removing may break other users).
