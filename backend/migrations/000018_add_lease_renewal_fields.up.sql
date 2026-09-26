-- Proposed renewal terms live here, separate from the lease's real,
-- active monthly_rent/end_date — an "Offered" proposal must never
-- affect the active lease until it's Accepted and a new lease record
-- is generated from it (see LeaseService.GenerateRenewalLease).
--
-- renewed_into_lease_id / renewed_from_lease_id are the two ends of
-- the link a renewal creates: the original lease points at the new
-- one it was renewed into, and the new lease points back at the one
-- it renewed. Plain nullable FKs rather than a join table since it's
-- always exactly one renewal event per pair.

ALTER TABLE leases ADD COLUMN proposed_rent NUMERIC(12,2);
ALTER TABLE leases ADD COLUMN proposed_end_date DATE;
ALTER TABLE leases ADD COLUMN offer_sent_date DATE;
ALTER TABLE leases ADD COLUMN renewed_into_lease_id UUID REFERENCES leases(id) ON DELETE SET NULL;
ALTER TABLE leases ADD COLUMN renewed_from_lease_id UUID REFERENCES leases(id) ON DELETE SET NULL;

ALTER TABLE leases ADD CONSTRAINT leases_proposed_rent_check CHECK (proposed_rent IS NULL OR proposed_rent >= 0);

CREATE INDEX idx_leases_renewed_from_lease_id ON leases(renewed_from_lease_id);
