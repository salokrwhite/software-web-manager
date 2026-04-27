ALTER TABLE users ADD COLUMN system_role varchar(32) NOT NULL DEFAULT 'none';

ALTER TABLE orgs ADD COLUMN status varchar(32) NOT NULL DEFAULT 'active';
ALTER TABLE orgs ADD COLUMN approved_by char(36) NULL;
ALTER TABLE orgs ADD COLUMN approved_at datetime NULL;
