-- Tracks when a unit most recently became vacant, so a "vacant units"
-- view can show days-vacant. Nullable: null means "not currently
-- vacant" or "became vacant before this column existed" — see
-- UnitService's stamping logic (unit.go's VacatedAt doc comment).
ALTER TABLE units ADD COLUMN vacated_at TIMESTAMPTZ;
