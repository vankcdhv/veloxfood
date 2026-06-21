CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =====================
-- RBAC foundation
-- =====================

CREATE TABLE IF NOT EXISTS roles (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code        VARCHAR(60) UNIQUE NOT NULL,
  name        VARCHAR(100) NOT NULL,
  description TEXT,
  scope_type  VARCHAR(10) NOT NULL DEFAULT 'global'
              CHECK (scope_type IN ('global','vendor')),
  is_system   BOOL NOT NULL DEFAULT FALSE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS permissions (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  code        VARCHAR(80) UNIQUE NOT NULL,
  name        VARCHAR(100) NOT NULL,
  description TEXT,
  resource    VARCHAR(40) NOT NULL,
  action      VARCHAR(40) NOT NULL,
  is_system   BOOL NOT NULL DEFAULT FALSE,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS role_permissions (
  role_id       UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  permission_id UUID NOT NULL REFERENCES permissions(id) ON DELETE CASCADE,
  granted_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  granted_by    UUID,
  PRIMARY KEY (role_id, permission_id)
);
CREATE INDEX IF NOT EXISTS idx_role_permissions_perm ON role_permissions(permission_id);

-- =====================
-- Core user tables
-- =====================

CREATE TABLE IF NOT EXISTS users (
  id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  email                 VARCHAR(255) UNIQUE,
  phone                 VARCHAR(50) UNIQUE,
  password_hash         VARCHAR(255),
  status                VARCHAR(20) NOT NULL DEFAULT 'pending'
                        CHECK (status IN ('pending','active','suspended','deactivated')),
  full_name             VARCHAR(255) NOT NULL,
  dob                   DATE,
  gender                VARCHAR(10) CHECK (gender IN ('male','female','other')),
  avatar_url            TEXT,
  failed_login_attempts INT NOT NULL DEFAULT 0,
  locked_until          TIMESTAMPTZ,
  created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_users_email  ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_phone  ON users(phone);
CREATE INDEX IF NOT EXISTS idx_users_status ON users(status);

CREATE TABLE IF NOT EXISTS identities (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  provider         VARCHAR(20) NOT NULL
                   CHECK (provider IN ('school_sso','google','local','guest')),
  external_subject VARCHAR(255) NOT NULL,
  raw_profile      JSONB,
  created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (provider, external_subject)
);
CREATE INDEX IF NOT EXISTS idx_identities_user_id ON identities(user_id);

CREATE TABLE IF NOT EXISTS user_roles (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  role_id    UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
  scope_type VARCHAR(10) NOT NULL CHECK (scope_type IN ('global','vendor')),
  scope_id   UUID,
  granted_by UUID REFERENCES users(id) ON DELETE SET NULL,
  granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  expires_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_roles_global ON user_roles(user_id, role_id)
  WHERE scope_type = 'global';
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_roles_vendor ON user_roles(user_id, role_id, scope_id)
  WHERE scope_type = 'vendor';
CREATE INDEX IF NOT EXISTS idx_user_roles_lookup ON user_roles(user_id, scope_type, scope_id);

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

CREATE TABLE IF NOT EXISTS vendor_memberships (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  vendor_id      UUID NOT NULL,
  role_in_vendor VARCHAR(20) NOT NULL
                 CHECK (role_in_vendor IN ('OWNER','MANAGER','KITCHEN','CASHIER')),
  status         VARCHAR(20) NOT NULL DEFAULT 'invited'
                 CHECK (status IN ('invited','active','left')),
  invited_by     UUID REFERENCES users(id) ON DELETE SET NULL,
  invited_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  joined_at      TIMESTAMPTZ,
  UNIQUE (user_id, vendor_id)
);
CREATE INDEX IF NOT EXISTS idx_vendor_memberships_vendor ON vendor_memberships(vendor_id);

CREATE TABLE IF NOT EXISTS invitations (
  id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  vendor_id           UUID NOT NULL,
  email               VARCHAR(255),
  phone               VARCHAR(50),
  role_in_vendor      VARCHAR(20) NOT NULL
                      CHECK (role_in_vendor IN ('OWNER','MANAGER','KITCHEN','CASHIER')),
  token_hash          VARCHAR(128) NOT NULL UNIQUE,
  expires_at          TIMESTAMPTZ NOT NULL,
  status              VARCHAR(20) NOT NULL DEFAULT 'pending'
                      CHECK (status IN ('pending','accepted','expired','revoked')),
  accepted_by_user_id UUID REFERENCES users(id) ON DELETE SET NULL,
  created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_invitations_vendor ON invitations(vendor_id, status);
CREATE INDEX IF NOT EXISTS idx_invitations_token  ON invitations(token_hash);

CREATE TABLE IF NOT EXISTS card_identifiers (
  id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  kind         VARCHAR(10) NOT NULL CHECK (kind IN ('rfid','face')),
  identifier   TEXT NOT NULL,
  issued_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  revoked_at   TIMESTAMPTZ,
  last_used_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_card_active ON card_identifiers(kind, identifier)
  WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_card_user ON card_identifiers(user_id);

CREATE TABLE IF NOT EXISTS password_reset_tokens (
  id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  token_hash VARCHAR(128) NOT NULL UNIQUE,
  expires_at TIMESTAMPTZ NOT NULL,
  used_at    TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS otp_codes (
  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id     UUID REFERENCES users(id) ON DELETE CASCADE,
  purpose     VARCHAR(30) NOT NULL
              CHECK (purpose IN ('login','register_verify','password_reset','phone_verify')),
  destination VARCHAR(255) NOT NULL,
  code_hash   VARCHAR(128) NOT NULL,
  expires_at  TIMESTAMPTZ NOT NULL,
  used_at     TIMESTAMPTZ,
  attempts    INT NOT NULL DEFAULT 0,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_otp_lookup ON otp_codes(destination, purpose, used_at);

-- =====================
-- Seed: system roles (9)
-- =====================
INSERT INTO roles (id, code, name, scope_type, is_system) VALUES
  (gen_random_uuid(), 'SUPER_ADMIN',         'Super Admin',          'global', TRUE),
  (gen_random_uuid(), 'SCHOOL_ADMIN',         'School Admin',         'global', TRUE),
  -- Fixed id: the web client embeds this UUID (ROLE_IDS.vendorOwner) to filter
  -- store-owner accounts in the admin "create store" picker. Keep in sync.
  ('754a8e4d-4e3b-4241-ba40-619f411aba3b', 'VENDOR_OWNER', 'Vendor Owner', 'vendor', TRUE),
  (gen_random_uuid(), 'VENDOR_STAFF_KITCHEN', 'Vendor Staff Kitchen', 'vendor', TRUE),
  (gen_random_uuid(), 'VENDOR_STAFF_CASHIER', 'Vendor Staff Cashier', 'vendor', TRUE),
  (gen_random_uuid(), 'STUDENT',              'Student',              'global', TRUE),
  (gen_random_uuid(), 'FACULTY',              'Faculty',              'global', TRUE),
  (gen_random_uuid(), 'GUEST',                'Guest',                'global', TRUE),
  (gen_random_uuid(), 'SHIPPER',              'Shipper',              'global', TRUE)
ON CONFLICT (code) DO NOTHING;

-- =====================
-- Seed: system permissions (26)
-- =====================
INSERT INTO permissions (id, code, name, resource, action, is_system) VALUES
  (gen_random_uuid(), 'user.read',              'Read User',            'user',         'read',              TRUE),
  (gen_random_uuid(), 'user.create',            'Create User',          'user',         'create',            TRUE),
  (gen_random_uuid(), 'user.update',            'Update User',          'user',         'update',            TRUE),
  (gen_random_uuid(), 'user.suspend',           'Suspend User',         'user',         'suspend',           TRUE),
  (gen_random_uuid(), 'user.assign_role',       'Assign Role to User',  'user',         'assign_role',       TRUE),
  (gen_random_uuid(), 'vendor.approve',         'Approve Vendor',       'vendor',       'approve',           TRUE),
  (gen_random_uuid(), 'vendor.reject',          'Reject Vendor',        'vendor',       'reject',            TRUE),
  (gen_random_uuid(), 'vendor.suspend',         'Suspend Vendor',       'vendor',       'suspend',           TRUE),
  (gen_random_uuid(), 'vendor_member.invite',   'Invite Vendor Member', 'vendor_member','invite',            TRUE),
  (gen_random_uuid(), 'vendor_member.remove',   'Remove Vendor Member', 'vendor_member','remove',            TRUE),
  (gen_random_uuid(), 'vendor_member.list',     'List Vendor Members',  'vendor_member','list',              TRUE),
  (gen_random_uuid(), 'role.read',              'Read Role',            'role',         'read',              TRUE),
  (gen_random_uuid(), 'role.create',            'Create Role',          'role',         'create',            TRUE),
  (gen_random_uuid(), 'role.update',            'Update Role',          'role',         'update',            TRUE),
  (gen_random_uuid(), 'role.delete',            'Delete Role',          'role',         'delete',            TRUE),
  (gen_random_uuid(), 'role.assign_permission', 'Assign Permission',    'role',         'assign_permission', TRUE),
  (gen_random_uuid(), 'permission.read',        'Read Permission',      'permission',   'read',              TRUE),
  (gen_random_uuid(), 'permission.create',      'Create Permission',    'permission',   'create',            TRUE),
  (gen_random_uuid(), 'permission.update',      'Update Permission',    'permission',   'update',            TRUE),
  (gen_random_uuid(), 'permission.delete',      'Delete Permission',    'permission',   'delete',            TRUE),
  (gen_random_uuid(), 'audit.read',             'Read Audit Log',       'audit',        'read',              TRUE),
  (gen_random_uuid(), 'order.create',           'Create Order',         'order',        'create',            TRUE),
  (gen_random_uuid(), 'order.read',             'Read Order',           'order',        'read',              TRUE),
  (gen_random_uuid(), 'order.update_status',    'Update Order Status',  'order',        'update_status',     TRUE),
  (gen_random_uuid(), 'menu.manage',            'Manage Menu',          'menu',         'manage',            TRUE),
  (gen_random_uuid(), 'payment.refund',         'Process Refund',       'payment',      'refund',            TRUE)
ON CONFLICT (code) DO NOTHING;

-- =====================
-- Seed: default role-permission mapping via CTE
-- =====================
WITH r AS (SELECT id, code FROM roles),
     p AS (SELECT id, code FROM permissions)
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM r, p WHERE r.code = 'SUPER_ADMIN'
UNION ALL
SELECT r.id, p.id FROM r, p WHERE
  r.code = 'SCHOOL_ADMIN' AND p.code IN (
    'vendor.approve','vendor.reject','vendor.suspend',
    'user.suspend','user.assign_role','user.read',
    'role.read','audit.read'
  )
UNION ALL
SELECT r.id, p.id FROM r, p WHERE
  r.code = 'VENDOR_OWNER' AND p.code IN (
    'vendor_member.invite','vendor_member.remove','vendor_member.list',
    'menu.manage','payment.refund'
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
