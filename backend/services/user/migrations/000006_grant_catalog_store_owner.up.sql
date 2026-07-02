-- Grant the seeded operator account VENDOR_OWNER on the 20 catalog vendors.
-- The catalog stores were seeded with owner_user_id = ops, but no matching
-- vendor-scoped role grant existed, so HasVendorPermission(store.manage) returned
-- false and every store-management action (menu, avatar, ship-fee, promotions…)
-- was rejected with 403. This backfills the grants so the owner console works.
--
-- The ops user id and VENDOR_OWNER role id are resolved by email/code rather
-- than hard-coded: seed 000005 generates the user id with gen_random_uuid(),
-- so a fixed UUID here breaks the users(id) FK on any freshly-migrated
-- database. No-ops when either row is absent.
INSERT INTO user_roles (user_id, role_id, scope_type, scope_id)
SELECT u.id, r.id, 'vendor', v.id
FROM users u
JOIN roles r ON r.code = 'VENDOR_OWNER'
CROSS JOIN (VALUES
    ('5eed0000-0000-0000-0000-000000000001'::uuid),
    ('5eed0000-0000-0000-0000-000000000002'::uuid),
    ('5eed0000-0000-0000-0000-000000000003'::uuid),
    ('5eed0000-0000-0000-0000-000000000004'::uuid),
    ('5eed0000-0000-0000-0000-000000000005'::uuid),
    ('5eed0000-0000-0000-0000-000000000006'::uuid),
    ('5eed0000-0000-0000-0000-000000000007'::uuid),
    ('5eed0000-0000-0000-0000-000000000008'::uuid),
    ('5eed0000-0000-0000-0000-000000000009'::uuid),
    ('5eed0000-0000-0000-0000-000000000010'::uuid),
    ('5eed0000-0000-0000-0000-000000000011'::uuid),
    ('5eed0000-0000-0000-0000-000000000012'::uuid),
    ('5eed0000-0000-0000-0000-000000000013'::uuid),
    ('5eed0000-0000-0000-0000-000000000014'::uuid),
    ('5eed0000-0000-0000-0000-000000000015'::uuid),
    ('5eed0000-0000-0000-0000-000000000016'::uuid),
    ('5eed0000-0000-0000-0000-000000000017'::uuid),
    ('5eed0000-0000-0000-0000-000000000018'::uuid),
    ('5eed0000-0000-0000-0000-000000000019'::uuid),
    ('5eed0000-0000-0000-0000-000000000020'::uuid)
) AS v(id)
WHERE u.email = 'ops@velox.test'
  AND NOT EXISTS (
    SELECT 1 FROM user_roles ur
    WHERE ur.user_id = u.id
      AND ur.role_id = r.id
      AND ur.scope_type = 'vendor'
      AND ur.scope_id = v.id
);
