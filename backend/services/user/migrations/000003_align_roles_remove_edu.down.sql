-- Best-effort reversal of 000003 (restores schema + legacy roles; data is not recovered).

-- =========================================================
-- 1. Recreate education-domain tables
-- =========================================================
CREATE TABLE IF NOT EXISTS student_profiles (
  user_id        UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  student_code   VARCHAR(50) UNIQUE NOT NULL,
  faculty        VARCHAR(100),
  class          VARCHAR(50),
  cohort_year    INT,
  allergies      JSONB,
  dormitory_room VARCHAR(50),
  synced_at      TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS faculty_profiles (
  user_id                 UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
  staff_code              VARCHAR(50) UNIQUE NOT NULL,
  department              VARCHAR(100),
  position                VARCHAR(100),
  allow_payroll_deduction BOOLEAN NOT NULL DEFAULT FALSE,
  synced_at               TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS card_identifiers (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind         VARCHAR(10) NOT NULL CHECK (kind IN ('rfid','face')),
  identifier   TEXT NOT NULL,
  issued_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  revoked_at   TIMESTAMPTZ,
  last_used_at TIMESTAMPTZ
);

-- =========================================================
-- 2. Revert role_in_vendor check constraints
-- =========================================================
ALTER TABLE vendor_memberships DROP CONSTRAINT IF EXISTS vendor_memberships_role_in_vendor_check;
ALTER TABLE vendor_memberships ADD CONSTRAINT vendor_memberships_role_in_vendor_check
  CHECK (role_in_vendor IN ('OWNER','MANAGER','KITCHEN','CASHIER'));
ALTER TABLE invitations DROP CONSTRAINT IF EXISTS invitations_role_in_vendor_check;
ALTER TABLE invitations ADD CONSTRAINT invitations_role_in_vendor_check
  CHECK (role_in_vendor IN ('OWNER','MANAGER','KITCHEN','CASHIER'));

-- =========================================================
-- 3. Restore legacy roles
-- =========================================================
UPDATE roles SET code = 'SCHOOL_ADMIN', name = 'School Admin' WHERE code = 'ADMIN';
DELETE FROM role_permissions WHERE role_id IN (SELECT id FROM roles WHERE code IN ('CUSTOMER','VENDOR_STAFF'));
DELETE FROM roles WHERE code IN ('CUSTOMER','VENDOR_STAFF');

INSERT INTO roles (id, code, name, scope_type, is_system) VALUES
  (gen_random_uuid(), 'VENDOR_STAFF_KITCHEN', 'Vendor Staff Kitchen', 'vendor', TRUE),
  (gen_random_uuid(), 'VENDOR_STAFF_CASHIER', 'Vendor Staff Cashier', 'vendor', TRUE),
  (gen_random_uuid(), 'STUDENT',              'Student',              'global', TRUE),
  (gen_random_uuid(), 'FACULTY',              'Faculty',              'global', TRUE),
  (gen_random_uuid(), 'GUEST',                'Guest',                'global', TRUE)
ON CONFLICT (code) DO NOTHING;

-- =========================================================
-- 4. Remove permissions added in 000003
-- =========================================================
DELETE FROM permissions WHERE code IN (
  'system.config','payment.config','admin.manage','shipper.approve','shipper.reject',
  'location.manage','review.moderate','dispute.read','dispute.resolve','payout.read','payout.execute'
);

-- =========================================================
-- 5. Restore original (000001) role-permission mapping
-- =========================================================
DELETE FROM role_permissions;

WITH r AS (SELECT id, code FROM roles),
     p AS (SELECT id, code FROM permissions)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM r, p WHERE r.code = 'SUPER_ADMIN'
UNION ALL
SELECT r.id, p.id FROM r, p WHERE
  r.code = 'SCHOOL_ADMIN' AND p.code IN (
    'vendor.approve','vendor.reject','vendor.suspend',
    'user.suspend','user.assign_role','user.read','role.read','audit.read'
  )
UNION ALL
SELECT r.id, p.id FROM r, p WHERE
  r.code = 'VENDOR_OWNER' AND p.code IN (
    'vendor_member.invite','vendor_member.remove','vendor_member.list','menu.manage','payment.refund'
  )
UNION ALL
SELECT r.id, p.id FROM r, p WHERE
  r.code = 'VENDOR_STAFF_KITCHEN' AND p.code IN ('order.read','order.update_status')
UNION ALL
SELECT r.id, p.id FROM r, p WHERE
  r.code = 'VENDOR_STAFF_CASHIER' AND p.code IN ('order.create','order.read','payment.refund')
UNION ALL
SELECT r.id, p.id FROM r, p WHERE
  r.code IN ('STUDENT','FACULTY','GUEST') AND p.code IN ('order.create','order.read')
ON CONFLICT (role_id, permission_id) DO NOTHING;
