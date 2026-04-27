SET @has_external_download_url := (
  SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'releases' AND COLUMN_NAME = 'external_download_url'
);
SET @sql := IF(@has_external_download_url > 0,
  'ALTER TABLE releases DROP COLUMN external_download_url',
  'SELECT 1');
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

