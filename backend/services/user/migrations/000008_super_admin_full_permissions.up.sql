-- SUPER_ADMIN must hold EVERY permission — it outranks ADMIN. The original full
-- grant was applied by hand and never captured in a migration, so on a fresh DB
-- SUPER_ADMIN was missing store.approve / store.manage / SHIPPER. The missing
-- store.approve left the super-admin account unable to load the platform
-- analytics dashboard (the reporting service gates /admin/analytics on it).
-- Grant SUPER_ADMIN all permissions. Idempotent.
INSERT INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id
FROM roles r
CROSS JOIN permissions p
WHERE r.code = 'SUPER_ADMIN'
  AND NOT EXISTS (
    SELECT 1 FROM role_permissions rp
    WHERE rp.role_id = r.id AND rp.permission_id = p.id
  );
