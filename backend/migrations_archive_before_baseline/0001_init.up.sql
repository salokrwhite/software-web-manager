CREATE TABLE IF NOT EXISTS users (
  id char(36) PRIMARY KEY,
  email varchar(255) NOT NULL UNIQUE,
  password_hash varchar(255) NOT NULL,
  status varchar(32) NOT NULL DEFAULT 'active',
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS orgs (
  id char(36) PRIMARY KEY,
  name varchar(255) NOT NULL,
  plan varchar(32) NOT NULL DEFAULT 'free',
  created_by char(36) NOT NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (created_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS org_members (
  org_id char(36) NOT NULL,
  user_id char(36) NOT NULL,
  role varchar(32) NOT NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (org_id, user_id),
  FOREIGN KEY (org_id) REFERENCES orgs(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS apps (
  id char(36) PRIMARY KEY,
  org_id char(36) NOT NULL,
  name varchar(255) NOT NULL,
  slug varchar(255) NOT NULL,
  description text,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_apps_org_slug (org_id, slug),
  FOREIGN KEY (org_id) REFERENCES orgs(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS channels (
  id char(36) PRIMARY KEY,
  app_id char(36) NOT NULL,
  name varchar(255) NOT NULL,
  code varchar(100) NOT NULL,
  is_default boolean NOT NULL DEFAULT false,
  min_supported_version varchar(64),
  preview_token varchar(255),
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_channels_app_code (app_id, code),
  FOREIGN KEY (app_id) REFERENCES apps(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS releases (
  id char(36) PRIMARY KEY,
  app_id char(36) NOT NULL,
  version varchar(100) NOT NULL,
  version_code int,
  notes text,
  status varchar(32) NOT NULL DEFAULT 'draft',
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (app_id) REFERENCES apps(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS release_channels (
  id char(36) PRIMARY KEY,
  release_id char(36) NOT NULL,
  channel_id char(36) NOT NULL,
  rollout_percent int NOT NULL DEFAULT 100,
  mandatory boolean NOT NULL DEFAULT false,
  whitelist_json json,
  published_at datetime,
  UNIQUE KEY uq_release_channels_release_channel (release_id, channel_id),
  FOREIGN KEY (release_id) REFERENCES releases(id),
  FOREIGN KEY (channel_id) REFERENCES channels(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS artifacts (
  id char(36) PRIMARY KEY,
  release_id char(36) NOT NULL,
  platform varchar(32) NOT NULL,
  arch varchar(32) NOT NULL,
  file_type varchar(32) NOT NULL,
  size bigint NOT NULL,
  checksum_sha256 varchar(64) NOT NULL,
  storage_driver varchar(32) NOT NULL,
  storage_path text NOT NULL,
  download_url text,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_artifacts_release_platform_arch (release_id, platform, arch),
  FOREIGN KEY (release_id) REFERENCES releases(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS api_keys (
  id char(36) PRIMARY KEY,
  org_id char(36) NOT NULL,
  app_id char(36) NOT NULL,
  key_hash varchar(64) NOT NULL UNIQUE,
  name varchar(255) NOT NULL,
  last_used_at datetime,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (org_id) REFERENCES orgs(id),
  FOREIGN KEY (app_id) REFERENCES apps(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS devices (
  id char(36) PRIMARY KEY,
  app_id char(36) NOT NULL,
  device_id varchar(255) NOT NULL,
  platform varchar(32) NOT NULL,
  arch varchar(32) NOT NULL,
  os_version varchar(64),
  first_seen_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_seen_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_devices_app_device (app_id, device_id),
  FOREIGN KEY (app_id) REFERENCES apps(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS events (
  id char(36) PRIMARY KEY,
  org_id char(36) NOT NULL,
  app_id char(36) NOT NULL,
  device_id varchar(255) NOT NULL,
  event_name varchar(255) NOT NULL,
  event_time datetime NOT NULL,
  channel_code varchar(100) NOT NULL,
  properties_jsonb json,
  FOREIGN KEY (org_id) REFERENCES orgs(id),
  FOREIGN KEY (app_id) REFERENCES apps(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS daily_metrics (
  date date NOT NULL,
  app_id char(36) NOT NULL,
  channel_id char(36) NOT NULL,
  event_name varchar(255) NOT NULL,
  count bigint NOT NULL,
  PRIMARY KEY (date, app_id, channel_id, event_name),
  FOREIGN KEY (app_id) REFERENCES apps(id),
  FOREIGN KEY (channel_id) REFERENCES channels(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS audit_logs (
  id char(36) PRIMARY KEY,
  org_id char(36) NOT NULL,
  user_id char(36) NOT NULL,
  action varchar(255) NOT NULL,
  target_type varchar(255) NOT NULL,
  target_id char(36),
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (org_id) REFERENCES orgs(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_releases_app ON releases(app_id);
CREATE INDEX idx_events_app_time ON events(app_id, event_time);
CREATE INDEX idx_events_name_time ON events(event_name, event_time);
