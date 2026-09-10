CREATE TABLE properties (
    id         UUID PRIMARY KEY,
    address    TEXT NOT NULL,
    unit_count INTEGER NOT NULL CHECK (unit_count > 0),
    status     TEXT NOT NULL CHECK (status IN ('active', 'inactive', 'maintenance')),
    owner_id   UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_properties_owner_id ON properties(owner_id);
