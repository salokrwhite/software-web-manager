ALTER TABLE orgs ADD COLUMN rejection_reason text NULL;
ALTER TABLE orgs ADD COLUMN allow_resubmit boolean NOT NULL DEFAULT false;
ALTER TABLE orgs ADD COLUMN rejected_by char(36) NULL;
ALTER TABLE orgs ADD COLUMN rejected_at datetime NULL;
