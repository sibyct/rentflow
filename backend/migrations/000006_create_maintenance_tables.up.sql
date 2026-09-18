-- Maintenance: work orders, their activity/status history, and
-- recurring maintenance rules that generate new work orders on a
-- schedule.
--
-- Like Leases, there is no Vendor/Staff/Tenant entity yet, so who
-- reported a work order and who it's assigned to are freeform text
-- (reported_by/assigned_to), not foreign keys — same "don't fabricate a
-- link to something that doesn't exist" policy as
-- leases.primary_resident_name and units.tenant_name. There is also no
-- file-storage feature anywhere in this codebase, so photo_link and
-- invoice_link are freeform URL text a manager can paste a link into,
-- not real uploads. And since an account has exactly one user (no
-- team/staff-members feature), activity entries don't track "who" —
-- there is only ever one possible answer, so a column for it would be
-- redundant, not informative.

-- Created before work_orders because work_orders references it.
CREATE TABLE maintenance_recurring_rules (
    id                  UUID PRIMARY KEY,
    property_id         UUID NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    unit_id             UUID REFERENCES units(id) ON DELETE SET NULL,
    title               TEXT NOT NULL,
    description         TEXT NOT NULL DEFAULT '',
    category            TEXT NOT NULL,
    frequency_interval  INTEGER NOT NULL,
    frequency_unit      TEXT NOT NULL,
    next_due_date       DATE NOT NULL,
    active              BOOLEAN NOT NULL DEFAULT true,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT maintenance_recurring_rules_category_check CHECK (category IN ('plumbing', 'electrical', 'hvac', 'appliance', 'pest_control', 'general', 'other')),
    CONSTRAINT maintenance_recurring_rules_frequency_unit_check CHECK (frequency_unit IN ('days', 'weeks', 'months')),
    CONSTRAINT maintenance_recurring_rules_frequency_interval_check CHECK (frequency_interval > 0)
);

CREATE INDEX idx_maintenance_recurring_rules_property_id ON maintenance_recurring_rules(property_id);

CREATE TABLE work_orders (
    id                   UUID PRIMARY KEY,
    property_id          UUID NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    unit_id              UUID REFERENCES units(id) ON DELETE SET NULL,
    title                TEXT NOT NULL,
    description          TEXT NOT NULL DEFAULT '',
    category             TEXT NOT NULL,
    priority             TEXT NOT NULL DEFAULT 'medium',
    status               TEXT NOT NULL DEFAULT 'new',
    reported_by          TEXT NOT NULL DEFAULT '',
    reported_by_contact  TEXT NOT NULL DEFAULT '',
    assigned_to          TEXT NOT NULL DEFAULT '',
    assigned_to_contact  TEXT NOT NULL DEFAULT '',
    access_instructions  TEXT NOT NULL DEFAULT '',
    scheduled_start      TIMESTAMPTZ,
    scheduled_end        TIMESTAMPTZ,
    due_date             DATE,
    estimated_cost       NUMERIC(12,2),
    actual_cost          NUMERIC(12,2),
    photo_link           TEXT NOT NULL DEFAULT '',
    invoice_link         TEXT NOT NULL DEFAULT '',
    internal_notes       TEXT NOT NULL DEFAULT '',
    recurring_rule_id    UUID REFERENCES maintenance_recurring_rules(id) ON DELETE SET NULL,
    completed_at         TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT work_orders_category_check CHECK (category IN ('plumbing', 'electrical', 'hvac', 'appliance', 'pest_control', 'general', 'other')),
    CONSTRAINT work_orders_priority_check CHECK (priority IN ('low', 'medium', 'high', 'emergency')),
    CONSTRAINT work_orders_status_check CHECK (status IN ('new', 'assigned', 'in_progress', 'on_hold', 'completed', 'cancelled')),
    CONSTRAINT work_orders_estimated_cost_check CHECK (estimated_cost IS NULL OR estimated_cost >= 0),
    CONSTRAINT work_orders_actual_cost_check CHECK (actual_cost IS NULL OR actual_cost >= 0)
);

CREATE INDEX idx_work_orders_property_id ON work_orders(property_id);
CREATE INDEX idx_work_orders_unit_id ON work_orders(unit_id);
CREATE INDEX idx_work_orders_status ON work_orders(status);
CREATE INDEX idx_work_orders_recurring_rule_id ON work_orders(recurring_rule_id);

-- Unifies three things the spec asked for into one timeline instead of
-- three separate tables: internal notes (kind='note',
-- visibility='internal'), a tenant-facing update thread (kind='note',
-- visibility='tenant_visible' — there's no tenant portal to actually
-- show these in yet, so this only labels intent), and the auto-inserted
-- status history (kind='status_change', written by the service
-- whenever a work order's status changes, never by a client directly).
CREATE TABLE maintenance_activity (
    id             UUID PRIMARY KEY,
    work_order_id  UUID NOT NULL REFERENCES work_orders(id) ON DELETE CASCADE,
    kind           TEXT NOT NULL,
    visibility     TEXT NOT NULL DEFAULT 'internal',
    message        TEXT NOT NULL DEFAULT '',
    old_value      TEXT,
    new_value      TEXT,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT maintenance_activity_kind_check CHECK (kind IN ('note', 'status_change')),
    CONSTRAINT maintenance_activity_visibility_check CHECK (visibility IN ('internal', 'tenant_visible'))
);

CREATE INDEX idx_maintenance_activity_work_order_id ON maintenance_activity(work_order_id);
