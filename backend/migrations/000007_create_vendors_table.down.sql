ALTER TABLE work_orders DROP CONSTRAINT work_orders_rating_check;
ALTER TABLE work_orders DROP COLUMN rating;
ALTER TABLE work_orders DROP COLUMN vendor_id;
DROP TABLE vendor_properties;
DROP TABLE vendors;
