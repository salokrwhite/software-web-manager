-- Per-app authorization signing keys. Replaces the single platform-wide Ed25519
-- key with one independent keypair per app (tenant): the private seed never
-- leaves the server (AES-GCM encrypted with APP_SECRET_MASTER_KEY), the public
-- key is handed to the developer to embed in their client. Multiple lifecycle
-- states (pending/active/retired) coexist so a key can be published to clients
-- before the server starts signing with it (zero-downtime rotation).
CREATE TABLE IF NOT EXISTS app_authz_keys (
  id char(36) PRIMARY KEY,
  app_id char(36) NOT NULL,
  key_id varchar(64) NOT NULL,                  -- public identifier embedded in clients; unique per app
  algorithm varchar(32) NOT NULL DEFAULT 'ed25519',
  private_key_ciphertext text NOT NULL,         -- ed25519 seed (hex), AES-GCM encrypted (APP_SECRET_MASTER_KEY)
  public_key varchar(128) NOT NULL,             -- hex public key (non-secret), embedded by developers
  status varchar(16) NOT NULL DEFAULT 'pending',-- pending | active | retired
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  activated_at datetime NULL,
  rotated_at datetime NULL,
  revoked_at datetime NULL,
  updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_app_authz_key_id (app_id, key_id),
  KEY idx_app_authz_app (app_id),
  KEY idx_app_authz_app_status (app_id, status, revoked_at),
  CONSTRAINT fk_app_authz_app FOREIGN KEY (app_id) REFERENCES apps(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
