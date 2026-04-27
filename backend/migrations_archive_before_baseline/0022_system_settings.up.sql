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
  FOREIGN KEY (updated_by) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_system_settings_updated_by ON system_settings(updated_by);
