-- Property owners: the real-world people/entities a managed property is
-- managed *for*, distinct from users (the logged-in manager account —
-- properties.owner_id). Owner Statements are generated per property
-- and addressed to one of these.
--
-- Named property_owners (not owners) and scoped by owner_id like every
-- other table, because "owner" already means "the account that owns this
-- row" throughout the codebase; a second meaning would make every
-- ownership check ambiguous to read.
--
-- properties.owner_name (freeform text) is left in place as the fallback
-- for properties nobody has linked to a record yet — the same
-- "real nullable FK alongside the freeform fallback" pattern Vendors
-- used for work_orders.assigned_to.
--
-- management_fee_bps is basis points (100 = 1.00%) so the fee is an
-- integer like every other money-ish number in the ledger.

CREATE TABLE property_owners (
    id                  UUID PRIMARY KEY,
    owner_id            UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name                TEXT NOT NULL,
    email               TEXT NOT NULL DEFAULT '',
    phone               TEXT NOT NULL DEFAULT '',
    management_fee_bps  INTEGER NOT NULL DEFAULT 0,
    auto_statements     BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT property_owners_fee_check CHECK (management_fee_bps BETWEEN 0 AND 10000)
);

CREATE INDEX idx_property_owners_owner_id ON property_owners(owner_id);

ALTER TABLE properties
    ADD COLUMN property_owner_id UUID REFERENCES property_owners(id) ON DELETE SET NULL;

CREATE INDEX idx_properties_property_owner_id ON properties(property_owner_id);
