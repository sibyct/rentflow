-- Lease-specific documents (signed agreements, addenda, notices) live
-- only here, on their originating lease — never duplicated onto the
-- unit's unit_documents. Mirrors unit_documents' exact shape.
CREATE TABLE lease_documents (
    id            UUID PRIMARY KEY,
    lease_id      UUID NOT NULL REFERENCES leases(id) ON DELETE CASCADE,
    attachment_id UUID NOT NULL REFERENCES attachments(id),
    category      TEXT NOT NULL DEFAULT 'other',
    uploaded_by   UUID NOT NULL REFERENCES users(id),
    is_automated  BOOLEAN NOT NULL DEFAULT false,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT lease_documents_category_check CHECK (category IN ('lease_agreement', 'inspection', 'notice', 'renewal', 'other'))
);

CREATE INDEX idx_lease_documents_lease_id ON lease_documents(lease_id);

-- A unit-level document may optionally reference the specific lease it
-- relates to (e.g. a move-out inspection generated during that lease's
-- termination). is_automated distinguishes a system-generated document
-- from one a staff member manually uploaded and tagged.
ALTER TABLE unit_documents ADD COLUMN related_lease_id UUID REFERENCES leases(id) ON DELETE SET NULL;
ALTER TABLE unit_documents ADD COLUMN is_automated BOOLEAN NOT NULL DEFAULT false;

CREATE INDEX idx_unit_documents_related_lease_id ON unit_documents(related_lease_id) WHERE related_lease_id IS NOT NULL;
