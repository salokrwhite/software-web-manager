-- Layer 1: composite index so app_started/update_failed lookups seek directly
-- to (app_id, event_name, time-range) instead of scanning all events of an app.
-- NOTE: on a large prod `events` table this ALTER can take a while; prefer a
-- low-traffic window or an online-DDL tool (pt-online-schema-change / gh-ost).
ALTER TABLE events ADD INDEX idx_events_app_name_time (app_id, event_name, event_time);

-- Layer 2: daily pre-aggregation of high-cardinality event dimensions
-- (app_started -> version, update_failed -> reason), mirroring daily_metrics.
-- Lets the version/failure analytics read a small rollup with no JSON parsing.
CREATE TABLE IF NOT EXISTS daily_event_dimensions (
  date date NOT NULL,
  app_id char(36) NOT NULL,
  event_name varchar(64) NOT NULL,
  dim_key varchar(32) NOT NULL,
  dim_value varchar(191) NOT NULL,
  count bigint NOT NULL DEFAULT 0,
  PRIMARY KEY (date, app_id, event_name, dim_key, dim_value),
  KEY idx_ded_lookup (app_id, event_name, dim_key, date),
  CONSTRAINT fk_ded_app FOREIGN KEY (app_id) REFERENCES apps(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
