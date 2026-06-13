package jobs

import (
	"context"
	"time"

	"gorm.io/gorm"
)

const defaultInterval = time.Hour

type Logger interface {
	Printf(format string, v ...interface{})
}

func Start(ctx context.Context, conn *gorm.DB, interval time.Duration, logger Logger) {
	if conn == nil || logger == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if interval < time.Minute {
		interval = defaultInterval
	}

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Printf("metrics aggregator panic recovered: %v", r)
			}
		}()

		RunOnce(conn, time.Now(), logger)

		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				logger.Printf("metrics aggregator stopped")
				return
			case now := <-ticker.C:
				RunOnce(conn, now, logger)
			}
		}
	}()
}

func RunOnce(conn *gorm.DB, now time.Time, logger Logger) {
	if conn == nil || logger == nil {
		return
	}
	end := now
	start := end.Truncate(24 * time.Hour).AddDate(0, 0, -1)
	logger.Printf("aggregate metrics: start=%s end=%s", start.Format(time.RFC3339), end.Format(time.RFC3339))
	if err := aggregateRange(conn, start, end); err != nil {
		logger.Printf("aggregate failed: %v", err)
	}
}

// dimensionRollups defines high-cardinality event dimensions that are
// pre-aggregated daily into daily_event_dimensions so the version/failure
// analytics endpoints never have to scan events + parse JSON at request time.
var dimensionRollups = []struct {
	eventName string
	jsonPath  string
	dimKey    string
}{
	{eventName: "app_started", jsonPath: "$.version", dimKey: "version"},
	{eventName: "update_failed", jsonPath: "$.reason", dimKey: "reason"},
}

func AggregateAppRange(conn *gorm.DB, appID string, start, end time.Time) (int64, error) {
	if conn == nil {
		return 0, nil
	}
	result := conn.Exec(`
		INSERT INTO daily_metrics (date, app_id, channel_id, event_name, count)
		SELECT DATE(e.event_time) as metric_date, e.app_id, c.id as channel_id, e.event_name, COUNT(1) as count
		FROM events e
		JOIN channels c ON c.app_id = e.app_id AND c.code = e.channel_code
		WHERE e.app_id = ? AND e.event_time >= ? AND e.event_time < ?
		GROUP BY metric_date, e.app_id, c.id, e.event_name
		ON DUPLICATE KEY UPDATE count = VALUES(count)
	`, appID, start, end)
	if result.Error != nil {
		return result.RowsAffected, result.Error
	}
	if err := aggregateDimensions(conn, start, end, appID); err != nil {
		return result.RowsAffected, err
	}
	return result.RowsAffected, nil
}

func aggregateRange(conn *gorm.DB, start, end time.Time) error {
	if err := conn.Exec(`
		INSERT INTO daily_metrics (date, app_id, channel_id, event_name, count)
		SELECT DATE(e.event_time) as metric_date, e.app_id, c.id as channel_id, e.event_name, COUNT(1) as count
		FROM events e
		JOIN channels c ON c.app_id = e.app_id AND c.code = e.channel_code
		WHERE e.event_time >= ? AND e.event_time < ?
		GROUP BY metric_date, e.app_id, c.id, e.event_name
		ON DUPLICATE KEY UPDATE count = VALUES(count)
	`, start, end).Error; err != nil {
		return err
	}
	return aggregateDimensions(conn, start, end, "")
}

// aggregateDimensions rolls up the configured event dimensions for [start, end).
// If appID is non-empty the rollup is scoped to that app, otherwise all apps.
func aggregateDimensions(conn *gorm.DB, start, end time.Time, appID string) error {
	for _, d := range dimensionRollups {
		sql := `
			INSERT INTO daily_event_dimensions (date, app_id, event_name, dim_key, dim_value, count)
			SELECT DATE(e.event_time) as metric_date, e.app_id, ?, ?,
			       LEFT(COALESCE(JSON_UNQUOTE(JSON_EXTRACT(e.properties_jsonb, ?)), 'unknown'), 191) as dim_value,
			       COUNT(1) as count
			FROM events e
			WHERE e.event_time >= ? AND e.event_time < ? AND e.event_name = ?`
		args := []interface{}{d.eventName, d.dimKey, d.jsonPath, start, end, d.eventName}
		if appID != "" {
			sql += " AND e.app_id = ?"
			args = append(args, appID)
		}
		sql += `
			GROUP BY metric_date, e.app_id, dim_value
			ON DUPLICATE KEY UPDATE count = VALUES(count)`
		if err := conn.Exec(sql, args...).Error; err != nil {
			return err
		}
	}
	return nil
}
