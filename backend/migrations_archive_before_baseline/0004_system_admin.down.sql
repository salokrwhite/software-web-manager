ALTER TABLE orgs DROP COLUMN approved_at;
ALTER TABLE orgs DROP COLUMN approved_by;
ALTER TABLE orgs DROP COLUMN status;

ALTER TABLE users DROP COLUMN system_role;
