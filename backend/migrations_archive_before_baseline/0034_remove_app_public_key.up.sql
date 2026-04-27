SET @has_public_key := (
  SELECT COUNT(*)
  FROM INFORMATION_SCHEMA.COLUMNS
  WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'apps'
    AND COLUMN_NAME = 'public_key'
);
SET @ddl := IF(
  @has_public_key = 0,
  'ALTER TABLE apps ADD COLUMN public_key text AFTER description',
  'SELECT ''apps.public_key already exists, skip add'''
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
