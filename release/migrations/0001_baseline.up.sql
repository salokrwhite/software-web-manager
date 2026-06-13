-- Final baseline schema for software-web-manager.
-- This replaces the historical incremental migrations with one fresh-install migration.

CREATE TABLE IF NOT EXISTS users (
  id char(36) PRIMARY KEY,
  email varchar(255) NOT NULL UNIQUE,
  password_hash varchar(255) NOT NULL,
  avatar_path varchar(512) DEFAULT '',
  status varchar(32) NOT NULL DEFAULT 'active',
  system_role varchar(32) NOT NULL DEFAULT 'none',
  otp_secret varchar(128) NULL,
  otp_enabled tinyint(1) NOT NULL DEFAULT 0,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS orgs (
  id char(36) PRIMARY KEY,
  name varchar(255) NOT NULL,
  plan varchar(32) NOT NULL DEFAULT 'free',
  org_type varchar(32) NOT NULL DEFAULT 'enterprise',
  status varchar(32) NOT NULL DEFAULT 'active',
  created_by char(36) NOT NULL,
  approved_by char(36) NULL,
  approved_at datetime NULL,
  rejection_reason text NULL,
  allow_resubmit tinyint(1) NOT NULL DEFAULT 0,
  resubmit_token char(36) NULL,
  rejected_by char(36) NULL,
  rejected_at datetime NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_orgs_created_by (created_by),
  KEY idx_orgs_status (status),
  KEY idx_orgs_org_type (org_type),
  CONSTRAINT fk_orgs_created_by FOREIGN KEY (created_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

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

CREATE TABLE IF NOT EXISTS org_invites (
  id char(36) PRIMARY KEY,
  org_id char(36) NOT NULL,
  email varchar(255) NOT NULL,
  role varchar(32) NOT NULL,
  token_hash varchar(255) NOT NULL UNIQUE,
  expires_at datetime NULL,
  created_by char(36) NOT NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  used_at datetime NULL,
  revoked_at datetime NULL,
  KEY idx_org_invites_org (org_id),
  KEY idx_org_invites_email (email),
  CONSTRAINT fk_org_invites_org FOREIGN KEY (org_id) REFERENCES orgs(id),
  CONSTRAINT fk_org_invites_created_by FOREIGN KEY (created_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS org_join_requests (
  id char(36) PRIMARY KEY,
  org_id char(36) NOT NULL,
  user_id char(36) NOT NULL,
  reason text,
  status varchar(32) NOT NULL DEFAULT 'pending',
  review_reason text,
  reviewed_by char(36) NULL,
  reviewed_at datetime NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_org_join_requests_org (org_id),
  KEY idx_org_join_requests_user (user_id),
  KEY idx_org_join_requests_reviewed_by (reviewed_by),
  KEY idx_org_join_requests_status (status),
  CONSTRAINT fk_org_join_requests_org FOREIGN KEY (org_id) REFERENCES orgs(id),
  CONSTRAINT fk_org_join_requests_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS org_roles (
  id char(36) PRIMARY KEY,
  org_id char(36) NOT NULL,
  role_name varchar(64) NOT NULL,
  is_builtin tinyint(1) NOT NULL DEFAULT 0,
  description varchar(255) NULL,
  status varchar(32) NOT NULL DEFAULT 'active',
  created_by char(36) NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uk_org_roles_org_role_name (org_id, role_name),
  KEY idx_org_roles_org_id (org_id),
  KEY idx_org_roles_status (status),
  CONSTRAINT fk_org_roles_org FOREIGN KEY (org_id) REFERENCES orgs(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS permission_catalog (
  permission_code varchar(128) PRIMARY KEY,
  module varchar(64) NOT NULL,
  name varchar(128) NOT NULL,
  description varchar(255) NULL,
  status varchar(32) NOT NULL DEFAULT 'active',
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_permission_catalog_module (module),
  KEY idx_permission_catalog_status (status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS org_role_permissions (
  id char(36) PRIMARY KEY,
  org_id char(36) NOT NULL,
  role_name varchar(64) NOT NULL,
  permission_code varchar(128) NOT NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uk_org_role_permissions (org_id, role_name, permission_code),
  KEY idx_org_role_permissions_org_id (org_id),
  KEY idx_org_role_permissions_role_name (role_name),
  KEY idx_org_role_permissions_permission_code (permission_code),
  CONSTRAINT fk_org_role_permissions_org FOREIGN KEY (org_id) REFERENCES orgs(id),
  CONSTRAINT fk_org_role_permissions_permission FOREIGN KEY (permission_code) REFERENCES permission_catalog(permission_code)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS apps (
  id char(36) PRIMARY KEY,
  org_id char(36) NOT NULL,
  name varchar(255) NOT NULL,
  slug varchar(255) NOT NULL,
  description text,
  public_key text,
  app_secret_ciphertext text NOT NULL,
  app_secret_updated_at datetime NULL,
  app_secret_scopes json NULL,
  app_secret_expires_at datetime NULL,
  app_secret_name varchar(128) NOT NULL DEFAULT 'app_secret',
  region_rules_json json NULL,
  feedback_enabled tinyint(1) NOT NULL DEFAULT 1,
  heartbeat_interval_seconds int NOT NULL DEFAULT 60,
  online_enabled tinyint(1) NOT NULL DEFAULT 1,
  status varchar(32) NOT NULL DEFAULT 'active',
  submitted_at datetime NULL,
  approved_by char(36) NULL,
  approved_at datetime NULL,
  rejected_by char(36) NULL,
  rejected_at datetime NULL,
  rejection_reason text NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_apps_org_slug (org_id, slug),
  KEY idx_apps_org (org_id),
  KEY idx_apps_status (status),
  CONSTRAINT fk_apps_org FOREIGN KEY (org_id) REFERENCES orgs(id)
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

CREATE TABLE IF NOT EXISTS app_secrets (
  id char(36) PRIMARY KEY,
  app_id char(36) NOT NULL,
  name varchar(128) NOT NULL DEFAULT 'app_secret',
  secret_ciphertext text NOT NULL,
  scopes_json json NULL,
  expires_at datetime NULL,
  last_used_at datetime NULL,
  revoked_at datetime NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_app_secrets_app_id (app_id),
  KEY idx_app_secrets_app_id_revoked_at (app_id, revoked_at),
  CONSTRAINT fk_app_secrets_app FOREIGN KEY (app_id) REFERENCES apps(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS channels (
  id char(36) PRIMARY KEY,
  app_id char(36) NOT NULL,
  name varchar(255) NOT NULL,
  code varchar(100) NOT NULL,
  is_default tinyint(1) NOT NULL DEFAULT 0,
  min_supported_version varchar(64),
  preview_token varchar(255),
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_channels_app_code (app_id, code),
  KEY idx_channels_app (app_id),
  CONSTRAINT fk_channels_app FOREIGN KEY (app_id) REFERENCES apps(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS release_templates (
  id char(36) PRIMARY KEY,
  org_id char(36) NOT NULL,
  name varchar(255) NOT NULL,
  schedule_at datetime NULL,
  window_start datetime NULL,
  window_end datetime NULL,
  emergency tinyint(1) NOT NULL DEFAULT 0,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_release_templates_org (org_id),
  CONSTRAINT fk_release_templates_org FOREIGN KEY (org_id) REFERENCES orgs(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS releases (
  id char(36) PRIMARY KEY,
  app_id char(36) NOT NULL,
  version varchar(100) NOT NULL,
  version_code int NULL,
  notes text,
  external_download_url varchar(2048) DEFAULT '',
  release_template_id char(36) NULL,
  status varchar(32) NOT NULL DEFAULT 'draft',
  submitted_at datetime NULL,
  approved_at datetime NULL,
  approved_by char(36) NULL,
  published_at datetime NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_releases_app (app_id),
  KEY idx_releases_status (status),
  KEY idx_releases_template (release_template_id),
  CONSTRAINT fk_releases_app FOREIGN KEY (app_id) REFERENCES apps(id),
  CONSTRAINT fk_releases_template FOREIGN KEY (release_template_id) REFERENCES release_templates(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS release_channels (
  id char(36) PRIMARY KEY,
  release_id char(36) NOT NULL,
  channel_id char(36) NOT NULL,
  rollout_percent int NOT NULL DEFAULT 100,
  mandatory tinyint(1) NOT NULL DEFAULT 0,
  whitelist_json json NULL,
  region_rules_json json NULL,
  status varchar(32) NOT NULL DEFAULT 'inactive',
  paused tinyint(1) NOT NULL DEFAULT 0,
  targeting_rules_json json NULL,
  rollout_start_at datetime NULL,
  rollout_end_at datetime NULL,
  published_at datetime NULL,
  UNIQUE KEY uq_release_channels_release_channel (release_id, channel_id),
  KEY idx_release_channels_release (release_id),
  KEY idx_release_channels_channel (channel_id),
  CONSTRAINT fk_release_channels_release FOREIGN KEY (release_id) REFERENCES releases(id),
  CONSTRAINT fk_release_channels_channel FOREIGN KEY (channel_id) REFERENCES channels(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS artifacts (
  id char(36) PRIMARY KEY,
  release_id char(36) NOT NULL,
  platform varchar(32) NOT NULL,
  arch varchar(32) NOT NULL,
  file_type varchar(32) NOT NULL,
  size bigint NOT NULL,
  checksum_sha256 varchar(64) NOT NULL,
  signature text,
  storage_driver varchar(32) NOT NULL,
  storage_path text NOT NULL,
  download_url text,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uq_artifacts_release_platform_arch (release_id, platform, arch),
  KEY idx_artifacts_release (release_id),
  CONSTRAINT fk_artifacts_release FOREIGN KEY (release_id) REFERENCES releases(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS devices (
  id char(36) PRIMARY KEY,
  app_id char(36) NOT NULL,
  device_id varchar(255) NOT NULL,
  platform varchar(32) NOT NULL,
  arch varchar(32) NOT NULL,
  os_version varchar(64),
  country varchar(32),
  app_version varchar(64),
  user_id varchar(128),
  last_ip varchar(64),
  first_seen_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  last_seen_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_devices_app_device (app_id, device_id),
  KEY idx_devices_app_seen (app_id, last_seen_at),
  CONSTRAINT fk_devices_app FOREIGN KEY (app_id) REFERENCES apps(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS device_controls (
  id char(36) PRIMARY KEY,
  app_id char(36) NOT NULL,
  device_id varchar(255) NOT NULL,
  blocked tinyint(1) NOT NULL DEFAULT 1,
  reason varchar(255) NULL,
  blocked_at datetime NULL,
  blocked_by char(36) NULL,
  unblocked_at datetime NULL,
  unblocked_by char(36) NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_device_controls_app_device (app_id, device_id),
  KEY idx_device_controls_app_blocked (app_id, blocked),
  KEY idx_device_controls_device_id (device_id),
  CONSTRAINT fk_device_controls_app FOREIGN KEY (app_id) REFERENCES apps(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS events (
  id char(36) PRIMARY KEY,
  org_id char(36) NOT NULL,
  app_id char(36) NOT NULL,
  release_id char(36) NULL,
  device_id varchar(255) NOT NULL,
  event_name varchar(255) NOT NULL,
  event_time datetime NOT NULL,
  channel_code varchar(100) NOT NULL,
  properties_jsonb json,
  KEY idx_events_org (org_id),
  KEY idx_events_app (app_id),
  KEY idx_events_time (event_time),
  KEY idx_events_release_id (release_id),
  KEY idx_events_release_channel_time (release_id, channel_code, event_time),
  CONSTRAINT fk_events_org FOREIGN KEY (org_id) REFERENCES orgs(id),
  CONSTRAINT fk_events_app FOREIGN KEY (app_id) REFERENCES apps(id),
  CONSTRAINT fk_events_release FOREIGN KEY (release_id) REFERENCES releases(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS daily_metrics (
  date date NOT NULL,
  app_id char(36) NOT NULL,
  channel_id char(36) NOT NULL,
  event_name varchar(255) NOT NULL,
  count bigint NOT NULL DEFAULT 0,
  PRIMARY KEY (date, app_id, channel_id, event_name),
  KEY idx_daily_metrics_app (app_id),
  KEY idx_daily_metrics_channel (channel_id),
  CONSTRAINT fk_daily_metrics_app FOREIGN KEY (app_id) REFERENCES apps(id),
  CONSTRAINT fk_daily_metrics_channel FOREIGN KEY (channel_id) REFERENCES channels(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS audit_logs (
  id char(36) PRIMARY KEY,
  org_id char(36) NOT NULL,
  user_id char(36) NOT NULL,
  action varchar(255) NOT NULL,
  target_type varchar(255) NOT NULL,
  target_id char(36),
  ip_address varchar(64),
  user_agent varchar(255),
  before_json json,
  after_json json,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_audit_logs_org (org_id),
  KEY idx_audit_logs_user (user_id),
  KEY idx_audit_logs_org_time (org_id, created_at),
  CONSTRAINT fk_audit_logs_org FOREIGN KEY (org_id) REFERENCES orgs(id),
  CONSTRAINT fk_audit_logs_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS system_settings (
  id char(36) PRIMARY KEY,
  setting_key varchar(100) NOT NULL,
  setting_value text NOT NULL,
  value_type varchar(32) NOT NULL DEFAULT 'string',
  description varchar(255),
  updated_by char(36),
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  UNIQUE KEY uq_system_settings_key (setting_key),
  KEY idx_system_settings_updated_by (updated_by),
  CONSTRAINT fk_system_settings_updated_by FOREIGN KEY (updated_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS email_verification_codes (
  id char(36) PRIMARY KEY,
  email varchar(255) NOT NULL,
  purpose varchar(64) NOT NULL,
  code_hash varchar(128) NOT NULL,
  expires_at datetime NOT NULL,
  used_at datetime NULL,
  request_ip varchar(64) DEFAULT '',
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_email_verification_codes_lookup (email, purpose, created_at),
  KEY idx_email_verification_codes_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS tickets (
  id char(36) PRIMARY KEY,
  org_id char(36) NOT NULL,
  created_by char(36) NOT NULL,
  title varchar(255) NOT NULL,
  description text,
  status varchar(32) NOT NULL DEFAULT 'submitted',
  assignee_type varchar(32) NOT NULL DEFAULT 'system',
  assignee_user_id char(36) NULL,
  in_progress_at datetime NULL,
  resolved_at datetime NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_tickets_org (org_id),
  KEY idx_tickets_status (status),
  KEY idx_tickets_assignee (assignee_user_id),
  KEY idx_tickets_created_at (created_at),
  CONSTRAINT fk_tickets_org FOREIGN KEY (org_id) REFERENCES orgs(id),
  CONSTRAINT fk_tickets_created_by FOREIGN KEY (created_by) REFERENCES users(id),
  CONSTRAINT fk_tickets_assignee FOREIGN KEY (assignee_user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ticket_messages (
  id char(36) PRIMARY KEY,
  ticket_id char(36) NOT NULL,
  org_id char(36) NOT NULL,
  user_id char(36) NOT NULL,
  sender_type varchar(32) NOT NULL,
  content text,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_ticket_messages_ticket (ticket_id),
  KEY idx_ticket_messages_org (org_id),
  KEY idx_ticket_messages_user (user_id),
  KEY idx_ticket_messages_created_at (created_at),
  CONSTRAINT fk_ticket_messages_ticket FOREIGN KEY (ticket_id) REFERENCES tickets(id),
  CONSTRAINT fk_ticket_messages_org FOREIGN KEY (org_id) REFERENCES orgs(id),
  CONSTRAINT fk_ticket_messages_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS feedbacks (
  id char(36) PRIMARY KEY,
  org_id char(36) NOT NULL,
  app_id char(36) NOT NULL,
  device_id varchar(255) NOT NULL,
  channel_code varchar(64),
  app_version varchar(128),
  rating int,
  content text,
  contact varchar(255),
  metadata_json json,
  status varchar(32) NOT NULL DEFAULT 'open',
  internal_note text,
  handled_by char(36) NULL,
  handled_at datetime NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_feedbacks_org (org_id),
  KEY idx_feedbacks_app (app_id),
  KEY idx_feedbacks_created_at (created_at),
  KEY idx_feedbacks_rating (rating),
  KEY idx_feedbacks_status (status),
  KEY idx_feedbacks_updated_at (updated_at),
  CONSTRAINT fk_feedbacks_org FOREIGN KEY (org_id) REFERENCES orgs(id),
  CONSTRAINT fk_feedbacks_app FOREIGN KEY (app_id) REFERENCES apps(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS attachments (
  id char(36) PRIMARY KEY,
  owner_type varchar(64) NOT NULL,
  owner_id char(36) NOT NULL,
  org_id char(36) NULL,
  file_name varchar(255) NOT NULL,
  content_type varchar(255) NOT NULL,
  size bigint NOT NULL,
  storage_driver varchar(32) NOT NULL,
  storage_path varchar(1024) NOT NULL,
  created_by char(36) NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_attachments_owner (owner_type, owner_id),
  KEY idx_attachments_org_created (org_id, created_at),
  KEY idx_attachments_storage_path (storage_path(255))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

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
