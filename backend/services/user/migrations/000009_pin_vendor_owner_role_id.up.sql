-- Pin the VENDOR_OWNER role id to the fixed UUID the web client embeds
-- (ROLE_IDS.vendorOwner). On DBs seeded before 000001 used a fixed id, the role
-- holds a random UUID, so the admin "create store" owner picker (which filters
-- users by this exact role_id) finds nobody and store creation is impossible.
-- Realign in place. FKs are ON UPDATE NO ACTION, so drop + re-add around the swap.
-- No-op when the id already matches (e.g. fresh DBs seeded by the updated 000001).
DO $$
DECLARE
  old_id uuid;
  new_id uuid := '754a8e4d-4e3b-4241-ba40-619f411aba3b';
BEGIN
  SELECT id INTO old_id FROM roles WHERE code = 'VENDOR_OWNER';
  IF old_id IS NULL OR old_id = new_id THEN
    RETURN;
  END IF;

  ALTER TABLE role_permissions DROP CONSTRAINT role_permissions_role_id_fkey;
  ALTER TABLE user_roles       DROP CONSTRAINT user_roles_role_id_fkey;

  UPDATE roles            SET id      = new_id WHERE id      = old_id;
  UPDATE role_permissions SET role_id = new_id WHERE role_id = old_id;
  UPDATE user_roles       SET role_id = new_id WHERE role_id = old_id;

  ALTER TABLE role_permissions
    ADD CONSTRAINT role_permissions_role_id_fkey
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE;
  ALTER TABLE user_roles
    ADD CONSTRAINT user_roles_role_id_fkey
    FOREIGN KEY (role_id) REFERENCES roles(id) ON DELETE CASCADE;
END $$;
