ALTER TABLE apps ADD COLUMN status varchar(32) NOT NULL DEFAULT 'active';
ALTER TABLE apps ADD COLUMN submitted_at datetime NULL;
ALTER TABLE apps ADD COLUMN approved_by char(36) NULL;
ALTER TABLE apps ADD COLUMN approved_at datetime NULL;
ALTER TABLE apps ADD COLUMN rejected_by char(36) NULL;
ALTER TABLE apps ADD COLUMN rejected_at datetime NULL;
ALTER TABLE apps ADD COLUMN rejection_reason text NULL;
