-- Add release_id to events for release-level metrics
SET @col_exists := (
  SELECT COUNT(1)
  FROM INFORMATION_SCHEMA.COLUMNS
  WHERE table_schema = DATABASE()
    AND table_name = 'events'
    AND column_name = 'release_id'
);
SET @sql := IF(@col_exists = 0,
  'ALTER TABLE events ADD COLUMN release_id char(36) NULL',
  'SELECT 1'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

SET @idx_exists := (
  SELECT COUNT(1)
  FROM INFORMATION_SCHEMA.STATISTICS
  WHERE table_schema = DATABASE()
    AND table_name = 'events'
    AND index_name = 'idx_events_release_channel_time'
);
SET @sql := IF(@idx_exists = 0,
  'CREATE INDEX idx_events_release_channel_time ON events (release_id, channel_code(64), event_time)',
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
SET @sql := IF(@idx_exists = 0,
  'CREATE INDEX idx_events_release_id ON events (release_id)',
  'SELECT 1'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;

-- Clean up duplicate release_channel rows before enforcing unique constraint
DELETE rc1 FROM release_channels rc1
JOIN release_channels rc2
  ON rc1.release_id = rc2.release_id
 AND rc1.channel_id = rc2.channel_id
 AND (
   IFNULL(rc1.published_at, '1970-01-01') < IFNULL(rc2.published_at, '1970-01-01')
   OR (
     IFNULL(rc1.published_at, '1970-01-01') = IFNULL(rc2.published_at, '1970-01-01')
     AND rc1.id > rc2.id
   )
 );

-- Ensure unique constraint exists
SET @idx_exists := (
  SELECT COUNT(1)
  FROM INFORMATION_SCHEMA.STATISTICS
  WHERE table_schema = DATABASE()
    AND table_name = 'release_channels'
    AND index_name = 'uq_release_channels_release_channel'
);
SET @sql := IF(@idx_exists = 0,
  'ALTER TABLE release_channels ADD UNIQUE KEY uq_release_channels_release_channel (release_id, channel_id)',
  'SELECT 1'
);
PREPARE stmt FROM @sql; EXECUTE stmt; DEALLOCATE PREPARE stmt;
