-- Vendors: the contractors/service providers who get assigned to work
-- orders. Unlike Units/Leases, a vendor belongs directly to the owner
-- (like Properties) rather than being scoped through a property — a
-- vendor typically serves many properties, not one.
--
-- categories reuses the same trade values as work_orders.category
-- (plumbing, electrical, ...) so a vendor's trades line up 1:1 with the
-- kind of work order they can be matched to (see the Vendor Selector).
--
-- Like every other freeform-link field this session, coi_link and
-- tax_doc_link are pasted URLs, not real uploads — there is no
-- file-storage feature in this codebase.

CREATE TABLE vendors (
    id                    UUID PRIMARY KEY,
    owner_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    company_name          TEXT NOT NULL,
    categories            TEXT[] NOT NULL DEFAULT '{}',
    contact_person        TEXT NOT NULL DEFAULT '',
    phone                 TEXT NOT NULL DEFAULT '',
    email                 TEXT NOT NULL DEFAULT '',
    address               TEXT NOT NULL DEFAULT '',
    serves_all_properties BOOLEAN NOT NULL DEFAULT true,
    insurance_expiry      DATE,
    license_number        TEXT NOT NULL DEFAULT '',
    license_expiry        DATE,
    coi_link              TEXT NOT NULL DEFAULT '',
    tax_doc_link          TEXT NOT NULL DEFAULT '',
    rate_type             TEXT,
    rate_amount           NUMERIC(12,2),
    payment_terms         TEXT,
    internal_notes        TEXT NOT NULL DEFAULT '',
    active                BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT vendors_rate_type_check CHECK (rate_type IS NULL OR rate_type IN ('hourly', 'flat')),
    CONSTRAINT vendors_payment_terms_check CHECK (payment_terms IS NULL OR payment_terms IN ('net_15', 'net_30', 'net_45')),
    CONSTRAINT vendors_rate_amount_check CHECK (rate_amount IS NULL OR rate_amount >= 0)
);

CREATE INDEX idx_vendors_owner_id ON vendors(owner_id);

-- Specific properties a vendor serves, when serves_all_properties is
-- false. Rows here are ignored (a vendor serves everything) while
-- serves_all_properties is true — enforced in the service layer, not a
-- DB constraint, since a boolean-vs-rows XOR isn't cleanly expressible
-- as a CHECK across two tables.
CREATE TABLE vendor_properties (
    vendor_id   UUID NOT NULL REFERENCES vendors(id) ON DELETE CASCADE,
    property_id UUID NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    PRIMARY KEY (vendor_id, property_id)
);

CREATE INDEX idx_vendor_properties_property_id ON vendor_properties(property_id);

-- Links a work order to a real vendor record, alongside the existing
-- freeform assigned_to/assigned_to_contact (kept as the fallback for
-- internal staff, who aren't vendors). rating is the optional 1-5 star
-- rating prompted after marking a work order completed.
ALTER TABLE work_orders
    ADD COLUMN vendor_id UUID REFERENCES vendors(id) ON DELETE SET NULL,
    ADD COLUMN rating INTEGER;

ALTER TABLE work_orders ADD CONSTRAINT work_orders_rating_check CHECK (rating IS NULL OR rating BETWEEN 1 AND 5);

CREATE INDEX idx_work_orders_vendor_id ON work_orders(vendor_id);
