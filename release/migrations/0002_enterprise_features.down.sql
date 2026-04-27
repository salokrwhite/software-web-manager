DROP INDEX idx_devices_app_seen ON devices;
DROP INDEX idx_audit_logs_org_time ON audit_logs;

DROP TABLE IF EXISTS app_members;

ALTER TABLE devices DROP COLUMN last_ip;
ALTER TABLE devices DROP COLUMN user_id;
ALTER TABLE devices DROP COLUMN app_version;
ALTER TABLE devices DROP COLUMN country;

ALTER TABLE audit_logs DROP COLUMN after_json;
ALTER TABLE audit_logs DROP COLUMN before_json;
ALTER TABLE audit_logs DROP COLUMN user_agent;
ALTER TABLE audit_logs DROP COLUMN ip_address;

ALTER TABLE api_keys DROP COLUMN revoked_at;
ALTER TABLE api_keys DROP COLUMN expires_at;
ALTER TABLE api_keys DROP COLUMN scopes;

ALTER TABLE artifacts DROP COLUMN signature;

ALTER TABLE release_channels DROP COLUMN rollout_end_at;
ALTER TABLE release_channels DROP COLUMN rollout_start_at;
ALTER TABLE release_channels DROP COLUMN targeting_rules_json;
ALTER TABLE release_channels DROP COLUMN paused;
ALTER TABLE release_channels DROP COLUMN status;

ALTER TABLE releases DROP COLUMN published_at;
ALTER TABLE releases DROP COLUMN approved_by;
ALTER TABLE releases DROP COLUMN approved_at;
ALTER TABLE releases DROP COLUMN submitted_at;

ALTER TABLE apps DROP COLUMN public_key;
