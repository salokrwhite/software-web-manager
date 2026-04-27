ALTER TABLE apps
  ADD COLUMN app_secret_scopes json NULL AFTER app_secret_updated_at;

