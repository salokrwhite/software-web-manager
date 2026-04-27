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
  KEY idx_ticket_messages_created_at (created_at),
  FOREIGN KEY (ticket_id) REFERENCES tickets(id),
  FOREIGN KEY (org_id) REFERENCES orgs(id),
  FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ticket_message_attachments (
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
