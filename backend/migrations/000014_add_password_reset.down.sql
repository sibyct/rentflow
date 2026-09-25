DROP INDEX idx_users_password_reset_token_hash;

ALTER TABLE users
    DROP COLUMN password_reset_token_hash,
    DROP COLUMN password_reset_expires_at;
