-- Remove demo accounts + the RBAC rows seeded in the up migration.
DELETE FROM user_roles WHERE user_id IN (
  SELECT id FROM users WHERE email IN ('ops@velox.test', 'cust@velox.test', 'ship@velox.test')
);
DELETE FROM users WHERE email IN ('ops@velox.test', 'cust@velox.test', 'ship@velox.test');
DELETE FROM role_permissions WHERE permission_id IN (
  SELECT id FROM permissions WHERE code IN ('store.approve', 'store.manage', 'SHIPPER')
);
DELETE FROM permissions WHERE code IN ('store.approve', 'store.manage', 'SHIPPER');
