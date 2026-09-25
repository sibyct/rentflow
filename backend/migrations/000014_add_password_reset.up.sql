-- Admin-triggered password reset for staff: mirrors the invite token
-- pattern exactly (see 000012's invite_token_hash comment) — only a
-- SHA-256 hash of the raw reset token is ever stored, the raw token
-- only ever exists in the emailed link. Kept as its own pair of
-- columns rather than reusing invite_token_hash/invite_expires_at
-- because a reset must work on an already-active user without
-- disturbing their status the way accepting an invite does.

ALTER TABLE users
    ADD COLUMN password_reset_token_hash  TEXT,
    ADD COLUMN password_reset_expires_at  TIMESTAMPTZ;

-- One reset link is live at a time; triggering another overwrites it
-- (see StaffRepository.SetPasswordResetToken), so this only ever needs
-- to reject a (vanishingly unlikely) hash collision across users.
CREATE UNIQUE INDEX idx_users_password_reset_token_hash ON users(password_reset_token_hash) WHERE password_reset_token_hash IS NOT NULL;
