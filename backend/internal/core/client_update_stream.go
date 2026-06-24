package core

import (
	"context"
	"errors"
	"log"
	"strings"
	"time"

	"software-web-manager/backend/internal/models"
	"software-web-manager/backend/internal/services/clientupdate"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var errScheduledAlreadyActivated = errors.New("scheduled release already activated")

const (
	MaintenanceEventScheduled = "maintenance_scheduled"
	MaintenanceEventCancelled = "maintenance_cancelled"
)

// WithinRolloutWindow reports whether the current time falls inside the optional
// [start, end] rollout window. It is shared by the release activation engine and
// the client update-check selection logic.
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

func (h *Handler) EmitMaintenance(app models.App, eventType string) {
	if h == nil || h.ClientUpdateHub == nil {
		return
	}
	evt := clientupdate.Event{
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
	h.ClientUpdateHub.Publish(evt)
}

func (h *Handler) EmitReleaseClientUpdate(eventType, reason string, appID uuid.UUID, releaseID uuid.UUID, channelCode string, publishedAt time.Time) {
	if h == nil || h.ClientUpdateHub == nil || h.DB == nil {
		return
	}
	var app models.App
	if err := h.DB.Where("id = ?", appID).First(&app).Error; err != nil {
		return
	}
	channelCode = strings.ToLower(strings.TrimSpace(channelCode))
	if channelCode == "" {
		var channel models.Channel
		if err := h.DB.Where("id IN (SELECT channel_id FROM release_channels WHERE release_id = ? LIMIT 1)", releaseID).First(&channel).Error; err == nil {
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
	_ = h.DB.Model(&models.Artifact{}).Select("DISTINCT platform, arch").Where("release_id = ?", releaseID).Scan(&pairs).Error
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
		h.ClientUpdateHub.Publish(clientupdate.Event{
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

func (h *Handler) ShouldEmitImmediateReleaseChannel(rc models.ReleaseChannel) bool {
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

func (h *Handler) StartReleaseActivationWatcher(ctx context.Context, interval time.Duration, logger activationWatcherLogger) {
	if h == nil || h.DB == nil || h.ClientUpdateHub == nil {
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
				if err := h.activateScheduledReleaseChannels(now, logger); err != nil {
					logger.Printf("release activation watcher schedule failed: %v", err)
				}
				rows, err := h.loadEffectiveReleaseChannels(now)
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
					h.EmitReleaseClientUpdate(
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

func (h *Handler) activateScheduledReleaseChannels(now time.Time, logger activationWatcherLogger) error {
	if h == nil || h.DB == nil {
		return nil
	}
	var rows []scheduledReleaseChannelRow
	if err := h.DB.Raw(`
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
		if err := h.DB.Transaction(func(tx *gorm.DB) error {
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
		h.EmitReleaseClientUpdate(
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

func (h *Handler) loadEffectiveReleaseChannels(now time.Time) ([]effectiveReleaseChannelRow, error) {
	if h == nil || h.DB == nil {
		return nil, nil
	}
	var rows []effectiveReleaseChannelRow
	err := h.DB.Raw(`
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
