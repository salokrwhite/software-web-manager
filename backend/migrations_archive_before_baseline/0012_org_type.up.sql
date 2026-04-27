ALTER TABLE orgs
  ADD COLUMN org_type varchar(32) NOT NULL DEFAULT 'enterprise';
