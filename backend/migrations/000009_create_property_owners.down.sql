DROP INDEX IF EXISTS idx_properties_property_owner_id;
ALTER TABLE properties DROP COLUMN IF EXISTS property_owner_id;
DROP TABLE IF EXISTS property_owners;
