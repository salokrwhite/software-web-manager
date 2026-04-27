CREATE TABLE IF NOT EXISTS release_templates (
  id char(36) PRIMARY KEY,
  org_id char(36) NOT NULL,
  name varchar(255) NOT NULL,
  schedule_at datetime NULL,
  window_start datetime NULL,
  window_end datetime NULL,
  emergency boolean NOT NULL DEFAULT false,
  created_at datetime NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY (org_id) REFERENCES orgs(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_release_templates_org ON release_templates(org_id);

ALTER TABLE releases ADD COLUMN release_template_id char(36) NULL;
CREATE INDEX idx_releases_template ON releases(release_template_id);
