ALTER TABLE vendors
    DROP COLUMN coi_attachment_id,
    DROP COLUMN tax_doc_attachment_id,
    ADD COLUMN coi_link     TEXT NOT NULL DEFAULT '',
    ADD COLUMN tax_doc_link TEXT NOT NULL DEFAULT '';

ALTER TABLE work_orders
    DROP COLUMN photo_attachment_id,
    DROP COLUMN invoice_attachment_id,
    ADD COLUMN photo_link   TEXT NOT NULL DEFAULT '',
    ADD COLUMN invoice_link TEXT NOT NULL DEFAULT '';
