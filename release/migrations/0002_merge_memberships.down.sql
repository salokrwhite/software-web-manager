-- Revert the memberships merge: restore separate org_members / app_members tables.

CREATE TABLE IF NOT EXISTS org_members (
  org_id char(36) NOT NULL,
  user_id char(36) NOT NULL,
  role varchar(32) NOT NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (org_id, user_id),
  KEY idx_org_members_user (user_id),
  CONSTRAINT fk_org_members_org FOREIGN KEY (org_id) REFERENCES orgs(id),
  CONSTRAINT fk_org_members_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS app_members (
  app_id char(36) NOT NULL,
  user_id char(36) NOT NULL,
  role varchar(32) NOT NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (app_id, user_id),
  KEY idx_app_members_user (user_id),
  CONSTRAINT fk_app_members_app FOREIGN KEY (app_id) REFERENCES apps(id),
  CONSTRAINT fk_app_members_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO org_members (org_id, user_id, role, created_at)
SELECT scope_id, user_id, role, created_at FROM memberships WHERE scope_type = 'org';

INSERT INTO app_members (app_id, user_id, role, created_at)
SELECT scope_id, user_id, role, created_at FROM memberships WHERE scope_type = 'app';

DROP TABLE memberships;
