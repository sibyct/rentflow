-- work_orders.photo_link/invoice_link and vendors.coi_link/tax_doc_link
-- were freeform URL text because there was no file-storage feature in
-- this codebase yet (see their own migrations' comments). That's no
-- longer true: 000010_create_accounting_core added a generic
-- attachments table (owner-keyed, not accounting-specific — see
-- backend/internal/domain/attachment.go). These four fields now point
-- at real uploaded files through it, the same way transactions,
-- deposit_deductions and owner_statements already do.
--
-- No data migration: there's no meaningful production data behind
-- these columns yet, and a pasted URL couldn't be turned into an
-- attachments row (it was never uploaded to this app's storage) even
-- if there were.

ALTER TABLE work_orders
    DROP COLUMN photo_link,
    DROP COLUMN invoice_link,
    ADD COLUMN photo_attachment_id   UUID REFERENCES attachments(id) ON DELETE SET NULL,
    ADD COLUMN invoice_attachment_id UUID REFERENCES attachments(id) ON DELETE SET NULL;

ALTER TABLE vendors
    DROP COLUMN coi_link,
    DROP COLUMN tax_doc_link,
    ADD COLUMN coi_attachment_id     UUID REFERENCES attachments(id) ON DELETE SET NULL,
    ADD COLUMN tax_doc_attachment_id UUID REFERENCES attachments(id) ON DELETE SET NULL;
