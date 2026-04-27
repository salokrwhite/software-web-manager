CREATE TABLE IF NOT EXISTS tickets (
  id char(36) PRIMARY KEY,
  org_id char(36) NOT NULL,
  created_by char(36) NOT NULL,
  title varchar(255) NOT NULL,
  description text,
  status varchar(32) NOT NULL DEFAULT 'submitted',
  assignee_type varchar(32) NOT NULL DEFAULT 'system',
  assignee_user_id char(36),
  in_progress_at datetime,
  resolved_at datetime,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  KEY idx_tickets_org (org_id),
  KEY idx_tickets_status (status),
  KEY idx_tickets_assignee (assignee_user_id),
  KEY idx_tickets_created_at (created_at),
  FOREIGN KEY (org_id) REFERENCES orgs(id),
  FOREIGN KEY (created_by) REFERENCES users(id),
  FOREIGN KEY (assignee_user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS ticket_attachments (
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
