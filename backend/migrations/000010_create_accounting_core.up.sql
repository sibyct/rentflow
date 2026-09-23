-- Accounting core: one central ledger (transactions + transaction_payments)
-- that Rent Roll, Expenses and Charges are all *views* over, plus the
-- audit trail, attachments and bank accounts that hang off it.
--
-- All money is BIGINT cents (never NUMERIC/float): sums and balances
-- must not drift. Legacy dollar fields (leases.monthly_rent, ...) are
-- converted with ROUND(x*100) at the single point they enter the ledger.
--
-- Status is never stored. "Paid/Partial/Late/Unpaid" (income) and
-- "Unpaid/Paid/Overdue" (expense) are derived at read time from
-- transaction_payments + due_on, same rationale as leases.status vs
-- DisplayStatus: a stored "late" goes stale the day nobody re-saves it.
--
-- Ledger rows are voided (voided_at), never deleted, so the audit trail
-- and any statement built from them stay reconstructible. Foreign keys
-- to properties/units/leases are RESTRICT for the same reason: history
-- must be voided or the lease terminated instead of silently cascaded
-- away.

CREATE TABLE attachments (
    id            UUID PRIMARY KEY,
    owner_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    object_key    TEXT NOT NULL UNIQUE,
    filename      TEXT NOT NULL,
    content_type  TEXT NOT NULL,
    size_bytes    BIGINT NOT NULL,
    status        TEXT NOT NULL DEFAULT 'pending',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT attachments_status_check CHECK (status IN ('pending', 'ready')),
    CONSTRAINT attachments_size_check CHECK (size_bytes >= 0)
);

CREATE INDEX idx_attachments_owner_id ON attachments(owner_id);

-- provider/external_id are the slot-in point for a future bank-feed
-- integration (e.g. Plaid); v1 is provider = 'manual' with a manually
-- maintained balance.
CREATE TABLE bank_accounts (
    id                   UUID PRIMARY KEY,
    owner_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    nickname             TEXT NOT NULL,
    bank_name            TEXT NOT NULL DEFAULT '',
    account_type         TEXT NOT NULL,
    property_id          UUID REFERENCES properties(id) ON DELETE SET NULL,
    balance_cents        BIGINT NOT NULL DEFAULT 0,
    balance_as_of        DATE,
    last4                TEXT NOT NULL DEFAULT '',
    provider             TEXT NOT NULL DEFAULT 'manual',
    external_account_id  TEXT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT bank_accounts_type_check CHECK (account_type IN ('operating', 'security_deposit_trust'))
);

CREATE INDEX idx_bank_accounts_owner_id ON bank_accounts(owner_id);

CREATE TABLE transactions (
    id                     UUID PRIMARY KEY,
    owner_id               UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    property_id            UUID NOT NULL REFERENCES properties(id) ON DELETE RESTRICT,
    unit_id                UUID REFERENCES units(id) ON DELETE RESTRICT,
    kind                   TEXT NOT NULL,
    type                   TEXT NOT NULL,
    charge_type            TEXT,
    category               TEXT,
    amount_cents           BIGINT NOT NULL,
    incurred_on            DATE NOT NULL,
    due_on                 DATE,
    period                 DATE,
    lease_id               UUID REFERENCES leases(id) ON DELETE RESTRICT,
    work_order_id          UUID REFERENCES work_orders(id) ON DELETE SET NULL,
    vendor_id              UUID REFERENCES vendors(id) ON DELETE SET NULL,
    vendor_name            TEXT NOT NULL DEFAULT '',
    description            TEXT NOT NULL DEFAULT '',
    tax_deductible         BOOLEAN NOT NULL DEFAULT false,
    is_recurring           BOOLEAN NOT NULL DEFAULT false,
    recurrence_frequency   TEXT,
    recurrence_parent_id   UUID REFERENCES transactions(id) ON DELETE SET NULL,
    attachment_id          UUID REFERENCES attachments(id) ON DELETE SET NULL,
    source                 TEXT NOT NULL DEFAULT 'manual',
    voided_at              TIMESTAMPTZ,
    created_by             UUID NOT NULL REFERENCES users(id),
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_by             UUID REFERENCES users(id),
    updated_at             TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT transactions_kind_check CHECK (kind IN ('income', 'expense')),
    CONSTRAINT transactions_type_check CHECK (type IN ('rent', 'late_fee', 'charge', 'expense')),
    CONSTRAINT transactions_kind_type_check CHECK ((kind = 'expense') = (type = 'expense')),
    CONSTRAINT transactions_charge_type_check CHECK (
        charge_type IS NULL OR charge_type IN ('utility_rebill', 'damage', 'amenity', 'other')
    ),
    CONSTRAINT transactions_category_check CHECK (
        category IS NULL OR category IN (
            'repairs', 'utilities', 'insurance', 'property_tax', 'management_fee',
            'landscaping', 'supplies', 'legal', 'mortgage', 'other'
        )
    ),
    CONSTRAINT transactions_recurrence_check CHECK (
        recurrence_frequency IS NULL OR recurrence_frequency IN ('monthly', 'quarterly', 'yearly')
    ),
    CONSTRAINT transactions_source_check CHECK (
        source IN ('manual', 'auto_rent', 'auto_late_fee', 'work_order', 'recurring')
    ),
    CONSTRAINT transactions_amount_check CHECK (amount_cents > 0),
    -- Receivables always have a due date; an expense's due date is optional
    -- (an expense with none simply never becomes "overdue").
    CONSTRAINT transactions_income_due_check CHECK (kind = 'expense' OR due_on IS NOT NULL),
    -- Rent and late fees are always tied to a lease and a billing month.
    CONSTRAINT transactions_lease_link_check CHECK (
        type NOT IN ('rent', 'late_fee') OR (lease_id IS NOT NULL AND period IS NOT NULL)
    )
);

CREATE INDEX idx_transactions_owner_property ON transactions(owner_id, property_id, incurred_on);
CREATE INDEX idx_transactions_lease_period ON transactions(lease_id, period);
CREATE INDEX idx_transactions_work_order ON transactions(work_order_id);

-- Idempotent generation: the worker, the manual "Generate" button and any
-- lazy trigger can all run for the same month without creating a second
-- rent or late-fee row for a lease.
--
-- Deliberately NOT restricted to un-voided rows: a voided rent/late-fee
-- row is a tombstone ("waived for that month") that stops the generator
-- from quietly recreating it on the next run.
CREATE UNIQUE INDEX idx_transactions_one_rent_per_lease_period
    ON transactions(lease_id, period, type)
    WHERE type IN ('rent', 'late_fee');

-- A work order produces at most one auto-created expense (voided ones
-- included, for the same tombstone reason).
CREATE UNIQUE INDEX idx_transactions_one_expense_per_work_order
    ON transactions(work_order_id)
    WHERE source = 'work_order';

-- One occurrence per recurring template per incurred date.
CREATE UNIQUE INDEX idx_transactions_one_recurrence_per_date
    ON transactions(recurrence_parent_id, incurred_on)
    WHERE recurrence_parent_id IS NOT NULL;

-- One settlement of a ledger row, in either direction (rent received, or
-- an expense paid out). A row can have several (partial payments).
CREATE TABLE transaction_payments (
    id               UUID PRIMARY KEY,
    transaction_id   UUID NOT NULL REFERENCES transactions(id) ON DELETE CASCADE,
    amount_cents     BIGINT NOT NULL,
    paid_on          DATE NOT NULL,
    method           TEXT NOT NULL DEFAULT '',
    reference        TEXT NOT NULL DEFAULT '',
    bank_account_id  UUID REFERENCES bank_accounts(id) ON DELETE SET NULL,
    voided_at        TIMESTAMPTZ,
    recorded_by      UUID NOT NULL REFERENCES users(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT transaction_payments_amount_check CHECK (amount_cents > 0)
);

CREATE INDEX idx_transaction_payments_transaction ON transaction_payments(transaction_id);
CREATE INDEX idx_transaction_payments_paid_on ON transaction_payments(paid_on);

-- Per-account late-fee rule. A lease's own late_fee_amount /
-- late_fee_grace_days / rent_due_day (already on leases) override these
-- when set. late_fee_value is cents when late_fee_kind = 'flat' and
-- basis points of the month's rent when 'percent'; 0 disables late fees.
CREATE TABLE accounting_settings (
    owner_id             UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    late_fee_kind        TEXT NOT NULL DEFAULT 'flat',
    late_fee_value       BIGINT NOT NULL DEFAULT 0,
    grace_days           INTEGER NOT NULL DEFAULT 5,
    default_rent_due_day INTEGER NOT NULL DEFAULT 1,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT accounting_settings_kind_check CHECK (late_fee_kind IN ('flat', 'percent')),
    CONSTRAINT accounting_settings_value_check CHECK (late_fee_value >= 0),
    CONSTRAINT accounting_settings_grace_check CHECK (grace_days >= 0),
    CONSTRAINT accounting_settings_due_day_check CHECK (default_rent_due_day BETWEEN 1 AND 31)
);

-- Change log for every accounting entity (transactions, payments, bank
-- accounts, deposits, statements). Written in the same DB transaction as
-- the change it describes. changes is {"field": {"old": ..., "new": ...}}.
CREATE TABLE accounting_audit_log (
    id           UUID PRIMARY KEY,
    owner_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    entity_type  TEXT NOT NULL,
    entity_id    UUID NOT NULL,
    actor_id     UUID NOT NULL REFERENCES users(id),
    action       TEXT NOT NULL,
    changes      JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_accounting_audit_entity ON accounting_audit_log(entity_type, entity_id, created_at);
