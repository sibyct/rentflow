DROP INDEX IF EXISTS idx_properties_owner_id_status;

ALTER TABLE properties ADD COLUMN address TEXT NOT NULL DEFAULT '';
UPDATE properties SET address = address_line1;
ALTER TABLE properties ALTER COLUMN address DROP DEFAULT;

ALTER TABLE properties DROP CONSTRAINT IF EXISTS properties_type_check;
ALTER TABLE properties DROP CONSTRAINT IF EXISTS properties_ownership_check;
ALTER TABLE properties DROP CONSTRAINT IF EXISTS properties_year_built_check;

ALTER TABLE properties DROP CONSTRAINT properties_status_check;
ALTER TABLE properties ALTER COLUMN status DROP DEFAULT;
UPDATE properties SET status = 'inactive' WHERE status = 'archived';
ALTER TABLE properties ADD CONSTRAINT properties_status_check CHECK (status IN ('active', 'inactive', 'maintenance'));

ALTER TABLE properties RENAME COLUMN units TO unit_count;

ALTER TABLE properties
    DROP COLUMN name,
    DROP COLUMN type,
    DROP COLUMN address_line1,
    DROP COLUMN address_line2,
    DROP COLUMN city,
    DROP COLUMN state_province,
    DROP COLUMN postal_code,
    DROP COLUMN country,
    DROP COLUMN ownership,
    DROP COLUMN owner_name,
    DROP COLUMN year_built,
    DROP COLUMN onboard_date,
    DROP COLUMN amenities,
    DROP COLUMN notes;
