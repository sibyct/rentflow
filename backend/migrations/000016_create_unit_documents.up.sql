CREATE TABLE unit_documents (
    id            UUID PRIMARY KEY,
    unit_id       UUID NOT NULL REFERENCES units(id) ON DELETE CASCADE,
    attachment_id UUID NOT NULL REFERENCES attachments(id),
    category      TEXT NOT NULL DEFAULT 'other',
    uploaded_by   UUID NOT NULL REFERENCES users(id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT unit_documents_category_check CHECK (category IN ('inspection', 'manual', 'photo', 'other'))
);

CREATE INDEX idx_unit_documents_unit_id ON unit_documents(unit_id);
