-- A Lease is a time-bound contract binding one or more residents to a
-- unit. There is no Residents entity yet (see the Units feature's
-- tenant_name for the same precedent), so parties are freeform text,
-- not a foreign key — primary_resident_name/co_residents are display
-- labels a manager types in, not links to real records.
--
-- status stores only the three states a manager actually sets by
-- taking an action: 'draft' (not yet finalized), 'active' (the normal
-- lifecycle), 'terminated' (ended early). The richer display states a
-- lease list needs — upcoming / active / expiring_soon / expired — are
-- derived from start_date/end_date at read time (see
-- domain.Lease.DisplayStatus), not stored: a stored "expiring_soon"
-- would silently go stale the day nobody happens to re-save the row.
--
-- There is also no documents/e-signature/payments subsystem behind
-- this table: `signed` is a manual flag a manager checks once they have
-- a signed copy by whatever means, not the result of a real e-signature
-- integration, and there is no balance_due column — that would be a
-- number nothing keeps up to date without a real payments ledger.

CREATE TABLE leases (
    id                       UUID PRIMARY KEY,
    unit_id                  UUID NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    lease_type               TEXT NOT NULL,
    status                   TEXT NOT NULL DEFAULT 'active',
    start_date               DATE NOT NULL,
    end_date                 DATE,
    move_in_date             DATE,
    move_out_date            DATE,
    monthly_rent             NUMERIC(12,2) NOT NULL,
    security_deposit         NUMERIC(12,2),
    deposit_status           TEXT,
    rent_due_day             INTEGER,
    late_fee_amount          NUMERIC(12,2),
    late_fee_grace_days      INTEGER,
    primary_resident_name    TEXT NOT NULL,
    co_residents             TEXT[] NOT NULL DEFAULT '{}',
    emergency_contact        TEXT NOT NULL DEFAULT '',
    renewal_status           TEXT NOT NULL DEFAULT 'not_started',
    termination_reason       TEXT,
    termination_notice_date  DATE,
    signed                   BOOLEAN NOT NULL DEFAULT false,
    signed_date              DATE,
    notes                    TEXT NOT NULL DEFAULT '',
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT leases_type_check CHECK (lease_type IN ('fixed', 'month_to_month')),
    CONSTRAINT leases_status_check CHECK (status IN ('draft', 'active', 'terminated')),
    CONSTRAINT leases_deposit_status_check CHECK (deposit_status IS NULL OR deposit_status IN ('held', 'partially_returned', 'returned', 'forfeited')),
    CONSTRAINT leases_renewal_status_check CHECK (renewal_status IN ('not_started', 'offered', 'accepted', 'declined')),
    CONSTRAINT leases_termination_reason_check CHECK (termination_reason IS NULL OR termination_reason IN ('non_renewal', 'eviction', 'mutual', 'resident_notice', 'other')),
    CONSTRAINT leases_rent_due_day_check CHECK (rent_due_day IS NULL OR (rent_due_day BETWEEN 1 AND 31)),
    CONSTRAINT leases_late_fee_grace_days_check CHECK (late_fee_grace_days IS NULL OR late_fee_grace_days >= 0),
    CONSTRAINT leases_monthly_rent_check CHECK (monthly_rent >= 0),
    CONSTRAINT leases_security_deposit_check CHECK (security_deposit IS NULL OR security_deposit >= 0),
    CONSTRAINT leases_late_fee_amount_check CHECK (late_fee_amount IS NULL OR late_fee_amount >= 0),
    CONSTRAINT leases_dates_check CHECK (end_date IS NULL OR end_date >= start_date),
    -- A fixed-term lease needs a known end; month-to-month runs until terminated.
    CONSTRAINT leases_fixed_needs_end_date CHECK (lease_type <> 'fixed' OR end_date IS NOT NULL)
);

CREATE INDEX idx_leases_unit_id ON leases(unit_id);
CREATE INDEX idx_leases_unit_id_status ON leases(unit_id, status);

-- A unit can only be genuinely leased to one active tenancy at a time;
-- draft and terminated rows are exempt so lease history and drafts
-- don't collide with this. Enforced here (not just app-level) since a
-- race between two concurrent "activate this lease" requests is a real
-- data-integrity risk, not just a UX nicety.
CREATE UNIQUE INDEX idx_leases_one_active_per_unit ON leases(unit_id) WHERE status = 'active';
