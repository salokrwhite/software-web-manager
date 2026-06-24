package channel

import (
	"sort"
	"time"

	"github.com/google/uuid"
)

// MetricsResult carries the funnel summary and per-day timeline for a release
// channel.
type MetricsResult struct {
	ReleaseID        string
	ChannelCode      string
	Summary          map[string]int64
	Timeline         []map[string]interface{}
	HasReleaseEvents bool
}

// Metrics computes the update funnel summary and timeline for a release channel
// over the given date range. It returns ErrReleaseChannelNotFound when the
// channel does not belong to the app.
func (s *Service) Metrics(appID, relChannelID string, from, to time.Time) (*MetricsResult, error) {
	var row struct {
		ReleaseID   uuid.UUID
		ChannelCode string
	}
	if err := s.DB.Raw(`
		SELECT rc.release_id, c.code as channel_code
		FROM release_channels rc
		JOIN channels c ON c.id = rc.channel_id
		WHERE rc.id = ? AND c.app_id = ?
	`, relChannelID, appID).Scan(&row).Error; err != nil || row.ReleaseID == (uuid.UUID{}) {
		return nil, ErrReleaseChannelNotFound
	}

	funnelEvents := []string{"check_update", "update_available", "download_started", "download_completed", "install_completed", "update_failed", "app_started"}
	summaryRows := []struct {
		EventName string
		Count     int64
	}{}
	releaseID := row.ReleaseID.String()
	if err := s.DB.Raw(`
		SELECT event_name, COUNT(1) as count
		FROM events
		WHERE app_id = ? AND channel_code = ? AND release_id = ? AND event_time >= ? AND event_time <= ? AND event_name IN ?
		GROUP BY event_name
	`, appID, row.ChannelCode, releaseID, from, to, funnelEvents).Scan(&summaryRows).Error; err != nil {
		return nil, err
	}
	summary := map[string]int64{}
	var total int64
	for _, name := range funnelEvents {
		summary[name] = 0
	}
	for _, r := range summaryRows {
		summary[r.EventName] = r.Count
		total += r.Count
	}

	timelineRows := []struct {
		Date      time.Time
		EventName string
		Count     int64
	}{}
	if err := s.DB.Raw(`
		SELECT DATE(event_time) as date, event_name, COUNT(1) as count
		FROM events
		WHERE app_id = ? AND channel_code = ? AND release_id = ? AND event_time >= ? AND event_time <= ? AND event_name IN ?
		GROUP BY DATE(event_time), event_name
		ORDER BY date ASC
	`, appID, row.ChannelCode, releaseID, from, to, funnelEvents).Scan(&timelineRows).Error; err != nil {
		return nil, err
	}
	dateMap := map[string]map[string]int64{}
	for _, tr := range timelineRows {
		dateKey := tr.Date.Format("2006-01-02")
		if _, ok := dateMap[dateKey]; !ok {
			dateMap[dateKey] = map[string]int64{}
		}
		dateMap[dateKey][tr.EventName] = tr.Count
	}
	dates := make([]string, 0, len(dateMap))
	for d := range dateMap {
		dates = append(dates, d)
	}
	sort.Strings(dates)
	timeline := make([]map[string]interface{}, 0, len(dates))
	for _, d := range dates {
		item := map[string]interface{}{"date": d}
		for _, name := range funnelEvents {
			item[name] = dateMap[d][name]
		}
		timeline = append(timeline, item)
	}

	return &MetricsResult{
		ReleaseID:        releaseID,
		ChannelCode:      row.ChannelCode,
		Summary:          summary,
		Timeline:         timeline,
		HasReleaseEvents: total > 0,
	}, nil
}
