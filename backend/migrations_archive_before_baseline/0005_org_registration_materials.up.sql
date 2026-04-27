CREATE TABLE IF NOT EXISTS org_registration_materials (
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
