DROP TABLE IF EXISTS account_audit_log;
DROP TABLE IF EXISTS staff_property_access;

ALTER TABLE users
    DROP CONSTRAINT IF EXISTS users_account_owner_consistency,
    DROP CONSTRAINT IF EXISTS users_status_check,
    DROP CONSTRAINT IF EXISTS users_staff_role_check;

DROP INDEX IF EXISTS idx_users_invite_token_hash;
DROP INDEX IF EXISTS idx_users_account_owner_id;

ALTER TABLE users
    DROP COLUMN IF EXISTS last_login_at,
    DROP COLUMN IF EXISTS invite_token_hash,
    DROP COLUMN IF EXISTS invite_expires_at,
    DROP COLUMN IF EXISTS invited_at,
    DROP COLUMN IF EXISTS invited_by,
    DROP COLUMN IF EXISTS all_properties,
    DROP COLUMN IF EXISTS status,
    DROP COLUMN IF EXISTS staff_role,
    DROP COLUMN IF EXISTS account_owner_id,
    DROP COLUMN IF EXISTS name;
