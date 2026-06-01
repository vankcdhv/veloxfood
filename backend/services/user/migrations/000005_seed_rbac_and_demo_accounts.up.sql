-- Reproducible seed for RBAC rows + demo accounts that were previously inserted
-- by hand. Idempotent: safe to run on an existing dev DB (skips what's present)
-- and fully provisions a fresh DB. Password for all demo accounts: Password123!

-- ── Permissions (checked at runtime by store/delivery/promotion/review) ─────────
INSERT INTO permissions (id, code, name, resource, action, is_system) VALUES
  (gen_random_uuid(), 'store.approve', 'Approve Stores',        'store',    'approve', TRUE),
  (gen_random_uuid(), 'store.manage',  'Manage Store Settings', 'store',    'manage',  TRUE),
  (gen_random_uuid(), 'SHIPPER',       'Shipper Delivery Access','delivery','manage',  TRUE)
ON CONFLICT (code) DO NOTHING;

-- ── Role → permission grants ────────────────────────────────────────────────────
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE (
       (r.code = 'ADMIN'        AND p.code = 'store.approve')
    OR (r.code = 'VENDOR_OWNER' AND p.code = 'store.manage')
    OR (r.code = 'SHIPPER'      AND p.code = 'SHIPPER')
  )
  AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );

-- ── Demo users (bcrypt of "Password123!") ───────────────────────────────────────
INSERT INTO users (id, email, password_hash, status, full_name) VALUES
  (gen_random_uuid(), 'ops@velox.test',  '$2a$10$yxV9cqK6U1tNK3fpV68B3e8wvrT81yRlFJhroOAHfOqptJK6jNyba', 'active', 'Lê Vận Hành'),
  (gen_random_uuid(), 'cust@velox.test', '$2a$10$yxV9cqK6U1tNK3fpV68B3e8wvrT81yRlFJhroOAHfOqptJK6jNyba', 'active', 'Nguyễn Thị Khách'),
  (gen_random_uuid(), 'ship@velox.test', '$2a$10$yxV9cqK6U1tNK3fpV68B3e8wvrT81yRlFJhroOAHfOqptJK6jNyba', 'active', 'Trần Giao Hàng')
ON CONFLICT (email) DO NOTHING;

-- ── User → role assignments (global except ops' vendor-scoped owner role) ────────
INSERT INTO user_roles (id, user_id, role_id, scope_type, scope_id)
SELECT gen_random_uuid(), u.id, r.id, v.scope_type, v.scope_id
FROM (VALUES
  ('ops@velox.test',  'ADMIN',        'global', NULL::uuid),
  ('ops@velox.test',  'VENDOR_OWNER', 'vendor', '33333333-3333-3333-3333-333333333333'::uuid),
  ('cust@velox.test', 'CUSTOMER',     'global', NULL::uuid),
  ('ship@velox.test', 'SHIPPER',      'global', NULL::uuid)
) AS v(email, role_code, scope_type, scope_id)
JOIN users u ON u.email = v.email
JOIN roles r ON r.code = v.role_code
WHERE NOT EXISTS (
  SELECT 1 FROM user_roles ur
  WHERE ur.user_id = u.id AND ur.role_id = r.id
    AND ur.scope_type = v.scope_type
    AND ur.scope_id IS NOT DISTINCT FROM v.scope_id
);
