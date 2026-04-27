ALTER TABLE apps
  ADD COLUMN app_secret_ciphertext text NOT NULL AFTER public_key,
  ADD COLUMN app_secret_updated_at datetime NULL AFTER app_secret_ciphertext;
