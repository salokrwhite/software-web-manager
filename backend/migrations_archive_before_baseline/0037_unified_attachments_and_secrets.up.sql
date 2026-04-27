CREATE TABLE IF NOT EXISTS attachments (
  id char(36) NOT NULL,
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
  PRIMARY KEY (id),
  KEY idx_attachments_owner (owner_type, owner_id),
  KEY idx_attachments_org_created (org_id, created_at),
  KEY idx_attachments_storage_path (storage_path(255))
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO attachments (
  id, owner_type, owner_id, org_id, file_name, content_type, size,
  storage_driver, storage_path, created_by, created_at
)
SELECT
  ta.id, 'ticket', ta.ticket_id, t.org_id, ta.file_name, ta.content_type, ta.size,
  ta.storage_driver, ta.storage_path, t.created_by, ta.created_at
FROM ticket_attachments ta
JOIN tickets t ON t.id = ta.ticket_id;

INSERT IGNORE INTO attachments (
  id, owner_type, owner_id, org_id, file_name, content_type, size,
  storage_driver, storage_path, created_by, created_at
)
SELECT
  tma.id, 'ticket_message', tma.message_id, tm.org_id, tma.file_name, tma.content_type, tma.size,
  tma.storage_driver, tma.storage_path, tm.user_id, tma.created_at
FROM ticket_message_attachments tma
JOIN ticket_messages tm ON tm.id = tma.message_id;

INSERT IGNORE INTO attachments (
  id, owner_type, owner_id, org_id, file_name, content_type, size,
  storage_driver, storage_path, created_by, created_at
)
SELECT
  fa.id, 'feedback', fa.feedback_id, f.org_id, fa.file_name, fa.content_type, fa.size,
  fa.storage_driver, fa.storage_path, NULL, fa.created_at
FROM feedback_attachments fa
JOIN feedbacks f ON f.id = fa.feedback_id;

INSERT IGNORE INTO attachments (
  id, owner_type, owner_id, org_id, file_name, content_type, size,
  storage_driver, storage_path, created_by, created_at
)
SELECT
  orm.id, 'org_registration_material', orm.org_id, orm.org_id, orm.file_name, orm.content_type, orm.size,
  orm.storage_driver, orm.storage_path, NULL, orm.created_at
FROM org_registration_materials orm;

SET @legacy_attachment_count := (
  (SELECT COUNT(*) FROM ticket_attachments) +
  (SELECT COUNT(*) FROM ticket_message_attachments) +
  (SELECT COUNT(*) FROM feedback_attachments) +
  (SELECT COUNT(*) FROM org_registration_materials)
);

SET @unified_attachment_count := (
  SELECT COUNT(*) FROM attachments
  WHERE owner_type IN ('ticket', 'ticket_message', 'feedback', 'org_registration_material')
);

SET @validation_sql := IF(
  @legacy_attachment_count = @unified_attachment_count,
  'SELECT 1',
  'SIGNAL SQLSTATE ''45000'' SET MESSAGE_TEXT = ''attachments migration count mismatch'''
);
PREPARE validation_stmt FROM @validation_sql;
EXECUTE validation_stmt;
DEALLOCATE PREPARE validation_stmt;

RENAME TABLE ticket_attachments TO legacy_ticket_attachments;
RENAME TABLE ticket_message_attachments TO legacy_ticket_message_attachments;
RENAME TABLE feedback_attachments TO legacy_feedback_attachments;
RENAME TABLE org_registration_materials TO legacy_org_registration_materials;
RENAME TABLE api_keys TO legacy_api_keys;
