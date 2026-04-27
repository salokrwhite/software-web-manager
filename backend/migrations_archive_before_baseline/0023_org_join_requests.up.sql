CREATE TABLE IF NOT EXISTS org_join_requests (
  id CHAR(36) PRIMARY KEY,
  org_id CHAR(36) NOT NULL,
  user_id CHAR(36) NOT NULL,
  reason TEXT NULL,
  status VARCHAR(32) NOT NULL DEFAULT 'pending',
  review_reason TEXT NULL,
  reviewed_by CHAR(36) NULL,
  reviewed_at DATETIME NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_org_join_requests_org_id (org_id),
  INDEX idx_org_join_requests_user_id (user_id),
  INDEX idx_org_join_requests_status (status),
  INDEX idx_org_join_requests_reviewed_by (reviewed_by)
);
