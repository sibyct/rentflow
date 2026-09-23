-- Staff accounts: lets more than one login share one portfolio.
--
-- account_owner_id NULL means this users row IS an account — the
-- existing one-user-per-account model every other table's owner_id FK
-- already assumes, completely unchanged for anyone who never invites
-- staff. account_owner_id set means this row is staff invited by that
-- account; properties.owner_id and every other resource's owner_id
-- keeps pointing at the ROOT row (see domain.User.AccountID()) — staff
-- never get their own portfolio, they act within the root's.
--
-- staff_role is intentionally a separate column from the existing
-- (legacy, coarse) `role` column rather than extending it: `role` has
-- meant admin|manager|tenant since the very first migration and nothing
-- here changes what it means for a root account. A root row never has
-- staff_role set — it's implicitly "admin" over its own account, which
-- is why there's no "admin" row to accidentally deactivate: the
-- last-active-admin safeguard only ever needs to reason about staff
-- rows (see StaffService), not the root.
--
-- status is the three states someone actually sets by taking an action
-- (active, invited, deactivated) — same "richer display state is
-- derived, not stored" rationale as leases.status/DisplayStatus:
-- "invite expired" is computed from invite_expires_at at read time
-- (see domain.User.DisplayStatus), not a fourth stored value that could
-- go stale.
--
-- invite_token_hash stores only a SHA-256 hash of the raw token that
-- goes out in the invite email/link — the same "never store the secret
-- itself" pattern a password hash already follows here.

ALTER TABLE users
    ADD COLUMN name               TEXT NOT NULL DEFAULT '',
    ADD COLUMN account_owner_id   UUID REFERENCES users(id) ON DELETE CASCADE,
    ADD COLUMN staff_role         TEXT,
    ADD COLUMN status             TEXT NOT NULL DEFAULT 'active',
    ADD COLUMN all_properties     BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN invited_by         UUID REFERENCES users(id) ON DELETE SET NULL,
    ADD COLUMN invited_at         TIMESTAMPTZ,
    ADD COLUMN invite_expires_at  TIMESTAMPTZ,
    ADD COLUMN invite_token_hash  TEXT,
    ADD COLUMN last_login_at      TIMESTAMPTZ;

ALTER TABLE users
    ADD CONSTRAINT users_staff_role_check CHECK (
        staff_role IS NULL OR staff_role IN ('admin', 'property_manager', 'maintenance_coordinator', 'accountant')
    ),
    ADD CONSTRAINT users_status_check CHECK (status IN ('active', 'invited', 'deactivated')),
    -- A root account (no owner) never carries staff-only fields; a
    -- staff row always has a role assigned.
    ADD CONSTRAINT users_account_owner_consistency CHECK (
        (account_owner_id IS NULL AND staff_role IS NULL) OR (account_owner_id IS NOT NULL AND staff_role IS NOT NULL)
    );

CREATE INDEX idx_users_account_owner_id ON users(account_owner_id);

-- One token is live per invite; re-inviting overwrites it (see
-- StaffRepository.ResendInvite), so this only ever needs to reject a
-- (vanishingly unlikely) hash collision across different users.
CREATE UNIQUE INDEX idx_users_invite_token_hash ON users(invite_token_hash) WHERE invite_token_hash IS NOT NULL;

-- Specific properties a staff member can see, when all_properties is
-- false — same "rows ignored while the boolean is true, enforced in the
-- service layer" pattern vendors.serves_all_properties/vendor_properties
-- already uses.
CREATE TABLE staff_property_access (
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    property_id UUID NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, property_id)
);

CREATE INDEX idx_staff_property_access_property_id ON staff_property_access(property_id);

-- Change log for staff management (invites, role/access changes,
-- deactivate/reactivate) — the account-level counterpart to
-- accounting_audit_log, kept separate since it's about who can log in
-- and act, not about money.
CREATE TABLE account_audit_log (
    id               UUID PRIMARY KEY,
    account_owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    actor_id         UUID NOT NULL REFERENCES users(id),
    action           TEXT NOT NULL,
    subject_user_id  UUID REFERENCES users(id) ON DELETE SET NULL,
    subject_name     TEXT NOT NULL DEFAULT '',
    changes          JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_account_audit_log_account ON account_audit_log(account_owner_id, created_at DESC);
