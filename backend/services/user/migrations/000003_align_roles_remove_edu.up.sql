-- Align User Service with VeloxFood food-delivery domain.
-- Removes education-domain leftovers (student/faculty/card, STUDENT/FACULTY/GUEST/SCHOOL_ADMIN
-- roles) and establishes the standard role taxonomy + least-privilege admin split.

-- =========================================================
-- 1. Drop education-domain tables
-- =========================================================
DROP TABLE IF EXISTS card_identifiers;
DROP TABLE IF EXISTS student_profiles;
DROP TABLE IF EXISTS faculty_profiles;

-- =========================================================
-- 2. Consolidate vendor-staff roles (KITCHEN/CASHIER -> single VENDOR_STAFF)
-- =========================================================
-- New consolidated role.
INSERT INTO roles (id, code, name, scope_type, is_system)
VALUES (gen_random_uuid(), 'VENDOR_STAFF', 'Vendor Staff', 'vendor', TRUE)
ON CONFLICT (code) DO NOTHING;

-- Re-point existing user_roles from the two old staff roles to VENDOR_STAFF.
UPDATE user_roles SET role_id = (SELECT id FROM roles WHERE code = 'VENDOR_STAFF')
WHERE role_id IN (SELECT id FROM roles WHERE code IN ('VENDOR_STAFF_KITCHEN', 'VENDOR_STAFF_CASHIER'));

-- Drop old staff roles (and their permission mappings).
DELETE FROM role_permissions WHERE role_id IN
  (SELECT id FROM roles WHERE code IN ('VENDOR_STAFF_KITCHEN', 'VENDOR_STAFF_CASHIER'));
DELETE FROM roles WHERE code IN ('VENDOR_STAFF_KITCHEN', 'VENDOR_STAFF_CASHIER');

-- =========================================================
-- 3. Rename SCHOOL_ADMIN -> ADMIN, add CUSTOMER, remove STUDENT/FACULTY/GUEST
-- =========================================================
UPDATE roles SET code = 'ADMIN', name = 'Admin' WHERE code = 'SCHOOL_ADMIN';

INSERT INTO roles (id, code, name, scope_type, is_system)
VALUES (gen_random_uuid(), 'CUSTOMER', 'Customer', 'global', TRUE)
ON CONFLICT (code) DO NOTHING;

DELETE FROM user_roles WHERE role_id IN
  (SELECT id FROM roles WHERE code IN ('STUDENT', 'FACULTY', 'GUEST'));
DELETE FROM role_permissions WHERE role_id IN
  (SELECT id FROM roles WHERE code IN ('STUDENT', 'FACULTY', 'GUEST'));
DELETE FROM roles WHERE code IN ('STUDENT', 'FACULTY', 'GUEST');

-- =========================================================
-- 4. Consolidate vendor membership/invitation role_in_vendor to OWNER/STAFF
-- =========================================================
UPDATE vendor_memberships SET role_in_vendor = 'STAFF'
WHERE role_in_vendor IN ('MANAGER', 'KITCHEN', 'CASHIER');
UPDATE invitations SET role_in_vendor = 'STAFF'
WHERE role_in_vendor IN ('MANAGER', 'KITCHEN', 'CASHIER');

ALTER TABLE vendor_memberships DROP CONSTRAINT IF EXISTS vendor_memberships_role_in_vendor_check;
ALTER TABLE vendor_memberships ADD CONSTRAINT vendor_memberships_role_in_vendor_check
  CHECK (role_in_vendor IN ('OWNER', 'STAFF'));
ALTER TABLE invitations DROP CONSTRAINT IF EXISTS invitations_role_in_vendor_check;
ALTER TABLE invitations ADD CONSTRAINT invitations_role_in_vendor_check
  CHECK (role_in_vendor IN ('OWNER', 'STAFF'));

-- =========================================================
-- 5. New permissions (system config split for SUPER_ADMIN + operational ones)
-- =========================================================
INSERT INTO permissions (id, code, name, resource, action, is_system) VALUES
  (gen_random_uuid(), 'system.config',    'Configure System',     'system',  'config',  TRUE),
  (gen_random_uuid(), 'payment.config',   'Configure Payment',    'payment', 'config',  TRUE),
  (gen_random_uuid(), 'admin.manage',     'Manage Admins',        'admin',   'manage',  TRUE),
  (gen_random_uuid(), 'shipper.approve',  'Approve Shipper',      'shipper', 'approve', TRUE),
  (gen_random_uuid(), 'shipper.reject',   'Reject Shipper',       'shipper', 'reject',  TRUE),
  (gen_random_uuid(), 'location.manage',  'Manage Locations',     'location','manage',  TRUE),
  (gen_random_uuid(), 'review.moderate',  'Moderate Reviews',     'review',  'moderate',TRUE),
  (gen_random_uuid(), 'dispute.read',     'Read Disputes',        'dispute', 'read',    TRUE),
  (gen_random_uuid(), 'dispute.resolve',  'Resolve Disputes',     'dispute', 'resolve', TRUE),
  (gen_random_uuid(), 'payout.read',      'Read Payouts',         'payout',  'read',    TRUE),
  (gen_random_uuid(), 'payout.execute',   'Execute Payouts',      'payout',  'execute', TRUE)
ON CONFLICT (code) DO NOTHING;

-- =========================================================
-- 6. Re-seed role_permissions for the new taxonomy (wipe + rebuild)
-- =========================================================
DELETE FROM role_permissions;

WITH r AS (SELECT id, code FROM roles),
     p AS (SELECT id, code FROM permissions)
INSERT INTO role_permissions (role_id, permission_id)
-- SUPER_ADMIN: everything
SELECT r.id, p.id FROM r, p WHERE r.code = 'SUPER_ADMIN'
UNION ALL
-- ADMIN: operational — everything except the SUPER_ADMIN-only sensitive perms
SELECT r.id, p.id FROM r, p WHERE
  r.code = 'ADMIN' AND p.code NOT IN ('system.config', 'payment.config', 'admin.manage')
UNION ALL
-- VENDOR_OWNER
SELECT r.id, p.id FROM r, p WHERE
  r.code = 'VENDOR_OWNER' AND p.code IN (
    'vendor_member.invite','vendor_member.remove','vendor_member.list',
    'menu.manage','payment.refund','order.read','order.update_status'
  )
UNION ALL
-- VENDOR_STAFF
SELECT r.id, p.id FROM r, p WHERE
  r.code = 'VENDOR_STAFF' AND p.code IN ('order.create','order.read','order.update_status')
UNION ALL
-- CUSTOMER
SELECT r.id, p.id FROM r, p WHERE
  r.code = 'CUSTOMER' AND p.code IN ('order.create','order.read')
UNION ALL
-- SHIPPER
SELECT r.id, p.id FROM r, p WHERE
  r.code = 'SHIPPER' AND p.code IN ('order.read')
ON CONFLICT (role_id, permission_id) DO NOTHING;
