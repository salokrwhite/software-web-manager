CREATE TABLE IF NOT EXISTS org_roles (
  id CHAR(36) PRIMARY KEY,
  org_id CHAR(36) NOT NULL,
  role_name VARCHAR(64) NOT NULL,
  is_builtin TINYINT(1) NOT NULL DEFAULT 0,
  description VARCHAR(255) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_by CHAR(36) NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_org_roles_org_role (org_id, role_name),
  INDEX idx_org_roles_org_id (org_id),
  INDEX idx_org_roles_status (status)
);
