-- Drop in reverse FK order (cascade handles seed data)
DROP TABLE IF EXISTS otp_codes;
DROP TABLE IF EXISTS password_reset_tokens;
DROP TABLE IF EXISTS card_identifiers;
DROP TABLE IF EXISTS invitations;
DROP TABLE IF EXISTS vendor_memberships;
DROP TABLE IF EXISTS faculty_profiles;
DROP TABLE IF EXISTS student_profiles;
DROP TABLE IF EXISTS user_roles;
DROP TABLE IF EXISTS identities;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
-- pgcrypto extension is intentionally kept
