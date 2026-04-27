CREATE TABLE IF NOT EXISTS org_invites (
  id char(36) PRIMARY KEY,
  org_id char(36) NOT NULL,
  email varchar(255) NOT NULL,
  role varchar(32) NOT NULL,
  token_hash varchar(64) NOT NULL UNIQUE,
  expires_at datetime,
  created_by char(36) NOT NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  used_at datetime,
  revoked_at datetime,
  FOREIGN KEY (org_id) REFERENCES orgs(id),
  FOREIGN KEY (created_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_org_invites_org ON org_invites(org_id);
CREATE INDEX idx_org_invites_email ON org_invites(email);
