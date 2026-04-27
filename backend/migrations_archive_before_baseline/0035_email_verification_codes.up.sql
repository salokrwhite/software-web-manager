CREATE TABLE IF NOT EXISTS email_verification_codes (
  id CHAR(36) PRIMARY KEY,
  email VARCHAR(255) NOT NULL,
  purpose VARCHAR(64) NOT NULL,
  code_hash VARCHAR(128) NOT NULL,
  expires_at DATETIME NOT NULL,
  used_at DATETIME NULL,
  request_ip VARCHAR(64) DEFAULT '',
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_email_verification_codes_lookup (email, purpose, created_at),
  INDEX idx_email_verification_codes_expires_at (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
