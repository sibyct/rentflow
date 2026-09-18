-- Units are the rentable thing inside a property (an apartment, a
-- suite, or — for a residential_single_unit property — the property
-- itself, represented by exactly one implicit unit row so occupancy/
-- rent stats have a single source of truth regardless of property
-- type; see PropertyService.CreateProperty).
--
-- There is no separate tenant/org scoping column here: this schema
-- scopes everything by properties.owner_id (see 000002), so a unit's
-- tenancy is reached by joining through property_id — never trust a
-- unit_id alone without checking its parent property's owner_id.

CREATE TABLE units (
    id                UUID PRIMARY KEY,
    property_id       UUID NOT NULL REFERENCES properties(id) ON DELETE CASCADE,
    unit_name         TEXT NOT NULL,
    floor             TEXT NOT NULL DEFAULT '',
    unit_type         TEXT NOT NULL,
    bedrooms          INTEGER,
    bathrooms         NUMERIC(3,1),
    sqft              INTEGER,
    furnished         TEXT,
    status            TEXT NOT NULL DEFAULT 'vacant',
    -- Both rent columns are nullable: the property form doesn't collect
    -- per-unit rent today, so an auto-created implicit unit (and a
    -- freshly bulk-added row) starts with no rent set rather than a
    -- fabricated 0.
    market_rent       NUMERIC(12,2),
    current_rent      NUMERIC(12,2),
    security_deposit  NUMERIC(12,2),
    rent_due_day      INTEGER,
    -- Freeform, like properties.owner_name: there's no tenant/lease
    -- feature yet to link a real record to, so this is a display-only
    -- label the manager types in, not a foreign key.
    tenant_name       TEXT NOT NULL DEFAULT '',
    notes             TEXT NOT NULL DEFAULT '',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT units_property_id_unit_name_key UNIQUE (property_id, unit_name),
    CONSTRAINT units_type_check CHECK (unit_type IN ('studio', '1br', '2br', '3br_plus', 'commercial_suite', 'other')),
    CONSTRAINT units_furnished_check CHECK (furnished IS NULL OR furnished IN ('unfurnished', 'furnished', 'partial')),
    CONSTRAINT units_status_check CHECK (status IN ('vacant', 'occupied', 'maintenance', 'off_market')),
    CONSTRAINT units_bedrooms_check CHECK (bedrooms IS NULL OR bedrooms >= 0),
    CONSTRAINT units_bathrooms_check CHECK (bathrooms IS NULL OR bathrooms >= 0),
    CONSTRAINT units_sqft_check CHECK (sqft IS NULL OR sqft > 0),
    CONSTRAINT units_market_rent_check CHECK (market_rent IS NULL OR market_rent >= 0),
    CONSTRAINT units_current_rent_check CHECK (current_rent IS NULL OR current_rent >= 0),
    CONSTRAINT units_security_deposit_check CHECK (security_deposit IS NULL OR security_deposit >= 0),
    CONSTRAINT units_rent_due_day_check CHECK (rent_due_day IS NULL OR (rent_due_day BETWEEN 1 AND 31))
);

CREATE INDEX idx_units_property_id ON units(property_id);
CREATE INDEX idx_units_property_id_status ON units(property_id, status);
