SET @has_public_key := (
  SELECT COUNT(*)
  FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'apps'
    AND COLUMN_NAME = 'public_key'
);
SET @ddl := IF(
  @has_public_key > 0,
  'ALTER TABLE apps DROP COLUMN public_key',
  'SELECT ''apps.public_key not found, skip drop'''
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
