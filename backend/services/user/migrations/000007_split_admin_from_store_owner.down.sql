-- Revert: remove the dedicated admin accounts and restore the combined operator.

-- Drop role grants + the new accounts.
DELETE FROM user_roles ur
USING users u
WHERE ur.user_id = u.id
  AND u.email IN ('admin@velox.test', 'super-admin@velox.test');

DELETE FROM users
WHERE email IN ('admin@velox.test', 'super-admin@velox.test');

-- Restore the operator identity.
UPDATE users
SET email = 'ops@velox.test', full_name = 'Lê Vận Hành'
WHERE email = 'shop@velox.test';

-- Re-grant the global ADMIN role to the operator.
INSERT INTO user_roles (id, user_id, role_id, scope_type, scope_id)
SELECT gen_random_uuid(), u.id, r.id, 'global', NULL::uuid
FROM users u, roles r
WHERE u.email = 'ops@velox.test' AND r.code = 'ADMIN'
  AND NOT EXISTS (
    SELECT 1 FROM user_roles ur
    WHERE ur.user_id = u.id AND ur.role_id = r.id AND ur.scope_type = 'global'
  );
