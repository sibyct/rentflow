-- Audit trail for lease lifecycle events (rent changes, renewal
-- generation, termination) — who, when, before/after. Its own table
-- rather than reusing accounting_audit_log, matching the precedent
-- account_audit_log already set for staff changes: same shape, kept
-- separate because it's about a different subsystem. Keyed directly by
-- lease_id (not a polymorphic entity_type/entity_id pair) since every
-- action in scope always has exactly one lease as its subject.
CREATE TABLE lease_audit_log (
    id         UUID PRIMARY KEY,
    owner_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    lease_id   UUID NOT NULL REFERENCES leases(id) ON DELETE CASCADE,
    actor_id   UUID NOT NULL REFERENCES users(id),
    action     TEXT NOT NULL,
    changes    JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_lease_audit_log_lease ON lease_audit_log(lease_id, created_at DESC);
