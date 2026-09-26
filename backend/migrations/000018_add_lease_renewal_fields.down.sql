DROP INDEX idx_leases_renewed_from_lease_id;

ALTER TABLE leases DROP CONSTRAINT leases_proposed_rent_check;

ALTER TABLE leases DROP COLUMN renewed_from_lease_id;
ALTER TABLE leases DROP COLUMN renewed_into_lease_id;
ALTER TABLE leases DROP COLUMN offer_sent_date;
ALTER TABLE leases DROP COLUMN proposed_end_date;
ALTER TABLE leases DROP COLUMN proposed_rent;
