CREATE TABLE IF NOT EXISTS app_secrets (
  id char(36) NOT NULL,
  app_id char(36) NOT NULL,
  name varchar(128) NOT NULL DEFAULT 'app_secret',
  secret_ciphertext text NOT NULL,
  scopes_json json NULL,
  expires_at datetime NULL,
  last_used_at datetime NULL,
  revoked_at datetime NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  KEY idx_app_secrets_app_id (app_id),
  KEY idx_app_secrets_app_id_revoked_at (app_id, revoked_at)
);

INSERT INTO app_secrets (
  id,
  app_id,
  name,
  secret_ciphertext,
  scopes_json,
  expires_at,
  last_used_at,
  revoked_at,
  created_at,
  updated_at
)
SELECT
  UUID(),
  id,
  COALESCE(NULLIF(TRIM(app_secret_name), ''), 'app_secret'),
  app_secret_ciphertext,
  app_secret_scopes,
  app_secret_expires_at,
  NULL,
  NULL,
  COALESCE(app_secret_updated_at, created_at),
  COALESCE(app_secret_updated_at, created_at)
FROM apps
WHERE app_secret_ciphertext IS NOT NULL AND TRIM(app_secret_ciphertext) <> '';
