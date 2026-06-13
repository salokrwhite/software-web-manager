-- Merge org_members + app_members into a single polymorphic `memberships` table.
-- scope_type discriminates org vs app membership; scope_id holds the org/app id.
-- All application code queries `memberships` directly with a scope_type filter.

CREATE TABLE IF NOT EXISTS memberships (
  scope_type varchar(16) NOT NULL,
  scope_id   char(36) NOT NULL,
  user_id    char(36) NOT NULL,
  role       varchar(32) NOT NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (scope_type, scope_id, user_id),
  KEY idx_memberships_user (user_id),
  KEY idx_memberships_scope (scope_type, scope_id),
  CONSTRAINT fk_memberships_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO memberships (scope_type, scope_id, user_id, role, created_at)
SELECT 'org', org_id, user_id, role, created_at FROM org_members;

INSERT INTO memberships (scope_type, scope_id, user_id, role, created_at)
SELECT 'app', app_id, user_id, role, created_at FROM app_members;

DROP TABLE app_members;
DROP TABLE org_members;
