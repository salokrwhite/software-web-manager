SET @has_external_download_url := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'releases' AND COLUMN_NAME = 'external_download_url'
);
SET @has_notes_url := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'releases' AND COLUMN_NAME = 'notes_url'
);
SET @sql := IF(
  @has_external_download_url = 0,
  IF(
    @has_notes_url > 0,
    'ALTER TABLE releases ADD COLUMN external_download_url VARCHAR(2048) NULL AFTER notes_url',
    'ALTER TABLE releases ADD COLUMN external_download_url VARCHAR(2048) NULL'
  ),
  'SELECT 1'
);
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
