SET @has_base_role := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'org_roles' AND COLUMN_NAME = 'base_role'
);
SET @add_base_role_sql := IF(@has_base_role = 0,
  "ALTER TABLE org_roles ADD COLUMN base_role VARCHAR(32) NOT NULL DEFAULT 'viewer' AFTER role_name",
  'SELECT 1');
PREPARE stmt FROM @add_base_role_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE org_roles
SET base_role = CASE
  WHEN role_name IN ('admin', 'dev', 'viewer') THEN role_name
  ELSE 'viewer'
END;

SET @has_is_builtin := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'org_roles' AND COLUMN_NAME = 'is_builtin'
);
SET @drop_is_builtin_sql := IF(@has_is_builtin > 0,
  'ALTER TABLE org_roles DROP COLUMN is_builtin',
  'SELECT 1');
PREPARE stmt FROM @drop_is_builtin_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

DROP TABLE IF EXISTS org_role_permissions;
DROP TABLE IF EXISTS permission_catalog;
