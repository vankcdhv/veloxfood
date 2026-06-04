-- Enable accent-folding and trigram extensions (idempotent).
CREATE EXTENSION IF NOT EXISTS unaccent;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- IMMUTABLE wrapper required to use unaccent() inside an expression index.
-- unaccent() itself is STABLE, which is insufficient for index expressions.
CREATE OR REPLACE FUNCTION f_unaccent(text)
    RETURNS text
    LANGUAGE sql
    IMMUTABLE PARALLEL SAFE STRICT
AS $$
    SELECT public.unaccent('public.unaccent', $1)
$$;

-- GIN trigram index on the accent-folded, lower-cased name for fast ILIKE + similarity queries.
CREATE INDEX IF NOT EXISTS idx_menu_items_name_trgm
    ON menu_items
    USING gin (f_unaccent(lower(name)) gin_trgm_ops);
