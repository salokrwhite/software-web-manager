CREATE TABLE IF NOT EXISTS permission_catalog (
  permission_code VARCHAR(128) PRIMARY KEY,
  module VARCHAR(64) NOT NULL,
  name VARCHAR(128) NOT NULL,
  description VARCHAR(255) NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'active',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_permission_catalog_module (module),
  INDEX idx_permission_catalog_status (status)
);

CREATE TABLE IF NOT EXISTS org_role_permissions (
  id CHAR(36) PRIMARY KEY,
  org_id CHAR(36) NOT NULL,
  role_name VARCHAR(64) NOT NULL,
  permission_code VARCHAR(128) NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_org_role_permissions (org_id, role_name, permission_code),
  INDEX idx_org_role_permissions_org_id (org_id),
  INDEX idx_org_role_permissions_role_name (role_name),
  INDEX idx_org_role_permissions_permission_code (permission_code)
);

SET @has_is_builtin := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'org_roles' AND COLUMN_NAME = 'is_builtin'
);
SET @sql := IF(@has_is_builtin = 0,
  'ALTER TABLE org_roles ADD COLUMN is_builtin TINYINT(1) NOT NULL DEFAULT 0 AFTER role_name',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @has_base_role := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'org_roles' AND COLUMN_NAME = 'base_role'
);

INSERT INTO permission_catalog (permission_code, module, name, description, status)
VALUES
  ('role.viewer', 'role', '查看权限', '基础查看权限', 'active'),
  ('role.dev', 'role', '开发权限', '应用与发布操作权限', 'active'),
  ('role.admin', 'role', '管理权限', '组织管理权限', 'active'),
  ('role.owner', 'role', '所有者权限', '组织所有者权限', 'active'),
  ('org_management.view', 'org_management', '查看组织信息', '查看组织信息', 'active'),
  ('org_management.update', 'org_management', '更新组织信息', '更新组织基础设置', 'active'),
  ('org_management.transfer_owner', 'org_management', '转移所有者', '转移组织所有者', 'active'),
  ('org_management.delete', 'org_management', '删除组织', '删除当前组织', 'active'),
  ('member_manage.view', 'member_manage', '查看成员', '查看组织成员列表', 'active'),
  ('member_manage.create', 'member_manage', '新增成员', '创建组织成员', 'active'),
  ('member_manage.update', 'member_manage', '编辑成员', '编辑成员角色与状态', 'active'),
  ('member_manage.delete', 'member_manage', '删除成员', '移除组织成员', 'active'),
  ('member_invite.manage', 'member_invite', '邀请管理', '管理成员邀请', 'active'),
  ('org_join_request.review', 'org_join_request', '审批加入申请', '审批用户加入组织申请', 'active'),
  ('org_join_request.manage_own', 'org_join_request', '管理我的申请', '查看与撤回我的加入申请', 'active'),
  ('role_manage.view', 'role_manage', '查看角色', '查看权限类型', 'active'),
  ('role_manage.edit', 'role_manage', '管理角色', '管理权限类型与权限点', 'active'),
  ('app.manage', 'app', '应用管理', '创建与编辑应用', 'active'),
  ('release.manage', 'release', '发布管理', '创建与发布版本', 'active'),
  ('ticket.manage', 'ticket', '工单管理', '处理工单', 'active'),
  ('audit_log.view', 'audit_log', '查看审计日志', '查看审计日志', 'active')
ON DUPLICATE KEY UPDATE
  module = VALUES(module),
  name = VALUES(name),
  description = VALUES(description),
  status = VALUES(status);

SET @insert_viewer_sql := IF(@has_base_role > 0,
  "INSERT INTO org_roles (id, org_id, role_name, base_role, is_builtin, description, status, created_by, created_at, updated_at)
   SELECT UUID(), o.id, 'viewer', 'viewer', 1, '系统内置角色', 'active', o.created_by, NOW(), NOW()
   FROM orgs o
   WHERE NOT EXISTS (
     SELECT 1 FROM org_roles r WHERE r.org_id = o.id AND r.role_name = 'viewer'
   )",
  "INSERT INTO org_roles (id, org_id, role_name, is_builtin, description, status, created_by, created_at, updated_at)
   SELECT UUID(), o.id, 'viewer', 1, '系统内置角色', 'active', o.created_by, NOW(), NOW()
   FROM orgs o
   WHERE NOT EXISTS (
     SELECT 1 FROM org_roles r WHERE r.org_id = o.id AND r.role_name = 'viewer'
   )"
);
PREPARE stmt FROM @insert_viewer_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @insert_dev_sql := IF(@has_base_role > 0,
  "INSERT INTO org_roles (id, org_id, role_name, base_role, is_builtin, description, status, created_by, created_at, updated_at)
   SELECT UUID(), o.id, 'dev', 'dev', 1, '系统内置角色', 'active', o.created_by, NOW(), NOW()
   FROM orgs o
   WHERE NOT EXISTS (
     SELECT 1 FROM org_roles r WHERE r.org_id = o.id AND r.role_name = 'dev'
   )",
  "INSERT INTO org_roles (id, org_id, role_name, is_builtin, description, status, created_by, created_at, updated_at)
   SELECT UUID(), o.id, 'dev', 1, '系统内置角色', 'active', o.created_by, NOW(), NOW()
   FROM orgs o
   WHERE NOT EXISTS (
     SELECT 1 FROM org_roles r WHERE r.org_id = o.id AND r.role_name = 'dev'
   )"
);
PREPARE stmt FROM @insert_dev_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @insert_admin_sql := IF(@has_base_role > 0,
  "INSERT INTO org_roles (id, org_id, role_name, base_role, is_builtin, description, status, created_by, created_at, updated_at)
   SELECT UUID(), o.id, 'admin', 'admin', 1, '系统内置角色', 'active', o.created_by, NOW(), NOW()
   FROM orgs o
   WHERE NOT EXISTS (
     SELECT 1 FROM org_roles r WHERE r.org_id = o.id AND r.role_name = 'admin'
   )",
  "INSERT INTO org_roles (id, org_id, role_name, is_builtin, description, status, created_by, created_at, updated_at)
   SELECT UUID(), o.id, 'admin', 1, '系统内置角色', 'active', o.created_by, NOW(), NOW()
   FROM orgs o
   WHERE NOT EXISTS (
     SELECT 1 FROM org_roles r WHERE r.org_id = o.id AND r.role_name = 'admin'
   )"
);
PREPARE stmt FROM @insert_admin_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE org_roles
SET is_builtin = 1
WHERE role_name IN ('viewer', 'dev', 'admin');

DELETE FROM org_role_permissions;

SET @role_expr := IF(@has_base_role > 0, "LOWER(COALESCE(NULLIF(r.base_role,''), r.role_name))", "LOWER(r.role_name)");

SET @viewer_sql := CONCAT(
  "INSERT IGNORE INTO org_role_permissions (id, org_id, role_name, permission_code, created_at) ",
  "SELECT UUID(), r.org_id, r.role_name, p.permission_code, NOW() ",
  "FROM org_roles r ",
  "JOIN permission_catalog p ON p.permission_code IN ('role.viewer','org_management.view','member_manage.view','org_join_request.manage_own','role_manage.view','audit_log.view') ",
  "WHERE ", @role_expr, " = 'viewer'"
);
PREPARE stmt FROM @viewer_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @dev_sql := CONCAT(
  "INSERT IGNORE INTO org_role_permissions (id, org_id, role_name, permission_code, created_at) ",
  "SELECT UUID(), r.org_id, r.role_name, p.permission_code, NOW() ",
  "FROM org_roles r ",
  "JOIN permission_catalog p ON p.permission_code IN ('role.viewer','role.dev','org_management.view','member_manage.view','org_join_request.manage_own','role_manage.view','audit_log.view','app.manage','release.manage','ticket.manage') ",
  "WHERE ", @role_expr, " = 'dev'"
);
PREPARE stmt FROM @dev_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @admin_sql := CONCAT(
  "INSERT IGNORE INTO org_role_permissions (id, org_id, role_name, permission_code, created_at) ",
  "SELECT UUID(), r.org_id, r.role_name, p.permission_code, NOW() ",
  "FROM org_roles r ",
  "JOIN permission_catalog p ON p.permission_code IN ('role.viewer','role.dev','role.admin','org_management.view','org_management.update','member_manage.view','member_manage.create','member_manage.update','member_manage.delete','member_invite.manage','org_join_request.review','org_join_request.manage_own','role_manage.view','role_manage.edit','audit_log.view','app.manage','release.manage','ticket.manage') ",
  "WHERE ", @role_expr, " = 'admin'"
);
PREPARE stmt FROM @admin_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

SET @drop_base_role_sql := IF(@has_base_role > 0,
  'ALTER TABLE org_roles DROP COLUMN base_role',
  'SELECT 1');
PREPARE stmt FROM @drop_base_role_sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
