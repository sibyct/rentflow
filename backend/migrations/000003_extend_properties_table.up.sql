-- Extends properties to match the actual product: a name, a property
-- type, a structured address, ownership/onboarding detail, amenities,
-- and notes — the fields the "Add Property" flow collects. The single
-- freeform `address` column is replaced by structured components so the
-- API can filter/sort/search on them; a formatted single-line address
-- for display is composed in Go (domain.Property.FormattedAddress),
-- not stored redundantly.

ALTER TABLE properties
    ADD COLUMN name            TEXT NOT NULL DEFAULT '',
    ADD COLUMN type             TEXT NOT NULL DEFAULT 'residential_single_unit',
    ADD COLUMN address_line1    TEXT NOT NULL DEFAULT '',
    ADD COLUMN address_line2    TEXT NOT NULL DEFAULT '',
    ADD COLUMN city             TEXT NOT NULL DEFAULT '',
    ADD COLUMN state_province   TEXT NOT NULL DEFAULT '',
    ADD COLUMN postal_code      TEXT NOT NULL DEFAULT '',
    ADD COLUMN country          TEXT NOT NULL DEFAULT '',
    ADD COLUMN ownership        TEXT,
    ADD COLUMN owner_name       TEXT NOT NULL DEFAULT '',
    ADD COLUMN year_built       INTEGER,
    ADD COLUMN onboard_date     DATE,
    ADD COLUMN amenities        TEXT[] NOT NULL DEFAULT '{}',
    ADD COLUMN notes            TEXT NOT NULL DEFAULT '';

-- Best-effort preservation of existing rows' only address data.
UPDATE properties SET address_line1 = address WHERE address_line1 = '';

ALTER TABLE properties RENAME COLUMN unit_count TO units;

-- The old inactive/maintenance statuses don't correspond to anything in
-- the current product (a property is "onboarding" until its first
-- lease, "active" once occupied, or "archived" when retired). Fold them
-- into archived rather than losing rows when the constraint changes.
UPDATE properties SET status = 'archived' WHERE status IN ('inactive', 'maintenance');

ALTER TABLE properties DROP CONSTRAINT properties_status_check;
ALTER TABLE properties ADD CONSTRAINT properties_status_check CHECK (status IN ('onboarding', 'active', 'archived'));
ALTER TABLE properties ALTER COLUMN status SET DEFAULT 'onboarding';

ALTER TABLE properties ADD CONSTRAINT properties_type_check
    CHECK (type IN ('residential_single_unit', 'residential_multi_unit', 'commercial', 'mixed_use'));
ALTER TABLE properties ADD CONSTRAINT properties_ownership_check
    CHECK (ownership IS NULL OR ownership IN ('owned', 'managed'));
ALTER TABLE properties ADD CONSTRAINT properties_year_built_check
    CHECK (year_built IS NULL OR (year_built BETWEEN 1800 AND 2100));

-- name/type/address_line1 are required for every row going forward;
-- the '' defaults above only existed to backfill pre-existing rows.
ALTER TABLE properties ALTER COLUMN name DROP DEFAULT;
ALTER TABLE properties ALTER COLUMN type DROP DEFAULT;
ALTER TABLE properties ALTER COLUMN address_line1 DROP DEFAULT;

ALTER TABLE properties DROP COLUMN address;

CREATE INDEX idx_properties_owner_id_status ON properties(owner_id, status);
