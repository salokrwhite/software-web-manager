DROP TABLE IF EXISTS daily_event_dimensions;
ALTER TABLE events DROP INDEX idx_events_app_name_time;
