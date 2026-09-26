DROP INDEX idx_unit_documents_related_lease_id;
ALTER TABLE unit_documents DROP COLUMN is_automated;
ALTER TABLE unit_documents DROP COLUMN related_lease_id;

DROP TABLE lease_documents;
