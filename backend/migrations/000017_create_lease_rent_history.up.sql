-- Every rent change on an active lease is a dated, reasoned amendment
-- appended here — never an overwrite of leases.monthly_rent. Once a
-- lease has ever gone active, leases.monthly_rent stays frozen at its
-- creation-time value forever; the *effective* rent as of any given
-- date is read-time-derived from the latest row here whose
-- effective_date has arrived (see domain.Lease's doc comment). This
-- mirrors how units.current_rent already gets overridden by an active
-- lease's rent at read time (see unitColumnsForRead) — this table is
-- what makes that same trick correct for the lease side too.
--
-- is_correction exists for the one narrow exception to "effective date
-- must be today or later": fixing a data-entry mistake. It is a
-- separate, explicitly-flagged path, not a backdoor for ordinary
-- backdated changes.

CREATE TABLE lease_rent_history (
    id             UUID PRIMARY KEY,
    lease_id       UUID NOT NULL REFERENCES leases(id) ON DELETE CASCADE,
    amount         NUMERIC(12,2) NOT NULL,
    effective_date DATE NOT NULL,
    reason         TEXT NOT NULL DEFAULT '',
    is_correction  BOOLEAN NOT NULL DEFAULT false,
    created_by     UUID NOT NULL REFERENCES users(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT lease_rent_history_amount_check CHECK (amount >= 0)
);

CREATE INDEX idx_lease_rent_history_lease_id ON lease_rent_history(lease_id, effective_date DESC);
