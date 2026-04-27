ALTER TABLE apps
  ADD COLUMN app_secret_expires_at datetime NULL AFTER app_secret_scopes;

