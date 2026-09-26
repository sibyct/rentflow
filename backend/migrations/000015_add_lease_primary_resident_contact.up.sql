-- Structured contact info for a lease's primary resident, alongside
-- the existing freeform primary_resident_name/emergency_contact —
-- there is still no Residents entity (see 000005's comment), so this
-- stays freeform text too, just two more fields of it.

ALTER TABLE leases ADD COLUMN primary_resident_phone TEXT NOT NULL DEFAULT '';
ALTER TABLE leases ADD COLUMN primary_resident_email TEXT NOT NULL DEFAULT '';
