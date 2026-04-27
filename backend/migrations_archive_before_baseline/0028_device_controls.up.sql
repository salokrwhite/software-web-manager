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
