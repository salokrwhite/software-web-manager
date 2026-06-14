-- App maintenance mode: schedule a downtime window so clients can warn users
-- with a countdown and exit when it starts.
ALTER TABLE apps ADD COLUMN maintenance_enabled TINYINT(1) NOT NULL DEFAULT 0;
ALTER TABLE apps ADD COLUMN maintenance_start_at DATETIME NULL;
ALTER TABLE apps ADD COLUMN maintenance_message VARCHAR(500) NOT NULL DEFAULT '';
