CREATE TABLE IF NOT EXISTS legacy_ticket_attachments (
  id char(36) PRIMARY KEY,
  ticket_id char(36) NOT NULL,
  file_name varchar(255) NOT NULL,
  content_type varchar(255) NOT NULL,
  size bigint NOT NULL,
  storage_driver varchar(32) NOT NULL,
  storage_path varchar(1024) NOT NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_ticket_attachments_ticket (ticket_id),
  FOREIGN KEY (ticket_id) REFERENCES tickets(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO legacy_ticket_attachments (
  id, ticket_id, file_name, content_type, size, storage_driver, storage_path, created_at
)
SELECT id, owner_id, file_name, content_type, size, storage_driver, storage_path, created_at
FROM attachments
WHERE owner_type = 'ticket';

CREATE TABLE IF NOT EXISTS legacy_ticket_message_attachments (
  id char(36) PRIMARY KEY,
  message_id char(36) NOT NULL,
  file_name varchar(255) NOT NULL,
  content_type varchar(255) NOT NULL,
  size bigint NOT NULL,
  storage_driver varchar(32) NOT NULL,
  storage_path varchar(1024) NOT NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_ticket_message_attachments_message (message_id),
  FOREIGN KEY (message_id) REFERENCES ticket_messages(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO legacy_ticket_message_attachments (
  id, message_id, file_name, content_type, size, storage_driver, storage_path, created_at
)
SELECT id, owner_id, file_name, content_type, size, storage_driver, storage_path, created_at
FROM attachments
WHERE owner_type = 'ticket_message';

CREATE TABLE IF NOT EXISTS legacy_feedback_attachments (
  id char(36) PRIMARY KEY,
  feedback_id char(36) NOT NULL,
  file_name varchar(255) NOT NULL,
  content_type varchar(255) NOT NULL,
  size bigint NOT NULL,
  storage_driver varchar(32) NOT NULL,
  storage_path varchar(1024) NOT NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_feedback_attachments_feedback (feedback_id),
  FOREIGN KEY (feedback_id) REFERENCES feedbacks(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO legacy_feedback_attachments (
  id, feedback_id, file_name, content_type, size, storage_driver, storage_path, created_at
)
SELECT id, owner_id, file_name, content_type, size, storage_driver, storage_path, created_at
FROM attachments
WHERE owner_type = 'feedback';

CREATE TABLE IF NOT EXISTS legacy_org_registration_materials (
  id char(36) NOT NULL,
  org_id char(36) NOT NULL,
  file_name varchar(255) NOT NULL,
  content_type varchar(255) NOT NULL,
  size bigint NOT NULL,
  storage_driver varchar(32) NOT NULL,
  storage_path varchar(1024) NOT NULL,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  INDEX idx_org_registration_materials_org_id (org_id),
  CONSTRAINT fk_org_registration_materials_org_id FOREIGN KEY (org_id) REFERENCES orgs(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT IGNORE INTO legacy_org_registration_materials (
  id, org_id, file_name, content_type, size, storage_driver, storage_path, created_at
)
SELECT id, owner_id, file_name, content_type, size, storage_driver, storage_path, created_at
FROM attachments
WHERE owner_type = 'org_registration_material';

CREATE TABLE IF NOT EXISTS legacy_api_keys (
  id char(36) PRIMARY KEY,
  org_id char(36) NOT NULL,
  app_id char(36) NOT NULL,
  key_hash varchar(64) NOT NULL UNIQUE,
  name varchar(255) NOT NULL,
  last_used_at datetime,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  scopes json,
  expires_at datetime,
  revoked_at datetime,
  FOREIGN KEY (org_id) REFERENCES orgs(id),
  FOREIGN KEY (app_id) REFERENCES apps(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
