ALTER TABLE apps ADD COLUMN public_key text;

ALTER TABLE releases ADD COLUMN submitted_at datetime NULL;
ALTER TABLE releases ADD COLUMN approved_at datetime NULL;
ALTER TABLE releases ADD COLUMN approved_by char(36) NULL;
ALTER TABLE releases ADD COLUMN published_at datetime NULL;

ALTER TABLE release_channels ADD COLUMN status varchar(32) NOT NULL DEFAULT 'inactive';
ALTER TABLE release_channels ADD COLUMN paused boolean NOT NULL DEFAULT false;
ALTER TABLE release_channels ADD COLUMN targeting_rules_json json;
ALTER TABLE release_channels ADD COLUMN rollout_start_at datetime;
ALTER TABLE release_channels ADD COLUMN rollout_end_at datetime;

ALTER TABLE artifacts ADD COLUMN signature text;

ALTER TABLE api_keys ADD COLUMN scopes json;
ALTER TABLE api_keys ADD COLUMN expires_at datetime;
ALTER TABLE api_keys ADD COLUMN revoked_at datetime;

ALTER TABLE audit_logs ADD COLUMN ip_address varchar(64);
ALTER TABLE audit_logs ADD COLUMN user_agent varchar(255);
ALTER TABLE audit_logs ADD COLUMN before_json json;
ALTER TABLE audit_logs ADD COLUMN after_json json;

ALTER TABLE devices ADD COLUMN country varchar(32);
ALTER TABLE devices ADD COLUMN app_version varchar(64);
ALTER TABLE devices ADD COLUMN user_id varchar(128);
ALTER TABLE devices ADD COLUMN last_ip varchar(64);

CREATE TABLE IF NOT EXISTS app_members (
  app_id char(36) NOT NULL,
  user_id char(36) NOT NULL,
  role varchar(32) NOT NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (app_id, user_id),
  FOREIGN KEY (app_id) REFERENCES apps(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_audit_logs_org_time ON audit_logs(org_id, created_at);
CREATE INDEX idx_devices_app_seen ON devices(app_id, last_seen_at);
