-- Reverses 0001_baseline (consolidated). Teardown runs newest-change first.

-- ========================= 0007_user_token_version (down) =========================
ALTER TABLE users DROP COLUMN token_version;


-- ========================= 0006_app_authz_keys (down) =========================
DROP TABLE IF EXISTS app_authz_keys;


-- ========================= 0005_app_maintenance (down) =========================
ALTER TABLE apps DROP COLUMN maintenance_message;
ALTER TABLE apps DROP COLUMN maintenance_start_at;
ALTER TABLE apps DROP COLUMN maintenance_enabled;


-- ========================= 0004_sso_sub (down) =========================
DROP INDEX uniq_users_sso_sub ON users;
ALTER TABLE users DROP COLUMN sso_sub;


-- ========================= 0003_event_dimension_metrics (down) =========================
DROP TABLE IF EXISTS daily_event_dimensions;
ALTER TABLE events DROP INDEX idx_events_app_name_time;


-- ========================= 0002_merge_memberships (down) =========================
-- Revert the memberships merge: restore separate org_members / app_members tables.

CREATE TABLE IF NOT EXISTS org_members (
  org_id char(36) NOT NULL,
  user_id char(36) NOT NULL,
  role varchar(32) NOT NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (org_id, user_id),
  KEY idx_org_members_user (user_id),
  CONSTRAINT fk_org_members_org FOREIGN KEY (org_id) REFERENCES orgs(id),
  CONSTRAINT fk_org_members_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS app_members (
  app_id char(36) NOT NULL,
  user_id char(36) NOT NULL,
  role varchar(32) NOT NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (app_id, user_id),
  KEY idx_app_members_user (user_id),
  CONSTRAINT fk_app_members_app FOREIGN KEY (app_id) REFERENCES apps(id),
  CONSTRAINT fk_app_members_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO org_members (org_id, user_id, role, created_at)
SELECT scope_id, user_id, role, created_at FROM memberships WHERE scope_type = 'org';

INSERT INTO app_members (app_id, user_id, role, created_at)
SELECT scope_id, user_id, role, created_at FROM memberships WHERE scope_type = 'app';

DROP TABLE memberships;


-- ========================= 0001_baseline (down) =========================
-- Roll back the final baseline schema.

DROP TABLE IF EXISTS attachments;
DROP TABLE IF EXISTS feedbacks;
DROP TABLE IF EXISTS ticket_messages;
DROP TABLE IF EXISTS tickets;
DROP TABLE IF EXISTS email_verification_codes;
DROP TABLE IF EXISTS system_settings;
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS daily_metrics;
DROP TABLE IF EXISTS events;
DROP TABLE IF EXISTS device_controls;
DROP TABLE IF EXISTS devices;
DROP TABLE IF EXISTS artifacts;
DROP TABLE IF EXISTS release_channels;
DROP TABLE IF EXISTS releases;
DROP TABLE IF EXISTS release_templates;
DROP TABLE IF EXISTS channels;
DROP TABLE IF EXISTS app_secrets;
DROP TABLE IF EXISTS app_members;
DROP TABLE IF EXISTS apps;
DROP TABLE IF EXISTS org_role_permissions;
DROP TABLE IF EXISTS permission_catalog;
DROP TABLE IF EXISTS org_roles;
DROP TABLE IF EXISTS org_join_requests;
DROP TABLE IF EXISTS org_invites;
DROP TABLE IF EXISTS org_members;
DROP TABLE IF EXISTS orgs;
DROP TABLE IF EXISTS users;


