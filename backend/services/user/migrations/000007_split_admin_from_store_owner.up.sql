-- Separate the platform-admin identity from the store-owner identity.
-- The operator account previously held both ADMIN (global) and VENDOR_OWNER
-- (vendor) roles. Rename it to the store-owner identity, strip its admin role,
-- and provision dedicated admin + super-admin accounts.
-- Keyed by email so it is robust to per-environment UUIDs. Idempotent.
-- Password for the new accounts: Password123!

-- 1) Rename the operator account to the store-owner identity.
UPDATE users
SET email = 'shop@velox.test', full_name = 'Chủ Cửa Hàng'
WHERE email = 'ops@velox.test';

-- 2) Strip the global ADMIN role from the store-owner (keep vendor-scoped owner roles).
DELETE FROM user_roles ur
USING users u, roles r
WHERE ur.user_id = u.id
  AND ur.role_id = r.id
  AND u.email = 'shop@velox.test'
  AND r.code = 'ADMIN'
  AND ur.scope_type = 'global';

-- 3) Dedicated admin + super-admin accounts (bcrypt of "Password123!").
INSERT INTO users (id, email, password_hash, status, full_name) VALUES
  (gen_random_uuid(), 'admin@velox.test',       '$2a$10$yxV9cqK6U1tNK3fpV68B3e8wvrT81yRlFJhroOAHfOqptJK6jNyba', 'active', 'Quản Trị Viên'),
  (gen_random_uuid(), 'super-admin@velox.test', '$2a$10$yxV9cqK6U1tNK3fpV68B3e8wvrT81yRlFJhroOAHfOqptJK6jNyba', 'active', 'Super Admin')
ON CONFLICT (email) DO NOTHING;

-- 4) Global role grants for the new accounts.
INSERT INTO user_roles (id, user_id, role_id, scope_type, scope_id)
SELECT gen_random_uuid(), u.id, r.id, 'global', NULL::uuid
FROM (VALUES
  ('admin@velox.test',       'ADMIN'),
  ('super-admin@velox.test', 'SUPER_ADMIN')
) AS v(email, role_code)
JOIN users u ON u.email = v.email
JOIN roles r ON r.code = v.role_code
WHERE NOT EXISTS (
  SELECT 1 FROM user_roles ur
  WHERE ur.user_id = u.id AND ur.role_id = r.id AND ur.scope_type = 'global'
);
