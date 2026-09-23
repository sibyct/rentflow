-- Security deposits, bank reconciliation, owner statements, and the
-- outbound-email / worker bookkeeping tables.
--
-- Deposit status (Held / Partially Refunded / Fully Refunded / Forfeited)
-- is derived at read time from settled_at / refund_cents / forfeited_at
-- plus the itemized deductions, not stored — see domain.SecurityDeposit.

CREATE TABLE security_deposits (
    id                    UUID PRIMARY KEY,
    owner_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    lease_id              UUID NOT NULL UNIQUE REFERENCES leases(id) ON DELETE RESTRICT,
    property_id           UUID NOT NULL REFERENCES properties(id) ON DELETE RESTRICT,
    unit_id               UUID NOT NULL REFERENCES units(id) ON DELETE RESTRICT,
    amount_cents          BIGINT NOT NULL,
    collected_on          DATE NOT NULL,
    held_in_account_id    UUID REFERENCES bank_accounts(id) ON DELETE SET NULL,
    settled_at            TIMESTAMPTZ,
    refund_cents          BIGINT NOT NULL DEFAULT 0,
    refund_method         TEXT NOT NULL DEFAULT '',
    refunded_on           DATE,
    forfeited_at          TIMESTAMPTZ,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT security_deposits_amount_check CHECK (amount_cents > 0),
    CONSTRAINT security_deposits_refund_check CHECK (refund_cents >= 0 AND refund_cents <= amount_cents)
);

CREATE INDEX idx_security_deposits_owner_id ON security_deposits(owner_id);
CREATE INDEX idx_security_deposits_property_id ON security_deposits(property_id);

CREATE TABLE deposit_deductions (
    id             UUID PRIMARY KEY,
    deposit_id     UUID NOT NULL REFERENCES security_deposits(id) ON DELETE CASCADE,
    description    TEXT NOT NULL,
    amount_cents   BIGINT NOT NULL,
    attachment_id  UUID REFERENCES attachments(id) ON DELETE SET NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT deposit_deductions_amount_check CHECK (amount_cents > 0)
);

CREATE INDEX idx_deposit_deductions_deposit_id ON deposit_deductions(deposit_id);

-- Backfill a deposit record for every existing lease that already
-- carries a security_deposit amount. Mapped from the legacy bare-flag
-- leases.deposit_status without inventing numbers: a lease marked
-- 'partially_returned' never recorded how much came back, so it lands
-- settled with refund 0 rather than a made-up figure.
INSERT INTO security_deposits (
    id, owner_id, lease_id, property_id, unit_id, amount_cents, collected_on,
    settled_at, refund_cents, refunded_on, forfeited_at
)
SELECT
    gen_random_uuid(), p.owner_id, l.id, p.id, u.id,
    ROUND(l.security_deposit * 100)::BIGINT,
    l.start_date,
    CASE WHEN l.deposit_status IN ('returned', 'partially_returned') THEN now() END,
    CASE WHEN l.deposit_status = 'returned' THEN ROUND(l.security_deposit * 100)::BIGINT ELSE 0 END,
    CASE WHEN l.deposit_status IN ('returned', 'partially_returned') THEN CURRENT_DATE END,
    CASE WHEN l.deposit_status = 'forfeited' THEN now() END
FROM leases l
JOIN units u ON u.id = l.unit_id
JOIN properties p ON p.id = u.property_id
WHERE l.security_deposit IS NOT NULL AND l.security_deposit > 0;

-- Imported/manual bank-statement lines to reconcile against recorded
-- payments. source/external_id let a future feed (Plaid) write here
-- idempotently; the reconciliation UI only ever reads this table.
CREATE TABLE bank_statement_lines (
    id                   UUID PRIMARY KEY,
    owner_id             UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    bank_account_id      UUID NOT NULL REFERENCES bank_accounts(id) ON DELETE CASCADE,
    posted_on            DATE NOT NULL,
    description          TEXT NOT NULL DEFAULT '',
    amount_cents         BIGINT NOT NULL,
    source               TEXT NOT NULL DEFAULT 'manual',
    external_id          TEXT,
    matched_payment_id   UUID REFERENCES transaction_payments(id) ON DELETE SET NULL,
    matched_at           TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT bank_statement_lines_source_check CHECK (source IN ('manual', 'csv', 'plaid')),
    CONSTRAINT bank_statement_lines_amount_check CHECK (amount_cents <> 0)
);

CREATE INDEX idx_bank_statement_lines_account ON bank_statement_lines(bank_account_id, posted_on);
CREATE UNIQUE INDEX idx_bank_statement_lines_external
    ON bank_statement_lines(bank_account_id, external_id) WHERE external_id IS NOT NULL;
-- A payment can back at most one statement line.
CREATE UNIQUE INDEX idx_bank_statement_lines_payment
    ON bank_statement_lines(matched_payment_id) WHERE matched_payment_id IS NOT NULL;

-- Owner statements are immutable snapshots: everything shown (line items,
-- totals, owner name/fee at generation time) is frozen into snapshot so a
-- later edit to a payment, expense or the owner's fee never changes a
-- statement that has already gone out.
CREATE TABLE owner_statements (
    id                  UUID PRIMARY KEY,
    owner_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    property_owner_id   UUID REFERENCES property_owners(id) ON DELETE SET NULL,
    property_id         UUID NOT NULL REFERENCES properties(id) ON DELETE RESTRICT,
    period_start        DATE NOT NULL,
    period_end          DATE NOT NULL,
    status              TEXT NOT NULL DEFAULT 'draft',
    snapshot            JSONB NOT NULL,
    income_cents        BIGINT NOT NULL,
    expenses_cents      BIGINT NOT NULL,
    fee_cents           BIGINT NOT NULL,
    net_payout_cents    BIGINT NOT NULL,
    pdf_attachment_id   UUID REFERENCES attachments(id) ON DELETE SET NULL,
    generated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at             TIMESTAMPTZ,
    paid_cents          BIGINT,
    paid_on             DATE,
    pay_method          TEXT NOT NULL DEFAULT '',
    pay_reference       TEXT NOT NULL DEFAULT '',

    CONSTRAINT owner_statements_status_check CHECK (status IN ('draft', 'sent', 'paid')),
    CONSTRAINT owner_statements_period_check CHECK (period_end >= period_start),
    CONSTRAINT owner_statements_unique_period UNIQUE (property_id, period_start, period_end)
);

CREATE INDEX idx_owner_statements_owner_id ON owner_statements(owner_id);

-- Outbound email queue. Rows are written by the API and drained by the
-- worker, so a slow or down SMTP server never fails a user request.
CREATE TABLE email_outbox (
    id             UUID PRIMARY KEY,
    owner_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    to_email       TEXT NOT NULL,
    subject        TEXT NOT NULL,
    body_html      TEXT NOT NULL,
    attachment_id  UUID REFERENCES attachments(id) ON DELETE SET NULL,
    statement_id   UUID REFERENCES owner_statements(id) ON DELETE SET NULL,
    status         TEXT NOT NULL DEFAULT 'pending',
    attempts       INTEGER NOT NULL DEFAULT 0,
    last_error     TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    sent_at        TIMESTAMPTZ,

    CONSTRAINT email_outbox_status_check CHECK (status IN ('pending', 'sent', 'failed'))
);

CREATE INDEX idx_email_outbox_pending ON email_outbox(created_at) WHERE status = 'pending';

-- Idempotency ledger for scheduled worker jobs: (job, run_key) — e.g.
-- ('rent-and-late-fees', '2026-09-22') — is inserted before a run so two
-- worker machines (or a restart) never repeat the same day's job.
CREATE TABLE worker_runs (
    job      TEXT NOT NULL,
    run_key  TEXT NOT NULL,
    ran_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (job, run_key)
);
