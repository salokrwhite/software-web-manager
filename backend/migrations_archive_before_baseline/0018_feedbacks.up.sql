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
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  KEY idx_feedbacks_org (org_id),
  KEY idx_feedbacks_app (app_id),
  KEY idx_feedbacks_created_at (created_at),
  KEY idx_feedbacks_rating (rating),
  FOREIGN KEY (org_id) REFERENCES orgs(id),
  FOREIGN KEY (app_id) REFERENCES apps(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS feedback_attachments (
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
