-- Revert the backfilled grants that brought SUPER_ADMIN up to the full set.
DELETE FROM role_permissions rp
USING roles r, permissions p
WHERE rp.role_id = r.id
  AND rp.permission_id = p.id
  AND r.code = 'SUPER_ADMIN'
  AND p.code IN ('store.approve', 'store.manage', 'SHIPPER');
