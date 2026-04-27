ALTER TABLE apps
  ADD COLUMN app_secret_name varchar(128) NOT NULL DEFAULT 'app_secret' AFTER app_secret_expires_at;

