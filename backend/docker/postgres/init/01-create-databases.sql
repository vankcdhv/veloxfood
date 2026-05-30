-- Database-per-service provisioning (single Postgres instance, multiple DBs).
-- Runs once on first init of an empty data volume (docker-entrypoint-initdb.d).
-- user_db is created by POSTGRES_DB; create the rest here.
-- NOTE: plain CREATE DATABASE has no IF NOT EXISTS — keep this list disjoint
-- from POSTGRES_DB and re-run only against a fresh volume (make docker-reset).
CREATE DATABASE location_db;
CREATE DATABASE store_db;
CREATE DATABASE order_db;
CREATE DATABASE delivery_db;
CREATE DATABASE payment_db;
CREATE DATABASE promotion_db;
CREATE DATABASE review_db;
CREATE DATABASE notification_db;
CREATE DATABASE reporting_db;
