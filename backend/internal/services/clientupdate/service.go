package clientupdate

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"software-web-manager/backend/internal/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var errScheduledAlreadyActivated = errors.New("scheduled release already activated")

const (
	MaintenanceEventScheduled = "maintenance_scheduled"
	MaintenanceEventCancelled = "maintenance_cancelled"
)

// Service publishes release/maintenance client-update events over the hub and
// runs the scheduled-release activation watcher. It owns DB + hub access.
type Service struct {
	DB  *gorm.DB
	Hub *Hub
}

// NewService builds a clientupdate engine over a database handle and hub.
func NewService(db *gorm.DB, hub *Hub) *Service {
	return &Service{DB: db, Hub: hub}
}

// WithinRolloutWindow reports whether the current time falls inside the optional
// [start, end] rollout window. Shared by the activation engine and the client
// update-check selection logic.
func WithinRolloutWindow(start, end *time.Time) bool {
	now := time.Now()
	if start != nil && now.Before(*start) {
		return false
	}
	if end != nil && now.After(*end) {
		return false
	}
	return true
}

func (s *Service) EmitMaintenance(app models.App, eventType string) {
	if s == nil || s.Hub == nil {
		return
	}
	evt := Event{
		ID:          uuid.NewString(),
		EventType:   eventType,
		OrgID:       app.OrgID.String(),
		AppID:       app.ID.String(),
		ChannelCode: "",
		Platform:    "universal",
		Arch:        "universal",
		PublishedAt: time.Now(),
		Reason:      "maintenance",
		Message:     strings.TrimSpace(app.MaintenanceMessage),
	}
	if eventType == MaintenanceEventScheduled && app.MaintenanceStartAt != nil {
		startCopy := app.MaintenanceStartAt.UTC()
		evt.MaintenanceStartAt = &startCopy
	}
	s.Hub.Publish(evt)
}

func (s *Service) EmitReleaseClientUpdate(eventType, reason string, appID uuid.UUID, releaseID uuid.UUID, channelCode string, publishedAt time.Time) {
	if s == nil || s.Hub == nil || s.DB == nil {
		return
	}
	var app models.App
	if err := s.DB.Where("id = ?", appID).First(&app).Error; err != nil {
		return
	}
	channelCode = strings.ToLower(strings.TrimSpace(channelCode))
	if channelCode == "" {
		var channel models.Channel
		if err := s.DB.Where("id IN (SELECT channel_id FROM release_channels WHERE release_id = ? LIMIT 1)", releaseID).First(&channel).Error; err == nil {
			channelCode = strings.ToLower(strings.TrimSpace(channel.Code))
		}
	}
	if channelCode == "" {
		channelCode = "stable"
	}

	type pair struct {
		Platform string `gorm:"column:platform"`
		Arch     string `gorm:"column:arch"`
	}
	var pairs []pair
	_ = s.DB.Model(&models.Artifact{}).Select("DISTINCT platform, arch").Where("release_id = ?", releaseID).Scan(&pairs).Error
	if len(pairs) == 0 {
		pairs = append(pairs, pair{Platform: "universal", Arch: "universal"})
	}

	for _, p := range pairs {
		platform := strings.ToLower(strings.TrimSpace(p.Platform))
		arch := strings.ToLower(strings.TrimSpace(p.Arch))
		if platform == "" {
			platform = "universal"
		}
		if arch == "" {
			arch = "universal"
		}
		s.Hub.Publish(Event{
			ID:          uuid.NewString(),
			EventType:   eventType,
			OrgID:       app.OrgID.String(),
			AppID:       app.ID.String(),
			ChannelCode: channelCode,
			Platform:    platform,
			Arch:        arch,
			ReleaseID:   releaseID.String(),
			PublishedAt: publishedAt,
			Reason:      reason,
		})
	}
}

func (s *Service) ShouldEmitImmediateReleaseChannel(rc models.ReleaseChannel) bool {
	if strings.ToLower(strings.TrimSpace(rc.Status)) != "active" {
		return false
	}
	if rc.Paused {
		return false
	}
	return WithinRolloutWindow(rc.RolloutStartAt, rc.RolloutEndAt)
}

type activationWatcherLogger interface {
	Printf(format string, v ...interface{})
}

func (s *Service) StartReleaseActivationWatcher(ctx context.Context, interval time.Duration, logger activationWatcherLogger) {
	if s == nil || s.DB == nil || s.Hub == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if interval < 10*time.Second {
		interval = 30 * time.Second
	}
	if logger == nil {
		logger = log.Default()
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		prev := map[string]struct{}{}

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				now := time.Now()
				if err := s.activateScheduledReleaseChannels(now, logger); err != nil {
					logger.Printf("release activation watcher schedule failed: %v", err)
				}
				rows, err := s.loadEffectiveReleaseChannels(now)
				if err != nil {
					logger.Printf("release activation watcher failed: %v", err)
					continue
				}
				current := make(map[string]struct{}, len(rows))
				for _, row := range rows {
					current[row.ID] = struct{}{}
					if _, existed := prev[row.ID]; existed {
						continue
					}
					s.EmitReleaseClientUpdate(
						"release_activated",
						"schedule_activation",
						row.AppID,
						row.ReleaseID,
						row.ChannelCode,
						now,
					)
				}
				prev = current
			}
		}
	}()
}

type scheduledReleaseChannelRow struct {
	ID          uuid.UUID  `gorm:"column:id"`
	AppID       uuid.UUID  `gorm:"column:app_id"`
	ReleaseID   uuid.UUID  `gorm:"column:release_id"`
	ChannelID   uuid.UUID  `gorm:"column:channel_id"`
	ChannelCode string     `gorm:"column:channel_code"`
	ActivateAt  *time.Time `gorm:"column:activate_at"`
}

func (s *Service) activateScheduledReleaseChannels(now time.Time, logger activationWatcherLogger) error {
	if s == nil || s.DB == nil {
		return nil
	}
	var rows []scheduledReleaseChannelRow
	if err := s.DB.Raw(`
		SELECT rc.id, r.app_id, rc.release_id, rc.channel_id, c.code AS channel_code, rc.published_at AS activate_at
		FROM release_channels rc
		JOIN releases r ON r.id = rc.release_id
		JOIN channels c ON c.id = rc.channel_id
		WHERE rc.status = 'scheduled'
		  AND rc.paused = 0
		  AND rc.published_at IS NOT NULL
		  AND rc.published_at <= ?
		  AND r.status IN ('approved', 'published')
		ORDER BY rc.published_at DESC
	`, now).Scan(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return nil
	}

	// Only keep the newest due plan for each channel to avoid flip-flop when multiple plans are queued.
	selectedByChannel := map[uuid.UUID]scheduledReleaseChannelRow{}
	for _, row := range rows {
		existing, ok := selectedByChannel[row.ChannelID]
		if !ok {
			selectedByChannel[row.ChannelID] = row
			continue
		}
		if row.ActivateAt != nil && (existing.ActivateAt == nil || row.ActivateAt.After(*existing.ActivateAt)) {
			selectedByChannel[row.ChannelID] = row
		}
	}

	for _, row := range selectedByChannel {
		current := row
		if err := s.DB.Transaction(func(tx *gorm.DB) error {
			update := tx.Model(&models.ReleaseChannel{}).Where("id = ? AND status = ?", current.ID, "scheduled").Updates(map[string]interface{}{
				"status":       "active",
				"published_at": now,
			})
			if update.Error != nil {
				return update.Error
			}
			if update.RowsAffected == 0 {
				return errScheduledAlreadyActivated
			}
			if err := tx.Model(&models.ReleaseChannel{}).Where("channel_id = ? AND id <> ?", current.ChannelID, current.ID).
				Update("status", "inactive").Error; err != nil {
				return err
			}
			return tx.Model(&models.Release{}).Where("id = ?", current.ReleaseID).Updates(map[string]interface{}{
				"status":       "published",
				"published_at": now,
			}).Error
		}); err != nil {
			if errors.Is(err, errScheduledAlreadyActivated) {
				continue
			}
			if logger != nil {
				logger.Printf("activate scheduled release failed, channel=%s release=%s: %v", current.ChannelID.String(), current.ReleaseID.String(), err)
			}
			continue
		}
		s.EmitReleaseClientUpdate(
			"release_published",
			"scheduled_publish",
			current.AppID,
			current.ReleaseID,
			current.ChannelCode,
			now,
		)
	}
	return nil
}

type effectiveReleaseChannelRow struct {
	ID          string    `gorm:"column:id"`
	AppID       uuid.UUID `gorm:"column:app_id"`
	ReleaseID   uuid.UUID `gorm:"column:release_id"`
	ChannelCode string    `gorm:"column:channel_code"`
}

func (s *Service) loadEffectiveReleaseChannels(now time.Time) ([]effectiveReleaseChannelRow, error) {
	if s == nil || s.DB == nil {
		return nil, nil
	}
	var rows []effectiveReleaseChannelRow
	err := s.DB.Raw(`
		SELECT rc.id, r.app_id, rc.release_id, c.code AS channel_code
		FROM release_channels rc
		JOIN releases r ON r.id = rc.release_id
		JOIN channels c ON c.id = rc.channel_id
		WHERE rc.status = 'active'
		  AND rc.paused = 0
		  AND r.status = 'published'
		  AND (rc.rollout_start_at IS NULL OR rc.rollout_start_at <= ?)
		  AND (rc.rollout_end_at IS NULL OR rc.rollout_end_at >= ?)
	`, now, now).Scan(&rows).Error
	return rows, err
}
