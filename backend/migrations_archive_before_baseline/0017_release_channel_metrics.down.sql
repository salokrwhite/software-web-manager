-- Drop indexes and column added for release metrics
SET @idx_exists := (
  SELECT COUNT(1)
  FROM INFORMATION_SCHEMA.STATISTICS
  WHERE table_schema = DATABASE()
    AND table_name = 'events'
    AND index_name = 'idx_events_release_channel_time'
);
SET @sql := IF(@idx_exists > 0,
  'DROP INDEX idx_events_release_channel_time ON events',
  'SELECT 1'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @idx_exists := (
  SELECT COUNT(1)
  FROM INFORMATION_SCHEMA.STATISTICS
  WHERE table_schema = DATABASE()
    AND table_name = 'events'
    AND index_name = 'idx_events_release_id'
);
SET @sql := IF(@idx_exists > 0,
  'DROP INDEX idx_events_release_id ON events',
  'SELECT 1'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @col_exists := (
  SELECT COUNT(1)
  FROM INFORMATION_SCHEMA.COLUMNS
  WHERE table_schema = DATABASE()
    AND table_name = 'events'
    AND column_name = 'release_id'
);
SET @sql := IF(@col_exists > 0,
  'ALTER TABLE events DROP COLUMN release_id',
  'SELECT 1'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
